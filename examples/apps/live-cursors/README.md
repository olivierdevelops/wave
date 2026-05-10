# live-cursors

Multi-cursor demo. The page sends your mouse position to `/cursor`
(throttled to ~30Hz). The Wave broker fans every move out to all
subscribers, who render one another's cursors on a canvas.

## Run it
```sh
wave serve examples/apps/live-cursors/server.yaml --port 8207
```
Open http://127.0.0.1:8207/ in two or three tabs and wave the mouse around.

## Try it
```sh
# Inject a synthetic cursor from a script:
curl -X POST http://127.0.0.1:8207/cursor \
  -H 'Content-Type: application/json' \
  -d '{"user":"bot","x":300,"y":200}'
```

## What to look at
- `connections.cursors` — `buffer_size` is bumped to 256 because cursor
  events are chatty.
- `web/index.html` — `EventSource` + `requestAnimationFrame` render loop,
  ~50 LOC of vanilla JS.
- `event_type: cursor` — the SSE frame uses a custom event so the client
  can subscribe with `addEventListener('cursor', …)` rather than the
  default `message` event.

## Caveats
- No history; users joining mid-session don't see stale cursors.
- The client throttles to one POST per 30ms; lower for smoother demos
  but expect quadratic chatter as more tabs join.
- Cursor coordinates leak across all subscribers — don't ship this as-is
  in a multi-tenant app.
