# tagged-search

Items with a JSON-array `tags` column, queryable by tag membership
using SQLite's `json1` extension (`json_each`).

## Run it

```sh
wave serve examples/apps/tagged-search/server.yaml --port 8107
```

## Try it

```sh
curl -s -X POST localhost:8107/items \
  -H 'content-type: application/json' \
  -d '{"name":"router","tags":["network","hardware"]}'

curl -s -X POST localhost:8107/items \
  -H 'content-type: application/json' \
  -d '{"name":"manual","tags":["docs"]}'

# All items
curl -s localhost:8107/items
# Filtered
curl -s 'localhost:8107/items?tag=network'
curl -s localhost:8107/items/1
```

## What to look at

- `tags` is stored as a JSON text column; the POST route wraps the
  declared `tags` input with `json(...)` to validate/normalise.
- The list query uses `json_each(i.tags)` to filter by a single tag.
- `({{tag}} = '' OR EXISTS …)` is the canonical way to make the
  query clause optional inside a single SQL string.

## Caveats

- The `inputs:` spec for `tags` accepts any JSON value (object/array);
  passing a non-array will produce an SQLite error at INSERT time.
- For high-volume tag queries you'd want a separate `item_tags(item_id,
  tag)` table with an index; this is the pattern, not the optimisation.
