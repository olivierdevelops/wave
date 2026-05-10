# rate-limited-public-api

Two named rate limits in the top-level registry, composed onto routes
by name. Authenticated users get their own bucket; anonymous traffic
shares one strict bucket.

## What it shows off

- Top-level `limits:` registry with two named entries.
- `case: rate_limited` + `rps`/`burst`/`key_claim`.
- `on_fail.status` + `headers` to control the 429 response shape.
- Route-level `limits: [name]` composition (no inline rate config).

## Run

```sh
SECRET_KEY=devsecret \
  wave serve examples/apps/rate-limited-public-api/server.yaml --port 8604

# Anonymous: hammer to see 429 within seconds.
for i in $(seq 1 20); do curl -s -o /dev/null -w "%{http_code}\n" \
  http://127.0.0.1:8604/api/anon; done

# Authenticated (forge a JWT — requires sub claim):
#   curl -H 'Authorization: Bearer <jwt>' http://127.0.0.1:8604/api/cheap
```

## Caveats

- `key_claim: sub` falls back to client IP when the request is anonymous.
- The default JWT secret is `dev-secret-change-me`; replace via env.
