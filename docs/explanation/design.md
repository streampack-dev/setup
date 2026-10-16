# Why It's Built This Way

## It Doesn't Know About Integrations

The tool asks for no Mattermost, GitLab or IRC details. Those integrations are set up at the console, with the server's own commands, exactly as on a server set up by hand. There are two reasons:

- **One way of doing it.** Every integration already has console commands, documented in streampack's [Admin Text Operations](https://github.com/streampack-dev/streampack/blob/main/docs/reference/admin-text-operations.md). If the tool asked its own questions, there would be a second way, and it would drift from the first.
- **The server knows what it needs.** A credential typed at the console is stored, and the server can say which environment variable it belongs in (`GET /admin/secrets`). The tool asks for exactly those variables. A new integration in a later server release needs no change to the tool.

Discord is the exception. It needs its three values before the server starts, so `init` asks for them.

## It Signs In with a Token File

The tool has no password to type and keeps no API key. The server writes a short-lived super-admin token to `/run/streampack/admin-token`, on a `tmpfs` inside its own container, and the tool reads it with `docker compose exec`. Anyone who can do that can also read `.env` and the database, so the file gives nobody access they didn't already have. The token is never on the host's disk, and it expires an hour after it's written. streampack's [The Local Admin Token](https://github.com/streampack-dev/streampack/blob/main/docs/explanation/local-admin-token.md) has the whole argument.

## Credentials Arrive in Two Steps

A credential is typed at the console, then moved into `.env`. The server can't write `.env`: that file belongs to the host, and the server runs in a container. The tool can, so it does the second step. streampack's [How Credentials Reach the Environment](https://github.com/streampack-dev/streampack/blob/main/docs/explanation/credentials-and-the-environment.md) explains why credentials live in the environment at all.

You type each credential twice, once at the console and again when the tool asks. That's deliberate: the server never sends a credential back, not even to the super-admin.

## `up -d`, Not `restart`

Compose reads `.env` when it creates a container, not when it restarts one. After `.env` changes, the tool runs `docker compose up -d`, which recreates the server container because its configuration changed. The database container is left as it is. podman-compose doesn't see a change to `.env`, so with it the tool runs `up -d --force-recreate`, which recreates the database container too; its data is in the volume, and stays.

## Every Step Is a Compose Command

The tool runs Compose in the deployment directory and prints each command as it runs it. Nothing it does is hidden: you can run any step by hand, read `compose.yaml`, or stop using the tool altogether. The directory is an ordinary Compose deployment, the same one streampack's [Run the Published Image with Docker Compose](https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/run-the-published-image.md) builds by hand.

## Any Container Engine

The tool doesn't mind whether Docker or Podman runs the containers. It finds out at the start of each run, and uses that engine's own commands, so what it prints is what ran. It asks each engine only what both answer the same way. Docker's `{{.Label}}` template function, for example, is unknown to Podman, and Podman's `{{index .Labels}}` fails on Docker, so the tool filters by labels and never prints them. Where the engines differ, the tool works with both:

- podman-compose puts the containers in a pod, `pod_<project>`. A hard reset removes it.
- podman-compose turns `depends_on` into a dependency Podman enforces: it won't remove the database while the server needs it. The tool removes containers with `--depend`, which takes the dependents too.
- podman-compose labels a volume with its project but not its name, so the tool finds a volume by its full name, `<project>_<name>`, which both name it.
- Podman won't guess the registry for a short image name, so `compose.yaml` names `postgres` in full.
- Fedora, RHEL and Rocky Linux enforce SELinux, so the host directories are mounted with `z`. Elsewhere, `z` does nothing.

[What It Uses](../reference/server-interface.md) lists the commands.

## Repeatable

Each step checks what's already done. `init` asks only for what `.env` lacks, and never overwrites `compose.yaml`. `up` on a running deployment changes nothing unless the configuration has. So the answer to "what now?" is always to run the tool again.
