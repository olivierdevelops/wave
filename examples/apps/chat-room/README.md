# chat-room

One global chat room. No login, no signup, no rooms — just type a
nick and start chatting. Last 100 messages persist in SQLite so
you have history when you join.

## Run it

```sh
wave serve examples/apps/chat-room/server.yaml --port 8505
```

## Try it

Open http://127.0.0.1:8505/app in two browser tabs, type messages
in one, see them appear instantly in the other.

```sh
curl -s localhost:8505/history | jq
curl -s -X POST localhost:8505/messages \
  -H 'content-type: application/json' \
  -d '{"nick":"curl-bot","body":"hello world"}'
curl -N localhost:8505/events/room
```

## What to look at

- One SQLite table, one SSE connection, three storage routes.
- The frontend POSTs each message twice: once to `/messages` for
  persistence, once to `/broadcast` for real-time fan-out. They're
  decoupled so a broker hiccup never loses a message.
- `EventSource` + `addEventListener('msg', ...)` matches the
  `event_type: msg` set in the stream-publish route.

## Caveats

- No moderation. No auth. Trivially exploitable.
- Old messages stay forever; add a sweep job if you care.

## What it shows off

SSE broker · stream-publish · `inputs:` validation · backfill+live
pattern · vanilla JS frontend with no build step.
