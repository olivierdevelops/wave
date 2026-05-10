# pomodoro

A shared 25-minute timer per room. Anyone in `?room=team-a` sees
the same clock; start/pause/reset broadcasts to every viewer.

## Run it

```sh
wave serve examples/apps/pomodoro/server.yaml --port 8507
```

## Try it

- Open http://127.0.0.1:8507/app?room=team-a in two tabs.
- Hit "start" in one — both count down in lock-step.
- Pause in one tab — both pause. Reset returns to 25:00.

```sh
curl -s localhost:8507/rooms/team-a | jq
curl -s -X POST localhost:8507/rooms/team-a/start
curl -s -X POST localhost:8507/rooms/team-a/pause
```

## What to look at

- The server stores only three things per room: state, remaining
  seconds, and the epoch-second the timer last started. The
  client computes the live tick locally so the server only needs
  to broadcast on state changes — not every second.
- The "auto-create on first read" pattern uses
  `INSERT … ON CONFLICT(name) DO NOTHING` so GET is idempotent
  even for brand-new rooms.
- `/rooms/notify` is a separate stream-publish route so the SQL
  state mutation and the broker fan-out are independent.

## Caveats

- No auth; anyone who knows the room name controls the timer.
- The 25-minute duration is hard-coded; tweak `remaining` in the
  reset SQL and the `1500` in the rooms table default.

## What it shows off

server-as-state-store · ON CONFLICT upsert · path-param routes ·
SSE state-change fan-out · client-side time interpolation.
