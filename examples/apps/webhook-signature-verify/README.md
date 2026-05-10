# webhook-signature-verify

Demonstrates Wave's `provider: generic` HMAC verifier. Three routes share
one secret but differ in algorithm / header value format.

## Run it
```sh
WAVE_HMAC_SECRET=topsecret \
  wave serve examples/apps/webhook-signature-verify/server.yaml --port 8205
```

## Try it
```sh
SECRET=topsecret
BODY='{"order":42}'

# /v1/sha256 — bare hex
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')
curl -X POST http://127.0.0.1:8205/v1/sha256 \
  -H "X-Signature: $SIG" -H "Content-Type: application/json" -d "$BODY"

# /v1/sha1
SIG=$(printf '%s' "$BODY" | openssl dgst -sha1 -hmac "$SECRET" | awk '{print $2}')
curl -X POST http://127.0.0.1:8205/v1/sha1 \
  -H "X-Signature: $SIG" -H "Content-Type: application/json" -d "$BODY"

# /v1/sha256-prefixed — value is "sha256=<hex>"
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')
curl -X POST http://127.0.0.1:8205/v1/sha256-prefixed \
  -H "X-Signature: sha256=$SIG" -H "Content-Type: application/json" -d "$BODY"
```

Subscribe in another terminal to see verified bodies arrive:
```sh
curl -N http://127.0.0.1:8205/events/hmac
```

## What to look at
- Each `webhook_sig:` block in `server.yaml`.
- `infra/webhooksig/webhooksig.go` — the `genericVerifier`.

## Caveats
- The built-in generic verifier supports `sha256` and `sha1`. SHA-512
  isn't wired up in this build, so this example uses a `sha256-prefixed`
  variant in place of the originally planned `/v1/sha512`.
- Constant-time comparison is used internally — don't roll your own.
