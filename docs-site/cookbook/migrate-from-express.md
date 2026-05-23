# Migrating from Express incrementally

You have an Express monolith. You want to move parts of it to Wave
because they'd be shorter, safer, and easier to maintain there.
A big-bang rewrite is out. Here's the route-at-a-time path.

## The strategy

1. **Don't change the Express app at all** at first.
2. **Add Wave alongside** on a different port.
3. **Pick the easiest route family** (usually auth or webhooks).
4. **Move it to Wave** — write the YAML equivalent.
5. **Point your load balancer / frontend** at Wave for that prefix
   only. Everything else keeps hitting Express.
6. **Delete the migrated routes from Express.**
7. **Repeat** with the next route family until Express is empty or
   only holds genuinely complex domain code.

Each step is a 1-day change. You can stop at any point.

## Why this works

The routing happens at the **path prefix** level (in your load
balancer, nginx, Caddy, Fly's [http_options.routes], or Cloudflare).
The client sees one origin; Wave and Express each handle the
prefixes they own.

## Example: migrating an Express app

### Starting point

```js
// app.js — Express monolith
import express from 'express'
import { signupRouter }  from './routes/signup.js'
import { loginRouter }   from './routes/login.js'
import { stripeWebhook } from './routes/stripe.js'
import { todosRouter }   from './routes/todos.js'
import { complexBilling } from './routes/billing.js'

const app = express()
app.use('/auth/signup',   signupRouter)        // 80 lines
app.use('/auth/login',    loginRouter)         // 60 lines
app.use('/webhooks/stripe', stripeWebhook)     // 50 lines
app.use('/api/todos',     todosRouter)         // 180 lines
app.use('/api/billing',   complexBilling)      // 600 lines — complex
app.listen(3000)
```

About 1000 lines + ~5 dependencies. Most of it is auth + webhook
+ CRUD ceremony.

### Step 1 — add Wave next to Express

`gateway.yaml`:

```yaml
default:
  port: 8080

env:
  EXPRESS_URL: { description: "Express service URL", default: "http://127.0.0.1:3000" }

routes:
  # Catch-all forwarder — everything still goes to Express by default.
  # We'll carve routes off this one at a time.
  - path: /
    type: forward
    forward:
      forward_url: "${env:EXPRESS_URL}"
```

Bind Express to localhost only:

```js
app.listen(3000, '127.0.0.1')
```

Point your load balancer at Wave (`:8080`). Nothing should appear
to change for clients. **Deploy. Validate.**

### Step 2 — migrate `/webhooks/stripe` (low risk, big win)

In Express, the Stripe handler does: parse raw body, verify
signature, parse event, dispatch by event type, log, respond. ~50
lines and 2 dependencies.

In Wave, it's:

```yaml
env:
  STRIPE_WEBHOOK_SECRET: { description: "whsec_…" }

# Add to routes (BEFORE the catch-all forward)
- path: /webhooks/stripe
  methods: [POST]
  type: stream-publish
  expected_content_type: application/json
  webhook_sig:
    provider: stripe
    secret: "${env:STRIPE_WEBHOOK_SECRET}"
    tolerance_sec: 300
  stream-publish:
    connection: events
    event_type_from: type
    store:
      source: app
      execute: |
        INSERT INTO stripe_events(kind, payload)
        VALUES ({{type}}, {{body_raw}})
```

(Plus the storage block and SSE connection — ~15 more lines.)

Delete `routes/stripe.js` from the Express app. **Deploy. Validate
with Stripe CLI.** Net code change: -50 LOC Express, +20 LOC Wave.

### Step 3 — migrate `/auth/*`

Magic-link or OAuth depending on what you had. Either replaces
~140 lines of Express + a passport.js dependency + ~3 helper files.

```yaml
auth:
  app:
    type: jwt
    secret: "${env:JWT_SECRET}"
    cookie_name: session

- path: /auth/signup
  methods: [POST]
  type: magic-link-request
  inputs: [{ name: email, source: body, type: email, required: true }]
  magic-link-request:
    for: app
    email_field: email
    callback_path: /auth/callback
    email_template: "Sign in: {{.link}}"

- path: /auth/callback
  methods: [GET]
  type: magic-link-consume
  magic-link-consume:
    for: app
    redirect_on_success: /app
```

Delete `routes/signup.js`, `routes/login.js`, passport
middleware. Update Express to read user id from `X-User-Id`
header that Wave sets when forwarding. **Deploy. Validate.**

### Step 4 — migrate `/api/todos`

Standard CRUD. ~180 lines of Express + Prisma → ~25 lines of Wave
YAML (see the [JSON API recipe](/cookbook/json-api)). Delete the
Prisma model for Todo too if it's not used elsewhere.

### Step 5 — keep complex billing in Express

`/api/billing` does real domain work — proration calculations,
multi-step state machines, third-party reconciliation. That stays
in Express. Wave forwards it:

```yaml
- path: /api/billing/
  methods: [GET, POST, PUT, DELETE]
  type: forward
  auth: [app]                          # Wave still gates with the new auth
  forward:
    forward_url: "${env:EXPRESS_URL}/api/billing/"
    include_headers:
      - ["X-User-Id", "{{getUser}}"]
```

Express keeps its billing routes; everything else has moved to Wave.

### Final shape

```
Wave (server.yaml)              Express (app.js)
  /webhooks/stripe                /api/billing/*
  /auth/*
  /api/todos
  /api/billing → forwards to Express's /api/billing
```

About 850 of the original 1000 lines are now ~80 lines of Wave
YAML. The Express service is down to ~600 lines of genuinely
complex business logic.

## Patterns that translate well

| Express | Wave |
|---|---|
| `app.use(express.json())` + Joi/Zod | `inputs:` with `type:` and validators |
| `app.use(cors({ origin: ... }))` | `cors_origins:` per route |
| `app.use(rateLimit({ ... }))` | `limits:` registry + `limits: [name]` |
| `app.use(helmet())` | Wave's built-in secure headers (always on) |
| Passport.js / NextAuth | `auth:` block + `type: magic-link-* / oauth-* / auth-login` |
| `req.user.id` | `{{getUser}}` |
| `csurf` middleware | `validate_csrf: true` |
| Stripe signature verification | `webhook_sig: { provider: stripe }` |
| `cron` library + cron job | `schedule:` block |
| `morgan` request logs | Built-in JSON request logging |
| `prom-client` exposing /metrics | Built-in `/metrics` |
| `node-cron` cleanup task | `schedule: { at: "03:30" }` |

## Patterns that don't translate

Keep these in Node (forward them):

- WebSockets that aren't just SSE (Wave does SSE first-class)
- gRPC servers
- Long-lived stateful in-process state machines
- Heavy compute (image processing, ML inference) — these belong
  in a [Python sidekick](/cookbook/python-sidekick) or a Go plugin

## Why incremental beats rewrite

- **Each step is reversible**: if a migrated route has issues,
  flip the load balancer back to Express in seconds.
- **No coordinated cutover**: deploy each migrated route on its
  own schedule.
- **Frontend doesn't change**: same URLs, same response shapes.
- **You can stop anywhere**: maybe only auth + webhooks make
  sense to move; the rest stays in Express forever.

## See also

- [Wave in your stack](/guide/wave-in-your-stack) — the four
  integration patterns
- [Wave in front of Node](/cookbook/node-gateway) — the gateway
  pattern in detail
- [Token efficiency](/ai/token-efficiency) — why this migration
  produces dramatically less generated code when AI-assisted
- [Comparison](/guide/comparison) — Wave vs Express trade-offs
