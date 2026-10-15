# Start Over

Use a hard reset when the deployment is broken and you don't want to repair it. For example:

- `.env` or `compose.yaml` is damaged, and `docker compose` can't read it.
- You lost the database password.
- A first run stopped halfway.

A hard reset deletes the database. Make a backup first if you need any of the data. See streampack's [Back Up and Restore PostgreSQL](https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/backup-and-restore-postgres.md).

## What It Deletes

- The deployment's containers, volumes and networks. The database is in a volume.
- `.env`.
- The compose file, under any name Compose reads (`compose.yaml`, `docker-compose.yml` and the others), and any override file (`compose.override.yaml` and the others).
- `media/`, with the posts' recordings. Only the database refers to them, and it's deleted too.

## What It Keeps

- `generative/`, with your prompt and mail templates.
- `ext/`, with your extension jars, banner and settings.
- The tool, and any other file in the directory.

On Linux, the recordings in `media/` belong to the container's root user. The reset deletes them from inside a container of the server's image, so you don't need `sudo`. If the server's container is already gone, the reset deletes `media/` directly. If that's refused, it stops and tells you to run `sudo rm -rf media`.

It finds the containers, volumes and networks by the Compose project's label. It doesn't read `.env` or the compose file, so it works when they're damaged.

## Steps

1. Go into the deployment directory:

   ```bash
   cd ~/streampack
   ```

2. Start the hard reset:

   ```bash
   ./streampack-setup -hard-reset
   ```

3. Read the list of what it will delete:

   ```text
   A hard reset deletes this deployment, and everything in its database:
     Containers: streampack-server-streampack-1, streampack-db-1
     Volumes: streampack_media, streampack_pgdata
     Networks: streampack_default
     Files in /home/you/streampack: .env, compose.yaml, media/
   Kept: ext/, generative/ (your templates and extensions)
   Type "streampack" to delete all of this:
   ```

4. Type the directory's name, and press Enter. Any other answer stops the reset, and nothing is deleted.

The tool shows each command as it deletes. Then it continues as a first run: it asks its questions, starts the server, and opens the console.

## Only the Reset

`./streampack-setup -hard-reset init` deletes the deployment, then asks the questions and writes the files. It doesn't start the server. Run `./streampack-setup` when you're ready.
