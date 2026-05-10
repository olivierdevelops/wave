# Wave Example Apps

**57 runnable example applications** showcasing what Wave can do. Each app is a directory with a `server.yaml`, a README, and any assets it needs. Boot any of them with:

```sh
wave serve examples/apps/<name>/server.yaml --port 8080
```

Apps without external dependencies boot cleanly out-of-the-box. Apps that need external services (Postgres, Vault, an OAuth provider, etc.) document the setup in their own README's **Caveats** section.

---

## Auth & Identity (8)

| App | What it shows |
|---|---|
| [`magic-link-login`](magic-link-login/) | Passwordless email magic-link flow |
| [`password-jwt`](password-jwt/) | Username/password → JWT in cookie, with seeded demo users |
| [`oauth-google`](oauth-google/) | Sign in with Google (`oauth-start` + `oauth-callback`) |
| [`oauth-github`](oauth-github/) | Sign in with GitHub, including profile claims |
| [`oauth-apple`](oauth-apple/) | Sign in with Apple — `.p8` ES256 client-secret JWT |
| [`oidc-okta`](oidc-okta/) | OIDC against any provider (Okta/Auth0/Entra/Google) |
| [`saml-sso`](saml-sso/) | Enterprise SAML SSO via the `saml-auth` plugin |
| [`totp-2fa`](totp-2fa/) | Magic-link login + TOTP enrollment + verification |

## Storage & Data (8)

| App | What it shows |
|---|---|
| [`notes-app`](notes-app/) | SQLite CRUD with a vanilla-JS frontend |
| [`url-shortener`](url-shortener/) | POST `/shorten` → slug; GET `/r/{slug}` → redirect |
| [`bookmarks`](bookmarks/) | Per-user data scoped by JWT claims |
| [`postgres-crud`](postgres-crud/) | CRUD over Postgres via the `postgres-storage` plugin |
| [`kv-store`](kv-store/) | Generic K/V HTTP API with binary bodies |
| [`file-uploads`](file-uploads/) | Multipart uploads → disk + SQLite metadata |
| [`tagged-search`](tagged-search/) | JSON-array tags with `json_each` filtering |
| [`soft-delete-pattern`](soft-delete-pattern/) | `deleted_at` tombstone with `/items/trash` admin view |

## Webhooks & Streaming (7)

| App | What it shows |
|---|---|
| [`stripe-webhook-receiver`](stripe-webhook-receiver/) | Stripe HMAC verify → SSE fan-out |
| [`github-webhook`](github-webhook/) | GitHub HMAC verify, event-type routing |
| [`sse-chat`](sse-chat/) | Vanilla-JS chat over `EventSource` |
| [`websocket-echo`](websocket-echo/) | `type: ws` connection with browser demo |
| [`webhook-signature-verify`](webhook-signature-verify/) | Generic HMAC: sha256, sha1, header-prefixed |
| [`event-fanout-hub`](event-fanout-hub/) | One ingest → 3 named broker connections |
| [`live-cursors`](live-cursors/) | Multi-cursor demo on a shared canvas via SSE |

## Reverse Proxy & Gateway (6)

| App | What it shows |
|---|---|
| [`basic-forward`](basic-forward/) | Simplest reverse proxy at `/api/*` |
| [`api-gateway-rate-limited`](api-gateway-rate-limited/) | Named `limits:` registry composed per route |
| [`cors-proxy`](cors-proxy/) | Browser-friendly CORS + OPTIONS preflight |
| [`jwt-validating-gateway`](jwt-validating-gateway/) | Validate inbound JWT before forwarding upstream |
| [`multi-tenant-routing`](multi-tenant-routing/) | `/{tenant}/*` → per-tenant upstream URL |
| [`static-and-api`](static-and-api/) | One server: SPA frontend + `/api/*` proxy backend |

## Static & Content (5)

| App | What it shows |
|---|---|
| [`blog-markdown`](blog-markdown/) | Markdown posts in `posts/`, Prism syntax highlighting |
| [`docs-site`](docs-site/) | Sidebar nav + markdown content area |
| [`spa-with-api`](spa-with-api/) | Vanilla-JS SPA bundled at startup + `/api/*` backend |
| [`photo-gallery`](photo-gallery/) | Manifest-driven thumbnail grid + lightbox |
| [`landing-page-with-form`](landing-page-with-form/) | Marketing page + `/contact` form → SQLite |

## Plugin Showcases (5)

| App | Plugin used |
|---|---|
| [`postgres-plugin-crud`](postgres-plugin-crud/) | `postgres-storage` |
| [`vault-secrets-fed`](vault-secrets-fed/) | `vault-secrets` (`${PLUGIN:vault:...}` markers) |
| [`saml-enterprise-sso`](saml-enterprise-sso/) | `saml-auth` |
| [`otel-tracing-demo`](otel-tracing-demo/) | `otel-exporter` (OTLP gRPC) |
| [`multi-plugin-stack`](multi-plugin-stack/) | All four plugins in one config — flagship demo |

