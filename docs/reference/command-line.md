# Command Line

```text
streampack-setup [-dir DIR] [-url URL] [-hard-reset] [command]
```

## Flags

| Flag | Default | Meaning |
|------|---------|---------|
| `-dir` | `.` | The deployment directory, holding `.env`, `compose.yaml`, `generative/`, `ext/` and `media/`. Every `docker compose` command runs here. |
| `-url` | `http://127.0.0.1:8080` | Where the server answers from this box. The compose file publishes the API on `127.0.0.1:8080`. |
| `-hard-reset` | off | First delete the deployment: its containers, volumes and networks, `.env`, the compose file and any override, and `media/`. It keeps `generative/` and `ext/`. You confirm by typing the directory's name. Then run as a first run. Goes with no command, or with `init`; with any other command it exits with status 2. See [Start Over](../how-to/start-over.md). |

Flags go before the command.

## Commands

| Command | What it does | Ends |
|---------|--------------|------|
| (none) | `init` if `.env` lacks `STREAMPACK_VERSION`, `DB_PASSWORD` or `JWT_SECRET`; then `up`; then `console`. | When the console is left, or at the first step that fails. |
| `init` | Asks for what `.env` lacks and writes `.env` and `compose.yaml` (if there is no compose file), and makes `generative/`, `ext/` and `media/`. See [The Deployment Directory](deployment-directory.md). | After writing. |
| `up` | `docker compose pull`, `docker compose up -d`, then waits for `GET /features`. A start refused by the secrets check is offered the variables it names, for up to three starts. | Once the server answers. It fails on any other startup failure, or after 5 minutes. |
| `console` | Opens the server's console as the super-admin. | On `:quit`, `:exit` or Ctrl-D, after offering to move waiting credentials. |
| `secrets` | Asks for the variables `GET /admin/secrets` names, writes them into `.env`, recreates the server, and checks again. | After the check. |
| `version` | Prints the tool's version. | |

Any other command prints the usage and exits with status 2.

## Console Commands

At the `>` prompt, a line that starts with `:` belongs to the tool. Anything else goes to the server, as at its own console.

| Command | What it does |
|---------|--------------|
| `:secrets` | Runs `secrets` now, without leaving. |
| `:help` | Lists these, with examples of server commands. |
| `:quit`, `:exit`, Ctrl-D | Leaves. If credentials are waiting, it asks whether to move them first; the default is yes. |

## Interrupting

Ctrl-C stops the current step and exits with status 0. Nothing half-written is left: `.env` is written in one go at the end of `init` and after each `secrets`.

## Exit Status

| Status | Meaning |
|--------|---------|
| 0 | Done, or interrupted. |
| 1 | A step failed, or a hard reset was not confirmed. The message starts with `streampack-setup:`. |
| 2 | Unknown command, or `-hard-reset` with a command other than `init`. |
