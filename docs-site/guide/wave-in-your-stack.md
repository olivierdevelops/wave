# Wave in your stack

Wave isn't trying to replace React, Node, or Python in your existing
project. It works **alongside** them, taking over the parts of your
backend that are mostly declarative — and letting your existing
codebase keep doing what it does well.

This page covers the four common shapes.

---

## Pattern 1 — Wave as a Backend-For-Frontend (BFF)

You have a React / Next.js / Vue / Svelte frontend. You want a
small, focused API: auth, a few resource endpoints, maybe a Stripe
webhook. The full Express+Prisma+Auth-library stack is overkill.

```
┌──────────────┐  fetch    ┌──────────────┐    SQL    ┌──────────┐
│  React app   │ ────────► │  Wave server │ ────────► │  SQLite  │
│  (Next.js,   │  /api/*   │  server.yaml │           │ Postgres │
│   Vite, etc.)│           └──────┬───────┘           └──────────┘
└──────────────┘                  │
                                  │ optional webhooks / SSE
                                  ▼
                          ┌──────────────┐
                          │  3rd parties │
                          │ (Stripe, etc)│
                          └──────────────┘
```

### Setup

1. Put your React build artifacts wherever your frontend host serves
   them (Vercel, Netlify, S3 + CloudFront). Wave doesn't need to know.
2. Run Wave on its own subdomain (`api.example.com`) or behind your
   frontend host's edge (Vercel's `rewrites:`, Next.js
   `next.config.js`'s `proxy:`).
3. Wave handles every backend concern; your frontend `fetch()`es it.

### CORS for browser requests

```yaml
- path: /api/me
  methods: [GET]                       # plural — preflight-safe
  type: storage-access
  auth: [app]
  cors_origins:
    - "https://my-app.vercel.app"
    - "http://localhost:3000"
  cors_credentials: true               # send/receive cookies cross-origin
  storage-access: { ... }
```

For local dev, point the frontend dev server at the Wave port:

```js
// next.config.js
module.exports = {
  async rewrites() {
    return [{ source: '/api/:path*', destination: 'http://localhost:8080/api/:path*' }]
  },
}
```

### Why this works well

- Wave's auth cookie + your frontend cookie store work natively
  cross-origin (no NextAuth/Clerk needed for the simple cases)
- Your frontend stays a pure presentation layer
- Backend deploy and frontend deploy are independent

### When to NOT use Wave as BFF

- You need server-side rendering with deep backend access (use
  Next.js API routes or a real BFF)
- Your frontend uses tRPC / gRPC-Web — Wave doesn't speak those

---

## Pattern 2 — Wave in front of your existing backend (gateway)

You already have a working Node / Python / Rails / Go backend. You
want to add auth, rate limiting, webhook signature verification,
multi-tenant routing, or audit logging — without changing the
backend.

```
┌──────────┐    ┌──────────────┐  /api/*    ┌──────────────────┐
│  Client  │ ──►│ Wave gateway │ ──────────►│  Your existing   │
│          │    │   - auth     │  /webhooks │  backend (Node,  │
│          │    │   - rate lim │  /admin    │  Python, Rails)  │
│          │    │   - audit    │            │                  │
│          │    │   - CORS     │            └──────────────────┘
└──────────┘    └──────────────┘
```

### Setup

A few lines of `server.yaml`:

```yaml
default:
  port: 8080

env:
  BACKEND_URL: { description: "URL of the existing service" }

auth:
  app:
    type: jwt
    secret: "${env:JWT_SECRET}"
    cookie_name: session

limits:
  global_rate:
    case: rate_limited
    rps: 100
    burst: 200

routes:
  # Public webhook receiver — verify signature, then forward
  - path: /webhooks/stripe
    methods: [POST]
    type: forward
    webhook_sig:
      provider: stripe
      secret: "${env:STRIPE_WEBHOOK_SECRET}"
    forward:
      forward_url: "${env:BACKEND_URL}/internal/stripe"

  # Authenticated user API — gate, audit, then forward
  - path: /api/
    methods: [GET, POST, PUT, PATCH, DELETE]
    type: forward
    auth: [app]
    limits: [global_rate]
    cors_origins: ["https://app.example.com"]
    forward:
      forward_url: "${env:BACKEND_URL}/api/"
      include_headers:
        - ["X-User-Id", "{{getUser}}"]
        - ["X-Forwarded-By", "wave"]

  # Admin — RBAC + audit log + IP allowlist
  - path: /admin/
    methods: [GET, POST]
    type: forward
    auth: [app]
    require_roles: [admin]
    ip_whitelist: ["10.0.0.0/8", "192.168.0.0/16"]
    forward:
      forward_url: "${env:BACKEND_URL}/admin/"
```

