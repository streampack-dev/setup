package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeTokens struct {
	tokens []string
	reads  int
}

func (f *fakeTokens) next(_ context.Context, fresh bool) (string, error) {
	if fresh || f.reads == 0 {
		f.reads++
	}
	return f.tokens[min(f.reads, len(f.tokens))-1], nil
}

func TestSecretsReadsTheListAndRetriesARefusedTokenOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/secrets" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer new" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"required":[{"variable":"MATTERMOST_WORK_TOKEN","description":"Mattermost server 'work' token","state":"STORED_IN_DATABASE"}]}`)
	}))
	defer server.Close()
	tokens := &fakeTokens{tokens: []string{"old", "new"}}
	client := &Client{BaseURL: server.URL, HTTP: server.Client(), Token: tokens.next}

	got, err := client.Secrets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Variable != "MATTERMOST_WORK_TOKEN" || got[0].State != "STORED_IN_DATABASE" {
		t.Fatalf("got %+v", got)
	}
	if tokens.reads != 2 {
		t.Fatalf("token read %d times", tokens.reads)
	}
}

func TestSecretsOnAnOlderServerIsNotSupported(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client(), Token: (&fakeTokens{tokens: []string{"t"}}).next}
	if _, err := client.Secrets(context.Background()); err != ErrNotSupported {
		t.Fatalf("got %v", err)
	}
}

func TestSubmitSendsTheLineAsTheConsoleExpects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/admin/console" || r.Header.Get("X-Web-Console") != "1" ||
			r.Header.Get("Content-Type") != "application/json" || string(body) != `{"line":"version"}` {
			t.Errorf("%s %s %v %s", r.Method, r.URL.Path, r.Header, body)
		}
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, `{"correlationId":"c1"}`)
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client(), Token: (&fakeTokens{tokens: []string{"t"}}).next}
	if err := client.Submit(context.Background(), "version"); err != nil {
		t.Fatal(err)
	}
}

func TestSubmitSaysWhatWasRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"detail":"Too many console commands"}`)
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client(), Token: (&fakeTokens{tokens: []string{"t"}}).next}
	err := client.Submit(context.Background(), "x")
	if err == nil || !strings.Contains(err.Error(), "Too many console commands") {
		t.Fatalf("got %v", err)
	}
}

func TestReadEventsParsesEventsAndSkipsHeartbeats(t *testing.T) {
	input := "event: ready\ndata: {\"username\":\"root\"}\n\n:heartbeat\n\nevent: result\ndata: line one\ndata: line two\n\n"
	var got []Event
	if err := readEvents(strings.NewReader(input), func(e Event) { got = append(got, e) }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "ready" || got[1].Data != "line one\nline two" {
		t.Fatalf("got %+v", got)
	}
}

func TestWaitReadyPollsUntilFeaturesAnswers(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	if err := client.WaitReady(context.Background(), time.Millisecond, nil); err != nil || calls != 3 {
		t.Fatalf("%v after %d calls", err, calls)
	}
}

func TestWaitReadyGivesUpWhenTheServerHasStopped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := &Client{BaseURL: server.URL, HTTP: server.Client()}
	stopped := io.ErrUnexpectedEOF
	if err := client.WaitReady(context.Background(), time.Millisecond, func() error { return stopped }); err != stopped {
		t.Fatalf("got %v", err)
	}
}
