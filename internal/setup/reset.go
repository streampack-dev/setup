package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/streampack-dev/setup/internal/compose"
)

// keptDirectories hold what you wrote or added, not what the tool made: a hard reset leaves them.
var keptDirectories = []string{"ext", "generative"}

// HardReset deletes the deployment: the project's containers, volumes and networks, its
// configuration (.env, the compose file and any override), and media/, so the next step is a first
// run (#7). media/'s recordings are reached only through the database, which goes too (#9). It
// finds them by the project's label and by name, never by reading .env or the compose file, which
// may be what's broken. What you wrote or added stays: generative/ and ext/ (#9), the tool, and any
// other file.
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
	media := filepath.Join(s.Dir, "media")
	if _, err := os.Lstat(media); err == nil {
		files = append(files, "media/")
	}
	var kept []string
	for _, dir := range keptDirectories {
		if _, err := os.Lstat(filepath.Join(s.Dir, dir)); err == nil {
			kept = append(kept, dir+"/")
		}
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
	if len(kept) > 0 {
		fmt.Fprintf(s.Out, "Kept: %s (your templates and extensions)\n", strings.Join(kept, ", "))
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
	// On Linux the server's recordings belong to the container's root, which you may not be, so
	// they're cleared from inside a container of the server's image when there is one.
	if slices.Contains(files, "media/") && resources.ServerImage != "" {
		if err := s.Docker.ClearDirectory(ctx, resources.ServerImage, media); err != nil {
			return fmt.Errorf("couldn't clear media/: %w", err)
		}
	}
	for _, file := range files {
		fmt.Fprintln(s.Out, "$ rm -rf "+file)
		if err := os.RemoveAll(filepath.Join(s.Dir, strings.TrimSuffix(file, "/"))); err != nil {
			if file == "media/" && errors.Is(err, fs.ErrPermission) {
				return fmt.Errorf("couldn't delete media/, whose recordings belong to the container's "+
					"root: delete it with `sudo rm -rf %s`, then run this again", media)
			}
			return err
		}
	}
	fmt.Fprintln(s.Out, "Deleted. Starting again from nothing.")
	return nil
}
