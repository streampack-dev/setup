// Package compose holds the compose file the setup tool writes: the one in streampack's
// docs/how-to/deploy/run-the-published-image.md, kept the same.
package compose

import (
	_ "embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed compose.yaml
var File []byte

// Service is the server's service name in File.
const Service = "server-streampack"

// TokenPath is where, inside the server's container, the local admin token is written.
const TokenPath = "/run/streampack/admin-token"

// Names are the file names Compose looks for, in its order of preference.
var Names = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

// Overrides are the override files Compose merges into the compose file when they're there.
var Overrides = []string{"compose.override.yaml", "compose.override.yml", "docker-compose.override.yaml", "docker-compose.override.yml"}

// Write puts File in dir as compose.yaml unless dir has a compose file by any of Names already; it
// says whether it wrote.
func Write(dir string) (bool, error) {
	for _, name := range Names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return false, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	}
	return true, os.WriteFile(filepath.Join(dir, Names[0]), File, 0o644)
}
