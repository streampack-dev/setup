// Package setup is what the tool does: ask what starting a server needs and write .env and the
// compose file, bring the server up, open a console on it, and move the credentials given at that
// console into .env. It knows nothing about particular integrations: those are set up at the
// console, with their own commands, and the server says which variables they need.
package setup

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/streampack-dev/setup/internal/compose"
	"github.com/streampack-dev/setup/internal/docker"
	"github.com/streampack-dev/setup/internal/envfile"
	"github.com/streampack-dev/setup/internal/prompt"
	"github.com/streampack-dev/setup/internal/server"
)

// Setup is one deployment directory and the things the flows use.
type Setup struct {
	Dir    string
	Ask    prompt.Asker
	Docker docker.Compose
	Server *server.Client
	Out    io.Writer
	// Latest finds the newest server release, for the default version.
	Latest func(ctx context.Context) (string, error)
	// Random returns n random bytes; crypto/rand when nil.
	Random func(n int) []byte
	// ReadyTimeout is how long a start is waited for; five minutes when zero.
	ReadyTimeout time.Duration
	// Poll is how often readiness is checked; two seconds when zero.
	Poll time.Duration
}

func (s *Setup) envPath() string { return filepath.Join(s.Dir, ".env") }

// Configured says whether the directory already has a .env with what starting needs.
func (s *Setup) Configured() (bool, error) {
	env, err := envfile.Read(s.envPath())
	if err != nil {
		return false, err
	}
	for _, key := range []string{"STREAMPACK_VERSION", "DB_PASSWORD", "JWT_SECRET"} {
		if value, ok := env.Get(key); !ok || value == "" {
			return false, nil
		}
	}
	return true, nil
}

// adapter is an integration that can be turned on at the first run. Its credentials are given at
// the console later, except for the few that must be in the environment from the start.
type adapter struct {
	name, variable string
	on             bool
	// keys the adapter needs in .env before it starts: generated, or asked for
	generated []string
	asked     []string
}

var adapters = []adapter{
	{name: "Mattermost", variable: "MATTERMOST_ENABLED", on: true},
	{name: "GitLab", variable: "GITLAB_ENABLED", on: true, generated: []string{"GITLAB_WEBHOOK_SECRET_KEY"}},
	{name: "IRC", variable: "IRC_ENABLED"},
	{name: "Slack", variable: "SLACK_ENABLED"},
	{name: "Discord", variable: "DISCORD_ENABLED", asked: []string{"DISCORD_APPLICATION_ID", "DISCORD_PUBLIC_KEY", "DISCORD_BOT_TOKEN"}},
}

