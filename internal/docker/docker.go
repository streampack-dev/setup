// Package docker runs `docker compose` in the deployment directory, as a command a person could
// type, so every step the tool takes can be rerun by hand.
package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Compose is what the setup flows need from Docker Compose.
type Compose interface {
	// Pull fetches the services' images.
	Pull(ctx context.Context) error
	// Up creates or recreates the services, reading .env afresh.
	Up(ctx context.Context) error
	// Exec runs a command in service's running container and returns its output.
	Exec(ctx context.Context, service string, args ...string) (string, error)
	// Logs returns service's last lines of log.
	Logs(ctx context.Context, service string, lines int) (string, error)
	// Volume returns the full name of the project's volume called name, or "" when there's none.
	// It works without .env, before the compose file can be read.
	Volume(ctx context.Context, name string) (string, error)
	// RemoveVolume removes the project's containers, which hold the volume, and then the volume.
	RemoveVolume(ctx context.Context, volume string) error
	// Resources lists what the project has in Docker, found by its label, so it works however
	// broken .env and the compose file are.
	Resources(ctx context.Context) (Resources, error)
	// RemoveResources removes them: containers, then volumes and networks.
	RemoveResources(ctx context.Context, r Resources) error
	// ClearDirectory empties the host directory dir from inside a throwaway container of image, as
	// root, since what the server wrote there belongs to the container's root.
	ClearDirectory(ctx context.Context, image, dir string) error
}

// Resources are a project's containers, volumes and networks, by name, and the image its server
// container runs, if it has one.
type Resources struct {
	Containers, Volumes, Networks []string
	ServerImage                   string
}

// Empty says whether there's nothing.
func (r Resources) Empty() bool {
	return len(r.Containers)+len(r.Volumes)+len(r.Networks) == 0
}

// CLI is Compose by the docker command, in Dir; progress goes to Out.
type CLI struct {
	Dir string
	Out io.Writer
}

func (c CLI) Pull(ctx context.Context) error { return c.run(ctx, "pull") }
func (c CLI) Up(ctx context.Context) error   { return c.run(ctx, "up", "-d") }

func (c CLI) Exec(ctx context.Context, service string, args ...string) (string, error) {
	return c.output(ctx, append([]string{"exec", "-T", service}, args...)...)
}

func (c CLI) Logs(ctx context.Context, service string, lines int) (string, error) {
	return c.output(ctx, "logs", "--no-color", "--tail", strconv.Itoa(lines), service)
}

// Project is the Compose project name for Dir: COMPOSE_PROJECT_NAME when it's set, otherwise the
// directory's name as Compose normalizes it (lower case; letters, digits, '-' and '_').
func (c CLI) Project() string {
	if name := os.Getenv("COMPOSE_PROJECT_NAME"); name != "" {
		return name
	}
	return ProjectName(filepath.Base(c.Dir))
}

// ProjectName normalizes a directory's name as Compose does for a project name.
func ProjectName(dir string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(dir) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.TrimLeft(b.String(), "-_")
}

func (c CLI) Volume(ctx context.Context, name string) (string, error) {
	out, err := c.docker(ctx, "volume", "ls", "-q",
		"--filter", "label=com.docker.compose.project="+c.Project(),
		"--filter", "label=com.docker.compose.volume="+name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.SplitN(out, "\n", 2)[0]), nil
}

func (c CLI) RemoveVolume(ctx context.Context, volume string) error {
	containers, err := c.docker(ctx, "ps", "-aq", "--filter", "label=com.docker.compose.project="+c.Project())
	if err != nil {
		return err
	}
	if ids := strings.Fields(containers); len(ids) > 0 {
		fmt.Fprintf(c.Out, "$ docker rm -f %s\n", strings.Join(ids, " "))
		if _, err := c.docker(ctx, append([]string{"rm", "-f"}, ids...)...); err != nil {
			return err
		}
	}
	fmt.Fprintf(c.Out, "$ docker volume rm %s\n", volume)
	_, err = c.docker(ctx, "volume", "rm", volume)
	return err
}

func (c CLI) Resources(ctx context.Context) (Resources, error) {
	label := "label=com.docker.compose.project=" + c.Project()
	var r Resources
	containers, err := c.docker(ctx, "ps", "-a", "--filter", label, "--format",
		`{{.Names}}\t{{.Image}}\t{{.Label "com.docker.compose.service"}}`)
	if err != nil {
		return Resources{}, err
	}
	for _, line := range strings.Split(strings.TrimSpace(containers), "\n") {
		fields := strings.Split(line, "\t")
		if fields[0] == "" {
			continue
		}
		r.Containers = append(r.Containers, fields[0])
		if len(fields) == 3 && fields[2] == "server-streampack" {
			r.ServerImage = fields[1]
		}
	}
	for _, list := range []struct {
		into *[]string
		args []string
	}{
		{&r.Volumes, []string{"volume", "ls", "--filter", label, "--format", "{{.Name}}"}},
		{&r.Networks, []string{"network", "ls", "--filter", label, "--format", "{{.Name}}"}},
	} {
		out, err := c.docker(ctx, list.args...)
		if err != nil {
			return Resources{}, err
		}
		*list.into = strings.Fields(out)
	}
	return r, nil
}

func (c CLI) RemoveResources(ctx context.Context, r Resources) error {
	for _, step := range []struct {
		args  []string
		names []string
	}{
		{[]string{"rm", "-f"}, r.Containers},
		{[]string{"volume", "rm"}, r.Volumes},
		{[]string{"network", "rm"}, r.Networks},
	} {
		if len(step.names) == 0 {
			continue
		}
		args := append(append([]string{}, step.args...), step.names...)
		fmt.Fprintf(c.Out, "$ docker %s\n", strings.Join(args, " "))
		if _, err := c.docker(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

func (c CLI) ClearDirectory(ctx context.Context, image, dir string) error {
	args := []string{"run", "--rm", "-v", dir + ":/clear", "--entrypoint", "sh", image, "-c",
		"rm -rf /clear/* /clear/.[!.]* /clear/..?*"}
	fmt.Fprintf(c.Out, "$ docker %s\n", shellJoin(args))
	_, err := c.docker(ctx, args...)
	return err
}

// shellJoin joins args as a shell would read them back, quoting any with spaces or globs, so a
// printed command can be copied and run.
func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " *?[]$'\"") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}

// docker runs a plain docker command (not compose) and returns its output. It needs no working
// directory, so it runs before init has made Dir.
func (c CLI) docker(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("docker %s: %w: %s", strings.Join(args[:2], " "), err, strings.TrimSpace(errOut.String()))
	}
	return out.String(), nil
}

func (c CLI) run(ctx context.Context, args ...string) error {
	fmt.Fprintf(c.Out, "$ docker compose %s\n", strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = c.Dir, c.Out, c.Out
	return cmd.Run()
}

func (c CLI) output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
	var out, errOut bytes.Buffer
	cmd.Dir, cmd.Stdout, cmd.Stderr = c.Dir, &out, &errOut
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("docker compose %s: %w: %s", args[0], err, strings.TrimSpace(errOut.String()))
	}
	return out.String(), nil
}