Your existing backend keeps running — it just stops dealing with
auth, signature verification, CORS, rate limits, RBAC, and IP
filtering.

### What this saves you

- ~40% of typical backend code that's auth/middleware ceremony
- One place to audit security (not 47 individual route handlers)
- One config file your platform team can review

### Demos

- [`api-gateway-rate-limited`](https://github.com/luowensheng/wave/tree/main/examples/apps/api-gateway-rate-limited)
- [`jwt-validating-gateway`](https://github.com/luowensheng/wave/tree/main/examples/apps/jwt-validating-gateway)

---

## Pattern 3 — Wave as a sidekick to a Python ML / data service

You have a Python service (FastAPI, Flask, or a model server). You
want it on the public internet with proper auth, rate limiting,
and observability — without re-implementing them in Python or
deploying a separate API gateway.

```
┌──────────┐    ┌──────────────┐    ┌────────────────┐
│  Client  │ ──►│ Wave         │ ──►│  Python ML     │
│          │    │ - auth       │    │  (FastAPI on   │
│          │    │ - rate-limit │    │   localhost or │
│          │    │ - SSE bridge │    │   k8s service) │
│          │    │ - audit      │    │                │
└──────────┘    └──────────────┘    └────────────────┘
                       │
                       ▼ each call
                ┌──────────────┐
                │  SQLite for  │
                │  audit + req │
                │  counts      │
                └──────────────┘
```

Plus `type: task` lets you turn synchronous Python ML calls into
202 + SSE-streamed progress for free.

### Variants

- **Python is HTTP**: use `type: forward` to proxy
- **Python is a worker**: use `type: plugin` with `kind: http`
- **Python is local**: use `type: plugin` with `kind: subprocess`
  for stateless calls, or `kind: longlived` for stateful

### Demos

- [`background-task-demo`](https://github.com/luowensheng/wave/tree/main/examples/apps/background-task-demo) —
  task → SSE pattern, plugin is Python
- [`streaming-file-processor`](https://github.com/luowensheng/wave/tree/main/examples/apps/streaming-file-processor)

---

## Pattern 4 — Carving off a sub-system incrementally

You have a Node monolith. You want to move just the auth flow (or
the webhook receivers, or the admin dashboard) into Wave because
it's easier to maintain there.

The migration is a route at a time:

1. Add Wave alongside, listening on a different port
2. Move one route family (`/auth/*`) to Wave
3. Point your frontend / load balancer at Wave for that prefix
4. Delete the equivalent code from the Node app
5. Repeat with the next route family

There's no big-bang migration. Each step is a 1-day change.

The reverse also works — if Wave doesn't fit a feature, carve that
*back out* into your Node service and have Wave forward to it.

### Concrete recipe

→ [Migrating from Express incrementally](/cookbook/migrate-from-express)

---

## What runs where: a decision tree

```
Need to do something on a request?
│
├─ Database CRUD with validation        → Wave route (storage-access)
├─ Webhook with signature verification  → Wave route (stream-publish)
├─ Auth flow (login/oauth/magic-link)   → Wave route (auth-* / oauth-* / magic-link-*)
├─ Reverse proxy with auth + rate limit → Wave route (forward)
├─ Cron job                             → Wave schedule
├─ Background work with progress events → Wave task + Python/Go plugin
├─ Static file serving                  → Your CDN, NOT Wave (better tools exist)
├─ React/Vue/Svelte rendering           → Your frontend host
├─ Heavy domain logic                   → Plugin (any language)
└─ Real-time game / streaming media     → Custom — not Wave's sweet spot
```

## When NOT to add Wave to an existing stack

- Your team is **happy** with their existing setup — don't add a
  third moving part for the sake of it.
- Your service is **mostly background processing** with little HTTP
  surface — Wave's value is in the HTTP layer.
- You're **already using a service mesh** (Istio, Linkerd) for
  auth + rate-limit — Wave overlaps with that.

## See also

- [Token efficiency](/ai/token-efficiency) — why AI-assisted
  development is cheaper in Wave
- [Comparison](/guide/comparison) — Wave vs Express, FastAPI,
  Caddy, Hasura
- [Cookbook](/cookbook/) — concrete patterns including
  [React + Wave](/cookbook/react-wave),
  [Wave in front of Node](/cookbook/node-gateway),
  [Wave with a Python service](/cookbook/python-sidekick),
  [Migrating from Express](/cookbook/migrate-from-express)