// Init asks what starting the server needs and writes .env and compose.yaml. An existing .env is
// kept: only what it lacks is asked for.
func (s *Setup) Init(ctx context.Context) error {
	env, err := envfile.Read(s.envPath())
	if err != nil {
		return err
	}
	fmt.Fprintln(s.Out, "Setting up a server-streampack deployment in "+s.Dir)

	version, _ := env.Get("STREAMPACK_VERSION")
	if version == "" {
		def := ""
		if s.Latest != nil {
			if latest, err := s.Latest(ctx); err == nil {
				def = latest
			} else {
				fmt.Fprintf(s.Out, "(Couldn't find the newest release: %v)\n", err)
			}
		}
		if version, err = s.required("Server version", def, nil); err != nil {
			return err
		}
		env.Set("STREAMPACK_VERSION", version)
	}
	if err := s.askInto(env, "ADMIN_EMAIL", "The super-admin's email address", "", validEmail); err != nil {
		return err
	}
	if err := s.askInto(env, "BASE_URL", "The API's public address", "http://localhost:8080", validURL); err != nil {
		return err
	}
	if err := s.askInto(env, "BLOG_BASE_URL", "The site's public address", "http://localhost:3000", validURL); err != nil {
		return err
	}

	if _, ok := env.Get("DB_PASSWORD"); !ok {
		env.Set("DB_PASSWORD", s.base64(24))
	}
	if _, ok := env.Get("JWT_SECRET"); !ok {
		env.Set("JWT_SECRET", s.base64(48))
	}
	// GitHub watching is always there; its webhooks need this key.
	if _, ok := env.Get("GITHUB_WEBHOOK_SECRET_KEY"); !ok {
		env.Set("GITHUB_WEBHOOK_SECRET_KEY", s.hex(32))
	}

	for _, a := range adapters {
		if _, ok := env.Get(a.variable); ok {
			continue
		}
		on, err := s.Ask.Confirm("Turn on "+a.name+"?", a.on)
		if err != nil {
			return err
		}
		env.Set(a.variable, fmt.Sprint(on))
		if !on {
			continue
		}
		for _, key := range a.generated {
			if _, ok := env.Get(key); !ok {
				env.Set(key, s.hex(32))
			}
		}
		for _, key := range a.asked {
			if _, ok := env.Get(key); ok {
				continue
			}
			value, err := s.Ask.Secret(key)
			if err != nil {
				return err
			}
			env.Set(key, value)
		}
	}

	if _, ok := env.Get("MAIL_ENABLED"); !ok {
		on, err := s.Ask.Confirm("Send mail (sign-in codes) through an SMTP server?", false)
		if err != nil {
			return err
		}
		env.Set("MAIL_ENABLED", fmt.Sprint(on))
		if on {
			for _, q := range []struct{ key, question, def string }{
				{"MAIL_HOST", "SMTP host", "host.docker.internal"},
				{"MAIL_PORT", "SMTP port", "25"},
				{"MAIL_FROM", "Sender address", ""},
			} {
				if err := s.askInto(env, q.key, q.question, q.def, nil); err != nil {
					return err
				}
			}
			user, err := s.Ask.Ask("SMTP username (blank for none)", "")
			if err != nil {
				return err
			}
			if user != "" {
				password, err := s.Ask.Secret("SMTP password")
				if err != nil {
					return err
				}
				env.Set("MAIL_USERNAME", user)
				env.Set("MAIL_PASSWORD", password)
				env.Set("MAIL_AUTH", "true")
				env.Set("MAIL_STARTTLS", "true")
			}
		}
	}

	if err := os.MkdirAll(filepath.Join(s.Dir, "generative"), 0o755); err != nil {
		return err
	}
	if wrote, err := compose.Write(s.Dir); err != nil {
		return err
	} else if wrote {
		fmt.Fprintln(s.Out, "Wrote compose.yaml")
	}
	if err := env.Write(s.envPath()); err != nil {
		return err
	}
	fmt.Fprintln(s.Out, "Wrote .env (readable by you alone)")
	return nil
}

// Up pulls the images and starts the services, then waits until the server answers. A start the
// server's secrets check stops is offered the missing variables, and tried again.
func (s *Setup) Up(ctx context.Context) error {
	if err := s.Docker.Pull(ctx); err != nil {
		return err
	}
	for attempt := 0; ; attempt++ {
		if err := s.Docker.Up(ctx); err != nil {
			return err
		}
		err := s.waitReady(ctx)
		var stopped *StartupStopped
		if !errors.As(err, &stopped) || len(stopped.Variables) == 0 || attempt == 2 {
			return err
		}
		fmt.Fprintln(s.Out, "The server stopped at its secrets check:\n"+stopped.Block)
		if changed, err := s.askFor(stopped.Variables, nil); err != nil {
			return err
		} else if !changed {
			return stopped
		}
	}
}

// StartupStopped is a start the server refused, with the log's explanation, and the variables it
// names when it was the secrets check.
type StartupStopped struct {
	Block     string
	Variables []string
}

func (e *StartupStopped) Error() string { return "the server didn't start:\n" + e.Block }

var named = regexp.MustCompile(`(?:requires environment variable|; set) ([A-Z][A-Z0-9_]+)`)

// startupStopped looks in the server's log for a refused start.
func startupStopped(log string) *StartupStopped {
	start := strings.LastIndex(log, "SECURITY STARTUP CHECK FAILED")
	if start < 0 {
		if i := strings.LastIndex(log, "APPLICATION FAILED TO START"); i >= 0 {
			return &StartupStopped{Block: strings.TrimSpace(log[i:])}
		}
		return nil
	}
	block := log[start:]
	var variables []string
	for _, m := range named.FindAllStringSubmatch(block, -1) {
		if !slices.Contains(variables, m[1]) {
			variables = append(variables, m[1])
		}
	}
	return &StartupStopped{Block: strings.TrimSpace(block), Variables: variables}
}

func (s *Setup) waitReady(ctx context.Context) error {
	timeout := s.ReadyTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	poll := s.Poll
	if poll == 0 {
		poll = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	fmt.Fprintln(s.Out, "Waiting for the server to answer at "+s.Server.BaseURL+"/features")
	err := s.Server.WaitReady(ctx, poll, func() error {
		log, err := s.Docker.Logs(ctx, compose.Service, 300)
		if err != nil {
			return nil
		}
		if stopped := startupStopped(log); stopped != nil {
			return stopped
		}
		return nil
	})
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("the server didn't answer within %s; see `docker compose logs %s`", timeout, compose.Service)
	}
	if err == nil {
		fmt.Fprintln(s.Out, "The server is ready.")
	}
	return err
}

