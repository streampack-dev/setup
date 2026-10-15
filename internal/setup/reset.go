package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/streampack-dev/setup/internal/compose"
)

// HardReset deletes the deployment: the project's containers, volumes and networks, and its
// configuration (.env, the compose file and any override, generative/), so the next step is a
// first run (#7). It finds them by the project's label and by name, never by reading .env or the
// compose file, which may be what's broken. The tool itself, and any other file, stay.
func (s *Setup) HardReset(ctx context.Context) error {
	resources, err := s.Docker.Resources(ctx)
	if err != nil {
		return err
	}
	var files []string
	for _, name := range append(append([]string{".env"}, compose.Names...), compose.Overrides...) {
		if _, err := os.Lstat(filepath.Join(s.Dir, name)); err == nil {
			files = append(files, name)
		}
	}
	if _, err := os.Lstat(filepath.Join(s.Dir, "generative")); err == nil {
		files = append(files, "generative/")
	}
	if resources.Empty() && len(files) == 0 {
		fmt.Fprintln(s.Out, "Nothing to delete: there's no deployment in "+s.Dir+".")
		return nil
	}

	fmt.Fprintln(s.Out, "A hard reset deletes this deployment, and everything in its database:")
	for _, list := range []struct {
		what  string
		names []string
	}{
		{"Containers", resources.Containers},
		{"Volumes", resources.Volumes},
		{"Networks", resources.Networks},
		{"Files in " + s.Dir, files},
	} {
		if len(list.names) > 0 {
			fmt.Fprintf(s.Out, "  %s: %s\n", list.what, strings.Join(list.names, ", "))
		}
	}
	name := filepath.Base(s.Dir)
	answer, err := s.Ask.Ask(fmt.Sprintf("Type %q to delete all of this", name), "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(answer) != name {
		return errors.New("hard reset stopped; nothing was deleted")
	}

	if err := s.Docker.RemoveResources(ctx, resources); err != nil {
		return err
	}
	for _, file := range files {
		fmt.Fprintln(s.Out, "$ rm -rf "+file)
		if err := os.RemoveAll(filepath.Join(s.Dir, strings.TrimSuffix(file, "/"))); err != nil {
			return err
		}
	}
	fmt.Fprintln(s.Out, "Deleted. Starting again from nothing.")
	return nil
}
