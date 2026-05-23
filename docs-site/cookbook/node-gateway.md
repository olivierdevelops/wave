# Wave in front of an existing Node / Express service

You have a working Node service. You want to add auth, rate
limiting, webhook signature verification, audit logging, CORS, and
multi-tenant routing — *without* touching the Node code.

Put Wave in front. The Node service stops dealing with security
ceremony; Wave handles it declaratively.

## Architecture

```
                ┌──────────────────┐         ┌──────────────────┐
   client  ───► │  Wave gateway    │   ───►  │  Your Node       │
                │  - auth          │  HTTP   │  service         │
                │  - rate limit    │  on     │  (Express,       │
                │  - webhook sig   │  127.0  │   Fastify,       │
                │  - CORS          │  .0.1   │   Hono, …)       │
                │  - audit log     │         │  unchanged       │
                │  - host routing  │         │                  │
                └──────────────────┘         └──────────────────┘
```

Bind your Node service to `127.0.0.1` only; Wave is the public
entry.

## Step 1 — keep your Node service exactly as it is

```js
// existing-node-app/server.js — no changes
import express from 'express'

const app = express()
app.use(express.json())

app.get('/items',       async (req, res) => { /* ... */ })
app.post('/items',      async (req, res) => { /* ... */ })
app.delete('/items/:id', async (req, res) => { /* ... */ })

// Only listen on localhost — Wave handles public traffic
app.listen(9000, '127.0.0.1', () => console.log('node up on :9000'))
```

## Step 2 — Wave in front

`gateway.yaml`:

```yaml
default:
  port: 8080                            # the public port

env:
  JWT_SECRET:              { description: "session HMAC" }
  STRIPE_WEBHOOK_SECRET:   { description: "whsec_…" }
  NODE_URL:                { description: "Node backend URL", default: "http://127.0.0.1:9000" }

auth:
  app:
    type: jwt
    secret: "${env:JWT_SECRET}"
    cookie_name: session

limits:
  public_100rpm:
    case: rate_limited
    rps: 1.66
    burst: 30
    on_fail:
      status_code: 429
      headers: [["Retry-After", "60"], ["Content-Type", "application/json"]]
      body: '{"error":"rate limited"}'

  big_upload:
    case: body_too_large
    max_bytes: 52428800                  # 50 MB for /upload

routes:
  # Public webhook receiver — Wave verifies the signature, then
  # forwards the (verified) body to Node. Node never sees a bad
  # signature.
  - path: /webhooks/stripe
    methods: [POST]
    type: forward
    expected_content_type: application/json
    webhook_sig:
      provider: stripe
      secret: "${env:STRIPE_WEBHOOK_SECRET}"
      tolerance_sec: 300
    forward:
      forward_url: "${env:NODE_URL}/internal/stripe"
      include_headers:
        - ["X-Verified-By", "wave-gateway"]

  # Authenticated user API — auth + rate limit + audit, then forward
  - path: /api/
    methods: [GET, POST, PUT, PATCH, DELETE]
    type: forward
    auth: [app]
    limits: [public_100rpm]
    cors_origins: ["https://app.example.com"]
    cors_credentials: true
    forward:
      forward_url: "${env:NODE_URL}/api/"
      # Pass user identity to Node as a header — Node trusts the
      # gateway and doesn't re-validate the JWT
      include_headers:
        - ["X-User-Id",       "{{getUser}}"]
        - ["X-Forwarded-For", "{{getClientIP}}"]

  # File upload route — bigger body limit, same auth + forward
  - path: /api/uploads
    methods: [POST]
    type: forward
    auth: [app]
    limits: [public_100rpm, big_upload]
    cors_origins: ["https://app.example.com"]
    cors_credentials: true
    expected_content_type: multipart/form-data
    forward:
      forward_url: "${env:NODE_URL}/api/uploads"

  # Admin — IP allowlist + admin role + audit
  - path: /admin/
    methods: [GET, POST]
    type: forward
    auth: [app]
    require_roles: [admin]
    ip_whitelist: ["10.0.0.0/8", "203.0.113.0/24"]
    forward:
      forward_url: "${env:NODE_URL}/admin/"
      include_headers:
        - ["X-User-Id", "{{getUser}}"]

  # Public assets — no auth, but rate-limited
  - path: /public/
    methods: [GET]
    type: forward
    limits: [public_100rpm]
    forward:
      forward_url: "${env:NODE_URL}/public/"
```

## Step 3 — run them together

```sh
# Terminal 1 — your existing Node service
node existing-node-app/server.js

# Terminal 2 — Wave gateway
JWT_SECRET=$(openssl rand -hex 32) \
STRIPE_WEBHOOK_SECRET=whsec_test \
wave serve gateway.yaml --port 8080
```

## What Wave does for you here

| Concern | Where it lives |
|---|---|
| HTTPS termination | Your hosting platform (Fly, Caddy, ALB) |
| Auth | Wave (`auth: [app]`) |
| Session cookies | Wave (JWT in HttpOnly cookie) |
| Rate limiting | Wave (`limits:`) |
| CORS preflight | Wave (`cors_origins`) |
| Webhook signature verification | Wave (`webhook_sig:`) |
| Body size limits | Wave (`limits[body_too_large]`) |
| IP allow/deny | Wave (`ip_whitelist`) |
| RBAC | Wave (`require_roles`) |
| Audit log | Wave (`audit:` or [the audit recipe](/cookbook/audit-log)) |
| **Domain logic** | **Your Node app — unchanged** |

## Trusting the X-User-Id header

Wave's gateway sets `X-User-Id: <verified user id>` on the
forwarded request. Your Node service reads it directly:

```js
app.get('/api/me', async (req, res) => {
  const userId = req.get('X-User-Id')
  if (!userId) return res.status(500).send('no upstream user id')
  // ... query by userId
})
```

**Important**: this only works because Node binds to 127.0.0.1.
Anything that could bypass Wave (a misconfigured load balancer,
another local user) could forge the header. Two defenses:

1. Bind Node to a Unix socket instead of a port (Wave forwards
   to a socket if you set `forward_url: unix:///tmp/node.sock`).
2. Add a shared `X-Gateway-Secret` header that Node validates.

## Production deploy

- **Single VM**: run both processes on the same Fly VM /
  systemd unit. Wave on :8080 public; Node on 127.0.0.1:9000.
- **Kubernetes**: Wave as a sidecar in the same pod as Node, or
  a separate Deployment in the same namespace. Node has no public
  Service; Wave does.
- **systemd**: two units, `node.service` (Restart=always, bind
  127.0.0.1) and `wave.service` (After=node.service).

## When to *not* use Wave as a gateway

- Your team already runs an API gateway (Kong, Tyk, AWS API Gateway,
  Cloudflare Workers). Adding Wave duplicates concerns.
- You're already on a service mesh (Istio, Linkerd) with mTLS +
  auth policies — keep using that.
- The Node service has very low traffic (~1 RPS); a separate
  process for auth/rate-limit is overkill.

## See also

- [Forward Stripe webhooks](/cookbook/stripe-webhooks)
- [Rate-limit an endpoint](/cookbook/rate-limit)
- [Audit log every mutation](/cookbook/audit-log)
- [Wave in your stack](/guide/wave-in-your-stack)
- Demo: [`api-gateway-rate-limited`](https://github.com/luowensheng/wave/tree/main/examples/apps/api-gateway-rate-limited),
  [`jwt-validating-gateway`](https://github.com/luowensheng/wave/tree/main/examples/apps/jwt-validating-gateway)
