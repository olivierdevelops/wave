<p align="center">
  <h1 align="center">Wave</h1>
  <p align="center">
    <strong>A declarative HTTP server framework — describe your backend in capy, ship a single binary.</strong>
  </p>
  <p align="center">
    <a href="https://github.com/olivierdevelops/wave/actions"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/olivierdevelops/wave/ci?label=ci"></a>
    <a href="https://github.com/olivierdevelops/wave/releases"><img alt="Release" src="https://img.shields.io/github/v/release/olivierdevelops/wave?display_name=tag&sort=semver"></a>
    <a href="https://github.com/olivierdevelops/wave/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/badge/license-Apache--2.0-blue.svg"></a>
    <a href="https://go.dev/"><img alt="Go" src="https://img.shields.io/badge/go-1.24+-00ADD8.svg"></a>
  </p>
  <p align="center">
    <a href="https://olivierdevelops.github.io/wave/"><strong>Docs</strong></a> •
    <a href="https://olivierdevelops.github.io/wave/guide/quickstart"><strong>Quickstart</strong></a> •
    <a href="https://olivierdevelops.github.io/wave/cookbook/"><strong>Cookbook</strong></a> •
    <a href="https://olivierdevelops.github.io/wave/ai/token-efficiency"><strong>For AI agents</strong></a>
  </p>
</p>

---

## A working JSON API in 18 lines of capy

```capy
storage app
    kind     sqlite
    location "./data.db"

route "/users"
    methods POST
    request
        content_type "application/json"
        body_field name
            type       text
            required   true
            min_length 1
    do
        created = on app do sql `INSERT INTO users (name) VALUES ({{request.name}})`
        match created
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}}}`
```

```sh
wave serve server.capy --listen :8080
curl -X POST -d '{"name":"ada"}' http://localhost:8080/users
# {"id": 1}
```

That's a working endpoint with **input validation**, **parameterised SQL**
(the `{{request.name}}` becomes a bound `?` — injection is impossible by
construction), a **JSON response**, and a built-in **`/healthz`** probe. No
Go code, no `node_modules`, no Docker Compose stack. Read it top to bottom:
*what it takes → what it does → what it returns.* The same `server.capy`
deploys as a single binary or a 25 MB distroless container.

## What ships in the box

**[→ Full feature inventory](https://olivierdevelops.github.io/wave/reference/features)** — every declaration, route shape, operation, response form, CLI subcommand, plugin kind, in one searchable page.

| | What |
|---|---|
| **One DSL** | `route` + `request` + `do`; operations `on TARGET do sql\|call_plugin\|broadcast`; `match` outcomes; `response`; `every` schedules; `background` tasks |
| **Demo apps** | Self-contained `server.capy` files under `examples/apps/` — chat, polls, todo, pastebin, multi-tenant SaaS, Stripe receiver, SSE chat, photo gallery, OAuth, magic-link, audit-logged admin, ML sidekick, … |
| **Cookbook recipes** | 30+ copy-paste patterns for the common needs |
| **CLI commands** | `serve`, `check`, `describe`, `docs`, `export`, `fmt`, `lsp`, `new`, `migrate`, `secrets`, `routes`, `test`, `version` |
| **Auth schemes** | session cookie, magic link (email), OAuth (Google/GitHub/Apple/OIDC), API-key header lookup, JWT |
| **Webhook handling** | Stripe, GitHub, Slack, generic HMAC — all with signature verification + replay protection |
| **Observability** | Prometheus `/metrics`, OpenTelemetry traces, JSON access logs, append-only audit log |
| **Reliability** | durable outbox, circuit breaker, rate limiter, body-size limits, response cache |
| **Tooling** | `wave describe` (project manifest → JSON/MD/OpenAPI), `wave export` (typed clients, mock server), `wave lsp` (editor language server) |
| **Deploy targets** | macOS / Linux / Windows binaries + distroless Docker image |

## Why Wave?

### 🚀 Ship faster

Most backends are 80% boilerplate — request parsing, validation, DB calls,
auth wiring, middleware ordering. Wave does that 80% declaratively. You write
Go (or any language, as a plugin) only where it actually matters.

### 🤖 5-10× fewer tokens for AI-assisted development

The same JSON API endpoint:

| Stack | Lines | Tokens |
|---|---:|---:|
| **Wave** | **18** | **~160** |
| FastAPI + Pydantic | 24 | ~360 |
| Gin (Go) | 38 | ~440 |
| Express + Zod + Prisma | 38 | ~520 |

More features per Cursor request, more state per Claude context window, fewer
hallucinations. Wave ships [`llms.txt`](llms.txt), a [Claude Code
skill](.claude/skills/wave.md), and a **self-describing grammar**
(`wave docs --format json`) so AI editors auto-complete and produce working
configs first try. See
[the full comparison](https://olivierdevelops.github.io/wave/ai/token-efficiency).

### 🔒 Safe by construction — not as an afterthought

- **SQL injection: impossible.** Inside `sql`, every `{{request.x}}` compiles
  to a bound `?` parameter — values never reach the SQL text. It's a property
  of the compiler, not a rule you must remember.
- **XSS: closed at the template boundary.** Response bodies auto-escape for
  their declared `content_type` — HTML responses HTML-escape, JSON responses
  JSON-escape.
- **CSRF**, **webhook signatures** (Stripe / GitHub / Slack), **rate limits**,
  **circuit breakers**, **body-size limits**, **input validation**, **secure
  headers** — all wired into the request pipeline.
- **RBAC** via `auth.user.roles`, **audit log** for every mutation.

### 📦 Real production primitives

- **`/healthz` + `/readyz`** built in
- **Prometheus** `/metrics` + **OpenTelemetry** traces
- **Durable outbox** for webhook delivery with retry + DLQ
- **Migrations** (`wave migrate server.capy up`)
- **Config check** (`wave check server.capy --format json`) for pre-flight
- **Functional test runner** (`wave test`) — capy-driven, in-process, no port

### 🧪 Testable end-to-end

```capy
# server.test.capy
import "server.capy"

