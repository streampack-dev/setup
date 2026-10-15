# Upgrade the Server

The server's version is `STREAMPACK_VERSION` in `.env`. The tool never changes it after `init`.

1. Back up the database first. See streampack's [Back Up and Restore PostgreSQL](https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/backup-and-restore-postgres.md).
2. Find the version you want in streampack's [releases](https://github.com/streampack-dev/streampack/releases), and read its notes.
3. Change the line in `.env`:

   ```text
   STREAMPACK_VERSION=1.16.0
   ```

4. Pull it and start it:

   ```bash
   streampack-setup up
   ```

The server applies its database migrations as it starts, and `up` waits for it.

To go back, put the old version in `.env` and restore the backup. A migration can't be undone by starting an older server.
