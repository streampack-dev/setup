package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/streampack-dev/setup/internal/envfile"
)

// configuredAt writes a configured .env pinned to version, and returns its path.
func configuredAt(t *testing.T, s *Setup, version string) string {
	t.Helper()
	path := filepath.Join(s.Dir, ".env")
	body := "STREAMPACK_VERSION=" + version + "\nDB_PASSWORD=pw\nJWT_SECRET=secret\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func pinned(t *testing.T, path string) string {
	t.Helper()
	env, err := envfile.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	v, _ := env.Get("STREAMPACK_VERSION")
	return v
}

// 1.3.11 goes to 1.4.1, across the minor version, once you confirm (setup#20).
func TestUpdateMovesToTheNewestReleaseOnceConfirmed(t *testing.T) {
	s, ask, out := newSetup(t, []string{"y"}, &fakeDocker{}, "")
	s.Latest = func(context.Context) (string, error) { return "1.4.1", nil }
	path := configuredAt(t, s, "1.3.11")

	if err := s.Update(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := pinned(t, path); got != "1.4.1" {
		t.Errorf("STREAMPACK_VERSION %q, want 1.4.1", got)
	}
	for _, said := range []string{"1.3.11 -> 1.4.1", "Back up the database", "backup-and-restore-postgres"} {
		if !strings.Contains(out.String(), said) {
			t.Errorf("didn't say %q:\n%s", said, out)
		}
	}
	if len(ask.asked) != 1 {
		t.Errorf("asked %v", ask.asked)
	}
	// The rest of .env is kept.
	env, _ := envfile.Read(path)
	if v, _ := env.Get("JWT_SECRET"); v != "secret" {
		t.Errorf("JWT_SECRET %q", v)
	}
}

func TestUpdateChangesNothingWhenDeclined(t *testing.T) {
	for _, answers := range [][]string{{"n"}, {""}, nil} {
		s, _, out := newSetup(t, answers, &fakeDocker{}, "")
		s.Latest = func(context.Context) (string, error) { return "1.4.1", nil }
		path := configuredAt(t, s, "1.3.11")

		err := s.Update(context.Background())
		if err == nil || !strings.Contains(err.Error(), "unchanged") {
			t.Errorf("%v: got %v\n%s", answers, err, out)
		}
		if got := pinned(t, path); got != "1.3.11" {
			t.Errorf("%v: STREAMPACK_VERSION %q", answers, got)
		}
	}
}

func TestUpdateOnTheNewestAsksNothing(t *testing.T) {
	s, ask, out := newSetup(t, nil, &fakeDocker{}, "")
	path := configuredAt(t, s, "1.16.0")

	if err := s.Update(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(ask.asked) != 0 || pinned(t, path) != "1.16.0" || !strings.Contains(out.String(), "already on the newest") {
		t.Errorf("asked %v, pinned %s:\n%s", ask.asked, pinned(t, path), out)
	}
}

func TestUpdateNeverMovesToAnOlderRelease(t *testing.T) {
	s, ask, out := newSetup(t, nil, &fakeDocker{}, "")
	path := configuredAt(t, s, "1.17.2")

	if err := s.Update(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(ask.asked) != 0 || pinned(t, path) != "1.17.2" || !strings.Contains(out.String(), "newer than") {
		t.Errorf("asked %v, pinned %s:\n%s", ask.asked, pinned(t, path), out)
	}
}

func TestUpdateStopsWhenTheRegistryCantBeReached(t *testing.T) {
	s, ask, _ := newSetup(t, nil, &fakeDocker{}, "")
	s.Latest = func(context.Context) (string, error) { return "", errors.New("no route to host") }
	path := configuredAt(t, s, "1.3.11")

	err := s.Update(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no route to host") || !strings.Contains(err.Error(), "unchanged") {
		t.Errorf("got %v", err)
	}
	if len(ask.asked) != 0 || pinned(t, path) != "1.3.11" {
		t.Errorf("asked %v, pinned %s", ask.asked, pinned(t, path))
	}
}

// A tag that isn't a release, such as latest, is moved to the newest number too.
func TestUpdateReplacesATagThatIsntARelease(t *testing.T) {
	s, _, out := newSetup(t, []string{"y"}, &fakeDocker{}, "")
	path := configuredAt(t, s, "latest")

	if err := s.Update(context.Background()); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if got := pinned(t, path); got != "1.16.0" {
		t.Errorf("STREAMPACK_VERSION %q", got)
	}
}

func TestUpdateNeedsAConfiguredDeployment(t *testing.T) {
	s, _, _ := newSetup(t, nil, &fakeDocker{}, "")
	if err := s.Update(context.Background()); err == nil || !strings.Contains(err.Error(), "init") {
		t.Errorf("got %v", err)
	}
}
