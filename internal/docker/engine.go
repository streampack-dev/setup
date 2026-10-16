package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Engine is the container engine on this box, and the Compose that drives it. The tool doesn't
// mind which: it finds out (Detect), and every command it runs is one both engines take.
type Engine struct {
	// Name is the command for containers, volumes and networks: docker or podman.
	Name string
	// Compose is the Compose command: docker compose, docker-compose, podman-compose or podman
	// compose.
	Compose []string
	// PodmanCompose says Compose is podman-compose, which needs telling to recreate the services
	// when .env changes, and groups them in a pod.
	PodmanCompose bool
}

// ComposeCommand is Compose as you'd type it.
func (e Engine) ComposeCommand() string { return strings.Join(e.Compose, " ") }

func (e Engine) String() string {
	return e.Name + ", with " + e.ComposeCommand()
}

// MinimumPodmanCompose is the first podman-compose that reads env_file itself, as Docker Compose
// does. Before it, podman-compose handed .env to Podman, which keeps quotes as part of a value.
var MinimumPodmanCompose = [3]int{1, 2, 0}

// Runner runs a command in dir: the real one runs it (Exec); tests replace it.
type Runner func(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error

// Exec runs the command for real.
func Exec(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = dir, stdout, stderr
	return cmd.Run()
}

// ErrNoEngine is a box with neither Docker nor Podman, or with neither running.
var ErrNoEngine = errors.New("no container engine")

// Detect finds the engine on this box. Docker counts only when its daemon answers; a docker
// command that is Podman's (podman-docker) is Podman. Podman is used by its own name, so the
// commands printed say what really ran. lookPath finds a command, as exec.LookPath does.
func Detect(ctx context.Context, lookPath func(string) (string, error), run Runner) (Engine, error) {
	output := func(name string, args ...string) (string, bool) {
		var out bytes.Buffer
		err := run(ctx, "", &out, &out, name, args...)
		return out.String(), err == nil
	}
	_, dockerErr := lookPath("docker")
	_, podmanErr := lookPath("podman")
	hasDocker, hasPodman := dockerErr == nil, podmanErr == nil
	if hasDocker {
		if v, _ := output("docker", "--version"); strings.Contains(strings.ToLower(v), "podman") {
			hasDocker = false
		}
	}

	var problems []string
	if hasDocker {
		if _, ok := output("docker", "version", "--format", "{{.Server.Version}}"); ok {
			return dockerCompose(lookPath, output)
		}
		problems = append(problems, "Docker is installed, but its daemon doesn't answer: start Docker")
	}
	if hasPodman {
		if _, ok := output("podman", "info"); ok {
			return podmanCompose(lookPath, output)
		}
		problems = append(problems, "Podman is installed, but `podman info` fails (on a Mac: `podman machine start`)")
	}
	if len(problems) == 0 {
		problems = append(problems, "install Docker or Podman, with Compose")
	}
	return Engine{}, fmt.Errorf("%w: %s", ErrNoEngine, strings.Join(problems, "; "))
}

func dockerCompose(lookPath func(string) (string, error), output func(string, ...string) (string, bool)) (Engine, error) {
	if _, ok := output("docker", "compose", "version"); ok {
		return Engine{Name: "docker", Compose: []string{"docker", "compose"}}, nil
	}
	if _, err := lookPath("docker-compose"); err == nil {
		return Engine{Name: "docker", Compose: []string{"docker-compose"}}, nil
	}
	return Engine{}, fmt.Errorf("%w: Docker has no Compose: install the Compose plugin", ErrNoEngine)
}

// podmanCompose prefers podman-compose by its own name; `podman compose` hands over to whichever
// provider is installed (podman-compose or docker-compose), with a notice on every run.
func podmanCompose(lookPath func(string) (string, error), output func(string, ...string) (string, bool)) (Engine, error) {
	candidates := [][]string{{"podman", "compose"}}
	if _, err := lookPath("podman-compose"); err == nil {
		candidates = append([][]string{{"podman-compose"}}, candidates...)
	}
	for _, compose := range candidates {
		v, ok := output(compose[0], append(compose[1:], "version")...)
		if !ok {
			continue
		}
		e := Engine{Name: "podman", Compose: compose}
		if found, at := podmanComposeVersion(v); found {
			if older(at, MinimumPodmanCompose) {
				return Engine{}, fmt.Errorf("%w: podman-compose %s is too old: it hands .env to Podman, which "+
					"keeps the quotes as part of each value. Install podman-compose %s or later",
					ErrNoEngine, dotted(at), dotted(MinimumPodmanCompose))
			}
			e.PodmanCompose = true
		}
		return e, nil
	}
	return Engine{}, fmt.Errorf("%w: Podman has no Compose: install podman-compose", ErrNoEngine)
}

var podmanComposeVersionLine = regexp.MustCompile(`podman-compose version:? v?(\d+)\.(\d+)\.(\d+)`)

// podmanComposeVersion reads podman-compose's version from what `version` printed, if it's
// podman-compose that printed it.
func podmanComposeVersion(output string) (bool, [3]int) {
	m := podmanComposeVersionLine.FindStringSubmatch(output)
	if m == nil {
		return false, [3]int{}
	}
	var v [3]int
	for i := range v {
		v[i], _ = strconv.Atoi(m[i+1])
	}
	return true, v
}

func older(v, than [3]int) bool {
	for i := range v {
		if v[i] != than[i] {
			return v[i] < than[i]
		}
	}
	return false
}

func dotted(v [3]int) string { return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]) }
