# analytics-collector

Privacy-respecting page-view collector. Sites POST page views; the
admin dashboard renders top paths + a per-day traffic chart. No
IPs, no PII — only what the client chooses to send.

## Run it

```sh
wave serve examples/apps/analytics-collector/server.yaml --port 8508
```

## Try it

- Open http://127.0.0.1:8508/demo and click a few buttons. They
  POST a page-view event for site `demo`.
- Open http://127.0.0.1:8508/dashboard to see counts and a 14-day
  traffic chart.

```sh
curl -s -X POST localhost:8508/collect \
  -H 'content-type: application/json' \
  -d '{"site_id":"mysite","path":"/home","referrer":"https://google.com","ua_hash":"abc12345"}'

# Dashboard HTML
curl -s localhost:8508/dashboard | head -40
```

## What to look at

- Declared `inputs:` is the entire schema contract for /collect —
  unknown fields are dropped, types coerced, lengths bounded.
- The dashboard SQL packs four aggregates into one row using
  correlated subqueries + `json_group_array`. The
  `output_template` then renders the HTML server-side and a tiny
  inline script draws the bar charts.
- The `ua_hash` field on the demo client uses the WebCrypto API
  to hash the UA in the browser before sending — the server never
  sees the raw string.

## Caveats

- `/dashboard` has no auth in this demo. Add an `auth:` block
  (jwt + magic-link, oidc) or wrap with `forward_auth:` for prod.
- No bot filtering, no sampling, no aggregation rollups; this is
  the "simplest thing that could possibly work" version.

## What it shows off

`infra/inputs` declared payload validation · `storage-access` for
write + read aggregations · server-side HTML templating ·
client-side hashing for privacy.
