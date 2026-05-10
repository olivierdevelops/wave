# pastebin

Anonymous code-paste service in ~120 lines of YAML. POST text, get
back a short slug, view it later with syntax highlighting.

## Run it

```sh
wave serve examples/apps/pastebin/server.yaml --port 8502
```

## Try it

Open http://127.0.0.1:8502/app for the paste form.

```sh
# Create a paste — returns {"id": N}
ID=$(curl -s -X POST localhost:8502/pastes \
  -H 'content-type: application/json' \
  -d '{"content":"console.log(\"hi\")","language":"javascript"}' | jq -r .id)

# Look up the slug
SLUG=$(curl -s localhost:8502/pastes/$ID/slug | jq -r '.slug // .[0].slug')

curl -s localhost:8502/raw/$SLUG
echo "View HTML at http://localhost:8502/p/$SLUG"

# With expiry + view limit
curl -s -X POST localhost:8502/pastes \
  -H 'content-type: application/json' \
  -d '{"content":"top secret","expires_at":"2030-01-01 00:00:00","view_limit":3}' | jq
```

## What to look at

- `server.yaml` — three storage-access routes do all the work.
- The slug is generated server-side via SQLite's `randomblob(5)`.
- `expires_at` and `view_limit` are filtered at read time; pastes
  silently 404 once they expire or hit their cap.
- Server-side rendering of the HTML view via `output_template` —
  no Node, no React, just Go templates.
- Prism is loaded from a CDN for highlighting.

## Caveats

- No CSRF tokens on /pastes — the API is meant to be open.
- `data.db` is created next to `server.yaml`.

## What it shows off

`storage-access` write + read · `inputs:` validation · server-side
HTML templating · path parameters · multi-statement SQL execute.
