# Move Credentials into `.env`

A credential typed at the console (a Mattermost token, a GitLab token, IRC SASL details) is stored in the server's database until it has an environment variable. The server's next start stops if it hasn't got one. Move it before then.

## When Leaving the Console

Leave with `:quit` or Ctrl-D. If credentials are waiting, the tool asks:

```text
2 credentials given at the console still need a place in .env; the next start fails without them. Move them now? [Y/n]:
```

Press Enter, then paste each value when its variable is named. Nothing you type is shown. A blank answer skips that variable.

## Without Leaving

At the `>` prompt:

```text
> :secrets
```

## From the Shell

```bash
./streampack-setup secrets
```

## What Happens

The tool writes the values into `.env`, then runs `docker compose up -d` to recreate the server. The server reads the new variables, replaces the stored credentials with `env://` references, and the tool checks that nothing is left:

```text
Done: every credential is in .env.
```

If it lists something under `Still needed:`, run `secrets` again for those.

Type the same value you gave at the console. The server doesn't compare the two: from now on it uses the one in `.env`.

## Checking First

`secrets` at the console (no colon) is the server's own command. It lists what's waiting and changes nothing:

```text
> secrets
```
