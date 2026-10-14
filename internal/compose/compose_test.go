package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheFileNamesWhatTheToolRelyOn(t *testing.T) {
	text := string(File)
	for _, want := range []string{
		"  " + Service + ":",
		"STREAMPACK_LOCAL_ADMIN_TOKEN_FILE: " + TokenPath,
		"- /run/streampack:mode=0700",
		"env_file: .env",
		"${STREAMPACK_VERSION:?",
		"${DB_PASSWORD:?",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("compose.yaml lacks %q", want)
		}
	}
}

func TestWriteLeavesAnExistingFileAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.yaml")
	if wrote, err := Write(path); err != nil || !wrote {
		t.Fatalf("first write: %v %v", wrote, err)
	}
	os.WriteFile(path, []byte("mine"), 0o644)
	if wrote, err := Write(path); err != nil || wrote {
		t.Fatalf("second write: %v %v", wrote, err)
	}
	if data, _ := os.ReadFile(path); string(data) != "mine" {
		t.Fatal("overwritten")
	}
}
