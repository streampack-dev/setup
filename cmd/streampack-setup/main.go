// Command streampack-setup brings up a server-streampack on a box with Docker, and then acts as its
// console. See README.md.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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

Usage: streampack-setup [flags] [command]

With -hard-reset, it deletes the deployment first, after you confirm, and then runs as a first run.

Commands:
  (none)    init if there's no .env yet, then up, then console
  init      ask what starting the server needs; write .env and compose.yaml
  up        docker compose pull and up -d, then wait for the server to answer
  console   open the server's console as the super-admin
  secrets   move credentials given at the console into .env, and recreate the server
  version   print this tool's version

Flags:
`

func main() {
	flags := flag.NewFlagSet("streampack-setup", flag.ExitOnError)
	dir := flags.String("dir", ".", "the deployment directory, where .env and compose.yaml are")
	url := flags.String("url", "http://127.0.0.1:8080", "the server's address from this box")
	hardReset := flags.Bool("hard-reset", false, "first delete the deployment (containers, volumes, .env, the compose file, generative/) and start from nothing")
	flags.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flags.PrintDefaults()
	}
	flags.Parse(os.Args[1:])
	command := flags.Arg(0)
	if command == "version" {
		fmt.Println("streampack-setup", version)
		return
	}

	absolute, err := filepath.Abs(*dir)
	if err != nil {
		fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	httpClient := &http.Client{}
	s := &setup.Setup{
		Dir:    absolute,
		Ask:    &prompt.Terminal{In: os.Stdin, Out: os.Stdout},
		Docker: docker.CLI{Dir: absolute, Out: os.Stdout},
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
			fmt.Fprintln(os.Stderr, "streampack-setup: -hard-reset goes with no command, or with init")
			os.Exit(2)
		}
		check(s.HardReset(ctx))
	}

	switch command {
	case "":
		configured, err := s.Configured()
		if err == nil && !configured {
			err = s.Init(ctx)
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
		check(s.Up(ctx))
	case "console":
		check(s.Console(ctx))
	case "secrets":
		check(s.MoveSecrets(ctx))
	default:
		flags.Usage()
		os.Exit(2)
	}
}

func check(err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "streampack-setup:", err)
	os.Exit(1)
}