## Real Apps (8)

| App | What it is |
|---|---|
| [`todo-app`](todo-app/) | Magic-link auth + per-user TODOs |
| [`pastebin`](pastebin/) | Anonymous code paste with syntax highlighting |
| [`polls`](polls/) | Create + vote, live counts via SSE |
| [`microblog`](microblog/) | 280-char posts, public timeline, per-user `/u/{handle}` |
| [`chat-room`](chat-room/) | Anonymous global chat with last-100 history |
| [`status-page`](status-page/) | Public status dashboard with admin updates |
| [`pomodoro`](pomodoro/) | Multi-user shared pomodoro timer per `?room=` |
| [`analytics-collector`](analytics-collector/) | Privacy-respecting page-view collector + dashboard |

## Patterns & Integrations (10)

| App | Pattern |
|---|---|
| [`magic-link-plus-totp`](magic-link-plus-totp/) | Magic-link establishes session, TOTP upgrades to "trusted" claim |
| [`multi-tenant-saas`](multi-tenant-saas/) | Subdomain → tenant scoping in storage queries |
| [`audit-logged-admin`](audit-logged-admin/) | All `/admin/*` mutations land in the audit log |
| [`rate-limited-public-api`](rate-limited-public-api/) | Per-user buckets via `key_claim`, anonymous fallback |
| [`streaming-file-processor`](streaming-file-processor/) | CSV upload → process line-by-line → SSE progress |
| [`slack-slash-command`](slack-slash-command/) | Slack signing-secret verify + slash-command response |
| [`discord-bot-webhook`](discord-bot-webhook/) | Outbox-backed reliable outbound webhook |
| [`email-to-api-ingest`](email-to-api-ingest/) | Inbound email parsing (Mailgun/SendGrid shape) |
| [`cron-data-refresh`](cron-data-refresh/) | `schedule:` block + cached fetch |
| [`outbox-reliability-demo`](outbox-reliability-demo/) | Atomic write + outbox drain → downstream |

---

## Boot status at a glance

**Boots out-of-the-box (no env, no external services):**
`magic-link-login`, `password-jwt`, `totp-2fa`, all 8 storage apps except `postgres-crud`, all 7 webhook/streaming apps, all 6 proxy apps, all 5 static apps, `saml-enterprise-sso`, `otel-tracing-demo`, all 8 real apps, `multi-tenant-saas`, `streaming-file-processor`, `discord-bot-webhook`, `email-to-api-ingest`, `cron-data-refresh`, `outbox-reliability-demo`.

**Need a `SECRET_KEY` env var** (Wave-wide JWT secret, set anything for dev):
Any app with an `auth:` block that uses cookies/JWT — e.g. `magic-link-plus-totp`, `audit-logged-admin`, `rate-limited-public-api`. Set `SECRET_KEY=dev` and they boot.

**Need third-party credentials to boot:**
`oauth-google`, `oauth-github`, `oauth-apple`, `oidc-okta` (issuer-discovery happens at boot), `saml-sso` (real IdP metadata).

**Need a built reference plugin binary:**
`postgres-crud`, `postgres-plugin-crud`, `vault-secrets-fed`, `saml-enterprise-sso`, `otel-tracing-demo`, `multi-plugin-stack`. Build them with:
```sh
cd examples/plugins/postgres-storage && go build -o /tmp/wave-postgres .
cd examples/plugins/vault-secrets    && go build -o /tmp/wave-vault    .
cd examples/plugins/saml-auth        && go build -o /tmp/wave-saml     .
cd examples/plugins/otel-exporter    && go build -o /tmp/wave-otel     .
```

**Need running external services to be useful (they boot but routes fail at request time):**
`stripe-webhook-receiver` (needs Stripe webhook secret to verify), `slack-slash-command` (signing secret), `vault-secrets-fed` (Vault dev server), `multi-plugin-stack` (the lot).

---

## How to read these as a tour

If you're new to Wave, walk in this order — each step adds one concept:

1. [`basic-forward`](basic-forward/) — minimum viable Wave config
2. [`notes-app`](notes-app/) — storage + JSON CRUD
3. [`magic-link-login`](magic-link-login/) — auth without passwords
4. [`api-gateway-rate-limited`](api-gateway-rate-limited/) — middleware composition
5. [`sse-chat`](sse-chat/) — real-time
6. [`stripe-webhook-receiver`](stripe-webhook-receiver/) — webhook verification + fan-out
7. [`todo-app`](todo-app/) — full-stack with auth + storage + frontend
8. [`postgres-plugin-crud`](postgres-plugin-crud/) — first plugin
9. [`multi-plugin-stack`](multi-plugin-stack/) — production-shaped setup with all 4 plugins

That's 9 apps in ~20 minutes and you've seen every major Wave feature.
