# soft-delete-pattern

The standard "tombstone" pattern: rows are never physically deleted —
DELETE sets `deleted_at`; reads filter out rows where the column is
not NULL. A `/items/trash` admin route exposes the tombstoned rows;
`/items/{id}/restore` undoes the soft-delete.

## Run it

```sh
wave serve examples/apps/soft-delete-pattern/server.yaml --port 8108
```

## Try it

```sh
curl -s -X POST localhost:8108/items \
  -H 'content-type: application/json' \
  -d '{"name":"keep me"}'

curl -s -X POST localhost:8108/items \
  -H 'content-type: application/json' \
  -d '{"name":"trash me"}'

curl -s -X DELETE localhost:8108/items/2

curl -s localhost:8108/items          # → only "keep me"
curl -s localhost:8108/items/trash    # → only "trash me"

curl -s -X POST localhost:8108/items/2/restore
curl -s localhost:8108/items          # → both back
```

## What to look at

- The DELETE route is an UPDATE under the hood — clients still call
  `DELETE /items/{id}`, so the API stays RESTful.
- The `idx_items_alive` partial index in `migrations/001_init.up.sql`
  keeps the live-set lookups fast even when the trash grows.

## Caveats

- Foreign-key cascades won't fire on soft delete. Audit relationships
  before adopting the pattern across an existing schema.
- A periodic vacuum job (not shown) is usually paired with the
  pattern to hard-delete tombstones older than N days.
