# api-gateway-rate-limited

Demonstrates Wave's **named limits** registry. Each entry in `limits:`
is a reusable policy bound to one `case` (rate_limited, body_too_large,
circuit_open, error, ...). Routes pull policies in by name, and
last-listed wins on conflict — so you compose bundles by listing.

## Run it

```sh
wave serve examples/apps/api-gateway-rate-limited/server.yaml
```

Listens on `http://localhost:8302`.

## Try it

A few normal calls:
```sh
curl http://localhost:8302/api/users
```

Hammer the admin route to trip the 5 rps limit:
```sh
for i in $(seq 1 30); do curl -s -o /dev/null -w "%{http_code}\n" \
  http://localhost:8302/api/admin; done
```

You should see a mix of `200` and `429` once the bucket drains.

Send a body bigger than 5 MB to trip `body_too_large`:
```sh
head -c 6000000 /dev/urandom | curl -X POST --data-binary @- \
  http://localhost:8302/api/orders
```

Returns `413`.

## What to look at

- `limits:` registry — named entries reused across routes.
- The admin route lists `[rate_100rps, rate_5rps, ...]` — both are case
  `rate_limited`, and the LATER one wins. This is the configuration
  cascade pattern.

## Caveats

Rate limits are per-IP by default. Use `key_claim:` on the limit entry
to scope by an authenticated claim instead.
