# websocket-echo

WebSocket fan-out: POST `/echo` (any JSON) goes out to every browser
connected to `/ws/echo`.

## Run it
```sh
wave serve examples/apps/websocket-echo/server.yaml --port 8204
```
Open http://127.0.0.1:8204/ in two tabs; messages sent from one appear in
both because they're fanned out via WebSocket.

## Try it
```sh
# Subscribe with websocat or browser; publish via plain HTTP:
curl -X POST http://127.0.0.1:8204/echo \
  -H 'Content-Type: application/json' \
  -d '{"hello":"ws"}'
```

## What to look at
- `connections.echo: { type: ws }` — switches the broker to WebSocket.
- `web/index.html` — minimal `WebSocket` subscriber.
- `infra/connections/ws.go` — the WS upgrade handler.

## Caveats
- No backpressure handling beyond the broker's per-client buffer.
- No auth; add `subscribe_auth: [...]` for token-gated subscribes.
