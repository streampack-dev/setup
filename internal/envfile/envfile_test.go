package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetReplacesInPlaceAndKeepsTheRest(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("# comment\nA=1\n\nB=2\n"), 0o644)

	f, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Set("A", "one")
	f.Set("C", "three")
	if err := f.Write(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if got, want := string(data), "# comment\nA=one\n\nB=2\nC=three\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v, want 0600", info.Mode().Perm())
	}
}

func TestValuesComeBackAsTheyWent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	values := map[string]string{
		"PLAIN":  "abc-123_x.y/z:@+=",
		"SPACES": "a b c",
		"HASH":   "a#b",
		"QUOTE":  `say "hi"`,
		"DOLLAR": "pa$word",
		"SLASH":  `back\slash`,
		"APOS":   `it's $5`,
	}
	f := &File{}
	for k, v := range values {
		f.Set(k, v)
	}
	if err := f.Write(path); err != nil {
		t.Fatal(err)
	}
	again, _ := Read(path)
	for k, v := range values {
		if got, ok := again.Get(k); !ok || got != v {
			t.Errorf("%s: got %q, want %q", k, got, v)
		}
	}
}

func TestAMissingFileReadsAsEmpty(t *testing.T) {
	f, err := Read(filepath.Join(t.TempDir(), "none"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.Get("A"); ok {
		t.Fatal("expected nothing")
	}
}

func TestReadsExportAndSingleQuotes(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(path, []byte("export A='x y'\n"), 0o600)
	f, _ := Read(path)
	if got, _ := f.Get("A"); got != "x y" {
		t.Fatalf("got %q", got)
	}
}
