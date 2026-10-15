# streampack-setup

Brings up a [server-streampack](https://github.com/streampack-dev/streampack) on a box with Docker, then acts as its console: the commands you'd type at the server's own console, from your shell, as the super-admin.

It asks only what starting the server needs. Integrations (Mattermost, GitLab, IRC, Slack, …) are set up afterwards at the console, with the server's own commands. When those commands store credentials, the server says which environment variables they belong in, and the tool asks for the values, writes them into `.env`, and recreates the server.

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

Use `linux-arm64` on an ARM box, or `darwin-amd64` or `darwin-arm64` on a Mac. Or build it: `go build ./cmd/streampack-setup` (Go 1.26).

## Use

```bash
mkdir ~/streampack && cd ~/streampack
streampack-setup
```

The first run asks its questions and writes `.env` and `compose.yaml`, then starts the server and opens the console. Each later run skips what's done and goes to the console.

| Command | What it does |
|---------|--------------|
| (none) | `init` if needed, then `up`, then `console` |
| `init` | Ask what starting needs; write `.env` and `compose.yaml` |
| `up` | `docker compose pull` and `up -d`; wait for the server |
| `console` | The server's console, as the super-admin |
| `secrets` | Move credentials given at the console into `.env`; recreate the server |
| `version` | The tool's version |

## Documentation

Start with [Your First Deployment](docs/tutorials/first-deployment.md). The [documentation](docs/README.md) has how-to guides, reference and explanation, including [How a Run Goes](docs/explanation/execution-flow.md).

## Development

```bash
go test -race ./...
go vet ./...
gofmt -l .
```

`internal/setup` holds the flows, and the packages beside it are what the flows use. Tests fake Docker, the server (`httptest`) and the person (a scripted `prompt.Asker`). Releasing: [Release the Tool](docs/how-to/release-the-tool.md).

## License

MIT
