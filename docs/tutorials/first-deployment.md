# Your First Deployment

In this tutorial you'll use `streampack-setup` to bring up a `server-streampack` with no integrations, sign in to its console, and take it down again. It takes about ten minutes, most of it spent pulling the images. Any machine with Docker will do: a Linux server, or a Mac with Docker Desktop.

Afterwards, streampack's [Deploy with Docker for Mattermost and GitLab](https://github.com/streampack-dev/streampack/blob/main/docs/tutorials/deploy-with-docker-for-mattermost-and-gitlab.md) does the same on a server, and connects it to Mattermost and GitLab.

## What You Need

- Docker with the Compose plugin, usable without `sudo`. On a Mac, Docker Desktop, running:

  ```bash
  docker compose version
  ```

- Port 8080 free on `127.0.0.1`. A development server-streampack (`just run` in streampack) uses it too, so stop that first.
- The tool, from the [releases](https://github.com/streampack-dev/setup/releases) for your machine (`linux-amd64`, `linux-arm64`, `darwin-amd64` or `darwin-arm64`):

  ```bash
  curl -fLo ~/streampack-setup https://github.com/streampack-dev/setup/releases/latest/download/streampack-setup-linux-amd64
  chmod +x ~/streampack-setup
  ```

  On an Apple Silicon Mac, that's `streampack-setup-darwin-arm64`. Download it with `curl`, as here: a browser's download is quarantined, and macOS won't run the unsigned binary until you clear it with `xattr -d com.apple.quarantine ~/streampack-setup`.

## 1. Answer the Questions

Make an empty directory and run the tool in it:

```bash
mkdir ~/streampack-trial && cd ~/streampack-trial
~/streampack-setup
```

Take the defaults by pressing Enter, except for your email address. Answer no to every integration:

```text
Setting up a server-streampack deployment in /home/you/streampack-trial
Server version [1.16.0]:
The super-admin's email address: you@example.com
The API's public address [http://localhost:8080]:
The site's public address [http://localhost:3000]:
Turn on Mattermost? [Y/n]: n
Turn on GitLab? [Y/n]: n
Turn on IRC? [y/N]:
Turn on Slack? [y/N]:
Turn on Discord? [y/N]:
Send mail (sign-in codes) through an SMTP server? [y/N]:
Wrote compose.yaml
Wrote .env (readable by you alone)
```

Look at what it wrote:

```bash
ls -a
```

`compose.yaml` is an ordinary Compose file. `.env` holds your answers and the secrets the tool generated. Don't print `.env` where anyone else can see it.

## 2. Watch It Start

The tool carries on by itself:

```text
$ docker compose pull
$ docker compose up -d
Waiting for the server to answer at http://127.0.0.1:8080/features
The server is ready.
Console open as you.
```

The lines starting with `$` are the commands it ran. You could have typed them yourself.

## 3. Use the Console

You're signed in as the super-admin. Nobody gave a password: the server wrote a short-lived token inside its container, and the tool read it from there.

Ask the server whether any credentials need moving into `.env`:

```text
> secrets
No environment variables are needed before the next start.
```

That's the server answering, not the tool. Anything you type without a leading `:` goes to the server. List the tool's own commands:

```text
> :help
```

## 4. Leave, and Come Back

Leave with `:quit`. Then run the tool again in the same directory:

```bash
~/streampack-setup
```

This time it asks nothing. `.env` is complete and the server is already up, so it goes straight to the console. Running it again is always safe: it skips what's done. Leave with `:quit` once more.

## 5. Take It Down

This is a trial, so remove the containers, the database and the media volume:

```bash
docker compose down -v
```

`-v` deletes the database. Never use it on a deployment you mean to keep.

## What You've Learned

- The tool asks only what starting needs, and generates the secrets itself.
- Each step is a `docker compose` command, run in the directory you chose.
- The console speaks the server's own commands, and signs in without a password.
- Run it again whenever you want the console back.

## Next

- [How a Run Goes](../explanation/execution-flow.md): what each step does.
- [Move Credentials into `.env`](../how-to/move-credentials-into-env.md): what happens once an integration has a token.
- streampack's [Deploy with Docker for Mattermost and GitLab](https://github.com/streampack-dev/streampack/blob/main/docs/tutorials/deploy-with-docker-for-mattermost-and-gitlab.md).
