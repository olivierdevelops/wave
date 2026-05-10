# notes-app

A minimal CRUD example: SQLite-backed notes with a tiny browser UI.

## Run it

```sh
wave serve examples/apps/notes-app/server.yaml --port 8101
```

Then open http://127.0.0.1:8101/app for the UI.

## Try it

```sh
curl -s localhost:8101/notes | jq
curl -s -X POST localhost:8101/notes \
  -H 'content-type: application/json' \
  -d '{"title":"hello","body":"first note"}'
curl -s localhost:8101/notes/1 | jq
curl -s -X DELETE localhost:8101/notes/1
```

## What to look at

- `server.yaml` — one route per CRUD verb, all of `type: storage-access`.
- `inputs:` declares typed path/body params; `{{.title}}` becomes a
  bound parameter so SQL injection is impossible.
- `web/index.html` — vanilla JS that calls the JSON API.

## Caveats

- Tables are auto-created from `storage.notes.tables`. The
  `migrations/` directory is shipped for the alternative
  `wave migrate up --db ./data.db --dir ./migrations` workflow.
- `data.db` is created next to `server.yaml` on first boot.
