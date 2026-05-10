# postgres-crud

CRUD over a Postgres `items` table via the `postgres-storage` reference
plugin. Demonstrates the `kind: storage` plugin pattern from
`docs/storage-plugins.md`.

## Build the plugin

The plugin binary is NOT shipped — you build it locally:

```sh
cd examples/plugins/postgres-storage
go build -o /tmp/wave-postgres .
```

## Prepare the DB

```sh
psql "$PG_DSN" -c "
CREATE TABLE IF NOT EXISTS items (
  id         SERIAL PRIMARY KEY,
  name       TEXT NOT NULL,
  qty        INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);"
```

## Run

```sh
PG_DSN=postgres://user:pass@localhost:5432/wave \
  wave serve examples/apps/postgres-crud/server.yaml --port 8104
```

## Try

```sh
curl -s -X POST localhost:8104/items \
  -H 'content-type: application/json' \
  -d '{"name":"widget","qty":3}'
curl -s localhost:8104/items
curl -s localhost:8104/items/1
```

## What to look at

- `plugins.pg_main` declares `kind: storage` and `transport: process`.
  Wave spawns the binary lazily, talks JSON-RPC, restarts on exit.
- The route `source: pg_main` falls through the built-in storage
  registry and resolves to the plugin.
- See `docs/storage-plugins.md` for the resolution order and the
  Phase 2 quoting caveat.

## Caveats

- Boot will FAIL until both `/tmp/wave-postgres` exists AND `PG_DSN`
  resolves to a reachable database. That's expected for the smoke
  test — the other 7 apps have no such requirement.
- The current plugin contract substitutes `text/template` (not `$1`)
  so SQL values must be wrapped in literal quotes inside the template.
