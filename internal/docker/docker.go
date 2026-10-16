// Package docker runs Compose in the deployment directory, as a command a person could type, so
// every step the tool takes can be rerun by hand. Docker or Podman, whichever is here (Detect).
package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
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
	// Resources lists what the project has in the engine, found by its label, so it works however
	// broken .env and the compose file are.
	Resources(ctx context.Context) (Resources, error)
	// RemoveResources removes them: containers, then pods, volumes and networks.
	RemoveResources(ctx context.Context, r Resources) error
	// ClearDirectory empties the host directory dir from inside a throwaway container of image, as
	// root, since what the server wrote there belongs to the container's root.
	ClearDirectory(ctx context.Context, image, dir string) error
	// ComposeCommand is Compose as you'd type it, for messages: docker compose, podman-compose…
	ComposeCommand() string
}

// Resources are a project's containers, pods (podman-compose's), volumes and networks, by name,
// and the image its server container runs, if it has one.
type Resources struct {
	Containers, Pods, Volumes, Networks []string
	ServerImage                         string
}

// Empty says whether there's nothing.
func (r Resources) Empty() bool {
	return len(r.Containers)+len(r.Pods)+len(r.Volumes)+len(r.Networks) == 0
}

// CLI is Compose by Engine's commands, in Dir; progress goes to Out. Run runs them; nil is Exec.
type CLI struct {
	Dir    string
	Out    io.Writer
	Engine Engine
	Run    Runner
}

func (c CLI) ComposeCommand() string { return c.Engine.ComposeCommand() }

func (c CLI) Pull(ctx context.Context) error { return c.run(ctx, "pull") }

// Up starts the services. podman-compose recreates a service only when the compose file changes,
// not .env, so it's told to.
func (c CLI) Up(ctx context.Context) error {
	if c.Engine.PodmanCompose {
		return c.run(ctx, "up", "-d", "--force-recreate")
	}
	return c.run(ctx, "up", "-d")
}

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

// Volume finds the volume by the project's label and the name both Composes give it,
// <project>_<name>: podman-compose doesn't label a volume with its own name.
func (c CLI) Volume(ctx context.Context, name string) (string, error) {
	out, err := c.docker(ctx, "volume", "ls", "--filter", "label=com.docker.compose.project="+c.Project(),
		"--format", "{{.Name}}")
	if err != nil {
		return "", err
	}
	want := c.Project() + "_" + name
	for _, volume := range strings.Fields(out) {
		if volume == want {
			return volume, nil
		}
	}
	return "", nil
}

func (c CLI) RemoveVolume(ctx context.Context, volume string) error {
	containers, err := c.docker(ctx, "ps", "-aq", "--filter", "label=com.docker.compose.project="+c.Project())
	if err != nil {
		return err
	}
	if ids := strings.Fields(containers); len(ids) > 0 {
		fmt.Fprintf(c.Out, "$ %s rm -f %s\n", c.Engine.Name, strings.Join(ids, " "))
		if _, err := c.docker(ctx, append([]string{"rm", "-f"}, ids...)...); err != nil {
			return err
		}
	}
	fmt.Fprintf(c.Out, "$ %s volume rm %s\n", c.Engine.Name, volume)
	_, err = c.docker(ctx, "volume", "rm", volume)
	return err
}

// Resources asks only for fields every engine's templates have (.Names, .Image, .Name), and finds
// the server's container by a second filter: Docker's {{.Label}} is unknown to Podman, and
// Podman's {{index .Labels}} fails on Docker.
func (c CLI) Resources(ctx context.Context) (Resources, error) {
	label := "label=com.docker.compose.project=" + c.Project()
	var r Resources
	containers, err := c.docker(ctx, "ps", "-a", "--filter", label, "--format", "{{.Names}}")
	if err != nil {
		return Resources{}, err
	}
	r.Containers = strings.Fields(containers)
	if len(r.Containers) > 0 {
		images, err := c.docker(ctx, "ps", "-a", "--filter", label,
			"--filter", "label=com.docker.compose.service=server-streampack", "--format", "{{.Image}}")
		if err != nil {
			return Resources{}, err
		}
		if fields := strings.Fields(images); len(fields) > 0 {
			r.ServerImage = fields[0]
		}
	}
	if c.Engine.Name == "podman" {
		// podman-compose puts the services in a pod named for the project, with no label.
		pods, err := c.docker(ctx, "pod", "ls", "--format", "{{.Name}}")
		if err != nil {
			return Resources{}, err
		}
		for _, pod := range strings.Fields(pods) {
			if pod == "pod_"+c.Project() {
				r.Pods = append(r.Pods, pod)
			}
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
		{[]string{"pod", "rm", "-f"}, r.Pods},
		{[]string{"volume", "rm"}, r.Volumes},
		{[]string{"network", "rm"}, r.Networks},
	} {
		if len(step.names) == 0 {
			continue
		}
		args := append(append([]string{}, step.args...), step.names...)
		fmt.Fprintf(c.Out, "$ %s %s\n", c.Engine.Name, strings.Join(args, " "))
		if _, err := c.docker(ctx, args...); err != nil {
			return err
		}
	}
	return nil
}

// ClearDirectory mounts dir with z, so SELinux lets the container at it; without SELinux, z does
// nothing.
func (c CLI) ClearDirectory(ctx context.Context, image, dir string) error {
	args := []string{"run", "--rm", "-v", dir + ":/clear:z", "--entrypoint", "sh", image, "-c",
		"rm -rf /clear/* /clear/.[!.]* /clear/..?*"}
	fmt.Fprintf(c.Out, "$ %s %s\n", c.Engine.Name, shellJoin(args))
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

// docker runs a plain engine command (not Compose) and returns its output. It needs no working
// directory, so it runs before init has made Dir.
func (c CLI) docker(ctx context.Context, args ...string) (string, error) {
	var out, errOut bytes.Buffer
	if err := c.runner()(ctx, "", &out, &errOut, c.Engine.Name, args...); err != nil {
		return out.String(), fmt.Errorf("%s %s: %w: %s", c.Engine.Name, strings.Join(args[:2], " "), err,
			strings.TrimSpace(errOut.String()))
	}
	return out.String(), nil
}

func (c CLI) run(ctx context.Context, args ...string) error {
	fmt.Fprintf(c.Out, "$ %s %s\n", c.ComposeCommand(), strings.Join(args, " "))
	name, full := c.compose(args)
	return c.runner()(ctx, c.Dir, c.Out, c.Out, name, full...)
}

func (c CLI) output(ctx context.Context, args ...string) (string, error) {
	var out, errOut bytes.Buffer
	name, full := c.compose(args)
	err := c.runner()(ctx, c.Dir, &out, &errOut, name, full...)
	if err != nil {
		return out.String(), fmt.Errorf("%s %s: %w: %s", c.ComposeCommand(), args[0], err, strings.TrimSpace(errOut.String()))
	}
	return out.String(), nil
}

// compose is the command and arguments that run Compose with args.
func (c CLI) compose(args []string) (string, []string) {
	return c.Engine.Compose[0], append(append([]string{}, c.Engine.Compose[1:]...), args...)
}

func (c CLI) runner() Runner {
	if c.Run != nil {
		return c.Run
	}
	return Exec
}
