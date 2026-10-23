# How a Run Goes

`streampack-setup` with no command does three things in order: `init` if the directory isn't configured yet, then `up`, then `console`. Each is also a command of its own, and every run can be repeated: a step that's already done is skipped.

Before any of them, the tool finds the container engine, Docker or Podman, and the Compose that goes with it. This page shows Docker's commands; with Podman, the tool runs `podman` and `podman-compose` in their place. See [What It Uses](../reference/server-interface.md#the-container-engine).

With `--hard-reset`, a step comes before all of these: the tool deletes the deployment, after you confirm, so the run that follows is a first run. See [Start Over](../how-to/start-over.md).

With `--update`, on a configured directory, a step comes before `up`: the tool sets `STREAMPACK_VERSION` to the registry's newest release, after you confirm. See [Upgrade the Server](../how-to/upgrade-the-server.md).

```mermaid
flowchart TD
    start([streampack-setup]) --> reset{"--hard-reset?"}
    reset -- "yes, and you type the<br>directory's name" --> wipe[delete containers, pods, volumes, networks,<br>.env, compose files, media/;<br>keep generative/ and ext/] --> configured
    reset -- "yes, any other answer" --> stopped([stop; nothing deleted])
    reset -- no --> configured{".env has STREAMPACK_VERSION,<br>DB_PASSWORD and JWT_SECRET?"}
    configured -- no --> volume{"no DB_PASSWORD, but the<br>database volume exists?"}
    volume -- "no" --> init[init: ask, generate, write .env and compose.yaml]
    volume -- "yes" --> old["ask for its password,<br>or delete the volume"] --> init
    configured -- yes --> update{"--update?"}
    update -- no --> up
    update -- "yes" --> newest{"registry's newest<br>newer than .env's?"}
    newest -- "no (already on it, or .env is newer)" --> up
    newest -- "yes, and you confirm" --> pin[write the newest number<br>to STREAMPACK_VERSION] --> up
    newest -- "you don't confirm, or<br>the registry can't be reached" --> unchanged([stop; .env unchanged])
    init --> up[up: docker compose pull, then up -d]
    up --> wait{"GET /features answers?"}
    wait -- "yes" --> console[console: read the token, open the stream]
    wait -- "log says SECURITY STARTUP CHECK FAILED" --> ask[ask for the variables it names, write .env]
    ask -- "something given (at most 3 starts)" --> up2[docker compose up -d] --> wait
    ask -- "nothing given" --> stop([stop, showing the log's explanation])
    wait -- "log says APPLICATION FAILED TO START<br>or password authentication failed,<br>or 5 minutes pass" --> stop
    console --> loop[read a line]
    loop -- "a server command" --> submit[POST /admin/console] --> loop
    loop -- ":secrets" --> move
    loop -- ":quit or Ctrl-D" --> left{"GET /admin/secrets:<br>anything waiting?"}
    left -- no --> done([done])
    left -- "yes, and you say move them" --> move[secrets: ask for each value, write .env,<br>docker compose up -d, wait, check again]
    move --> done
```

## `init`

`init` reads `.env` if there is one, and asks only for what it lacks, so running it on a configured directory asks nothing.

0. **An existing database.** If `.env` has no `DB_PASSWORD` but the deployment's database volume exists (a `.env` lost after an earlier run), `init` asks for that database's password, or offers to delete the volume. A generated password would never open it: Postgres keeps the one it was first given. See [After Losing `.env`](../how-to/recover-a-refused-start.md#after-losing-env).

1. **The server version.** It suggests the newest tag in the image registry. If the registry can't be reached, it says so and asks with no default.
2. **The system's name** (`STREAMPACK_NAME`), default `streampack`.
3. **The super-admin's email address, the API's public address and the site's public address.** Each is checked: an email address, and addresses that start with `http://` or `https://`.
4. **The site's name and its feed.** The site's name (`BLOG_SITE_NAME`) defaults to the system's name, the feed's title (`BLOG_FEED_TITLE`) to the site's name, and the feed's description (`BLOG_FEED_DESCRIPTION`) to "Posts from" and the feed's title.
5. **Generated values:** the database password, the JWT secret and GitHub's webhook key. They're random, made on this box, and never shown.
6. **Which integrations to turn on.** Each defaults to no: nothing is on unless you say so. GitLab gets a generated webhook key. Discord's three values are asked for straight away, because the server needs them before it starts. The others need nothing yet.
7. **Mail.** If you want it: the SMTP host, port, sender, and optionally a username and password.
8. **The files.** It writes `compose.yaml` if there isn't one, and never overwrites one that's there. It makes `generative/`, `ext/` and `media/`, and writes `.env` with mode 600.

Nothing is sent anywhere during `init`, except the registry lookup for the version.

## `up`

1. `docker compose pull`, then `docker compose up -d`.
2. It polls `GET /features` every 2 seconds until the server answers. Between polls it reads the last 300 lines of the server's log, looking for a start that was refused.
3. **A refused start.** The server's secrets check refuses to start when a credential stored in the database has no environment variable, and its log names those variables. `up` shows that part of the log, asks for each variable (hidden), writes the answers into `.env`, and starts the server again, at most three times in all. A database that refuses `DB_PASSWORD`, or an `application.yml` at the top of `ext/`, stops `up` with an explanation. Any other startup failure, or no answer within 5 minutes, stops `up` with the log's explanation.

## `console`

1. **The stream.** It opens the server's console stream (`GET /admin/console/stream`) as the super-admin. The server's first event names the account, and the tool prints "Console open as …".
2. **The token.** The tool reads it with `docker compose exec server-streampack cat /run/streampack/admin-token`. The server writes that file inside its own container once it's ready, and renews it. When a request comes back 401, because the token has been renewed or the server restarted, the tool reads the file again and retries once.
3. **The input loop.** Each line you type is posted to `/admin/console`. The answers come back on the stream, so a slow command's answer can arrive after the next prompt. When the stream drops, the tool says so once and reopens it every 2 seconds. Anything the server said in between is lost.
4. **Its own commands** start with `:`. `:secrets` moves credentials now, `:help` lists the commands, and `:quit` or Ctrl-D leaves.
5. **Leaving.** The tool stops the stream, then asks the server whether any credentials are still waiting. If some are, it offers to move them. Against a server too old to answer, it just leaves.

## `secrets`

1. **What's needed:** `GET /admin/secrets` returns each variable's name and what it's for. Values never come back.
2. **The values.** The tool asks for each one, hidden. A blank answer skips it.
3. **`.env`.** It writes the answers into `.env`.
4. **The restart.** It runs `docker compose up -d`, which recreates the container. A `restart` would keep the old environment. On this start the server finds each variable set and replaces the stored value with an `env://` reference. It doesn't compare the two, so from then on the value in `.env` is the one used, even if it was mistyped.
5. **The check.** It waits for the server, then asks again and reports anything still needed.

[Why It's Built This Way](design.md) covers the choices behind these steps.
