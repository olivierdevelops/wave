# status-page

Public status board like statuspage.io. Admins POST status updates;
the dashboard ticks live via SSE for every viewer.

## Run it

```sh
wave serve examples/apps/status-page/server.yaml --port 8506
```

## Try it

Open http://127.0.0.1:8506/app — the public dashboard.

```sh
# Set an initial status.
curl -s -X POST localhost:8506/admin/status \
  -H 'content-type: application/json' \
  -d '{"name":"api","status":"operational","reason":"healthy"}'

# Knock it down.
curl -s -X POST localhost:8506/admin/status \
  -H 'content-type: application/json' \
  -d '{"name":"api","status":"degraded","reason":"high p99 latency"}'

# Tell every open dashboard.
curl -s -X POST localhost:8506/admin/notify

curl -s localhost:8506/status  | jq
curl -s localhost:8506/history | jq
```

## What to look at

- The `/admin/status` route runs three SQL statements in one
  execute: upsert the service, append to the change log, return
  the new state.
- `connections.status` SSE broker · `/admin/notify` stream-publish
  fans events to every dashboard.
- Public `/status` and `/history` have no auth — anyone can read.

## Caveats

- The `/admin/*` routes have NO auth in this demo. For production,
  gate them with a real `auth:` block (jwt + magic-link, oidc) or
  wrap behind `forward_auth:` to oauth2-proxy / Authelia.
- `data.db` is created next to `server.yaml` on first boot.

## What it shows off

upsert with ON CONFLICT · multi-statement execute · SSE live
dashboard · stream-publish broker fan-out.
