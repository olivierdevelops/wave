# stripe-webhook-receiver

POST `/webhooks/stripe` with a valid `Stripe-Signature` header. Wave verifies
the HMAC, then fans the parsed body out to all SSE subscribers on
`/events/payments`.

## Run it
```sh
STRIPE_WEBHOOK_SECRET=whsec_test \
  wave serve examples/apps/stripe-webhook-receiver/server.yaml --port 8201
```

## Try it
In one terminal, subscribe:
```sh
curl -N http://127.0.0.1:8201/events/payments
```
In another, send a signed event with the helper below.

### `tools/sign.sh` (Stripe)
```sh
#!/usr/bin/env bash
SECRET="${STRIPE_WEBHOOK_SECRET:-whsec_test}"
BODY='{"id":"evt_1","amount":4200,"currency":"usd"}'
TS=$(date +%s)
SIG=$(printf "%s.%s" "$TS" "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | awk '{print $2}')
curl -s -X POST http://127.0.0.1:8201/webhooks/stripe \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: t=$TS,v1=$SIG" \
  -d "$BODY"
```

## What to look at
- `webhook_sig: { provider: stripe }` in `server.yaml`
- The `connections.payments` block — built-in SSE broker
- `infra/webhooksig/webhooksig.go` for the Stripe envelope format

## Caveats
- Keep `STRIPE_WEBHOOK_SECRET` out of source control in real deployments.
- Tolerance is 300s; replays older than that are rejected.
