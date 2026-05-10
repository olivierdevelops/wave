# basic-forward

The smallest useful Wave config: one `forward` route that proxies
`/api/*` straight through to a public test upstream.

## Run it

```sh
wave serve examples/apps/basic-forward/server.yaml
```

Listens on `http://localhost:8301`.

## Try it

```sh
curl http://localhost:8301/api/get
curl -X POST -d 'hello=world' http://localhost:8301/api/post
```

Both requests are forwarded to `https://httpbin.org/get` and
`https://httpbin.org/post` respectively. The response is whatever
httpbin returns — Wave just shuttles bytes.

## What to look at

`forward.forward_url` — the upstream base URL. The path under `/api/`
gets appended verbatim (so `/api/anything/foo` → `…/anything/foo`).

## Swap the upstream

Change `forward_url` to any HTTPS endpoint — `https://api.github.com`,
`http://localhost:9000`, an internal service, etc. No restart trickery
needed; Wave reads the config at boot.

## Caveats

The default `forward` proxy preserves the request method, body, and
headers. If your upstream needs auth, add `include_headers:` to the
`forward:` block.
