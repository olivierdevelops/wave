# streaming-file-processor

POST `/process` writes a `{processed, total, current_row}` event to the
`progress` SSE broker. Browser clients subscribe at `/events/progress`
and see updates as the producer publishes them.

## What it shows off

- `connections.progress` — SSE broker with keep-alive.
- `type: stream-publish` route fanning a request body to subscribers.
- `inputs:` validation (typed int + required string) on the producer.
- Inline browser UI under `/` consuming `EventSource`.

## Run

```sh
wave serve examples/apps/streaming-file-processor/server.yaml --port 8605
# Subscriber (browser or curl):
curl -N http://127.0.0.1:8605/events/progress
# Producer (loop to simulate row-by-row work):
for i in 1 2 3; do
  curl -s -X POST http://127.0.0.1:8605/process \
       -H 'Content-Type: application/json' \
       -d "{\"total\":3,\"processed\":$i,\"current_row\":\"row-$i\"}"
done
```

## Caveats

- Real CSV ingestion would live behind a plugin worker that publishes
  one row per iteration. This example keeps the wiring visible.
- Buffer size 64; slow subscribers get dropped events first.
