package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/streampack-dev/setup/internal/envfile"
	"github.com/streampack-dev/setup/internal/server"
)

// script answers each question with the next of its answers, whatever kind it is; io.EOF after.
type script struct {
	answers []string
	asked   []string
}

func (s *script) next(question string) (string, error) {
	s.asked = append(s.asked, question)
	if len(s.answers) == 0 {
		return "", io.EOF
	}
	a := s.answers[0]
	s.answers = s.answers[1:]
	return a, nil
}

func (s *script) Ask(q, def string) (string, error) {
	a, err := s.next(q)
	if a == "" && err == nil {
		return def, nil
	}
	return a, err
}
func (s *script) Secret(q string) (string, error) { return s.next(q) }
func (s *script) Line(p string) (string, error)   { return s.next(p) }
func (s *script) Confirm(q string, def bool) (bool, error) {
	a, err := s.next(q)
	if a == "" {
		return def, err
	}
	return a == "y", err
}

// fakeDocker records what was run; logs answers the server's log, and token the token file.
type fakeDocker struct {
	mu      sync.Mutex
	pulls   int
	ups     int
	logs    []string
	token   string
	onUp    func(n int)
	lastLog string
}

func (f *fakeDocker) Pull(context.Context) error { f.pulls++; return nil }
func (f *fakeDocker) Up(context.Context) error {
	f.mu.Lock()
	f.ups++
	n := f.ups
	f.mu.Unlock()
	if f.onUp != nil {
		f.onUp(n)
	}
	return nil
}
func (f *fakeDocker) Exec(_ context.Context, service string, args ...string) (string, error) {
	if service != "server-streampack" || strings.Join(args, " ") != "cat /run/streampack/admin-token" {
		return "", fmt.Errorf("unexpected exec %s %v", service, args)
	}
	return f.token + "\n", nil
}
func (f *fakeDocker) Logs(context.Context, string, int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastLog, nil
}

// lockedBuffer is output written by the console's stream and read by the test at once.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newSetup(t *testing.T, answers []string, docker *fakeDocker, url string) (*Setup, *script, *lockedBuffer) {
	t.Helper()
	ask := &script{answers: answers}
	out := &lockedBuffer{}
	s := &Setup{
		Dir:    t.TempDir(),
		Ask:    ask,
		Docker: docker,
		Out:    out,
		Latest: func(context.Context) (string, error) { return "1.16.0", nil },
		Random: func(n int) []byte { return bytes.Repeat([]byte{7}, n) },
		Poll:   time.Millisecond,
	}
	s.Server = &server.Client{BaseURL: url, HTTP: http.DefaultClient, Token: s.Token}
	return s, ask, out
}

func TestInitAsksWhatStartingNeedsAndWritesTheFiles(t *testing.T) {
	s, _, _ := newSetup(t, []string{
		"",                    // version: the newest release
		"not an address",      // refused
		"root@example.com",    // admin
		"",                    // API address, default
		"https://example.com", // site address
		"",                    // Mattermost: yes by default
		"",                    // GitLab: yes by default
		"",                    // IRC: no
		"y",                   // Slack
		"",                    // Discord: no
		"",                    // mail: no
	}, &fakeDocker{}, "")

	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	env, _ := envfile.Read(filepath.Join(s.Dir, ".env"))
	for key, want := range map[string]string{
		"STREAMPACK_VERSION": "1.16.0",
		"ADMIN_EMAIL":        "root@example.com",
		"BASE_URL":           "http://localhost:8080",
		"BLOG_BASE_URL":      "https://example.com",
		"MATTERMOST_ENABLED": "true",
		"GITLAB_ENABLED":     "true",
		"IRC_ENABLED":        "false",
		"SLACK_ENABLED":      "true",
		"DISCORD_ENABLED":    "false",
		"MAIL_ENABLED":       "false",
	} {
		if got, _ := env.Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	for _, key := range []string{"DB_PASSWORD", "JWT_SECRET", "GITLAB_WEBHOOK_SECRET_KEY", "GITHUB_WEBHOOK_SECRET_KEY"} {
		if got, _ := env.Get(key); len(got) < 32 {
			t.Errorf("%s = %q, want a generated value", key, got)
		}
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "compose.yaml")); err != nil {
		t.Error("no compose.yaml")
	}
	if configured, _ := s.Configured(); !configured {
		t.Error("not configured after init")
	}
}

