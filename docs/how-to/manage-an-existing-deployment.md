# Manage an Existing Deployment

The tool can take over a deployment that was set up by hand, for example from streampack's [Run the Published Image with Docker Compose](https://github.com/streampack-dev/streampack/blob/main/docs/how-to/deploy/run-the-published-image.md).

## What It Needs

- **A `compose.yaml` (or `docker-compose.yml`) in one directory with `.env`,** with a service named `server-streampack`.
- **The local admin token file.** Check that the server service has these, and add them if it hasn't:

  ```yaml
      environment:
        STREAMPACK_LOCAL_ADMIN_TOKEN_FILE: /run/streampack/admin-token
      tmpfs:
        - /run/streampack:mode=0700
  ```

- **`ADMIN_EMAIL` in `.env`,** naming a super-admin account.
- **A server with streampack#209.**

## Steps

Run `init` against the directory. It asks only for what `.env` lacks (`STREAMPACK_VERSION`, say, if your compose file names the tag directly), and it leaves your compose file alone:

```bash
streampack-setup -dir /srv/streampack init
```

Then use it as usual:

```bash
streampack-setup -dir /srv/streampack
```

If the API isn't published on `127.0.0.1:8080`, tell the tool where it is:

```bash
streampack-setup -dir /srv/streampack -url http://127.0.0.1:9000
```

`init` adds the keys it generates when they're missing, `DB_PASSWORD` among them. If your database already has a password that isn't in `.env`, add it to `.env` before running `init`. Otherwise `init` generates a new one, which won't match the existing database.
