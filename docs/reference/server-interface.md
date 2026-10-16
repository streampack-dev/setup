# What It Uses

Everything the tool does goes through Compose, the container engine, or the server's HTTP API.

## The Container Engine

The tool works with Docker or Podman. At the start of each run it finds which one is here, and prints it (`Containers: podman, with podman-compose`):

1. Docker, when the `docker` command is Docker's own and its daemon answers (`docker version`). Compose is `docker compose`, or `docker-compose` without the plugin.
2. Otherwise Podman, when `podman info` works. A `docker` command that is Podman's (`podman-docker`) counts as Podman, and the tool runs `podman` by its own name. Compose is `podman-compose`, or else `podman compose`.

podman-compose must be 1.2.0 or later: older ones give the server `.env`'s quotes as part of each value. The tool stops with that message.

Below, `<compose>` is the Compose command and `<engine>` is `docker` or `podman`. The tool asks the engine only for what both answer the same way: no label templates, and labels only in filters.

## Compose

Every command runs in the deployment directory. The commands that change something are printed as they run, starting with `$`.

| Command | Used by | Printed |
|---------|---------|---------|
| `<compose> pull` | `up` | yes |
| `<compose> up -d` (with podman-compose, `up -d --force-recreate`) | `up`, `secrets` | yes |
| `<compose> logs --no-color --tail 300 server-streampack` | `up`, while waiting | no |
| `<compose> exec -T server-streampack cat /run/streampack/admin-token` | `console`, `secrets` | no |

podman-compose makes `depends_on` a Podman dependency, and Podman won't remove a container that another still requires. So with Podman, `--depend` removes the dependents with it, and `--ignore` skips names already gone.

podman-compose recreates a container only when the compose file changes, not `.env`, so `up` tells it to.

Plain engine commands are used too, without the deployment directory, since `init` can run before it has anything in it:

| Command | Used by | Printed |
|---------|---------|---------|
| `<engine> volume ls --filter label=com.docker.compose.project=<project> --format {{.Name}}`, looking for `<project>_pgdata` | `init`, when `.env` has no `DB_PASSWORD` | no |
| `<engine> rm -f <the project's containers>` (Podman: `podman rm -f --depend --ignore`), then `<engine> volume rm <volume>` | `init`, when you agree to start the database afresh | yes |

A hard reset uses these, also without reading `.env` or the compose file:

| Command | Printed |
|---------|---------|
| `<engine> ps -a`, `<engine> volume ls` and `<engine> network ls`, each with `--filter label=com.docker.compose.project=<project>` | no |
| `<engine> ps -a`, also filtered by `label=com.docker.compose.service=server-streampack`, for the server's image | no |
| With Podman, `podman pod ls`, for podman-compose's pod `pod_<project>` | no |
| `<engine> rm -f <containers>` (Podman: `podman rm -f --depend --ignore`), `podman pod rm -f <pod>`, `<engine> volume rm <volumes>`, `<engine> network rm <networks>` | yes |
| `<engine> run --rm -v <dir>/media:/clear:z …`, to empty `media/` as the container's root | yes |

`<project>` is the Compose project name: `COMPOSE_PROJECT_NAME` if it's set in the environment, otherwise the directory's name, lower-cased, keeping only letters, digits, `-` and `_`.

## HTTP

All requests go to `--url` (by default `http://127.0.0.1:8080`). The admin endpoints carry `Authorization: Bearer <token>` with the local admin token. On a 401, the tool reads the token again and retries once.

| Request | Used by | For |
|---------|---------|-----|
| `GET /features` | `up`, `secrets` | Readiness: a 200 means ready. |
| `GET /admin/secrets` | `secrets`, leaving `console` | The variables the next start needs. A 404 means the server predates streampack#209. |
| `GET /admin/console/stream` (`text/event-stream`) | `console` | The console's output. The tool reads the `ready` event (the username) and `result` events. |
| `POST /admin/console` | `console` | One line. |

streampack's [Blog HTTP API](https://github.com/streampack-dev/streampack/blob/main/docs/reference/blog-http-api.md) documents these endpoints.

## The Registry

`init` looks up the newest version at `https://nexus.streampack.dev/v2/images/server-streampack/tags/list`. It takes the highest tag of the form `X.Y.Z` and waits at most 10 seconds.

## Server Versions

The local admin token and `GET /admin/secrets` arrived with streampack#209. Against an older server, `console` reports that there's no token, `secrets` says the server doesn't support it, and leaving the console skips the offer to move credentials.
