package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHighestComparesNumbersNotText(t *testing.T) {
	got, err := Highest([]string{"1.9.3", "1.15.7", "1.2.15", "latest", "1.1.9a", "1.15.10b"})
	if err != nil || got != "1.15.7" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := Highest([]string{"latest"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestLatestReadsTheTagList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"server-streampack","tags":["1.10.0","1.9.9","latest"]}`))
	}))
	defer server.Close()
	got, err := Latest(context.Background(), server.Client(), server.URL)
	if err != nil || got != "1.10.0" {
		t.Fatalf("got %q, %v", got, err)
	}
}