test "create user"
    request POST "/users"
        json { "name": "ada" }
    expect
        status 201
        json   { "id": "*" }
    capture id from json.id

test "read it back"
    request GET "/users/{{id}}"
    expect
        status 200
        json   { "name": "ada" }
```

```sh
wave test server.test.capy --format json    # CI-friendly, in-process, no port binding
```

[Full testing recipe →](https://olivierdevelops.github.io/wave/cookbook/testing)

### 🧩 Fits your existing stack

Wave isn't a replacement for React, Node, or Python. It's a complement:

- [**React / Next.js + Wave backend**](https://olivierdevelops.github.io/wave/cookbook/react-wave) — auth + persistence in capy, your frontend stays on Vercel
- [**Wave in front of an Express service**](https://olivierdevelops.github.io/wave/cookbook/node-gateway) — gateway pattern for auth, rate-limit, audit, webhook signatures
- [**Wave + Python ML service**](https://olivierdevelops.github.io/wave/cookbook/python-sidekick) — wrap a FastAPI/model server with auth + SSE + background tasks
- [**Migrate from Express incrementally**](https://olivierdevelops.github.io/wave/cookbook/migrate-from-express) — route-at-a-time, no big-bang

### 🛠️ Generate clients & docs from the source

```sh
wave describe server.capy               # API reference for your app (Markdown / JSON / OpenAPI)
wave export   server.capy --client ts   # a typed TypeScript client for your frontend
wave serve    server.capy --mock        # a mock server for frontend dev (no side effects)
```

The client a consumer imports is, by construction, the contract the server
enforces — both come from the same parse.

## Popular integrations — copy-paste recipes

The most-asked "how do I plug Wave into X?" recipes:

- [**Sign in with Google**](https://olivierdevelops.github.io/wave/cookbook/google-signin) — OAuth, domain-gated
- [**Stripe Checkout**](https://olivierdevelops.github.io/wave/cookbook/stripe-checkout) (and [webhooks](https://olivierdevelops.github.io/wave/cookbook/stripe-webhooks))
- [**OpenAI / Claude / Ollama chat**](https://olivierdevelops.github.io/wave/cookbook/openai-claude) — streaming tokens via background task + plugin
- [**Transactional email**](https://olivierdevelops.github.io/wave/cookbook/send-email) — Resend / SendGrid / Postmark / Mailgun
- [**SMS via Twilio**](https://olivierdevelops.github.io/wave/cookbook/twilio-sms) — verify codes, 2FA, alerts
- [**Slack slash command**](https://olivierdevelops.github.io/wave/cookbook/slack-slash-command) — signature-verified, sub-3-second response
- [**S3 / R2 / B2 uploads**](https://olivierdevelops.github.io/wave/cookbook/s3-r2-uploads) — pre-signed PUT, bypass-Wave for bytes
- [**Firebase Cloud Messaging**](https://olivierdevelops.github.io/wave/cookbook/firebase-fcm) — iOS / Android / web push
- [**Supabase / Neon / Railway Postgres**](https://olivierdevelops.github.io/wave/cookbook/supabase-postgres) — managed Postgres via `storage / kind postgres`

Same `on PLUGIN do call_plugin` / signature-verification / `storage`
primitives — once you've done one, the rest are copy-paste.

Don't see what you need? [**Build a plugin (any language)**](https://olivierdevelops.github.io/wave/cookbook/build-plugin) — same echo plugin in Go, Python, Node, Rust, and 9-line Bash.

## Pick your path

| You are… | Start here |
|---|---|
| **Trying it for the first time** | [Quickstart (5 min)](https://olivierdevelops.github.io/wave/guide/quickstart) |
| **Building a real app** | [Tutorial: build a todo API (30 min)](https://olivierdevelops.github.io/wave/guide/tutorial) |
| **An indie hacker** | [`wave new api`](https://olivierdevelops.github.io/wave/guide/quickstart) — scaffolded project with auth + Docker + Fly.io ready |
| **A backend engineer** | [Comparison vs Express / FastAPI / Gin](https://olivierdevelops.github.io/wave/guide/comparison) |
| **A platform / SRE engineer** | [Production checklist](https://olivierdevelops.github.io/wave/guide/deploy-checklist) + [Observability](https://olivierdevelops.github.io/wave/guide/concepts-observability) |
| **An AI agent builder** | [Token efficiency](https://olivierdevelops.github.io/wave/ai/token-efficiency) + [Claude skill](https://olivierdevelops.github.io/wave/ai/claude-code) |
| **Adding Wave to an existing app** | [Wave in your stack](https://olivierdevelops.github.io/wave/guide/wave-in-your-stack) |

## Install

```bash
# Pre-built binaries (macOS / Linux / Windows)
curl -sSfL https://olivierdevelops.github.io/wave/install.sh | sh

