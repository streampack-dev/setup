# Recover a Refused Start

`streampack-setup up` stops, with an explanation, when the server won't start. This page covers each reason it gives.

## A Credential Has No Variable

The server refuses to start when a credential stored in its database has no environment variable. That happens after `docker compose restart`, a reboot, or `up` run by hand, any of which can come before the credential has been moved into `.env`. The log says `SECURITY STARTUP CHECK FAILED` and names the variables.

### With the Tool

```bash
./streampack-setup up
```

`up` sees the refusal in the log, shows it, and asks for each variable it names. It writes your answers into `.env` and starts the server again, up to three starts in all:

```text
The server stopped at its secrets check:
SECURITY STARTUP CHECK FAILED (Mattermost secrets)
...
MATTERMOST_WORK_TOKEN (blank to skip):
Wrote them into .env.
$ docker compose up -d
The server is ready.
```

If you don't have a value to hand, leave it blank. `up` stops, and the server stays down until the variable is set.

## The Database Refuses the Password

```text
streampack-setup: the server didn't start: the database refused the password in .env (DB_PASSWORD).
```

The database keeps the password it was given when it was first made, and ignores `DB_PASSWORD` after that. This message means `.env` has a different one, usually because `.env` was deleted and written again while the database volume stayed. `up` stops as soon as the log shows it, rather than waiting out its 5 minutes.

- **To keep the data,** put the original password back in `.env` as `DB_PASSWORD`, from a backup of `.env`, and run `streampack-setup up`.
- **To start afresh,** losing everything in the database:

  ```bash
  docker compose down -v
  streampack-setup up
  ```

## After Losing `.env`

Running the tool in a directory whose `.env` is gone, but whose database is still there, doesn't generate a new database password. It stops and says so:

```text
This deployment's database already exists (the volume streampack_pgdata), but .env has no DB_PASSWORD.
The database keeps the password it was first given, so a new one wouldn't open it.
The existing database's password, blank if you haven't got it:
Delete streampack_pgdata, and everything in that database, to start afresh? [y/N]:
```

- Give the password, and the database is kept.
- Leave it blank and answer `y`, and the tool removes the deployment's containers and the database volume, then carries on as on a first run.
- Leave it blank and answer no, and it stops without changing anything.

Even with the database kept, a new `.env` lacks what the old one held:

- **Credentials moved there from the console.** The next start stops at the secrets check, and `up` asks for each one.
- **The webhook keys** (`GITHUB_WEBHOOK_SECRET_KEY`, `GITLAB_WEBHOOK_SECRET_KEY`). New ones are generated, and webhook secrets stored under the old keys can't be read. Set those repositories' and projects' webhooks up again.
- **`JWT_SECRET`.** Everyone is signed out.

Back up `.env` along with the database.

## Other Failures

For any other startup failure (`APPLICATION FAILED TO START` in the log), `up` stops and shows that part of the log. Fix what it describes, usually a setting in `.env`, and run `./streampack-setup up` again. The whole log:

```bash
docker compose logs server-streampack
```

If the server doesn't answer within 5 minutes and the log shows no failure, `up` gives up and says so. The server may still be starting, for example on a first start with a slow disk. Check the log, then run `up` again.
