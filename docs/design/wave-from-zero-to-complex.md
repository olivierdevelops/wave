# Wave — from zero to complex

A single end-to-end reference. Read top-to-bottom and you can build anything
Wave supports; jump to a section if you already know the basics. Every example
is a complete, runnable `server.capy` (or fragment of one) — no pseudo-code.

This doc is the practical companion to [`wave-capy-complete.md`](./wave-capy-complete.md)
(the full surface reference) and [`wave-capy-templating.md`](./wave-capy-templating.md)
(SQL & response templating safety). Where they catalogue, this one *builds*.

> **Language is `capy`, not YAML.** Every declaration is line-oriented;
> indentation forms blocks; values bind into SQL as `?` parameters
> automatically. There is no schema file separate from the language.

---

## Table of contents

**Foundations**
1. [Install & run](#1-install--run)
2. [Hello world](#2-hello-world)
3. [Anatomy of a route](#3-anatomy-of-a-route)
4. [Inputs — every shape](#4-inputs--every-shape)
5. [Templates — the two rules](#5-templates--the-two-rules)
6. [Responses](#6-responses)

**Storage**
7. [SQLite, Postgres, MySQL + first-class migrations](#7-sqlite-postgres-mysql)
8. [Reading rows](#8-reading-rows)
9. [Writing rows](#9-writing-rows)
10. [Pipelines (multi-step lookups)](#10-pipelines-multi-step-lookups)
11. [Search, filters, arrays](#11-search-filters-arrays)

**CRUD & failure handling**
12. [CRUD resource end-to-end](#12-crud-resource-end-to-end)
13. [Failure cases & `handle`](#13-failure-cases--handle)
14. [Server-wide defaults](#14-server-wide-defaults)

**Auth & authorization**
15. [Session cookies + password](#15-session-cookies--password)
16. [Magic-link login](#16-magic-link-login)
17. [OAuth (Google, GitHub, generic OIDC)](#17-oauth)
18. [JWT / header auth (API keys)](#18-jwt--header-auth)
19. [RBAC, per-resource ownership, claims](#19-rbac-per-resource-ownership-claims)

**Files**
20. [Upload (multipart)](#20-upload-multipart)
21. [Download (binary, ranges)](#21-download)
22. [Static & file-server](#22-static--file-server)

**Routing patterns**
23. [Redirects, URL shorteners](#23-redirects-url-shorteners)
24. [Forward / proxy / dynamic forward](#24-forward--proxy)
25. [Multi-tenant by host / path / cookie](#25-multi-tenant)
26. [A/B testing & device detection](#26-ab-testing--device-detection)

**Plugins**
27. [Plugin contract recap](#27-plugin-contract-recap)
28. [Subprocess plugin (one-shot)](#28-subprocess-plugin)
29. [Long-lived plugin (streaming)](#29-long-lived-plugin)
30. [HTTP plugin (call any external service)](#30-http-plugin)
31. [Storage & auth & secrets plugins](#31-storage--auth--secrets-plugins)

**Real-time**
32. [SSE — server-sent events](#32-sse)
33. [WebSocket](#33-websocket)
34. [Partitioned channels (per-user, per-tenant)](#34-partitioned-channels)

**Async & scheduled work**
35. [Background tasks](#35-background-tasks)
36. [Scheduled jobs (`every` / `at`)](#36-scheduled-jobs)
37. [Outbox pattern (durable webhooks)](#37-outbox-pattern)

**Integrations (worked examples)**
38. [Stripe Checkout + webhooks](#38-stripe)
39. [Send email (Resend / SendGrid / Postmark)](#39-send-email)
40. [Slack slash command](#40-slack-slash-command)
41. [LLM chat (OpenAI / Claude / Ollama)](#41-llm-chat)
42. [S3 / R2 image uploads](#42-s3-r2-image-uploads)

**Operations**
43. [Rate limiting, timeouts, body limits](#43-rate-limiting-timeouts-body-limits)
44. [Secrets & configuration](#44-secrets--configuration)
45. [Observability — metrics, traces, logs](#45-observability)
46. [Testing with `wave test`](#46-testing-with-wave-test)
47. [Deployment — Docker, Fly, Railway](#47-deployment)

**Reference appendix**
48. [Operation table](#48-operation-table)
49. [Match-case table](#49-match-case-table)
50. [Type & modifier table](#50-type--modifier-table)

---

# Foundations

## 1. Install & run

```bash
# Install (Go required for source build; binaries shipped per-release)
go install github.com/luowensheng/wave/cmd/wave@latest

# Or download a release binary
curl -fsSL https://wave.dev/install.sh | sh

# Create a new project
wave new my-app
cd my-app

# Run a server file with hot reload
wave serve server.capy --watch

# Validate without running (CI gate)
wave check server.capy

# Run the test suite
wave test server.test.capy
```

The single binary handles serve / check / test / fmt / describe / new / docs.
No external runtime, no Node, no Python — Wave itself is one executable.

---

## 2. Hello world

The smallest useful `server.capy`:

```capy
route "/"
    methods GET
    request
    do
        response
            status       200
            content_type "text/plain; charset=utf-8"
            body         `hello, wave`
```

Run with `wave serve server.capy`. Hit `http://localhost:8080/`. Done.

**What's already happening for you**: routing, method matching, 404 for
unknown paths, 405 + `Allow:` header for wrong methods, request IDs in
response headers, structured logs, panic recovery, graceful shutdown on
SIGTERM.

---

## 3. Anatomy of a route

```capy
route "/path/with/{param}"               # path; {param} are path parameters
    methods POST                          # one or more HTTP methods
    requires_authentication user          # optional — opt into an auth scheme
    rate_limit                            # optional — N requests per window
        per "1m"
        max 60
    timeout 5s                            # optional — hard wall-clock cap

    request                               # what the route accepts
        content_type "application/json"   # required when reading a body
        path_parameter param
            type     text
            required true
        body_field title
            type       text
            required   true
            min_length 1
            max_length 200

    do                                    # what the route does
        result = on main do sql `INSERT INTO items (title) VALUES ({{request.title}})`
        match result
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}}}`

    handle                                # optional — failure overrides
        case invalid_input(field, reason)
            response
                status       422
                content_type "application/json"
                body         `{"error":"invalid","field":"{{field}}","reason":"{{reason}}"}`
```

Five sections — header, `request`, `do`, optional `handle`. Read it
top-to-bottom: *what it takes → what it does → what it returns*.

---

## 4. Inputs — every shape

Every value the route reads is declared. Anything not declared can't be
referenced, in SQL or templates.

```capy
request
    content_type "application/json"        # required iff a body input follows

    # URL path: route is "/users/{id}/posts/{slug}"
    path_parameter id
        type     integer
        required true
    path_parameter slug
        type     text
        required true

    # URL query: ?q=…&limit=…
    query_parameter q
        type       text
        max_length 100
        pattern    "^[A-Za-z0-9 ]*$"
    query_parameter limit
        type    integer
        minimum 1
        maximum 100

    # Headers — referenced snake_case in templates (`request.x_api_key`)
    header X-Api-Key
        type     text
        required true

    # Cookies
    cookie session
        type text

    # JSON / form / multipart body field
    body_field title
        type       text
        required   true
        min_length 1
        max_length 200

    # Entire body as one JSON value
    body items
        type     json_array
        required true

    # Multipart file
    body_file avatar
        required  true
        max_bytes 1048576    # 1 MiB

    # Raw bytes (signed webhooks)
    body_raw payload
        type     bytes
        required true
```

**Types**: `text`, `integer`, `decimal`, `true_or_false`, `email`, `uuid`,
`file`, `bytes`, `json_array`, `json_object`.

**Modifiers**: `required`, `minimum` / `maximum` (numeric),
`min_length` / `max_length` (text), `pattern` (regex), `max_bytes`
(file / raw).

A request that fails any validator → `400` + `invalid_input(field, reason)`
case (or `415` for content-type mismatch, `413` for body-too-large).
Override in `handle`.

---

## 5. Templates — the two rules

Templates are inside backticks. Two contexts: SQL strings and response
bodies / headers. **The rules are different and the safety guarantee is
strong in both.**

### Rule 1 — In SQL, every value binds

```capy
on main do sql `SELECT * FROM users WHERE id = {{request.id}} AND active = {{request.active}}`
```

`{{request.id}}` and `{{request.active}}` compile to `?` placeholders; the
values are appended to the parameter list. There is no syntax to splice a
value into SQL text. **SQL injection is structurally impossible.**

The single exception: identifiers (table/column names) can't parameterize.
For dynamic identifiers, use `{{identifier x}}` — it validates against
`^[A-Za-z_][A-Za-z0-9_]*$` and rejects on violation. Use sparingly; prefer
configuration over request data.

### Rule 2 — In responses, output auto-escapes per `content_type`

```capy
response
    status       200
    content_type "text/html; charset=utf-8"
    body         `<p>Hello {{request.name}}</p>`     # HTML-escaped automatically
```

If `request.name` is `<script>alert(1)</script>`, it renders as
`&lt;script&gt;…`. **XSS is closed by default.** Same for JSON: declared
content type `application/json` JSON-escapes every interpolation.

Opt out with `{{raw x}}` (rare; you take responsibility).

### Helpers you'll use a lot

| In SQL | What it does |
|---|---|
| `{{now}}` | UTC timestamp (binds) |
| `{{add_days N}}` / `{{add_hours N}}` / `{{add_minutes N}}` | offset timestamp (binds) |
| `{{like_contains x}}` / `{{like_prefix x}}` / `{{like_suffix x}}` | LIKE pattern (binds) |
| `{{if has_value "name"}}…{{end}}` | include clause only when input present |
| `{{each "tags" as t, i}}…{{end}}` | iterate a `json_array` input |
| `{{secret "key"}}` | configured secret (binds) |
| `{{new_token}}` / `{{new_uuid}}` | random token / UUID (bind) |
| `{{hash_password x}}` | argon2id hash of x (binds) |
| `{{client_ip}}` | request IP (binds) |
| `{{identifier x}}` | validated identifier (NOT bound) |

| In responses | What it does |
|---|---|
| `{{to_json x}}` | JSON-encode a row, list, or map |
| `{{raw x}}` | bypass auto-escape (opt-in) |
| `{{join "," xs}}` | join a list |
| `{{format_time "layout" t}}` | format a timestamp |

Full list: [`wave-capy-templating.md`](./wave-capy-templating.md).

---

## 6. Responses

Every `response` declares status, content type, body.

```capy
# JSON
response
    status       200
    content_type "application/json"
    body         `{{to_json data}}`

# HTML
response
    status       200
    content_type "text/html; charset=utf-8"
    body         `<!doctype html><html><body><h1>{{title}}</h1></body></html>`

# Plain text
response
    status       200
    content_type "text/plain; charset=utf-8"
    body         `ok`

# Binary download (e.g. a file from storage)
response
    status                         200
    body_bytes                     file.blob
    content_type_from_extension_of file.name
    set_header Content-Disposition
        value `attachment; filename="{{file.name}}"`

# Redirect
response
    status       302
    content_type "text/plain; charset=utf-8"
    body         ``
    set_header Location
        value "/dashboard"

# Set / clear cookies
response
    status       200
    content_type "application/json"
    body         `{"ok":true}`
    set_cookie session
        value     session_token
        lifetime  24h
        http_only true
        secure    true
        same_site lax
    clear_cookie csrf

# Custom headers
response
    status       200
    content_type "application/json"
    body         `{"ok":true}`
    set_header X-Total-Count
        value count
    set_header Cache-Control
        value "public, max-age=60"
```

---

# Storage

## 7. SQLite, Postgres, MySQL

Declare one or more storages at the top of the file.

```capy
# SQLite (file-backed)
storage main
    kind     sqlite
    location "./app.db"

# In-memory SQLite (testing)
storage scratch
    kind     sqlite
    location ":memory:"

# Postgres
storage analytics
    kind     postgres
    location "postgres://wave:secret@db.internal:5432/analytics?sslmode=require"
    pool_max 20

# MySQL
storage legacy
    kind     mysql
    location "wave:secret@tcp(db.internal:3306)/legacy"
    pool_max 10
```

Multiple storages can coexist; each route picks one with `on <name> do sql`.

### Schema migrations

Declare migrations in the same `server.capy`. Wave applies pending
ones at boot, in declaration order, tracked per storage in a
wave-owned `__wave_migrations` ledger.

```capy
migration "001_initial" on main
    description "users table + email index"
    up `
        CREATE TABLE users (
            id            INTEGER PRIMARY KEY AUTOINCREMENT,
            email         TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at    TEXT NOT NULL
        );
        CREATE INDEX users_email ON users(email);
    `
    down `
        DROP INDEX IF EXISTS users_email;
        DROP TABLE  IF EXISTS users;
    `

migration "002_add_roles" on main
    up   `ALTER TABLE users ADD COLUMN roles TEXT NOT NULL DEFAULT '[]'`
    down `ALTER TABLE users DROP COLUMN roles`
```

Boot flow:

```
wave serve server.capy
  → open storage `main`
  → ensure __wave_migrations exists
  → applied = SELECT id FROM __wave_migrations
  → pending = declared - applied (in declaration order)
  → for each pending: BEGIN; up; INSERT ledger row; COMMIT
  → fail-fast on any error
  → start HTTP listener
```

Edited a committed migration? Boot fails with
`migration_checksum_mismatch` — revert or add a follow-up migration,
don't silently diverge.

Per-storage ledgers, environment tags (`tag dev, staging`), `requires`
graph, `transactional false` for `CREATE INDEX CONCURRENTLY`, and the
`wave migrate status / up / down / new / verify` CLI are covered in the
[full migrations reference](./wave-capy-migrations.md).

Common knobs:

| Need | How |
|---|---|
| Skip in prod | `tag dev, staging` on the migration |
| Concurrent index build (Postgres) | `transactional false` + `CREATE INDEX CONCURRENTLY` |
| Cross-file dependencies | `requires "001_initial"` |
| One-way data fix | omit `down` (also: keep it idempotent) |
| Long backfill (millions of rows) | use a scheduled job, not a migration |

---

## 8. Reading rows

### Single row (`LIMIT 1`)

```capy
route "/users/{id}"
    methods GET
    request
        path_parameter id
            type     integer
            required true
    do
        lookup = on main do sql `SELECT id, name, email FROM users WHERE id = {{request.id}} LIMIT 1`
        match lookup
            case empty
                response
                    status       404
                    content_type "application/json"
                    body         `{"error":"not found"}`
            case found(user)
                response
                    status       200
                    content_type "application/json"
                    body         `{"id":{{user.id}},"name":"{{user.name}}","email":"{{user.email}}"}`
```

With `LIMIT 1`, `case found(user)` binds a single map — access fields as
`user.id`.

### Multiple rows

```capy
route "/users"
    methods GET
    request
        query_parameter limit
            type    integer
            minimum 1
            maximum 200
    do
        result = on main do sql `
            SELECT id, name FROM users
             ORDER BY id
             LIMIT {{request.limit}}
        `
        match result
            case success(rows)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json rows}}`
            case error(err)
                response
                    status       500
                    content_type "application/json"
                    body         `{"error":"db","detail":"{{err}}"}`
```

Without `LIMIT 1`, `rows` is a list — use `{{to_json rows}}` or iterate.

---

## 9. Writing rows

### Insert

```capy
route "/items"
    methods POST
    request
        content_type "application/json"
        body_field name
            type     text
            required true
    do
        result = on main do sql `INSERT INTO items (name, created_at) VALUES ({{request.name}}, {{now}})`
        match result
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}}}`
            case error(err)
                response
                    status       500
                    content_type "application/json"
                    body         `{"error":"insert failed"}`
```

### Update

```capy
route "/items/{id}"
    methods PATCH
    request
        path_parameter id
            type     integer
            required true
        content_type "application/json"
        body_field name
            type     text
            required true
    do
        result = on main do sql `UPDATE items SET name = {{request.name}} WHERE id = {{request.id}}`
        match result
            case success(info)
                match info.rows_affected
                    case 0
                        response
                            status       404
                            content_type "application/json"
                            body         `{"error":"not found"}`
                    case _
                        response
                            status       200
                            content_type "application/json"
                            body         `{"updated":{{info.rows_affected}}}`
```

### Delete

```capy
route "/items/{id}"
    methods DELETE
    request
        path_parameter id
            type     integer
            required true
    do
        result = on main do sql `DELETE FROM items WHERE id = {{request.id}}`
        match result
            case success(info)
                response
                    status       204
                    content_type "application/json"
                    body         ``
```

### Upsert (Postgres `ON CONFLICT`)

```capy
on analytics do sql `
    INSERT INTO daily_counts (day, key, n)
    VALUES ({{day}}, {{key}}, 1)
    ON CONFLICT (day, key) DO UPDATE SET n = daily_counts.n + 1
`
```

### Multi-statement SQL

`;`-separated statements all execute; the last one drives the outcome.

```capy
on main do sql `
    UPDATE pastes SET views = views + 1 WHERE slug = {{request.slug}};
    SELECT slug, body, views FROM pastes WHERE slug = {{request.slug}} LIMIT 1
`
```

`LIMIT 1` on the last statement classifies the result as single-row.

---

## 10. Pipelines (multi-step lookups)

Use nested `match` blocks. Each subsequent SQL inherits names from the
enclosing `case`.

```capy
route "/users/{id}/orders"
    methods GET
    request
        path_parameter id
            type     integer
            required true
    do
        user_lookup = on main do sql `SELECT id, email FROM users WHERE id = {{request.id}} LIMIT 1`

        match user_lookup
            case empty
                response
                    status       404
                    content_type "application/json"
                    body         `{"error":"user not found"}`

            case found(user)
                orders = on main do sql `SELECT id, total FROM orders WHERE user_id = {{u_id}}`
                    bind u_id
                        from user.id

                match orders
                    case success(rows)
                        response
                            status       200
                            content_type "application/json"
                            body         `{"user":{{to_json user}},"orders":{{to_json rows}}}`
```

`bind u_id / from user.id` is how an inner SQL references the outer
`case`'s binding. Direct dot access into the *raw* row is fine, but `bind`
makes the dependency explicit and avoids accidental name collisions.

---

## 11. Search, filters, arrays

### Optional filters (presence guards)

```capy
route "/products"
    methods GET
    request
        query_parameter category
            type text
        query_parameter min_price
            type    decimal
            minimum 0
        query_parameter q
            type       text
            max_length 100
    do
        result = on main do sql `
            SELECT id, name, price FROM products
             WHERE 1=1
            {{if has_value "category"}}  AND category = {{request.category}}        {{end}}
            {{if has_value "min_price"}} AND price >= {{request.min_price}}         {{end}}
            {{if has_value "q"}}         AND name LIKE {{like_contains request.q}}  {{end}}
             ORDER BY id DESC
             LIMIT 50
        `
        match result
            case success(rows)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json rows}}`
```

### Array input → `json_each`

```capy
route "/items/lookup"
    methods POST
    request
        content_type "application/json"
        body ids
            type     json_array
            required true
    do
        result = on main do sql `
            SELECT id, name FROM items
             WHERE id IN (SELECT value FROM json_each({{request.ids}}))
             ORDER BY id
        `
        match result
            case success(rows)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json rows}}`
```

The `json_array` input binds as one JSON-text parameter; `json_each` (or
the equivalent Postgres `jsonb_array_elements`) walks it.

### Batch insert (loop)

```capy
route "/stock/restock"
    methods POST
    request
        content_type "application/json"
        body items
            type     json_array
            required true
    do
        result = on main do sql `
            INSERT INTO stock (sku, qty) VALUES
            {{each "items" as row, i}}
                {{if i}},{{end}}
                ({{row.sku}}, {{row.qty}})
            {{end}}
        `
        match result
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"inserted":{{info.rows_affected}}}`
```

---

# CRUD & failure handling

## 12. CRUD resource end-to-end

A complete `things` resource: list / create / get / update / delete.

```capy
storage main
    kind     sqlite
    location "./things.db"

migration "001_things" on main
    up `
        CREATE TABLE things (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            name       TEXT NOT NULL,
            created_at TEXT NOT NULL
        )
    `
    down `DROP TABLE things`

route "/things"
    methods GET
    request
        query_parameter limit
            type    integer
            minimum 1
            maximum 200
        query_parameter q
            type       text
            max_length 100
    do
        result = on main do sql `
            SELECT id, name, created_at FROM things
             WHERE 1=1
            {{if has_value "q"}} AND name LIKE {{like_contains request.q}} {{end}}
             ORDER BY id DESC
             LIMIT {{request.limit}}
        `
        match result
            case success(rows)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json rows}}`

route "/things"
    methods POST
    request
        content_type "application/json"
        body_field name
            type       text
            required   true
            min_length 1
            max_length 200
    do
        result = on main do sql `
            INSERT INTO things (name, created_at) VALUES ({{request.name}}, {{now}})
        `
        match result
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}}}`

route "/things/{id}"
    methods GET
    request
        path_parameter id
            type     integer
            required true
    do
        lookup = on main do sql `SELECT id, name, created_at FROM things WHERE id = {{request.id}} LIMIT 1`
        match lookup
            case empty
                response
                    status       404
                    content_type "application/json"
                    body         `{"error":"not found"}`
            case found(row)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json row}}`

route "/things/{id}"
    methods PATCH
    request
        path_parameter id
            type     integer
            required true
        content_type "application/json"
        body_field name
            type       text
            required   true
            min_length 1
            max_length 200
    do
        result = on main do sql `UPDATE things SET name = {{request.name}} WHERE id = {{request.id}}`
        match result
            case success(info)
                match info.rows_affected
                    case 0
                        response
                            status       404
                            content_type "application/json"
                            body         `{"error":"not found"}`
                    case _
                        response
                            status       200
                            content_type "application/json"
                            body         `{"updated":{{info.rows_affected}}}`

route "/things/{id}"
    methods DELETE
    request
        path_parameter id
            type     integer
            required true
    do
        result = on main do sql `DELETE FROM things WHERE id = {{request.id}}`
        match result
            case success(info)
                response
                    status       204
                    content_type "application/json"
                    body         ``
```

That's a complete REST resource — no Go code, no separate schema file.

---

## 13. Failure cases & `handle`

Every pre-handler stage has a default JSON response. Override per-route:

```capy
route "/items"
    methods POST
    request
        content_type "application/json"
        body_field name
            type       text
            required   true
            min_length 1
            max_length 200
    do
        # ...

    handle
        case wrong_content_type(expected, got)
            response
                status       415
                content_type "application/json"
                body         `{"error":"send_json","expected":"{{expected}}","got":"{{got}}"}`

        case invalid_input(field, reason)
            response
                status       422
                content_type "application/json"
                body         `{"error":"invalid","field":"{{field}}","reason":"{{reason}}"}`

        case body_too_large(max_bytes)
            response
                status       413
                content_type "application/json"
                body         `{"error":"too_large","max_bytes":{{max_bytes}}}`
```

### All 11 failure cases

| case | when | default status |
|---|---|---|
| `method_not_allowed(allowed)` | request method not in `methods` | 405 |
| `wrong_content_type(expected, got)` | body type mismatch | 415 |
| `body_too_large(max_bytes)` | body exceeds `max_body_bytes` | 413 |
| `invalid_body(detail)` | JSON parse failed / multipart broken | 400 |
| `invalid_input(field, reason)` | input failed validator | 400 |
| `unauthenticated` | `requires_authentication` failed | 401 |
| `invalid_credentials` | auth-login wrong password | 401 |
| `forbidden(reason)` | RBAC predicate failed | 403 |
| `rate_limited(retry_after)` | `rate_limit` exceeded | 429 |
| `timeout(elapsed)` | `timeout` exceeded | 504 |
| `server_error(err)` | unhandled / panic / DB error | 500 |

---

## 14. Server-wide defaults

Set every route's defaults in one place:

```capy
server
    listen           ":8080"
    request_timeout  30s
    max_body_bytes   1048576       # 1 MiB

defaults
    request
        content_type "application/json"
    response
        content_type "application/json"
        set_header X-Frame-Options
            value "DENY"
        set_header X-Content-Type-Options
            value "nosniff"
        set_header Referrer-Policy
            value "strict-origin-when-cross-origin"

    handle
        case server_error(err)
            response
                status       500
                content_type "application/json"
                body         `{"error":"internal","trace_id":"{{err.trace_id}}"}`
        case rate_limited(retry_after)
            response
                status       429
                content_type "application/json"
                body         `{"error":"slow_down","retry_after":{{retry_after}}}`
                set_header Retry-After
                    value retry_after
```

Routes inherit; `handle` blocks on a route override.

---

# Auth & authorization

## 15. Session cookies + password

```capy
authentication user
    kind             session_cookie
    cookie_name      "session"
    session_lifetime 24h
    same_site        lax
    secure           true
    http_only        true

route "/signup"
    methods POST
    request
        content_type "application/json"
        body_field email
            type     email
            required true
        body_field password
            type       text
            required   true
            min_length 8
    do
        insert = on main do sql `
            INSERT INTO users (email, password_hash, created_at)
            VALUES ({{request.email}}, {{hash_password request.password}}, {{now}})
        `
        match insert
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}}}`
            case error(err)
                response
                    status       409
                    content_type "application/json"
                    body         `{"error":"email taken"}`

route "/login"
    methods POST
    request
        content_type "application/json"
        body_field email
            type     email
            required true
        body_field password
            type     text
            required true
    do
        match on user do authenticate_with_password
            using email
                from request.email
            using password
                from request.password
            case success(session)
                response
                    status       200
                    content_type "application/json"
                    body         `{"ok":true}`
                    set_cookie session
                        value     session.token
                        lifetime  24h
                        http_only true
                        secure    true
                        same_site lax
            case error(err)
                response
                    status       401
                    content_type "application/json"
                    body         `{"error":"invalid credentials"}`

route "/logout"
    methods POST
    requires_authentication user
    request
    do
        on user do end_session
        response
            status       200
            content_type "application/json"
            body         `{"ok":true}`
            clear_cookie session

route "/me"
    methods GET
    requires_authentication user
    request
    do
        lookup = on main do sql `SELECT id, email FROM users WHERE id = {{auth.user.id}} LIMIT 1`
        match lookup
            case found(me)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json me}}`
```

`auth.user.id`, `auth.user.email`, `auth.user.roles`, `auth.user.claims.X`
are available in every authenticated route.

---

## 16. Magic-link login

```capy
plugin mailer
    kind         http
    endpoint_url "https://api.resend.com/emails"
    headers
        Authorization "Bearer {{secret resend_api_key}}"
        Content-Type  "application/json"

authentication user_email
    kind             magic_link_email
    session_lifetime 30d
    link_lifetime    15m
    plugin           mailer
    from_address     "no-reply@example.com"
    subject          "Sign in to ExampleApp"
    link_path        "/auth/consume"

route "/auth/request"
    methods POST
    request
        content_type "application/json"
        body_field email
            type     email
            required true
    do
        on user_email do send_magic_link
            using email
                from request.email
        response
            status       202
            content_type "application/json"
            body         `{"ok":true,"message":"check your inbox"}`

route "/auth/consume"
    methods GET
    request
        query_parameter token
            type     text
            required true
    do
        match on user_email do consume_magic_link
            using token
                from request.token
            case success(session)
                response
                    status       302
                    content_type "text/plain"
                    body         ``
                    set_header Location
                        value "/dashboard"
                    set_cookie session
                        value     session.token
                        lifetime  30d
                        http_only true
                        secure    true
                        same_site lax
            case error(err)
                response
                    status       401
                    content_type "application/json"
                    body         `{"error":"link expired or invalid"}`
```

---

## 17. OAuth

### Google

```capy
authentication google
    kind          oauth
    provider      google
    client_id     "{{secret google_client_id}}"
    client_secret "{{secret google_client_secret}}"
    callback_path "/auth/google/callback"
    scopes        "openid", "email", "profile"
    domain_allow  "example.com"           # optional: gate to a Workspace

route "/auth/google/start"
    methods GET
    request
    do
        on google do begin_oauth_flow
        # the operation issues a 302 to Google; no response block needed

route "/auth/google/callback"
    methods GET
    request
        query_parameter code
            type     text
            required true
        query_parameter state
            type     text
            required true
    do
        match on google do complete_oauth_flow
            using code
                from request.code
            using state
                from request.state
            case success(session)
                response
                    status       302
                    content_type "text/plain"
                    body         ``
                    set_header Location
                        value "/dashboard"
                    set_cookie session
                        value     session.token
                        lifetime  24h
                        http_only true
                        secure    true
                        same_site lax
            case error(err)
                response
                    status       401
                    content_type "application/json"
                    body         `{"error":"oauth failed","detail":"{{err}}"}`
```

### Generic OIDC

Swap `provider google` for `provider oidc` and add `issuer_url`. GitHub,
Apple, Auth0, Keycloak, Okta — same shape.

---

## 18. JWT / header auth

For API consumers (bots, CLIs, mobile apps):

```capy
authentication api_jwt
    kind            jwt
    header_name     "Authorization"
    header_prefix   "Bearer "
    public_key_pem  "{{secret jwt_public_pem}}"
    issuer          "https://issuer.example.com"
    audience        "wave-api"

# Or API keys backed by SQL
authentication api_key
    kind        header_lookup
    header_name "X-Api-Key"
    validation_sql `
        SELECT user_id, tier
          FROM api_keys
         WHERE key = {{request.key}} AND revoked_at IS NULL
         LIMIT 1
    `
```

Use as `requires_authentication api_jwt` (or `api_key`). For JWT,
`auth.user.claims.<name>` exposes every claim from the token.

---

## 19. RBAC, per-resource ownership, claims

### Role check

```capy
route "/admin/users"
    methods GET
    requires_authentication user
    request
    do
        match auth.user.roles
            case contains("admin")
                result = on main do sql `SELECT id, email, created_at FROM users ORDER BY id`
                match result
                    case success(rows)
                        response
                            status       200
                            content_type "application/json"
                            body         `{{to_json rows}}`
            case _
                response
                    status       403
                    content_type "application/json"
                    body         `{"error":"admin role required"}`
```

### Per-resource ownership

Encode as a SQL predicate. Don't separate the auth check from the query —
combine them so a missing row and a non-owned row are indistinguishable.

```capy
route "/notes/{id}"
    methods DELETE
    requires_authentication user
    request
        path_parameter id
            type     integer
            required true
    do
        result = on main do sql `
            DELETE FROM notes
             WHERE id = {{request.id}} AND owner_id = {{auth.user.id}}
        `
        match result
            case success(info)
                match info.rows_affected
                    case 0
                        response
                            status       404
                            content_type "application/json"
                            body         `{"error":"not found"}`
                    case _
                        response
                            status       204
                            content_type "application/json"
                            body         ``
```

### Claims-based (JWT / OIDC)

```capy
route "/tenant/{tenant_id}/data"
    methods GET
    requires_authentication api_jwt
    request
        path_parameter tenant_id
            type     text
            required true
    do
        match auth.user.claims.tenant
            case equals(request.tenant_id)
                result = on main do sql `
                    SELECT id, name FROM data
                     WHERE tenant_id = {{request.tenant_id}}
                `
                match result
                    case success(rows)
                        response
                            status       200
                            content_type "application/json"
                            body         `{{to_json rows}}`
            case _
                response
                    status       403
                    content_type "application/json"
                    body         `{"error":"wrong tenant"}`
```

---

# Files

## 20. Upload (multipart)

```capy
route "/avatars"
    methods POST
    requires_authentication user
    request
        content_type "multipart/form-data"
        body_file image
            required  true
            max_bytes 5242880      # 5 MiB
    do
        # Persist to local disk via a plugin or to SQL as a blob
        result = on main do sql `
            INSERT INTO avatars (user_id, file_name, content_type, bytes, created_at)
            VALUES ({{auth.user.id}}, {{request.image.name}}, {{request.image.content_type}}, {{request.image.bytes}}, {{now}})
        `
        match result
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}},"name":"{{request.image.name}}"}`
```

`request.image` exposes `name`, `content_type`, `bytes` (the file body),
`size`.

For S3 / R2 / B2 / Spaces uploads, see [§42](#42-s3-r2-image-uploads).

---

## 21. Download

```capy
route "/avatars/{id}"
    methods GET
    request
        path_parameter id
            type     integer
            required true
    do
        lookup = on main do sql `
            SELECT file_name, content_type, bytes FROM avatars WHERE id = {{request.id}} LIMIT 1
        `
        match lookup
            case empty
                response
                    status       404
                    content_type "application/json"
                    body         `{"error":"not found"}`
            case found(file)
                response
                    status                         200
                    body_bytes                     file.bytes
                    content_type_from_extension_of file.file_name
                    set_header Content-Disposition
                        value `attachment; filename="{{file.file_name}}"`
                    set_header Cache-Control
                        value "public, max-age=3600, immutable"
```

`body_bytes` streams the binary; `content_type_from_extension_of` picks
a sensible MIME by file extension (override with `content_type` if needed).

---

## 22. Static & file-server

For frontend assets, marketing pages, docs:

```capy
# Single file (e.g. a landing page)
route "/"
    methods GET, HEAD
    request
    do
        response
            status                         200
            body_bytes_from_file           "./web/index.html"
            content_type_from_extension_of "./web/index.html"

# Directory tree (auto-routes to /static/*)
file_server static
    root_dir   "./web"
    mount_at   "/static/"
    cache      "public, max-age=86400"
    index_file "index.html"
```

Path traversal is blocked by default; `..` segments rejected.

---

# Routing patterns

## 23. Redirects, URL shorteners

```capy
storage main
    kind     sqlite
    location "./shortener.db"

route "/{code}"
    methods GET
    request
        path_parameter code
            type    text
            pattern "^[a-zA-Z0-9_-]{1,16}$"
    do
        lookup = on main do sql `
            UPDATE links SET hits = hits + 1, last_hit_at = {{now}} WHERE code = {{request.code}};
            SELECT target FROM links WHERE code = {{request.code}} LIMIT 1
        `
        match lookup
            case empty
                response
                    status       404
                    content_type "text/plain"
                    body         `link not found`
            case found(link)
                response
                    status       302
                    content_type "text/plain"
                    body         ``
                    set_header Location
                        value link.target

route "/api/shorten"
    methods POST
    request
        content_type "application/json"
        body_field target
            type     text
            required true
            pattern  "^https?://"
    do
        result = on main do sql `
            INSERT INTO links (code, target, created_at) VALUES ({{new_token}}, {{request.target}}, {{now}})
        `
        match result
            case success(info)
                lookup = on main do sql `SELECT code FROM links WHERE id = {{info.last_insert_id}} LIMIT 1`
                match lookup
                    case found(link)
                        response
                            status       201
                            content_type "application/json"
                            body         `{"short":"/{{link.code}}"}`
```

---

## 24. Forward / proxy

Pass-through to an upstream:

```capy
# Static upstream
route "/api/v1/{rest...}"
    methods GET, POST, PUT, PATCH, DELETE
    request
    do
        on upstream do forward
            base_url    "https://internal.api.example.com"
            preserve_path true
            forward_headers Authorization, X-Request-Id
            timeout     30s
```

### Dynamic / multi-tenant upstream

```capy
route "/proxy/{tenant}/api/{rest...}"
    methods GET, POST
    request
        path_parameter tenant
            type    text
            pattern "^[a-z0-9-]{1,32}$"
    do
        on dynamic do forward
            base_url         `https://{{request.tenant}}.upstream.local`
            allow_hosts      "*.upstream.local"
            block_private_ips true
            preserve_path    true
            timeout          15s
```

SSRF protection is on by default — pinning private IP ranges off.

---

## 25. Multi-tenant

By Host header:

```capy
route "/api/items"
    methods GET
    request
        header Host
            type     text
            required true
    do
        match request.host
            case equals("acme.example.com")
                result = on acme_db do sql `SELECT id, name FROM items`
                ...
            case equals("globex.example.com")
                result = on globex_db do sql `SELECT id, name FROM items`
                ...
            case _
                response
                    status       404
                    content_type "application/json"
                    body         `{"error":"unknown tenant"}`
```

By path segment is simpler — see §24's dynamic forward.

By cookie / claim:

```capy
route "/dashboard"
    methods GET
    requires_authentication user
    request
    do
        match auth.user.claims.tenant
            case equals("acme")
                response
                    status       200
                    content_type "text/html"
                    body         `<h1>Acme dashboard</h1>`
            case equals("globex")
                response
                    status       200
                    content_type "text/html"
                    body         `<h1>Globex dashboard</h1>`
```

---

## 26. A/B testing & device detection

### Variant cookie split

```capy
route "/"
    methods GET
    request
        cookie ab_variant
            type text
    do
        match request.ab_variant
            case equals("B")
                response
                    status       200
                    content_type "text/html"
                    body         `<h1>Variant B</h1>`
            case _
                # Default + assignment
                response
                    status       200
                    content_type "text/html"
                    body         `<h1>Variant A</h1>`
                    set_cookie ab_variant
                        value    "A"
                        lifetime 30d
                        same_site lax
```

### Device detection

```capy
route "/"
    methods GET
    request
        header User-Agent
            type text
    do
        match request.user_agent
            case matches("(?i)iPhone|Android|Mobile")
                response
                    status       200
                    content_type "text/html"
                    body         `<link rel="stylesheet" href="/m.css"><h1>Mobile</h1>`
            case _
                response
                    status       200
                    content_type "text/html"
                    body         `<link rel="stylesheet" href="/d.css"><h1>Desktop</h1>`
```

---

# Plugins

## 27. Plugin contract recap

A plugin receives a JSON `Request` and returns a JSON `Response`:

```jsonc
// Request
{
  "trigger_key": "translate",
  "metadata":    {"remote_ip": "1.2.3.4", "request_id": "..."},
  "headers":     {"authorization": "..."},
  "query":       {"q": "..."},
  "body":        {/* arbitrary JSON */}
}

// Response
{
  "status":  200,
  "headers": {"Content-Type": "application/json"},
  "body":    {/* arbitrary JSON */}
}
```

Three transports: **subprocess** (spawn-per-call), **long_lived_subprocess**
(persistent stdin/stdout, JSON-RPC LSP framing), **http** (POST to an
endpoint). All speak the same envelope.

Full reference: [`docs-site/reference/plugin-contract.md`](../../docs-site/reference/plugin-contract.md).

---

## 28. Subprocess plugin

```capy
plugin translator
    kind    subprocess
    command "python3" "./plugins/translate.py"

route "/api/translate"
    methods POST
    request
        content_type "application/json"
        body_field text
            type     text
            required true
        body_field target_lang
            type    text
            pattern "^[a-z]{2}$"
    do
        result = on translator do call_plugin
            trigger "translate"
            pass_input text
                from request.text
            pass_input target_lang
                from request.target_lang
        match result
            case success(data)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json data}}`
            case error(err)
                response
                    status       502
                    content_type "application/json"
                    body         `{"error":"plugin failed","detail":"{{err}}"}`
```

`./plugins/translate.py` (the plugin):

```python
import sys, json
req = json.loads(sys.stdin.read())
text = req["body"]["text"]
lang = req["body"]["target_lang"]
# ... call your translation API ...
out = {"status": 200, "body": {"translated": "..."}}
print(json.dumps(out))
```

---

## 29. Long-lived plugin (streaming)

For LLMs, ML inference, anything where boot cost matters:

```capy
plugin llm
    kind        long_lived_subprocess
    command     "./plugins/llm_server"
    working_dir "./plugins"
    environment
        MODEL_PATH "./model.bin"
        DEVICE     "cuda:0"

route "/api/chat"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field prompt
            type     text
            required true
    do
        task = background
            stream = on llm do call_plugin
                trigger   "chat"
                streaming true
                pass_input prompt
                    from request.prompt

            for_each_emission(event) of stream
                on chat_feed do broadcast
                    event_name      "chunk"
                    data            `{{to_json event}}`
                    partition_value auth.user.id

                on main do sql `
                    INSERT INTO chat_chunks (user_id, task_id, content, at)
                    VALUES ({{auth.user.id}}, {{task.id}}, {{c}}, {{now}})
                `
                    bind c
                        from event.content

        response
            status       202
            content_type "application/json"
            body         `{"task_id":"{{task.id}}"}`
```

The plugin process boots once at server start; each call frames as JSON-RPC
LSP-style over stdio. Streaming emits one frame per chunk.

---

## 30. HTTP plugin

For any external API — Stripe, Slack, SendGrid, OpenAI, your own service:

```capy
plugin stripe
    kind         http
    endpoint_url "https://api.stripe.com/v1/checkout/sessions"
    headers
        Authorization "Bearer {{secret stripe_secret_key}}"
        Content-Type  "application/x-www-form-urlencoded"

route "/api/checkout"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field price_id
            type     text
            required true
    do
        result = on stripe do call_plugin
            trigger "create_session"
            pass_input price_id
                from request.price_id
            pass_input customer_id
                from auth.user.id

        match result
            case success(data)
                response
                    status       200
                    content_type "application/json"
                    body         `{"checkout_url":"{{data.url}}"}`
```

`secret stripe_secret_key` reads from `WAVE_SECRET_STRIPE_SECRET_KEY` env
var (or a configured secrets plugin).

---

## 31. Storage & auth & secrets plugins

Plugins aren't only handlers. They also extend storage backends, auth
schemes, and secret resolvers.

### Storage plugin

```capy
storage warehouse
    kind     plugin
    plugin   clickhouse_plugin
    config
        dsn "{{secret clickhouse_dsn}}"

plugin clickhouse_plugin
    kind    subprocess
    command "./plugins/clickhouse"
```

Routes use it like any other storage: `on warehouse do sql \`SELECT ...\``.
The plugin implements `storage.query` / `storage.get` / `storage.set` from
the [plugin contract](../../docs-site/reference/plugin-contract.md#kind-storage).

### Auth plugin

```capy
authentication saml
    kind   plugin
    plugin saml_idp_plugin

plugin saml_idp_plugin
    kind         http
    endpoint_url "https://saml-bridge.internal/wave"
```

The plugin implements `auth.authenticate` / `auth.refresh_claims` /
`auth.logout`.

### Secrets plugin

```capy
plugin vault
    kind         http
    endpoint_url "https://vault.internal/v1/wave"

# Anywhere a secret is needed:
plugin stripe
    kind         http
    endpoint_url "https://api.stripe.com/v1/charges"
    headers
        Authorization "Bearer {{secret vault: secret/data/stripe#api_key}}"
```

The `vault:` prefix routes through the named secrets plugin.

---

# Real-time

## 32. SSE

```capy
connection events
    kind           server_sent_events
    subscribe_path "/events"
    buffer_size    256

route "/announce"
    methods POST
    request
        content_type "application/json"
        body_field message
            type     text
            required true
    do
        on events do broadcast
            event_name "announcement"
            data       `{{to_json request.message}}`

        response
            status       204
            content_type "application/json"
            body         ``
```

Clients connect to `GET /events` with `Accept: text/event-stream`. The
route is auto-registered — don't declare it yourself. Recent events replay
from the ring buffer (`buffer_size`) on connect.

---

## 33. WebSocket

```capy
connection cursors
    kind           websocket
    subscribe_path "/ws/cursors"
    buffer_size    256

route "/api/cursor"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field x
            type     integer
            required true
        body_field y
            type     integer
            required true
    do
        on cursors do broadcast
            event_name "move"
            data       `{"user_id":{{auth.user.id}},"x":{{request.x}},"y":{{request.y}}}`

        response
            status       204
            content_type "application/json"
            body         ``
```

---

## 34. Partitioned channels

Each subscriber sees only their partition:

```capy
connection notify
    kind           server_sent_events
    subscribe_path "/notify"
    buffer_size    64
    partition_by   auth.user.id        # requires auth

route "/notify/send"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field to
            type     integer
            required true
        body_field text
            type     text
            required true
    do
        on notify do broadcast
            event_name      "message"
            data            `{"from":{{auth.user.id}},"text":"{{request.text}}"}`
            partition_value request.to
```

User A only receives broadcasts where `partition_value == A`. Subscribing
to `/notify` requires authentication (the partition expression has to
evaluate).

---

# Async & scheduled work

## 35. Background tasks

```capy
route "/api/process"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field file_id
            type     uuid
            required true
    do
        task = background
            on resize do call_plugin
                trigger "thumbnail"
                pass_input file_id
                    from request.file_id

            on main do sql `
                UPDATE files SET status = 'processed', processed_at = {{now}}
                 WHERE id = {{request.file_id}}
            `

            on events do broadcast
                event_name "processed"
                data       `{"file_id":"{{request.file_id}}"}`

        response
            status       202
            content_type "application/json"
            body         `{"task_id":"{{task.id}}"}`
```

The `do` body inside `background` runs after the response is sent. Errors
are logged; the client gets a `task_id` to poll status if it needs to.

---

## 36. Scheduled jobs

```capy
# Every 30 seconds — poll a webhook list
every 30s as poll_pending
    pending = on main do sql `SELECT id, payload FROM outbox WHERE status = 'pending' LIMIT 50`

    match pending
        case found(rows)
            for_each_row(row) of rows
                result = on webhook_target do call_plugin
                    trigger "post"
                    pass_input id
                        from row.id
                    pass_input payload
                        from row.payload

                match result
                    case success(data)
                        on main do sql `UPDATE outbox SET status='sent', sent_at={{now}} WHERE id={{id}}`
                            bind id
                                from row.id
                    case error(err)
                        on main do sql `
                            UPDATE outbox
                               SET attempts = attempts + 1,
                                   last_error = {{e}},
                                   next_attempt_at = {{add_minutes 5}}
                             WHERE id = {{id}}
                        `
                            bind id
                                from row.id
                            bind e
                                from err

# Daily at 04:30 local — purge old sessions
at "04:30" as session_cleanup
    on main do sql `DELETE FROM sessions WHERE expires_at < {{now}}`
```

Schedules use the same operations / `match` vocabulary as routes — no
parallel syntax.

---

## 37. Outbox pattern

Durable webhooks with retry + DLQ:

```capy
# Producer route — enqueue (transactional with the business write)
route "/api/orders"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body items
            type     json_array
            required true
    do
        result = on main do sql `
            INSERT INTO orders (user_id, items, status, created_at)
            VALUES ({{auth.user.id}}, {{request.items}}, 'placed', {{now}});

            INSERT INTO outbox (kind, payload, status, attempts, created_at)
            VALUES (
                'order.placed',
                {{request.items}},
                'pending',
                0,
                {{now}}
            );

            SELECT id FROM orders WHERE id = last_insert_rowid() LIMIT 1
        `
        match result
            case found(order)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{order.id}}}`

# Drainer — runs every 5s
every 5s as drain_outbox
    pending = on main do sql `
        SELECT id, kind, payload FROM outbox
         WHERE status = 'pending' AND attempts < 10
            AND (next_attempt_at IS NULL OR next_attempt_at <= {{now}})
         ORDER BY id
         LIMIT 25
    `

    match pending
        case found(rows)
            for_each_row(row) of rows
                result = on webhook do call_plugin
                    trigger "deliver"
                    pass_input kind
                        from row.kind
                    pass_input payload
                        from row.payload

                match result
                    case success(data)
                        on main do sql `UPDATE outbox SET status='sent', sent_at={{now}} WHERE id={{id}}`
                            bind id
                                from row.id
                    case error(err)
                        on main do sql `
                            UPDATE outbox
                               SET attempts        = attempts + 1,
                                   last_error      = {{e}},
                                   next_attempt_at = {{add_minutes 5}},
                                   status          = CASE WHEN attempts + 1 >= 10 THEN 'dlq' ELSE 'pending' END
                             WHERE id = {{id}}
                        `
                            bind id
                                from row.id
                            bind e
                                from err

# Dead-letter inspection route
route "/admin/outbox/dlq"
    methods GET
    requires_authentication user
    request
    do
        match auth.user.roles
            case contains("admin")
                result = on main do sql `SELECT * FROM outbox WHERE status='dlq' ORDER BY id DESC LIMIT 200`
                match result
                    case success(rows)
                        response
                            status       200
                            content_type "application/json"
                            body         `{{to_json rows}}`
            case _
                response
                    status       403
                    content_type "application/json"
                    body         `{"error":"admin only"}`
```

At-least-once with idempotency-key support: have the receiver dedupe on
`(kind, payload_hash)`, or include an `idempotency_key` field.

---

# Integrations (worked examples)

## 38. Stripe

### Checkout

```capy
plugin stripe
    kind         http
    endpoint_url "https://api.stripe.com/v1/checkout/sessions"
    headers
        Authorization "Bearer {{secret stripe_secret_key}}"
        Content-Type  "application/x-www-form-urlencoded"

route "/billing/checkout"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field price_id
            type     text
            required true
    do
        result = on stripe do call_plugin
            trigger "create"
            pass_input price_id
                from request.price_id
            pass_input customer_email
                from auth.user.email
            pass_input success_url
                from "https://example.com/billing/success"
            pass_input cancel_url
                from "https://example.com/billing/cancel"

        match result
            case success(data)
                response
                    status       200
                    content_type "application/json"
                    body         `{"url":"{{data.url}}"}`
```

### Webhooks (signature verify + persist + fan-out)

```capy
route "/billing/webhook"
    methods POST
    request
        content_type "application/json"
        header Stripe-Signature
            type     text
            required true
        body_raw payload
            type     bytes
            required true
        webhook_sig stripe
            secret   "{{secret stripe_webhook_secret}}"
            signature_header Stripe-Signature
            body     payload

    do
        # at this point signature has already been verified by the middleware
        on main do sql `
            INSERT INTO stripe_events (id, type, payload, received_at)
            VALUES ({{event.id}}, {{event.type}}, {{event.data}}, {{now}})
            ON CONFLICT (id) DO NOTHING
        `

        on stripe_events do broadcast
            event_name event.type
            data       `{{to_json event.data}}`

        response
            status       200
            content_type "application/json"
            body         `{"received":true}`
```

`webhook_sig stripe` is a built-in middleware that verifies the HMAC
signature; non-verified requests get `401` before reaching `do`.

---

## 39. Send email

Resend / SendGrid / Postmark / Mailgun — all the same shape.

```capy
plugin mailer
    kind         http
    endpoint_url "https://api.resend.com/emails"
    headers
        Authorization "Bearer {{secret resend_api_key}}"
        Content-Type  "application/json"

route "/api/contact"
    methods POST
    request
        content_type "application/json"
        body_field name
            type     text
            required true
        body_field email
            type     email
            required true
        body_field message
            type       text
            required   true
            min_length 1
            max_length 2000
    do
        result = on mailer do call_plugin
            trigger "send"
            pass_input from
                from "no-reply@example.com"
            pass_input to
                from "team@example.com"
            pass_input subject
                from `New contact: {{request.name}}`
            pass_input html
                from `<p>{{request.name}} &lt;{{request.email}}&gt; said:</p><p>{{request.message}}</p>`

        match result
            case success(data)
                response
                    status       202
                    content_type "application/json"
                    body         `{"ok":true}`
```

---

## 40. Slack slash command

```capy
route "/slack/deploy"
    methods POST
    request
        content_type "application/x-www-form-urlencoded"
        body_field text
            type text
        body_field user_id
            type     text
            required true
        body_raw raw
            type     bytes
            required true
        webhook_sig slack
            secret           "{{secret slack_signing_secret}}"
            signature_header X-Slack-Signature
            timestamp_header X-Slack-Request-Timestamp
            body             raw
    do
        task = background
            on deploy_worker do call_plugin
                trigger "run"
                pass_input target
                    from request.text
                pass_input requested_by
                    from request.user_id

        response
            status       200
            content_type "application/json"
            body         `{"response_type":"ephemeral","text":":rocket: deploying {{request.text}}..."}`
```

---

## 41. LLM chat

OpenAI / Anthropic / Ollama — through a long-lived plugin:

```capy
plugin llm
    kind        long_lived_subprocess
    command     "node" "./plugins/llm/server.js"
    environment
        OPENAI_API_KEY "{{secret openai_api_key}}"

connection chat_feed
    kind           server_sent_events
    subscribe_path "/chat/stream"
    buffer_size    64
    partition_by   auth.user.id

authentication user
    kind             session_cookie
    cookie_name      "session"
    session_lifetime 7d

route "/chat"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field prompt
            type     text
            required true
    do
        task = background
            stream = on llm do call_plugin
                trigger   "chat"
                streaming true
                pass_input model
                    from "gpt-4o-mini"
                pass_input prompt
                    from request.prompt

            for_each_emission(event) of stream
                on chat_feed do broadcast
                    event_name      "chunk"
                    data            `{{to_json event}}`
                    partition_value auth.user.id

                on main do sql `
                    INSERT INTO chat_log (user_id, role, content, at)
                    VALUES ({{auth.user.id}}, 'assistant', {{c}}, {{now}})
                `
                    bind c
                        from event.content

        response
            status       202
            content_type "application/json"
            body         `{"task_id":"{{task.id}}"}`
```

The frontend connects an SSE source to `/chat/stream` and renders chunks
as they arrive. The full transcript persists to SQL on the fly.

---

## 42. S3 / R2 image uploads

Pre-signed PUT URL pattern — the client uploads directly to S3/R2/B2/Spaces;
your server only mediates.

```capy
plugin s3_presigner
    kind        long_lived_subprocess
    command     "./plugins/s3_presigner"
    environment
        S3_ENDPOINT   "{{secret s3_endpoint}}"
        S3_BUCKET     "{{secret s3_bucket}}"
        S3_ACCESS_KEY "{{secret s3_access_key}}"
        S3_SECRET_KEY "{{secret s3_secret_key}}"

route "/uploads/presign"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field file_name
            type     text
            required true
        body_field content_type
            type     text
            required true
    do
        result = on s3_presigner do call_plugin
            trigger "presign_put"
            pass_input key
                from `uploads/{{auth.user.id}}/{{new_uuid}}-{{request.file_name}}`
            pass_input content_type
                from request.content_type
            pass_input lifetime_seconds
                from 300

        match result
            case success(data)
                on main do sql `
                    INSERT INTO upload_intents (user_id, key, content_type, requested_at)
                    VALUES ({{auth.user.id}}, {{k}}, {{ct}}, {{now}})
                `
                    bind k
                        from data.key
                    bind ct
                        from request.content_type

                response
                    status       200
                    content_type "application/json"
                    body         `{"upload_url":"{{data.url}}","key":"{{data.key}}"}`
```

---

# Operations

## 43. Rate limiting, timeouts, body limits

### Per-route

```capy
route "/api/login"
    methods POST
    rate_limit
        per "1m"
        max 10              # 10 attempts per minute per IP
    timeout 5s
    request
        content_type "application/json"
        body_field email
            type     email
            required true
        body_field password
            type       text
            required   true
            max_length 256
        # also bound by server-wide max_body_bytes
```

### Server-wide

```capy
server
    listen           ":8080"
    request_timeout  30s         # all routes default to this
    max_body_bytes   1048576     # 1 MiB default
```

Rate-limit scope is per principal when authenticated, per IP otherwise.

---

## 44. Secrets & configuration

```capy
plugin stripe
    kind         http
    endpoint_url "https://api.stripe.com/v1/checkout/sessions"
    headers
        Authorization "Bearer {{secret stripe_secret_key}}"
```

Secret resolution order:
1. `WAVE_SECRET_STRIPE_SECRET_KEY` env var (uppercased name).
2. Configured secrets plugin (`${PLUGIN:vault:secret/data/stripe#api_key}`).
3. Local `secrets.env` file (development only).

Generate / list / set secrets:

```bash
wave secrets list
wave secrets set stripe_secret_key
wave secrets generate session_signing_key --length 64
```

---

## 45. Observability

```capy
server
    listen ":8080"

observability
    metrics
        prometheus_path "/metrics"
    traces
        otlp_endpoint  "http://otel-collector:4318"
        sample_rate    0.1
    logs
        format "json"
        level  "info"
```

Wave exports per-route counters and histograms (`wave_http_requests_total`,
`wave_http_request_duration_seconds`), traces spans per request and per
operation, and structured logs with request IDs.

For custom sinks, register an `exporter` plugin (see
[plugin contract §exporter](../../docs-site/reference/plugin-contract.md#kind-exporter)).

---

## 46. Testing with `wave test`

```capy
# server.test.capy
import "server.capy"

test "create then read"
    request POST "/items"
        content_type "application/json"
        json { "name": "laptop" }
    expect
        status 201
        json   { "id": "*" }      # "*" = any value
    capture id from json.id

test "read it back"
    request GET "/items/{{id}}"
    expect
        status 200
        json   { "name": "laptop", "id": id }

test "404 on missing"
    request GET "/items/999999"
    expect
        status 404
        json   { "error": "not found" }

test "auth required"
    request GET "/me"
    expect
        status 401
        json   { "error": "authentication_required" }
```

Tests run in-process — no port collisions, no external services unless
your plugins need them. Run with `wave test server.test.capy`. In CI,
the gate is two shell commands:

```bash
wave check server.capy        # parse + validate (fails build on any error)
wave test  server.test.capy   # run the suite
```

---

## 47. Deployment

### Docker

```dockerfile
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=ghcr.io/luowensheng/wave:latest /usr/local/bin/wave /usr/local/bin/wave
COPY server.capy /app/server.capy
COPY data        /app/data
WORKDIR /app
EXPOSE 8080
CMD ["wave", "serve", "/app/server.capy"]
```

### Fly.io

```toml
# fly.toml
app = "my-wave-app"
primary_region = "iad"

[build]
  image = "ghcr.io/luowensheng/wave:latest"

[env]
  PORT = "8080"

[mounts]
  source = "data"
  destination = "/app/data"

[[services]]
  protocol = "tcp"
  internal_port = 8080

  [[services.ports]]
    handlers = ["http"]
    port = 80
  [[services.ports]]
    handlers = ["tls", "http"]
    port = 443

  [services.concurrency]
    type = "connections"
    hard_limit = 200
    soft_limit = 100
```

```bash
fly secrets set STRIPE_SECRET_KEY=sk_...
fly volumes create data --size 1
fly deploy
```

### Railway

Push the repo with `server.capy` at root; Railway autodetects via the
included `wave.json`. Set env vars in the dashboard. Persistent volume at
`/app/data` for SQLite.

### Render / VPS

Single binary + `server.capy` + a systemd unit. No language runtime.
SQLite goes on persistent disk; Postgres connection string in
`WAVE_SECRET_DB_DSN`.

---

# Reference appendix

## 48. Operation table

| Operation | Form |
|---|---|
| SQL | `[name =] on STORAGE do sql \`…\`` with optional `bind KEY / from PATH` |
| Plugin call | `[name =] on PLUGIN do call_plugin / trigger "T" [/ streaming true] / pass_input KEY / from PATH` |
| Broadcast | `on CONNECTION do broadcast / event_name "E" / data \`…\` [/ partition_value EXPR]` |
| Forward / proxy | `on PROXY do forward / base_url "…" / preserve_path true / forward_headers H1, H2` |
| Authenticate | `match on AUTH do authenticate_with_password / using FIELD / from request.x` |
| End session | `on AUTH do end_session` |
| Magic-link issue | `on AUTH do send_magic_link / using email / from request.email` |
| Magic-link consume | `match on AUTH do consume_magic_link / using token / from request.token` |
| OAuth begin | `on AUTH do begin_oauth_flow` |
| OAuth complete | `match on AUTH do complete_oauth_flow / using code / from request.code / using state / from request.state` |
| Background | `task = background / …` |
| Iterate rows | `for_each_row(NAME) of RESULT / …` |
| Iterate emissions | `for_each_emission(NAME) of STREAM / …` |

---

## 49. Match-case table

| case | what it binds | when |
|---|---|---|
| `error(err)` | err = message | the operation failed |
| `empty` | nothing | SELECT 0 rows / DML 0 rows affected / plugin returned empty |
| `found(data)` | data = row or list-of-rows | SELECT returned 1+ rows (LIMIT 1 → single map) |
| `success(info)` | info.last_insert_id, info.rows_affected | any non-error outcome (use for INSERT/UPDATE/DELETE) |

Match also works on **values**: `contains("x")`, `equals(y)`,
`matches("regex")`, integer literals, `_` (wildcard).

---

## 50. Type & modifier table

| Type | Coerces from | Modifiers |
|---|---|---|
| `text` | any string | `min_length`, `max_length`, `pattern` |
| `integer` | string, JSON number | `minimum`, `maximum` |
| `decimal` | string, JSON number | `minimum`, `maximum` |
| `true_or_false` | `"true"`/`"false"`, `1`/`0`, JSON bool | — |
| `email` | string | implicit format check |
| `uuid` | string | implicit format check |
| `file` | multipart file part | `max_bytes` |
| `bytes` | raw body | `max_bytes` |
| `json_array` | JSON array body / field | — |
| `json_object` | JSON object body / field | — |

`required` and a `default` value apply to every type.

---

## Where to go next

- [`wave-capy-complete.md`](./wave-capy-complete.md) — every shape, every
  field, in catalogue form.
- [`wave-capy-templating.md`](./wave-capy-templating.md) — the safety
  story behind SQL binding & response auto-escape.
- [`wave-capy-migrations.md`](./wave-capy-migrations.md) — full
  migrations reference: ledger, checksums, `requires` graph, CLI,
  per-env tags, round-trip testing.
- [`wave-capy-cookbook.md`](./wave-capy-cookbook.md) — 50+ task-oriented
  recipes.
- [`wave-capy-projects.md`](./wave-capy-projects.md) — 25 worked apps,
  full source.
- [`wave-capy-route.md`](./wave-capy-route.md) — the route lifecycle and
  every failure case.
- [`wave-capy-tooling.md`](./wave-capy-tooling.md) — `wave doctor`,
  `wave describe`, `wave export`, the LSP.
- [`../../examples/apps/INDEX.md`](../../examples/apps/INDEX.md) —
  64 runnable demos.
