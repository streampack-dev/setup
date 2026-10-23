# Upgrade the Server

The server's version is `STREAMPACK_VERSION` in `.env`. The tool changes it only when you ask, with `--update`.

## Update to the Newest Release

1. Back up the database first. See streampack's [Back Up and Restore PostgreSQL](https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/backup-and-restore-postgres.md).
2. Run the tool with `--update`:

   ```bash
   ./streampack-setup up --update
   ```

   It finds the newest release in the registry and shows the change:

   ```text
   Update the server: STREAMPACK_VERSION 1.3.11 -> 1.4.1
   ```

   The newest release can be in a different minor or major version. Read its notes in streampack's [releases](https://github.com/streampack-dev/streampack/releases).

3. Type `y` to update. The tool writes the new number in `.env`, then pulls the server and starts it. Any other answer stops the tool, and `.env` stays as it was.

The server applies its database migrations as it starts, and `up` waits for it.

Without `up`, `./streampack-setup --update` updates, starts the server, and then opens the console.

If the server is already on the newest release, the tool says so and starts it as usual. If the registry can't be reached, the tool stops and `.env` stays as it was. The tool never moves the server to an older release.

## Update to a Version You Choose

1. Back up the database first.
2. Find the version you want in streampack's [releases](https://github.com/streampack-dev/streampack/releases), and read its notes.
3. Change the line in `.env`:

   ```text
   STREAMPACK_VERSION=1.16.0
   ```

4. Pull it and start it:

   ```bash
   ./streampack-setup up
   ```

## Go Back

Put the old version in `.env` and restore the backup. A migration can't be undone by starting an older server.
