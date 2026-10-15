// Package docker runs `docker compose` in the deployment directory, as a command a person could
// type, so every step the tool takes can be rerun by hand.
package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
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
