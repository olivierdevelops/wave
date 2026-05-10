# todo-app

The "Hello World" of full-stack apps: per-user TODO list backed by
SQLite, with passwordless magic-link sign-in.

## Run it

```sh
wave serve examples/apps/todo-app/server.yaml --port 8501
```

## Try it

1. Open http://127.0.0.1:8501. Enter any email, hit "send link".
2. Look at the server's stderr — the dev-mode console mailer prints
   the full email, including the `?token=...` URL. Click it.
3. You're signed in. Add, check off, and delete TODOs. Each user
   sees only their own.

```sh
curl -s localhost:8501/todos -b cookies.txt | jq
curl -s -X POST localhost:8501/todos -b cookies.txt \
  -H 'content-type: application/json' -d '{"title":"buy milk"}'
```

## What to look at

- `auth.session` — JWT cookie session config.
- `magic-link-request` / `magic-link-consume` route types — full
  passwordless flow with no SMTP needed in dev.
- `storage-access` routes — every CRUD verb uses Wave's templated
  SQL with bound parameters; `(getUser).Username` scopes to the
  signed-in email.
- `inputs:` declared params keep the SQL free of injection.

## Caveats

- Dev mode uses the console mailer (no real email). Wire SMTP via
  `auth_flows.smtp` for production.
- `data.db` is created next to `server.yaml` on first boot.

## What it shows off

magic-link auth · session cookies · `storage-access` SQL templating ·
declared `inputs:` validation · per-user row scoping.
