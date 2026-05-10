# cors-proxy

A reverse proxy that adds permissive CORS headers so browser-side code
can hit upstream APIs that don't set CORS themselves.

## Run it

```sh
wave serve examples/apps/cors-proxy/server.yaml
```

Listens on `http://localhost:8303`.

## Try it

Simulate a browser preflight:
```sh
curl -i -X OPTIONS http://localhost:8303/proxy/get \
  -H 'Origin: https://example.com' \
  -H 'Access-Control-Request-Method: GET'
```

You should see `Access-Control-Allow-Origin: *` and a `204` (or `200`)
with no body.

Then the actual GET:
```sh
curl -i http://localhost:8303/proxy/get -H 'Origin: https://example.com'
```

The response includes the CORS headers and the upstream JSON body.

## What to look at

`cors_origins: ["*"]` on the route — Wave's per-route CORS handling
both:

1. Answers `OPTIONS` preflights itself (without round-tripping to the
   upstream).
2. Adds `Access-Control-Allow-Origin` to actual responses.

`include_headers` shows how to inject a constant header into upstream
requests.

## Caveats

`*` is fine for public, read-only data. For anything user-specific,
list explicit origins (`cors_origins: ["https://app.example.com"]`) —
`*` plus credentials is rejected by browsers.
