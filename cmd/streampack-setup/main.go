// Command streampack-setup brings up a server-streampack on a box with Docker, and then acts as its
// console. See README.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/streampack-dev/setup/internal/docker"
	"github.com/streampack-dev/setup/internal/prompt"
	"github.com/streampack-dev/setup/internal/registry"
	"github.com/streampack-dev/setup/internal/server"
	"github.com/streampack-dev/setup/internal/setup"
)

// version is set at build time: -ldflags "-X main.version=1.0.0".
var version = "development"

const usage = `streampack-setup brings up a server-streampack with Docker Compose, then acts as its console.

Usage: streampack-setup [flags] [command] [flags]

With --hard-reset, it deletes the deployment first, after you confirm, and then runs as a first run.
With --update, it moves the server to the newest release first, after you confirm, and then runs.

Commands:
  (none)    init if there's no .env yet, then up, then console
  init      ask what starting the server needs; write .env and compose.yaml
  up        docker compose pull and up -d, then wait for the server to answer
  console   open the server's console as the super-admin
  secrets   move credentials given at the console into .env, and recreate the server
  version   print this tool's version

Flags, before or after the command; one dash works as well as two:
  --dir DIR      the deployment directory, where .env and compose.yaml are (default ".")
  --url URL      the server's address from this box (default "http://127.0.0.1:8080")
  --hard-reset   first delete the deployment (containers, volumes, .env, the compose file,
                 media/) and start from nothing; it keeps generative/ and ext/
  --update       first set STREAMPACK_VERSION in .env to the newest server release, whatever
                 version it is on, after you confirm; with no command or with up
  --version, -v  print this tool's version
  --help, -h     print this help
`

// options is what the command line asks for.
type options struct {
	dir, url    string
	hardReset   bool
	update      bool
	showVersion bool
	command     string
}

// parseArgs reads the flags wherever they are on the line, before or after the command: Go's flag
// package alone stops at the first word that isn't one, so "init --hard-reset" would run a plain
// init (#11). More than one command is an error, as is a flag it doesn't know.
func parseArgs(args []string, stderr io.Writer) (options, error) {
	o := options{}
	flags := flag.NewFlagSet("streampack-setup", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { fmt.Fprint(stderr, usage) }
	flags.StringVar(&o.dir, "dir", ".", "")
	flags.StringVar(&o.url, "url", "http://127.0.0.1:8080", "")
	flags.BoolVar(&o.hardReset, "hard-reset", false, "")
	flags.BoolVar(&o.update, "update", false, "")
	flags.BoolVar(&o.showVersion, "version", false, "")
	flags.BoolVar(&o.showVersion, "v", false, "")
	var words []string
	for {
		if err := flags.Parse(args); err != nil {
			return o, err
		}
		rest := flags.Args()
		if len(rest) == 0 {
			break
		}
		words = append(words, rest[0])
		args = rest[1:]
	}
	if len(words) > 1 {
		return o, fmt.Errorf("one command at a time, not %q", strings.Join(words, " "))
	}
	if len(words) == 1 {
		o.command = words[0]
	}
	if o.command == "version" {
		o.showVersion = true
	}
	return o, nil
}

// banner names the tool and its version, first on every run, so a log shows what ran.
func banner() string { return "streampack-setup " + version }

func main() {
	o, err := parseArgs(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "streampack-setup:", err)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	fmt.Println(banner())
	if o.showVersion {
		return
	}
	command := o.command
	dir, url, hardReset := &o.dir, &o.url, &o.hardReset
	if !knownCommand(command) {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if o.update && !updateAllowed(command) {
		fmt.Fprintln(os.Stderr, "streampack-setup: --update goes with no command, or with up")
		os.Exit(2)
	}

	absolute, err := filepath.Abs(*dir)
	if err != nil {
		fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	engine, err := docker.Detect(ctx, exec.LookPath, docker.Exec)
	if err != nil {
		fail(err)
	}
	fmt.Println("Containers: " + engine.String())

	httpClient := &http.Client{}
	s := &setup.Setup{
		Dir:    absolute,
		Ask:    &prompt.Terminal{In: os.Stdin, Out: os.Stdout},
		Docker: docker.CLI{Dir: absolute, Out: os.Stdout, Engine: engine},
		Out:    os.Stdout,
		Latest: func(ctx context.Context) (string, error) {
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return registry.Latest(ctx, httpClient, registry.TagsURL)
		},
	}
	s.Server = &server.Client{BaseURL: *url, HTTP: httpClient, Token: s.Token}

	if *hardReset {
		// What follows a reset is a first run: everything else needs the deployment it deleted.
		if command != "" && command != "init" {
			fmt.Fprintln(os.Stderr, "streampack-setup: --hard-reset goes with no command, or with init")
			os.Exit(2)
		}
		check(s.HardReset(ctx))
	}

	switch command {
	case "":
		configured, err := s.Configured()
		if err == nil && !configured {
			// A first run's init already offers the newest release.
			err = s.Init(ctx)
		} else if err == nil && o.update {
			err = s.Update(ctx)
		}
		if err == nil {
			err = s.Up(ctx)
		}
		if err == nil {
			err = s.Console(ctx)
		}
		check(err)
	case "init":
		check(s.Init(ctx))
	case "up":
		var err error
		if o.update {
			err = s.Update(ctx)
		}
		if err == nil {
			err = s.Up(ctx)
		}
		check(err)
	case "console":
		check(s.Console(ctx))
	case "secrets":
		check(s.MoveSecrets(ctx))
	}
}

func knownCommand(c string) bool {
	switch c {
	case "", "init", "up", "console", "secrets":
		return true
	}
	return false
}

// updateAllowed says whether --update goes with command: an update is followed by a start, so
// it goes with no command or with up.
func updateAllowed(command string) bool { return command == "" || command == "up" }

func check(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "streampack-setup:", err)
	os.Exit(1)
}
