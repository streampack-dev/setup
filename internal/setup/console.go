package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/streampack-dev/setup/internal/server"
)

const consoleHelp = `Type commands as at the server's console, for example:
  mattermost connect work https://mattermost.example.com <token>
  gitlab instance add https://gitlab.example.com <token>
  secrets
Tool commands: :secrets moves credentials given here into .env; :help; :quit (or Ctrl-D) leaves.`

// Console opens the server's web console as the super-admin and passes lines to it until input
// ends or :quit. Leaving, it offers to move any credentials given into .env.
func (s *Setup) Console(ctx context.Context) error {
	streaming, stopStream := context.WithCancel(ctx)
	defer stopStream()

	ready := make(chan string, 1)
	go s.stream(streaming, ready)
	select {
	case username := <-ready:
		fmt.Fprintf(s.Out, "Console open as %s.\n%s\n", username, consoleHelp)
	case <-time.After(30 * time.Second):
		return errors.New("the console stream didn't open within 30 seconds")
	case <-ctx.Done():
		return ctx.Err()
	}

	for {
		line, err := s.Ask.Line("> ")
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(s.Out)
			break
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		switch line {
		case "":
			continue
		case ":quit", ":exit":
		case ":help":
			fmt.Fprintln(s.Out, consoleHelp)
			continue
		case ":secrets":
			if err := s.MoveSecrets(ctx); err != nil {
				fmt.Fprintln(s.Out, "error:", err)
			}
			continue
		default:
			if err := s.Server.Submit(ctx, line); err != nil {
				fmt.Fprintln(s.Out, "error:", err)
			}
			continue
		}
		break
	}
	// The handoff may recreate the server; the console is done with, so its stream needn't follow.
	stopStream()
	return s.offerSecrets(ctx)
}

// offerSecrets asks, on the way out, whether to move credentials still waiting into .env.
func (s *Setup) offerSecrets(ctx context.Context) error {
	needed, err := s.Server.Secrets(ctx)
	if errors.Is(err, server.ErrNotSupported) {
		return nil
	}
	if err != nil || len(needed) == 0 {
		return err
	}
	move, err := s.Ask.Confirm(fmt.Sprintf("%d credentials given at the console still need a place in .env; "+
		"the next start fails without them. Move them now?", len(needed)), true)
	if err != nil || !move {
		if err == nil {
			fmt.Fprintln(s.Out, "Run `streampack-setup secrets` before the server next restarts.")
		}
		return err
	}
	return s.MoveSecrets(ctx)
}

// stream prints the console's output, opening the stream again whenever it closes (an expired
// token, a restart) until ctx ends. It sends the username to ready the first time it opens.
func (s *Setup) stream(ctx context.Context, ready chan<- string) {
	opened := false
	said := ""
	for ctx.Err() == nil {
		err := s.Server.Stream(ctx, func(e server.Event) {
			switch e.Name {
			case "ready":
				var body struct {
					Username string `json:"username"`
				}
				json.Unmarshal([]byte(e.Data), &body)
				if !opened {
					opened = true
					ready <- body.Username
				}
			case "result":
				fmt.Fprintln(s.Out, formatResult(e.Data))
			}
		})
		if ctx.Err() != nil {
			return
		}
		// Said once, not every two seconds while the server restarts.
		message := "(console stream closed; opening it again. Output in between isn't replayed.)"
		if err != nil {
			message = "(console stream: " + err.Error() + "; trying again)"
		}
		if (err != nil || opened) && message != said {
			fmt.Fprintln(s.Out, message)
			said = message
		} else if err == nil {
			said = ""
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func formatResult(data string) string {
	var result server.Result
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return data
	}
	text := result.Text
	switch result.Status {
	case "error":
		text = "error: " + text
	case "unhandled":
		text = "(no command matched that)"
	}
	if result.CorrelationID == nil {
		text = "(notice) " + text
	}
	return text
}
