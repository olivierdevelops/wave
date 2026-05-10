# multi-tenant-routing

One Wave route definition that forwards each request to a *different*
upstream chosen at request time from a path parameter. Useful for
multi-tenant gateways where each tenant has its own backend.

## Run it

```sh
wave serve examples/apps/multi-tenant-routing/server.yaml
```

Listens on `http://localhost:8305`.

## Try it

```sh
curl http://localhost:8305/tenants/foo/anything
curl http://localhost:8305/tenants/bar/anything
```

The first request is forwarded to `https://httpbin.org/anything/tenant/foo`,
the second to `…/anything/tenant/bar` — the `{tenant}` placeholder in
`forward_url` is substituted from the URL path on each request.

## What to look at

- `forward.forward_url: https://httpbin.org/anything/tenant/{tenant}` —
  the curly-brace placeholder.
- The matching `{tenant}` segment in the route's `path` — Wave's mux
  exposes it via Go 1.22 `r.PathValue("tenant")`, which the forward
  proxy resolves per request.

In a real deployment you'd point this at internal hosts:
`forward_url: https://{tenant}.api.internal.svc/v1` — same mechanism.

## Caveats — read this before deploying

- **Closed allowlist required.** Letting users supply the tenant
  directly in the path (or worse, in a header) means an attacker can
  point you at any host they like. Either:
  - Validate `{tenant}` against a known allowlist before this route
    runs (e.g. a small `request_schema` regex / `inputs` enum), or
  - Use `dynamic-forward` with `allowed_domains` and
    `block_private_ips: true` set (see `usecases/dynamic_forward`).
- The example above uses `httpbin.org` for any tenant value — that's
  fine for a demo, **not** for production.
