# Recover a Refused Start

The server refuses to start when a credential stored in its database has no environment variable. That happens after `docker compose restart`, a reboot, or `up` run by hand, any of which can come before the credential has been moved into `.env`. The log says `SECURITY STARTUP CHECK FAILED` and names the variables.

## With the Tool

```bash
streampack-setup up
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

## Other Failures

For any other startup failure (`APPLICATION FAILED TO START` in the log), `up` stops and shows that part of the log. Fix what it describes, usually a setting in `.env`, and run `streampack-setup up` again. The whole log:

```bash
docker compose logs server-streampack
```

If the server doesn't answer within 5 minutes and the log shows no failure, `up` gives up and says so. The server may still be starting, for example on a first start with a slow disk. Check the log, then run `up` again.
