// Package envfile reads and edits a Docker Compose .env file in place: keys are replaced where
// they are, new ones are appended, and everything else (comments, order, blank lines) is kept.
package envfile

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var assignment = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$`)

// plain is a value that needs no quoting in a Compose .env file.
var plain = regexp.MustCompile(`^[A-Za-z0-9_./:@+=,-]*$`)

// File is a .env file's lines.
type File struct {
	lines []string
}

// Read loads path; a file that doesn't exist reads as empty.
func Read(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &File{}, nil
	}
	if err != nil {
		return nil, err
	}
	f := &File{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		f.lines = append(f.lines, scanner.Text())
	}
	return f, scanner.Err()
}

// Get returns key's value, unquoted, and whether it's set.
func (f *File) Get(key string) (string, bool) {
	for _, line := range f.lines {
		if m := assignment.FindStringSubmatch(line); m != nil && m[1] == key {
			return unquote(strings.TrimSpace(m[2])), true
		}
	}
	return "", false
}

// Set replaces key's value where it is, or appends it.
func (f *File) Set(key, value string) {
	line := key + "=" + quote(value)
	for i, existing := range f.lines {
		if m := assignment.FindStringSubmatch(existing); m != nil && m[1] == key {
			f.lines[i] = line
			return
		}
	}
	f.lines = append(f.lines, line)
}

// Append adds a line as it is, such as a comment.
func (f *File) Append(line string) { f.lines = append(f.lines, line) }

// Write saves the file at path, readable and writable by its owner alone, replacing what was there
// in one move so it's never seen half-written.
func (f *File) Write(path string) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".env-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	content := strings.Join(f.lines, "\n")
	if content != "" {
		content += "\n"
	}
	if _, err := temporary.WriteString(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporary.Name(), path)
}

// quote writes value so Compose reads it back unchanged: as it is when it can be; in single quotes,
// which Compose takes literally, when it has no single quote or newline of its own; otherwise in
// double quotes, with backslashes, quotes, newlines and dollar signs escaped.
func quote(value string) string {
	if plain.MatchString(value) {
		return value
	}
	if !strings.ContainsAny(value, "'\n") {
		return "'" + value + "'"
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `$$`, "\n", `\n`).Replace(value)
	return `"` + escaped + `"`
}

func unquote(value string) string {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return strings.NewReplacer(`\\`, `\`, `\"`, `"`, `$$`, `$`, `\n`, "\n").Replace(value[1 : len(value)-1])
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}
