# discord-bot-webhook

POST `/notify` accepts an event, broadcasts it to local SSE listeners,
AND posts a copy to a Discord webhook URL via the outbox — so transient
Discord outages don't drop events.

## What it shows off

- Top-level `outbox_db:` enables durable outbound delivery (SQLite).
- `stream-publish.forward_url` routes through the outbox automatically.
- `connections.notify` SSE broker for in-process subscribers.
- `inputs:` validation on the producer body.

## Run

```sh
DISCORD_WEBHOOK_URL='https://discord.com/api/webhooks/<id>/<token>' \
  wave serve examples/apps/discord-bot-webhook/server.yaml --port 8607

curl -X POST http://127.0.0.1:8607/notify \
  -H 'Content-Type: application/json' \
  -d '{"title":"deploy","body":":rocket: shipped"}'
```

Without a real webhook the default points at `httpbin.org/post`, so
deliveries succeed against an echo endpoint — useful for inspecting the
outbox flow.

## Caveats

- Real Discord wants a `{"content":"..."}` shape. Adjust `static_meta`
  / `output:` mappings if you want to reshape before delivery.
- Outbox retries are bounded (default 10); failures land in the DLQ.
