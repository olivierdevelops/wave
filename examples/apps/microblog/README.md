# microblog

Twitter-lite: 280-character posts, public timeline, per-user pages,
live updates over Server-Sent Events. Magic-link sign-in.

## Run it

```sh
wave serve examples/apps/microblog/server.yaml --port 8504
```

## Try it

1. Open http://127.0.0.1:8504/app, click "sign in", enter any
   email, copy the magic link from the server's stderr, open it.
2. Compose a quack and hit "quack". Open another tab on the same
   URL — your post appears live, no refresh.
3. Click any `@handle` to see that user's posts at `/app?u=<h>`.

```sh
curl -s localhost:8504/timeline | jq
curl -s localhost:8504/u/alice@example.com | jq
curl -N localhost:8504/events/timeline   # subscribe to live feed
```

## What to look at

- `magic-link-request` + `magic-link-consume` route types — full
  passwordless auth in five lines of YAML.
- `auth: ["session"]` gates `/posts` so only signed-in users can
  write; the public `/timeline` and `/u/{handle}` routes have no
  auth at all.
- `connections.timeline` is the SSE broker; the `/posts/notify`
  stream-publish route fans events out to every subscriber.
- Frontend listens with `EventSource` and reloads the feed.

## Caveats

- Console mailer in dev: real send needs `auth_flows.smtp` config.
- `data.db` is created next to `server.yaml`.
- The "@handle" is the user's email address (magic-link is the
  identity provider).

## What it shows off

magic-link auth · per-route auth gating · SSE fan-out · path
parameters · public + authed routes side-by-side.
