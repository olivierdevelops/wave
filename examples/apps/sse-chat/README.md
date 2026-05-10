# sse-chat

A live chat with no persistence: POST `/messages` is fanned out to every
EventSource subscribed to `/events/chat`.

## Run it
```sh
wave serve examples/apps/sse-chat/server.yaml --port 8203
```
Open http://127.0.0.1:8203/ in two browser tabs and type in either one.

## Try it (curl)
```sh
# Subscribe in one terminal:
curl -N http://127.0.0.1:8203/events/chat

# Publish in another:
curl -X POST http://127.0.0.1:8203/messages \
  -H 'Content-Type: application/json' \
  -d '{"text":"hello from curl"}'
```

## What to look at
- `connections.chat` — built-in SSE broker.
- `web/index.html` — vanilla `EventSource` subscriber, ~50 LOC.
- `stream-publish` route — body becomes the SSE `data:` payload.

## Caveats
- No history: subscribers only see messages sent after they connect.
- No moderation, no auth — for demos only.
