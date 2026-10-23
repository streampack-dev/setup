package setup

import (
	"context"
	"errors"
	"fmt"

	"github.com/streampack-dev/setup/internal/envfile"
	"github.com/streampack-dev/setup/internal/registry"
)

// backupGuide is streampack's page on backing up the database, shown before an update.
const backupGuide = "https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/backup-and-restore-postgres.md"

// releaseNotes lists streampack's releases and their notes.
const releaseNotes = "https://github.com/streampack-dev/streampack/releases"

// Update moves STREAMPACK_VERSION in .env to the newest server release, whatever version the
// deployment is on: 1.3.11 goes to 1.4.1 (#20). It writes the release's number, not the latest
// tag, so .env still says what runs. It shows the change and asks you to confirm before it writes
// anything. It never moves to an older release, and it changes nothing when the registry can't be
// reached. It doesn't pull or start the server: Up does that.
func (s *Setup) Update(ctx context.Context) error {
	configured, err := s.Configured()
	if err != nil {
		return err
	}
	if !configured {
		return errors.New("there's no deployment to update in " + s.Dir + ": run init first")
	}
	env, err := envfile.Read(s.envPath())
	if err != nil {
		return err
	}
	current, _ := env.Get("STREAMPACK_VERSION")
	if s.Latest == nil {
		return errors.New("can't find the newest release; .env is unchanged")
	}
	newest, err := s.Latest(ctx)
	if err != nil {
		return fmt.Errorf("couldn't find the newest release (%w); .env is unchanged", err)
	}
	if current == newest {
		fmt.Fprintf(s.Out, "The server is already on the newest release, %s.\n", current)
		return nil
	}
	if registry.IsRelease(current) && registry.Less(newest, current) {
		fmt.Fprintf(s.Out, "STREAMPACK_VERSION %s is newer than the registry's newest release, %s; it stays as it is.\n",
			current, newest)
		return nil
	}

	fmt.Fprintf(s.Out, "Update the server: STREAMPACK_VERSION %s -> %s\n", current, newest)
	fmt.Fprintln(s.Out, "Read the release notes first: "+releaseNotes)
	fmt.Fprintln(s.Out, "Back up the database first. A migration can't be undone by starting an older server: "+backupGuide)
	// No answer at all (input ended) is a no.
	if yes, err := s.Ask.Confirm("Update to "+newest+"?", false); err != nil || !yes {
		return errors.New("update stopped; .env is unchanged")
	}
	env.Set("STREAMPACK_VERSION", newest)
	if err := env.Write(s.envPath()); err != nil {
		return err
	}
	fmt.Fprintf(s.Out, "STREAMPACK_VERSION is now %s.\n", newest)
	return nil
}
