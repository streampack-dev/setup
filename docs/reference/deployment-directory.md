# The Deployment Directory

`init` writes these into the deployment directory. They make an ordinary Compose deployment, which you can manage by hand.

| Path | Written | Contents |
|------|---------|----------|
| `.env` | Created by `init`; `init` and `secrets` add keys to it | The server's settings and secrets. Mode 600. |
| `compose.yaml` | By `init`, only if the directory has no compose file | The server and PostgreSQL. |
| `generative/` | By `init`, only if absent | Prompt and mail template overrides, mounted read-only. Empty to begin with. See streampack's [Prompt and Message Templates](https://github.com/streampack-dev/streampack/blob/main/docs/reference/templates.md). |
| `ext/` | By `init`, only if absent | Extension jars and extra classpath files such as `banner.txt`, mounted read-only. Empty to begin with. See streampack's [The `ext/` Directory](https://github.com/streampack-dev/streampack/blob/main/docs/reference/ext-directory.md). |
| `media/` | By `init`, only if absent | Posts' recordings, for the audio streams. The server writes here. On Linux its files belong to the container's root user. |

## `.env` Keys

`init` asks for, or generates, only the keys that are missing. A key that's already there, even with an empty value, is left alone. The one exception is the questions marked *required*, which are asked again when the value is empty.

| Key | Source | Notes |
|-----|--------|-------|
| `STREAMPACK_VERSION` | Asked (required) | The image tag. The default is the registry's newest. |
| `STREAMPACK_NAME` | Asked (required) | The deployment's name: in `/features`, the IRC identity, the user agent and the MCP server's name. Default `nevet`. Letters, digits, `.`, `_` and `-`, with no spaces. |
| `ADMIN_EMAIL` | Asked (required) | The super-admin, created on the server's first start. |
| `BASE_URL` | Asked (required) | The API's public address. Default `http://localhost:8080`. |
| `BLOG_BASE_URL` | Asked (required) | The site's public address. Default `http://localhost:3000`. |
| `BLOG_SITE_NAME` | Asked (required) | The site's name, as readers see it: in pages, the mail and the feeds. Default: the system's name. |
| `BLOG_FEED_TITLE` | Asked (required) | The post feeds' title. Default: the site's name. |
| `BLOG_FEED_DESCRIPTION` | Asked (required) | The post feeds' description. Default: the server's, `JVM ecosystem news and community content`. |
| `DB_PASSWORD` | Generated | 24 random bytes, base64url. The server and PostgreSQL both use it. |
| `JWT_SECRET` | Generated | 48 random bytes, base64url. |
| `GITHUB_WEBHOOK_SECRET_KEY` | Generated | 32 random bytes, hex. |
| `MATTERMOST_ENABLED` | Asked, default yes | `true` or `false`. |
| `GITLAB_ENABLED` | Asked, default yes | `true` or `false`. |
| `GITLAB_WEBHOOK_SECRET_KEY` | Generated if GitLab is on | 32 random bytes, hex. |
| `IRC_ENABLED`, `SLACK_ENABLED` | Asked, default no | `true` or `false`. |
| `DISCORD_ENABLED` | Asked, default no | `true` or `false`. |
| `DISCORD_APPLICATION_ID`, `DISCORD_PUBLIC_KEY`, `DISCORD_BOT_TOKEN` | Asked, hidden, if Discord is on | Needed before the server starts. |
| `MAIL_ENABLED` | Asked, default no | `true` or `false`. |
| `MAIL_HOST`, `MAIL_PORT`, `MAIL_FROM` | Asked if mail is on | Defaults `host.docker.internal` and `25`. |
| `MAIL_USERNAME`, `MAIL_PASSWORD` | Asked if mail is on and a username is given | The password is hidden. `MAIL_AUTH` and `MAIL_STARTTLS` are set to `true` with them. |
| Credentials from the console (`MATTERMOST_WORK_TOKEN`, …) | Asked by `secrets`, hidden | The names come from the server. |

Every other server setting keeps the server's default. You can add any of them to `.env` yourself: see streampack's [Environment Variables](https://github.com/streampack-dev/streampack/blob/main/docs/reference/environment-variables.md). Then run `./streampack-setup up`.

## How `.env` Is Written

- **Mode and atomicity.** The file is written to a temporary file with mode 600 and renamed into place, so a reader never sees half a file.
- **Lines kept.** Comments, blank lines and the order of existing keys stay as they were. A changed key is replaced on its own line, and a new key is added at the end.
- **Quoting.** Values are quoted so that Compose reads them back exactly:
  - a plain value (letters, digits and `_ - . , : / @ + =`) as it is;
  - anything else in single quotes, which Compose doesn't interpret;
  - a value containing a single quote or a newline in double quotes, with escapes.
- **Reading.** `.env` files written by hand are read too: `export KEY=value`, and single- or double-quoted values.

## `compose.yaml`

It's the file from streampack's [Run the Published Image with Docker Compose](https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/run-the-published-image.md). The tool depends on these parts of it:

| Part | Why |
|------|-----|
| The service name `server-streampack` | `exec` and `logs` name it. |
| `STREAMPACK_LOCAL_ADMIN_TOKEN_FILE: /run/streampack/admin-token`, on a `tmpfs` at `/run/streampack` | The console signs in with this token. |
| `ports: "127.0.0.1:8080:8080"` | The default `--url`. If you change the port, pass `--url`. |
| `env_file: .env` | `secrets` writes credentials there. |
| `./media`, `./ext` and `./generative` mounted at `/opt/streampack/…`, with `z` | `init` makes the directories, and a hard reset knows which to keep. `z` lets the container read them under SELinux. |
| Images named in full (`docker.io/library/postgres:18`) | Podman refuses a short name when it can't ask which registry is meant. |

A compose file that's already in the directory, under any of the names Compose looks for (`compose.yaml`, `compose.yml`, `docker-compose.yaml`, `docker-compose.yml`), is never replaced, and none is added beside it. If yours lacks the token file, `console` and `secrets` say there's no token.
