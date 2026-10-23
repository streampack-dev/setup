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

func TestLessComparesReleasesAsNumbers(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"1.3.11", "1.4.1", true},
		{"1.4.1", "1.3.11", false},
		{"1.9.0", "1.10.0", true},
		{"1.99.9", "2.0.0", true},
		{"1.4.1", "1.4.1", false},
	} {
		if got := Less(c.a, c.b); got != c.want {
			t.Errorf("Less(%s, %s) = %v", c.a, c.b, got)
		}
	}
	if !IsRelease("1.4.1") || IsRelease("latest") || IsRelease("1.4.1a") {
		t.Error("IsRelease")
	}
}
