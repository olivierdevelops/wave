# event-fanout-hub

Three named SSE connections (`payments`, `users`, `audit`) and three
ingest routes that each fan out to a different broker. Subscribers pick
which streams they care about.

## Run it
```sh
wave serve examples/apps/event-fanout-hub/server.yaml --port 8206
```

## Try it
```sh
# Tail one (or all) of the streams:
curl -N http://127.0.0.1:8206/events/payments &
curl -N http://127.0.0.1:8206/events/users &
curl -N http://127.0.0.1:8206/events/audit &

# Push events:
curl -X POST http://127.0.0.1:8206/ingest/payments \
  -H 'Content-Type: application/json' \
  -d '{"type":"payment","amount":4200}'

curl -X POST http://127.0.0.1:8206/ingest/users \
  -H 'Content-Type: application/json' \
  -d '{"type":"user","action":"signup"}'

curl -X POST http://127.0.0.1:8206/ingest/audit \
  -H 'Content-Type: application/json' \
  -d '{"type":"audit","actor":"root"}'
```

## What to look at
- Three entries in `connections:` — each gets its own subscribe path.
- `request_schema:` rejects mismatched bodies at the door.
- `static_meta.topic` lets subscribers see which broker delivered a
  given frame even when streams are mux'd.

## Caveats
- The Wave `stream-publish` type does not yet support a single-ingest
  `condition:` filter that routes one POST to multiple brokers based on
  a payload field. This example uses one route per topic instead, which
  is the idiomatic shape today. If you need true field-conditional
  fan-out, layer a small plugin (`type: plugin`) in front that calls
  the per-topic ingest URLs.