func TestInitKeepsWhatTheEnvAlreadyHas(t *testing.T) {
	s, ask, _ := newSetup(t, []string{"", "", "", "", "", "", "", ""}, &fakeDocker{}, "")
	os.WriteFile(filepath.Join(s.Dir, ".env"), []byte("STREAMPACK_VERSION=1.15.7\nADMIN_EMAIL=me@example.com\nJWT_SECRET=kept\n"), 0o600)

	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	env, _ := envfile.Read(filepath.Join(s.Dir, ".env"))
	if v, _ := env.Get("STREAMPACK_VERSION"); v != "1.15.7" {
		t.Errorf("version %q", v)
	}
	if v, _ := env.Get("JWT_SECRET"); v != "kept" {
		t.Errorf("JWT_SECRET %q", v)
	}
	for _, q := range ask.asked {
		if strings.Contains(q, "version") || strings.Contains(q, "email") {
			t.Errorf("asked %q again", q)
		}
	}
}

func TestStartupStoppedNamesTheVariables(t *testing.T) {
	log := `starting
============================================================
SECURITY STARTUP CHECK FAILED (Mattermost secrets)
============================================================
Missing/invalid environment variables:
- Mattermost server 'work' token is stored as a literal; set MATTERMOST_WORK_TOKEN (the value is in the database, or issue a new token) so it can be externalized
- GitLab project 'g/p' token requires environment variable GITLAB_G_P_TOKEN
============================================================`
	stopped := startupStopped(log)
	if stopped == nil || strings.Join(stopped.Variables, ",") != "MATTERMOST_WORK_TOKEN,GITLAB_G_P_TOKEN" {
		t.Fatalf("got %+v", stopped)
	}
	if startupStopped("all fine") != nil {
		t.Fatal("expected nothing")
	}
	if s := startupStopped("APPLICATION FAILED TO START\nbad"); s == nil || len(s.Variables) != 0 {
		t.Fatalf("got %+v", s)
	}
}

// fakeServer answers /features once ready, /admin/secrets from needed (for the bearer "token"),
// and the console.
type fakeServer struct {
	mu        sync.Mutex
	ready     bool
	needed    []server.Requirement
	submitted []string
	results   chan string
}

func (f *fakeServer) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.URL.Path == "/features" {
			if !f.ready {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/admin/secrets":
			json.NewEncoder(w).Encode(map[string]any{"required": f.needed})
		case "/admin/console":
			var body struct{ Line string }
			json.NewDecoder(r.Body).Decode(&body)
			f.submitted = append(f.submitted, body.Line)
			w.WriteHeader(http.StatusAccepted)
			f.results <- body.Line
		case "/admin/console/stream":
			f.mu.Unlock()
			defer f.mu.Lock()
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "event: ready\ndata: {\"username\":\"root\"}\n\n")
			w.(http.Flusher).Flush()
			for {
				select {
				case line := <-f.results:
					fmt.Fprintf(w, "event: result\ndata: {\"correlationId\":\"c\",\"status\":\"success\",\"text\":\"did %s\"}\n\n", line)
					w.(http.Flusher).Flush()
				case <-r.Context().Done():
					return
				}
			}
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
}

