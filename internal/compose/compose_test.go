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
		"- ./media:/opt/streampack/media",
		"- ./ext:/opt/streampack/ext:ro",
		"- ./generative:/opt/streampack/generative:ro",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("compose.yaml lacks %q", want)
		}
	}
}

// Media is a directory on the host now, not a named volume (setup#9).
func TestMediaIsNotANamedVolume(t *testing.T) {
	if strings.Contains(string(File), "- media:/opt/streampack/media") || strings.Contains(string(File), "\n  media:") {
		t.Error("compose.yaml still has the named media volume")
	}
}

func TestWriteLeavesAnExistingFileAlone(t *testing.T) {
	dir := t.TempDir()
	if wrote, err := Write(dir); err != nil || !wrote {
		t.Fatalf("first write: %v %v", wrote, err)
	}
	path := filepath.Join(dir, "compose.yaml")
	os.WriteFile(path, []byte("mine"), 0o644)
	if wrote, err := Write(dir); err != nil || wrote {
		t.Fatalf("second write: %v %v", wrote, err)
	}
	if data, _ := os.ReadFile(path); string(data) != "mine" {
		t.Fatal("overwritten")
	}
}

// Compose prefers compose.yaml, so writing one beside a docker-compose.yml would replace it.
func TestWriteLeavesAComposeFileByAnotherNameAlone(t *testing.T) {
	for _, name := range []string{"compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, name), []byte("mine"), 0o644)
		if wrote, err := Write(dir); err != nil || wrote {
			t.Fatalf("%s: wrote %v, %v", name, wrote, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "compose.yaml")); err == nil {
			t.Fatalf("%s: compose.yaml written beside it", name)
		}
	}
}
