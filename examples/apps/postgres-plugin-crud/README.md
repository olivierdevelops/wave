# postgres-plugin-crud

Items CRUD over Postgres, served entirely through the `postgres-storage`
reference plugin. No built-in SQLite — every read and write travels
JSON-RPC to a separate plugin process that talks to Postgres via pgx.

## Setup

1. Build the plugin binary (path matches `command:` in `server.yaml`):

   ```sh
   cd examples/plugins/postgres-storage && go build -o /tmp/wave-postgres .
   ```

2. Have a Postgres instance reachable. Create the table:

   ```sql
   CREATE TABLE items (
     id    SERIAL PRIMARY KEY,
     name  TEXT NOT NULL,
     price NUMERIC NOT NULL DEFAULT 0
   );
   ```

3. Export the DSN:

   ```sh
   export PG_DSN="postgres://user:pass@localhost:5432/wave?sslmode=disable"
   ```

## Run it

```sh
wave serve examples/apps/postgres-plugin-crud/server.yaml --port 8701
```

## Try it

```sh
curl -XPOST localhost:8701/items -d '{"name":"widget","price":9.99}' -H 'content-type: application/json'
curl localhost:8701/items
curl -XDELETE localhost:8701/items/1
```

## What to look at

`plugins.pg_main` declares `kind: storage`; the route's `source: pg_main`
resolves to that plugin instead of a built-in storage. The plugin
process is spawned lazily on the first request and kept alive.

## Caveats

If `/tmp/wave-postgres` is missing or `PG_DSN` is unreachable the boot
fails fast with a clear error. Inputs are interpolated via `text/template`
— quote dynamic values in the SQL until parameter binding lands.