# Pin a version
curl -sSfL https://olivierdevelops.github.io/wave/install.sh | sh -s -- v0.1.0

# Or via Go (latest main, includes built-in SQLite)
go install github.com/olivierdevelops/wave/orchestrator@latest

# Or via Docker (sqlite-capable)
docker run --rm -p 8080:8080 \
  -v $(pwd)/server.capy:/server.capy \
  ghcr.io/olivierdevelops/wave:latest serve /server.capy --listen :8080
```

> Released binaries are built `nosqlite` for cross-platform simplicity.
> Use the Docker image or `go install` for built-in SQLite. Homebrew
> formula lands shortly.

## CLI at a glance

```sh
wave serve    server.capy --listen :8080   # run a server
wave check    server.capy                  # parse + validate (no server)
wave test     server.test.capy             # run a capy test suite
wave describe server.capy --format json    # document what this project does
wave docs     --format json                # document the language itself
wave export   server.capy --client ts      # generate a typed client
wave fmt      server.capy --check          # CI-safe formatter
wave routes   server.capy --format json    # print the route table
wave new      api ./my-project             # scaffold a starter
wave migrate  server.capy up               # apply migrations
wave lsp                                   # editor language server (stdio)
```

## The 30-second taste

```sh
git clone https://github.com/olivierdevelops/wave.git
cd wave

# Pick any demo
go run ./orchestrator serve examples/apps/url-shortener/server.capy --listen :8102
curl http://localhost:8102/healthz
# ok

# Or run its test suite
go run ./orchestrator test examples/apps/url-shortener/server.test.capy
#   PASS  built-in /healthz returns ok (200, 1ms)
#   PASS  unknown path returns framework 404 envelope (404, 0s)
#   PASS  POST /shorten validates target pattern (400, 0s)
#   …
#   8 passed, 0 failed, 0.00s
```

## What it's not

- Not a frontend framework (it serves your React/Vue/Svelte build, but
  doesn't render it).
- Not a service mesh (it sits at L7, in your app; pair with Istio/Linkerd
  if you need mTLS).
- Not a workflow engine (it has a scheduler; use Temporal/Airflow if you
  need durable multi-step workflows).
- Not a replacement for your domain code — it's the boring parts done
  declaratively so you can focus on the parts that aren't.

## Documentation

- **[olivierdevelops.github.io/wave](https://olivierdevelops.github.io/wave/)** — docs site
- **[CLAUDE.md](CLAUDE.md)** — full developer guide (architecture, conventions, every capy keyword)
- **[docs/](docs/)** — reference docs (the capy language, templating, tooling, route lifecycle)
- **[examples/apps/](examples/apps/)** — runnable demos
- **[llms.txt](llms.txt)** — LLM-friendly index for AI agents

## Community

- **[GitHub Discussions](https://github.com/olivierdevelops/wave/discussions)** — questions, ideas, show & tell
- **[Issues](https://github.com/olivierdevelops/wave/issues)** — bug reports and feature requests
- Discord — *coming soon*

## Status

**Pre-1.0.** Breaking changes are allowed between 0.x minors and are
documented in [CHANGELOG.md](CHANGELOG.md). Production usage is welcome —
pin a version and read the CHANGELOG before upgrading.

## Privacy

**Wave never phones home.** No telemetry, no analytics, no remote config
fetches. The single binary only contacts services *you* configure.

## Contributing

PRs welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md) for the process
and [CLAUDE.md](CLAUDE.md) for the architecture. Good first issues:
[`good-first-issue`](https://github.com/olivierdevelops/wave/labels/good-first-issue).

## License

Apache-2.0 — see [LICENSE](LICENSE).