func TestUpOffersTheVariablesAStoppedStartNames(t *testing.T) {
	fake := &fakeServer{}
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()
	docker := &fakeDocker{}
	docker.onUp = func(n int) {
		docker.mu.Lock()
		defer docker.mu.Unlock()
		if n == 1 {
			docker.lastLog = "SECURITY STARTUP CHECK FAILED\n- x requires environment variable MATTERMOST_WORK_TOKEN\n"
			return
		}
		docker.lastLog = "Started"
		fake.mu.Lock()
		fake.ready = true
		fake.mu.Unlock()
	}
	s, _, out := newSetup(t, []string{"mm-secret"}, docker, srv.URL)

	if err := s.Up(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if docker.pulls != 1 || docker.ups != 2 {
		t.Fatalf("pulls %d, ups %d", docker.pulls, docker.ups)
	}
	env, _ := envfile.Read(filepath.Join(s.Dir, ".env"))
	if v, _ := env.Get("MATTERMOST_WORK_TOKEN"); v != "mm-secret" {
		t.Fatalf("got %q", v)
	}
}

func TestMoveSecretsWritesWhatTheServerNeedsAndRecreatesIt(t *testing.T) {
	fake := &fakeServer{ready: true, needed: []server.Requirement{
		{Variable: "MATTERMOST_WORK_TOKEN", Description: "Mattermost server 'work' token", State: "STORED_IN_DATABASE"},
		{Variable: "GITLAB_G_P_TOKEN", Description: "GitLab project 'g/p' token", State: "STORED_IN_DATABASE"},
	}}
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()
	docker := &fakeDocker{token: "token"}
	docker.onUp = func(int) {
		fake.mu.Lock()
		fake.needed = fake.needed[1:] // the one skipped is still needed
		fake.mu.Unlock()
	}
	s, ask, out := newSetup(t, []string{"mm-secret", ""}, docker, srv.URL)

	if err := s.MoveSecrets(context.Background()); err != nil {
		t.Fatal(err)
	}
	env, _ := envfile.Read(filepath.Join(s.Dir, ".env"))
	if v, _ := env.Get("MATTERMOST_WORK_TOKEN"); v != "mm-secret" {
		t.Errorf("got %q", v)
	}
	if _, ok := env.Get("GITLAB_G_P_TOKEN"); ok {
		t.Error("a skipped variable was written")
	}
	if docker.ups != 1 {
		t.Errorf("ups %d", docker.ups)
	}
	if !strings.Contains(ask.asked[0], "Mattermost server 'work' token") {
		t.Errorf("asked %q", ask.asked[0])
	}
	if !strings.Contains(out.String(), "Still needed: GITLAB_G_P_TOKEN") {
		t.Errorf("out:\n%s", out)
	}
	if strings.Contains(out.String(), "mm-secret") {
		t.Error("a value was printed")
	}
}

func TestConsolePassesLinesAndPrintsResults(t *testing.T) {
	fake := &fakeServer{ready: true, results: make(chan string, 4)}
	srv := httptest.NewServer(fake.handler(t))
	defer srv.Close()
	docker := &fakeDocker{token: "token"}
	s, _, out := newSetup(t, []string{"version", "", ":quit"}, docker, srv.URL)
	// Give the result time to arrive before quitting.
	s.Ask = &slowScript{script: s.Ask.(*script), out: out, wait: "did version"}

	if err := s.Console(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Join(fake.submitted, ",") != "version" {
		t.Fatalf("submitted %v", fake.submitted)
	}
	if !strings.Contains(out.String(), "Console open as root") || !strings.Contains(out.String(), "did version") {
		t.Fatalf("out:\n%s", out)
	}
}

// slowScript waits, before :quit, until out shows wait.
type slowScript struct {
	*script
	out  *lockedBuffer
	wait string
}

func (s *slowScript) Line(p string) (string, error) {
	line, err := s.script.Line(p)
	if line == ":quit" {
		deadline := time.Now().Add(5 * time.Second)
		for !strings.Contains(s.out.String(), s.wait) && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
	}
	return line, err
}
