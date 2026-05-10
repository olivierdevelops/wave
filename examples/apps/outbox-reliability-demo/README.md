# outbox-reliability-demo

A bank-transfer-shaped write that demonstrates the outbox pattern: the
publish AND the forward-to-downstream are both governed by the durable
outbox, so the side-effect can't be lost on a downstream blip.

## What it shows off

- Top-level `outbox_db:` enabling the SQLite-backed durable outbox.
- `stream-publish.forward_url` routes through the outbox automatically.
- Worker drains in background, retries on failure, dead-letters after N.
- `inputs:` validation for typed amount + account names.

## Run

```sh
DOWNSTREAM_URL='https://httpbin.org/post' \
  wave serve examples/apps/outbox-reliability-demo/server.yaml --port 8610

curl -X POST http://127.0.0.1:8610/transfer \
  -H 'Content-Type: application/json' \
  -d '{"from_account":"A","to_account":"B","amount_cents":4200}'
```

## Why outbox > direct call

If we POSTed straight to the downstream from the request handler:
- A downstream 5xx would either fail the user request or silently drop.
- A process crash between the DB write and the POST loses the event.

With the outbox: the event is enqueued in the same SQLite the rest of
the app uses. Delivery is the worker's problem; restarts don't matter.
The DLQ (`outbox.db`) is your audit trail of what failed permanently.

## Caveats

- Default `DOWNSTREAM_URL` is `httpbin.org/post` — replace for prod.
- The orchestrator does not auto-INSERT into `transfers` for stream-publish
  routes; wire a small app-side route or plugin if you want both.
