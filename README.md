# streampack-setup

Brings up a [server-streampack](https://github.com/streampack-dev/streampack) on a box with Docker, then acts as its console: the commands you'd type at the server's own console, from your shell, as the super-admin.

It asks only what starting the server needs. Integrations (Mattermost, GitLab, IRC, Slack, …) are set up afterwards at the console, with the server's own commands. When those commands store credentials, the server says which environment variables they belong in, and the tool asks for the values, writes them into `.env`, and recreates the server.

## Needs

- Docker with the Compose plugin. On a Mac, this is Docker Desktop.
- A server-streampack release with streampack#209 (the local admin token and `GET /admin/secrets`).

## Install

Follow [Install the Tool](docs/how-to/install-the-tool.md). It has one section for each kind of computer: Mac with Apple Silicon, Mac with Intel, Linux on x86-64 and Linux on ARM.

The tool lives in the deployment directory, with `.env` and `compose.yaml`.

## Use

Run the tool from the deployment directory:

```bash
cd ~/streampack
./streampack-setup
```

The first run asks its questions and writes `.env` and `compose.yaml`. Then it starts the server and opens the console. Each later run skips the steps that are done, and opens the console.

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
just test     # gofmt check, go vet, go test -race, as CI does
just build    # ./streampack-setup, stamped with VERSION
```

`internal/setup` holds the flows, and the packages beside it are what the flows use. Tests fake Docker, the server (`httptest`) and the person (a scripted `prompt.Asker`). Releasing: [Release the Tool](docs/how-to/release-the-tool.md).

## License

MIT