// Token reads the local admin token from inside the server's container.
func (s *Setup) Token(ctx context.Context, _ bool) (string, error) {
	out, err := s.Docker.Exec(ctx, compose.Service, "cat", compose.TokenPath)
	token := strings.TrimSpace(out)
	if err != nil || token == "" {
		return "", fmt.Errorf("no local admin token at %s in the server's container yet (it's "+
			"written once the server is ready, and needs STREAMPACK_LOCAL_ADMIN_TOKEN_FILE, a "+
			"super-admin from ADMIN_EMAIL, and a server with streampack#209)", compose.TokenPath)
	}
	return token, nil
}

// MoveSecrets asks for the variables the server says its next start needs, writes them into .env,
// recreates the server so it reads them, and checks again.
func (s *Setup) MoveSecrets(ctx context.Context) error {
	needed, err := s.Server.Secrets(ctx)
	if err != nil {
		return err
	}
	if len(needed) == 0 {
		fmt.Fprintln(s.Out, "No credentials are waiting to move into .env.")
		return nil
	}
	fmt.Fprintln(s.Out, "These credentials were given at the console and need a place in .env before the next start:")
	variables := make([]string, 0, len(needed))
	descriptions := map[string]string{}
	for _, r := range needed {
		fmt.Fprintf(s.Out, "  %s: %s\n", r.Variable, r.Description)
		variables = append(variables, r.Variable)
		descriptions[r.Variable] = r.Description
	}
	changed, err := s.askFor(variables, descriptions)
	if err != nil || !changed {
		return err
	}
	fmt.Fprintln(s.Out, "Recreating the server so it reads them (up -d; restart wouldn't).")
	if err := s.Docker.Up(ctx); err != nil {
		return err
	}
	if err := s.waitReady(ctx); err != nil {
		return err
	}
	left, err := s.Server.Secrets(ctx)
	if err != nil {
		return err
	}
	if len(left) == 0 {
		fmt.Fprintln(s.Out, "Done: every credential is in .env.")
		return nil
	}
	for _, r := range left {
		fmt.Fprintf(s.Out, "Still needed: %s (%s)\n", r.Variable, r.Description)
	}
	return nil
}

// askFor asks for each variable's value, hidden, and writes those given into .env.
func (s *Setup) askFor(variables []string, descriptions map[string]string) (bool, error) {
	env, err := envfile.Read(s.envPath())
	if err != nil {
		return false, err
	}
	changed := false
	for _, variable := range variables {
		question := variable
		if d := descriptions[variable]; d != "" {
			question += " (" + d + ")"
		}
		value, err := s.Ask.Secret(question + ", blank to skip")
		if err != nil {
			return false, err
		}
		if value == "" {
			continue
		}
		env.Set(variable, value)
		changed = true
	}
	if changed {
		if err := env.Write(s.envPath()); err != nil {
			return false, err
		}
		fmt.Fprintln(s.Out, "Wrote them into .env.")
	}
	return changed, nil
}

func (s *Setup) askInto(env *envfile.File, key, question, def string, valid func(string) error) error {
	if value, ok := env.Get(key); ok && value != "" {
		return nil
	}
	value, err := s.required(question, def, valid)
	if err != nil {
		return err
	}
	env.Set(key, value)
	return nil
}

func (s *Setup) required(question, def string, valid func(string) error) (string, error) {
	for {
		value, err := s.Ask.Ask(question, def)
		if err != nil {
			return "", err
		}
		if value == "" {
			fmt.Fprintln(s.Out, "An answer is needed.")
			continue
		}
		if valid != nil {
			if err := valid(value); err != nil {
				fmt.Fprintln(s.Out, err)
				continue
			}
		}
		return value, nil
	}
}

func validEmail(value string) error {
	if address, err := mail.ParseAddress(value); err != nil || address.Address != value {
		return fmt.Errorf("%q isn't an email address", value)
	}
	return nil
}

func validURL(value string) error {
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return fmt.Errorf("%q should start with http:// or https://", value)
	}
	return nil
}

func (s *Setup) bytes(n int) []byte {
	if s.Random != nil {
		return s.Random(n)
	}
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func (s *Setup) base64(n int) string { return base64.RawURLEncoding.EncodeToString(s.bytes(n)) }
func (s *Setup) hex(n int) string    { return hex.EncodeToString(s.bytes(n)) }
