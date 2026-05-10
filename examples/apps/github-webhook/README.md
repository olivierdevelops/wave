# github-webhook

POST `/webhooks/github` — Wave verifies `X-Hub-Signature-256` (HMAC-SHA256
of the raw body), then republishes the payload on the `github` SSE broker.

To inspect events live, `curl -N` the SSE stream — that's the easiest way
to "pretty print" arrivals. Filter by `X-GitHub-Event` client-side or with
`jq` on the curl output.

## Run it
```sh
GITHUB_WEBHOOK_SECRET=ghsecret \
  wave serve examples/apps/github-webhook/server.yaml --port 8202
```

## Try it
```sh
# In one terminal — tail events:
curl -N http://127.0.0.1:8202/events/github

# In another — send a signed push event:
./tools/sign.sh push '{"ref":"refs/heads/main","head_commit":{"id":"abc"}}'
```

### `tools/sign.sh` (GitHub)
```sh
#!/usr/bin/env bash
SECRET="${GITHUB_WEBHOOK_SECRET:-ghsecret}"
EVENT="${1:-push}"
BODY="${2:-{\"zen\":\"hello\"}}"
SIG=$(printf "%s" "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')
curl -s -X POST http://127.0.0.1:8202/webhooks/github \
  -H "Content-Type: application/json" \
  -H "X-GitHub-Event: $EVENT" \
  -H "X-Hub-Signature-256: sha256=$SIG" \
  -d "$BODY"
```

## What to look at
- `webhook_sig: { provider: github }` — handles the `sha256=` prefix.
- `infra/webhooksig/webhooksig.go` — `githubVerifier`.

## Caveats
- The verifier reads/replaces `r.Body`; we don't currently echo the
  `X-GitHub-Event` header into the SSE payload, so use the streamed body
  + your terminal-side filter to distinguish event types.
