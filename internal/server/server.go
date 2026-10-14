// Package server talks to a running server-streampack: whether it's ready, which secrets its next
// start needs, and its web console.
package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNotSupported is a server too old for what was asked.
var ErrNotSupported = errors.New("the server doesn't support this; it predates streampack#209")

// Requirement is an environment variable the server's next start needs (GET /admin/secrets).
type Requirement struct {
	Variable    string `json:"variable"`
	Description string `json:"description"`
	State       string `json:"state"`
}

// Event is one of the console stream's events.
type Event struct {
	Name string
	Data string
}

// Result is a "result" event's data.
type Result struct {
	CorrelationID *string `json:"correlationId"`
	Status        string  `json:"status"`
	Text          string  `json:"text"`
}

// Client is a server at BaseURL, authenticated by whatever Token returns; fresh is true when the
// last token was refused, so it should be read again rather than reused.
type Client struct {
	BaseURL string
	HTTP    *http.Client
	Token   func(ctx context.Context, fresh bool) (string, error)
}

// WaitReady polls GET /features until it answers 200, ctx ends, or stopped says to give up.
func (c *Client) WaitReady(ctx context.Context, every time.Duration, stopped func() error) error {
	for {
		request, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/features", nil)
		if response, err := c.HTTP.Do(request); err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		if stopped != nil {
			if err := stopped(); err != nil {
				return err
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(every):
		}
	}
}

// Secrets lists the variables the next start needs.
func (c *Client) Secrets(ctx context.Context) ([]Requirement, error) {
	response, err := c.authorized(ctx, http.MethodGet, "/admin/secrets", nil, nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrNotSupported
	default:
		return nil, statusError(response)
	}
	var body struct {
		Required []Requirement `json:"required"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Required, nil
}

// Submit sends one console line; its output arrives on the stream.
func (c *Client) Submit(ctx context.Context, line string) error {
	body, _ := json.Marshal(map[string]string{"line": line})
	headers := map[string]string{"Content-Type": "application/json", "X-Web-Console": "1"}
	response, err := c.authorized(ctx, http.MethodPost, "/admin/console", body, headers)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return statusError(response)
	}
	return nil
}

// Stream reads the console's events, calling each, until the stream ends or ctx does. It returns
// nil when the server closed the stream (an expired token, say), so the caller can open another.
func (c *Client) Stream(ctx context.Context, each func(Event)) error {
	response, err := c.authorized(ctx, http.MethodGet, "/admin/console/stream", nil, map[string]string{"Accept": "text/event-stream"})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return statusError(response)
	}
	return readEvents(response.Body, each)
}

// readEvents parses server-sent events: "event:" and "data:" lines, ended by a blank line;
// comments (":heartbeat") are skipped.
func readEvents(r io.Reader, each func(Event)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var name string
	var data []string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if len(data) > 0 || name != "" {
				if name == "" {
					name = "message"
				}
				each(Event{Name: name, Data: strings.Join(data, "\n")})
			}
			name, data = "", nil
		case strings.HasPrefix(line, ":"):
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// authorized sends a request with the token, and once more with a fresh one if it's refused.
func (c *Client) authorized(ctx context.Context, method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		token, err := c.Token(ctx, attempt > 0)
		if err != nil {
			return nil, err
		}
		request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Authorization", "Bearer "+token)
		for k, v := range headers {
			request.Header.Set(k, v)
		}
		response, err := c.HTTP.Do(request)
		if err != nil {
			return nil, err
		}
		if response.StatusCode == http.StatusUnauthorized && attempt == 0 {
			response.Body.Close()
			continue
		}
		return response, nil
	}
}

func statusError(response *http.Response) error {
	var problem struct {
		Detail string `json:"detail"`
	}
	data, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	if json.Unmarshal(data, &problem) == nil && problem.Detail != "" {
		return fmt.Errorf("%s: %s", response.Status, problem.Detail)
	}
	return fmt.Errorf("%s", response.Status)
}
