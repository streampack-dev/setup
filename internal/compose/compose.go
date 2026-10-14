// Package compose holds the compose file the setup tool writes: the one in streampack's
// docs/how-to/deploy/run-the-published-image.md, kept the same.
package compose

import (
	_ "embed"
	"errors"
	"io/fs"
	"os"
)

//go:embed compose.yaml
var File []byte

// Service is the server's service name in File.
const Service = "server-streampack"

// TokenPath is where, inside the server's container, the local admin token is written.
const TokenPath = "/run/streampack/admin-token"

// Write puts File at path unless something is there already; it says whether it wrote.
func Write(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	return true, os.WriteFile(path, File, 0o644)
}
