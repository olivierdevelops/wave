# static-and-api

One Wave server hosting **both** a static SPA frontend and a
reverse-proxied backend API on the same port. The single-binary,
single-port deployment is the most common Wave use case.

## Run it

```sh
wave serve examples/apps/static-and-api/server.yaml
```

Listens on `http://localhost:8306`.

## Try it

In a browser, visit `http://localhost:8306/` and click the
**Ping /api/get** button. The fetch goes to `/api/get` on the same
origin (no CORS!), Wave proxies it to `https://httpbin.org/get`, and
the JSON response renders inline.

Or from the shell:
```sh
curl http://localhost:8306/             # → index.html
curl http://localhost:8306/api/get      # → forwarded to httpbin
```

## What to look at

- `routes:` lists the API route FIRST. Order matters: more-specific
  patterns must be registered before the catch-all `/` static route or
  they'll be shadowed.
- `static.dir: ./web` — directory served as the SPA root.
- The frontend uses a relative `/api/...` URL — so the same code works
  in dev, staging, and prod without environment-specific config. No
  CORS preflight, no cookie-domain headaches.

## Caveats

For a real SPA with client-side routing, you'd add a `not_found:`
fallback that serves `index.html` so deep links work. Out of scope for
this minimal example.
