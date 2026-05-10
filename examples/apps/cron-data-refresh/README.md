# cron-data-refresh

In-process scheduler hits an upstream API every 5 minutes via a plugin.
GET `/cached` reads the upstream synchronously; POST `/refresh`
triggers the same plugin manually.

## What it shows off

- Top-level `schedule:` block — `every: 5m` triggers a plugin call.
- HTTP-transport plugin (`transport: http`) talking to a real upstream.
- `type: forward` for an on-demand cached read.
- `storage.cache.tables` declared for an exporter plugin to land into.

## Run

```sh
wave serve examples/apps/cron-data-refresh/server.yaml --port 8609

curl http://127.0.0.1:8609/cached
curl -X POST http://127.0.0.1:8609/refresh
```

Watch stderr — the scheduler logs each tick with the plugin response.

## Caveats

- The 5-minute interval applies even on first boot; the first cron
  run happens after one full period.
- Persisting the plugin response to `cached_payload` would normally be
  done by an exporter plugin or a small app-side handler.
