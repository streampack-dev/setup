# What It Uses

Everything the tool does goes through Docker Compose or the server's HTTP API.

## Docker Compose

Every command runs in the deployment directory. The commands that change something are printed as they run, starting with `$`.

| Command | Used by | Printed |
|---------|---------|---------|
| `docker compose pull` | `up` | yes |
| `docker compose up -d` | `up`, `secrets` | yes |
| `docker compose logs --no-color --tail 300 server-streampack` | `up`, while waiting | no |
| `docker compose exec -T server-streampack cat /run/streampack/admin-token` | `console`, `secrets` | no |

## HTTP

All requests go to `-url` (by default `http://127.0.0.1:8080`). The admin endpoints carry `Authorization: Bearer <token>` with the local admin token. On a 401, the tool reads the token again and retries once.

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
