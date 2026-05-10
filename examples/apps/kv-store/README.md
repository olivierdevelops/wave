# kv-store

Tiny generic K/V API: `GET`, `PUT`, `DELETE` keyed by URL path. Backed
by a SQLite `kv(k, v)` table with `ON CONFLICT … DO UPDATE` for
upsert.

## Run it

```sh
wave serve examples/apps/kv-store/server.yaml --port 8105
```

## Try it

```sh
curl -s -X PUT localhost:8105/kv/greeting \
  -H 'content-type: text/plain' \
  --data-binary 'hello world'

curl -s localhost:8105/kv/greeting
curl -s localhost:8105/kv
curl -s -X DELETE localhost:8105/kv/greeting
```

## What to look at

- The `body` declared input grabs the raw request body as a string;
  combine with `--data-binary` for binary-safe payloads.
- The `ON CONFLICT(k) DO UPDATE` clause keeps the route idempotent.

## Caveats

- Storage type is BLOB but values come in as strings via the
  text/template path — for large or true-binary content, look at the
  `file-uploads` example which uses multipart instead.
