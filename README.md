# streampack-setup

Brings up a [server-streampack](https://github.com/streampack-dev/streampack) on a Linux box with Docker, then acts as its console: the commands you'd type at the server's own console, from your shell, as the super-admin.

It asks only what starting the server needs. Integrations (Mattermost, GitLab, IRC, Slack, …) are set up afterwards at the console, with their own commands, as on any server. When those commands store credentials, the server says which environment variables they belong in, and the tool asks for the values, writes them into `.env`, and recreates the server.

The walkthrough is streampack's tutorial, [Deploy with Docker for Mattermost and GitLab](https://github.com/streampack-dev/streampack/blob/main/docs/tutorials/deploy-with-docker-for-mattermost-and-gitlab.md). This page is the reference.

## Needs

- Docker with the Compose plugin (`docker compose version`), usable by you without `sudo`.
- A server-streampack release with streampack#209 (the local admin token and `GET /admin/secrets`).

## Install

Download the binary for the box from the [releases](https://github.com/streampack-dev/setup/releases), check it against `SHA256SUMS`, and make it executable:

```bash
curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/streampack-setup-linux-amd64
curl -fLO https://github.com/streampack-dev/setup/releases/latest/download/SHA256SUMS
sha256sum --check --ignore-missing SHA256SUMS
install -m 755 streampack-setup-linux-amd64 /usr/local/bin/streampack-setup
```

Use `linux-arm64` on an ARM box. Or build it: `go build ./cmd/streampack-setup` (Go 1.26).

## Usage

```text
streampack-setup [-dir DIR] [-url URL] [command]
```

Run it in the deployment's directory (or name it with `-dir`). `-url` is where the server answers from the box, `http://127.0.0.1:8080` by default.

| Command | What it does |
|---------|--------------|
| (none) | `init` if there's no `.env` yet, then `up`, then `console`. Run it again any time: it skips what's done. |
| `init` | Asks what starting needs and writes `.env` and `compose.yaml`. Keeps what `.env` already has. |
| `up` | `docker compose pull`, `docker compose up -d`, then waits for `/features`. |
| `console` | Opens the server's console as the super-admin. |
| `secrets` | Moves credentials given at the console into `.env`, and recreates the server. |
| `version` | Prints the tool's version. |

Every step it takes is a `docker compose` command, shown as it runs, so you can do any of them by hand.

## What `init` Asks, and Writes

- **The server version.** The newest release is the default, from the image registry.
- **The super-admin's email** (`ADMIN_EMAIL`). On first start the server creates the account. There's no password: the tool signs in with the local admin token, and the account signs in elsewhere with a one-time code.
- **The API's and the site's public addresses** (`BASE_URL`, `BLOG_BASE_URL`).
- **Which integrations to turn on:** Mattermost, GitLab, IRC, Slack, Discord (`*_ENABLED`). Discord's three settings are asked for if it's on, since it needs them at startup. The others are set up at the console.
- **Mail, if you want it:** an SMTP host, port, sender, and optionally a username and password.

It generates `DB_PASSWORD`, `JWT_SECRET` and the webhook keys (`GITLAB_WEBHOOK_SECRET_KEY`, `GITHUB_WEBHOOK_SECRET_KEY`).

It writes:

- **`.env`**, readable by you alone (mode 600). Values are written so Compose reads them back exactly.
- **`compose.yaml`**, if there isn't one. It's the file in streampack's [Run the Published Image with Docker Compose](https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/run-the-published-image.md): the server and PostgreSQL, the API on `127.0.0.1:8080` for a reverse proxy, and named volumes for the database and media.
- **`generative/`**, for prompt overrides.

## The Console

```text
Console open as root.
> mattermost connect work https://mattermost.example.com <token>
Connecting to 'work'...
> gitlab instance add https://gitlab.example.com <token>
> secrets
```

Lines go to the server's web console (`POST /admin/console`), and its output comes back on the console stream. When the stream closes (a token expires, or the server restarts), the tool opens it again. Anything said in between isn't replayed.

| Tool command | What it does |
|--------------|--------------|
| `:secrets` | Moves credentials into `.env` now. |
| `:help` | Lists these. |
| `:quit`, Ctrl-D | Leaves. If credentials are still waiting, it offers to move them first. |

## Credentials

A credential given at the console is stored in the server's database until it has an environment variable, and the server's next start fails without one. `secrets` (or `:secrets`, or answering yes on the way out) does the following:

1. Asks the server which variables are needed (`GET /admin/secrets`), by name.
2. Asks you for each value, hidden. Leave one blank to skip it.
3. Writes them into `.env`.
4. Recreates the server with `docker compose up -d` (`restart` wouldn't re-read `.env`), so it moves the stored values to `env://` references.
5. Checks again, and says what's still needed.

The server never sends a value back, and the tool never prints one.

If a start is stopped anyway (a restart outside the tool, say), `up` reads the stopped start's log, offers the variables it names, and tries again.

## How It Signs In

The compose file sets `STREAMPACK_LOCAL_ADMIN_TOKEN_FILE=/run/streampack/admin-token` on a `tmpfs`. When the server is ready, it writes a short-lived super-admin token there, and renews it at half its lifetime. The tool reads it with `docker compose exec server-streampack cat /run/streampack/admin-token` each time it needs one. The file never touches the host's disk, and reading it takes the same Docker access that can already read `.env`. Streampack's [The Local Admin Token](https://github.com/streampack-dev/streampack/blob/main/docs/explanation/local-admin-token.md) has the reasoning.

## Development

```bash
go test -race ./...
go vet ./...
gofmt -l .
```

`internal/setup` holds the flows, and the packages beside it are what the flows use. Tests fake Docker, the server (`httptest`) and the person (a scripted `prompt.Asker`).

Releases come from a tag `vX.Y.Z`. CI attaches static binaries for linux and darwin, amd64 and arm64, with `SHA256SUMS`.

## License

MIT
