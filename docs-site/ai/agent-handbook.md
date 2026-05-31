# Wave — agent handbook

A single self-contained guide for an AI coding agent (or a human new to
Wave) to build any backend Wave supports. Read this once; you should be
able to write a working `server.capy` for any realistic web-backend task.

This is the **agent handbook** — strict, vocabulary-locked, with
worked walkthroughs and pointers to deeper docs when (and only when)
you need them. Skim the table of contents, then read **§0 (must-knows)**
and **§1–§4 (the mental model)** before you touch anything else.

---

## Table of contents

- [§0 Must-knows (read this first)](#0-must-knows)
- [§1 What Wave is, and isn't](#1-what-wave-is)
- [§2 The mental model in one page](#2-mental-model)
- [§3 Capy — the language, strict rules](#3-capy-strict-rules)
- [§4 The six declaration kinds](#4-six-declarations)
- [§5 Building blocks](#5-building-blocks)
  - 5.1 [The skeleton of a route](#51-route-skeleton)
  - 5.2 [Every input source](#52-inputs)
  - 5.3 [Templates — SQL & response, the two rules](#53-templates)
  - 5.4 [Operations](#54-operations)
  - 5.5 [Match & pipelines](#55-match-pipelines)
  - 5.6 [Responses, headers, cookies](#56-responses)
  - 5.7 [Failure cases & `handle`](#57-failures)
- [§6 Walkthrough 1 — hello → todo API (45 min)](#6-walkthrough-1)
- [§7 Walkthrough 2 — auth + per-user data (60 min)](#7-walkthrough-2)
- [§8 Walkthrough 3 — realtime + background + plugin (90 min)](#8-walkthrough-3)
- [§9 Schema migrations (first-class)](#9-migrations)
- [§10 Plugins (when capy isn't enough)](#10-plugins)
- [§11 Auth schemes — when to pick which](#11-auth)
- [§12 Observability, secrets, deployment](#12-ops)
- [§13 Testing — `wave test`](#13-testing)
- [§14 Common mistakes & how to detect them](#14-mistakes)
- [§15 Self-verification checklist](#15-verify)
- [§16 Where to look next (every other doc, with purpose)](#16-more-info)
- [§17 Agent-specific tips](#17-agent-tips)

---

## §0. Must-knows

Read these eight lines before writing capy. If you internalise them,
you will not produce broken `server.capy`.

1. **The language is capy** — line-oriented, indentation-blocked. It
   is its own DSL, not a serialised config. Backticks delimit
   multi-line strings.
2. **Every input is declared in `request`.** Anything not declared is
   inaccessible; referencing it is a load-time error, not a runtime
   bug.
3. **`{{request.x}}` in SQL binds as a `?` parameter automatically.**
   There is no way to splice a value into SQL text. SQL injection is
   structurally impossible.
4. **Response bodies auto-escape per the declared `content_type`.**
   HTML-escape for `text/html`, JSON-escape for `application/json`.
   Opt out with `{{raw x}}`.
5. **`content_type` is mandatory** on any `request` that reads a body
   and on every `response`. Full MIME strings, no aliases.
6. **Operations are `[name =] on TARGET do OPERATION`.** Same shape
   for SQL, plugin calls, broadcasts. Read top-to-bottom.
7. **Outcomes are pattern-matched.** `case error(err)`, `case empty`,
   `case found(data)`, `case success(info)`. You don't have to handle
   every case — unhandled `error` becomes a 500 by default.
8. **File extension is `.capy` or `.wave`** — both load identically.
   `.capy` is the language; `.wave` is a friendlier alias because the
   product is Wave. Anything else (`.yaml`/`.yml`/`.json`) is passed
   straight to the YAML loader; capy syntax in a non-capy extension
   will fail to parse.
9. **`wave check server.capy`** parses + validates. Run it after every
   non-trivial edit. It catches missing inputs, broken matches,
   unknown storages, and migration drift before the server boots.

If any of those eight surprised you, slow down and read §3 + §5
before writing code.

---

## §1. What Wave is

Wave is **a single Go binary that runs a backend you described in
capy**. One `server.capy` file declares storage, plugins, connections,
routes, schedules, migrations, and auth. No hand-written Go for app
code. The binary ships your config; you can deploy it without
touching the build.

**What it is good at**: JSON/HTML APIs over SQLite / Postgres / MySQL,
auth flows (session cookie / magic-link / OAuth / OIDC / JWT / API
key), file uploads and serving, SSE / WebSocket pushes, scheduled
jobs, plugin extensions (subprocess / long-lived / HTTP), outbox-style
durable webhooks, multi-tenant routing, rate limiting, OpenTelemetry
and Prometheus out of the box.

**What it isn't**:

- A full programming language. Branching past simple `match` requires
  a plugin.
- An ORM. SQL goes through; you write SQL.
- A frontend framework. It serves files and JSON; you bring React /
  HTMX / curl.
- A general-purpose RPC server (use plugins for non-HTTP transports).

When you need behavior Wave can't express in capy, write a **plugin**
in any language and call it from a route (§10).

---

## §2. Mental model

A `server.capy` is a flat list of **declarations**. Six kinds:

| Declaration | What it makes available | Example |
|---|---|---|
| `storage` | A database the server can `on <name> do sql` against. | SQLite, Postgres, MySQL |
| `connection` | An SSE or WebSocket broker. | SSE channel for chat |
| `plugin` | A process or HTTP endpoint Wave can `call_plugin` on. | LLM worker, payment API |
| `authentication` | A scheme routes opt into via `requires_authentication`. | Session cookie, OAuth |
| `migration` | A versioned schema change applied at boot. | `CREATE TABLE …` |
| `route` / `every` / `at` | Triggers — HTTP routes, periodic, daily. | `GET /users/{id}` |

A **route** has five sections. Read them top-to-bottom:

```
route "/path"
    methods METHOD                    ← the header
    [requires_authentication SCHEME]
    [rate_limit ... ]
    [timeout DURATION]

    request                           ← what comes in
        content_type "..."
        path_parameter NAME ...
        body_field NAME ...

    do                                ← what happens
        result = on STORAGE do sql `...`
        match result
            case ...
                response ...          ← what goes out

    handle                            ← optional: failure overrides
        case ...
            response ...
```

The handler reads as a story:
*what it takes (`request`) → what it does (`do`) → what it returns
(`response` inside a `case`)*.

That's the whole model. Everything else is detail.

---

## §3. Capy — strict rules

### 3.1 Lexical rules

- **Line is the unit.** One key, one block opener, or one operation
  per line. Never two on one line.
- **Indent forms blocks.** Use spaces (4 is conventional). Mixing
  tabs and spaces fails to parse.
- **Backticks delimit multi-line strings.** Use them for SQL and any
  body that spans lines.
- **Double-quotes** delimit single-line strings (identifiers, paths,
  MIME types).
- **No commas inside blocks**, only between flag args (e.g. `methods
  GET, POST`, `scopes "openid", "email"`).
- **Comments**: `#` to end of line. Allowed anywhere.

### 3.2 Identifier conventions

- **Storage / connection / plugin / authentication names**:
  snake_case nouns. `main`, `analytics`, `chat_feed`, `user`,
  `mailer`.
- **Route paths**: standard URL syntax with `{name}` for path
  parameters and `{rest...}` for a catch-all.
- **Inputs**: snake_case. Referenced as `request.<name>` in templates
  and SQL.
- **Headers**: declare as the literal HTTP header name
  (`X-Api-Key`); reference snake-cased (`request.x_api_key`).
- **Migration ids**: `NNN_snake_case` (e.g. `001_initial`,
  `017_add_invoices_index`). Sort order should match declaration order.

### 3.3 The "no magic" guarantees

These are guarantees you can rely on without checking:

| Guarantee | How |
|---|---|
| **No SQL injection.** | Every `{{request.x}}` / `{{auth.user.id}}` / `{{now}}` becomes a `?` parameter. No syntax splices values into SQL text. |
| **No XSS by default.** | Response bodies auto-escape per `content_type`. Opt out with `{{raw x}}`. |
| **No undeclared input errors at runtime.** | Referencing `{{request.foo}}` where `foo` isn't declared in `request` fails at `wave check` time. |
| **No accidental route conflicts.** | Same path + same method → load-time error. Different methods on the same path → separate `route` blocks. |
| **No silent migration drift.** | Edited a committed `migration` body? Boot fails with `migration_checksum_mismatch`. |
| **No `init()` magic.** | Wave's Go side wires dependencies explicitly. There is no implicit boot-time mutation surface a plugin could inject into. |

If a check ever surprises you, the surprise is the bug. File it.

### 3.4 What you cannot do

These are constructs that **do not exist** in capy. Don't look for them.

| Want to do | Don't expect to find | Do instead |
|---|---|---|
| Branch on arbitrary expression | `if`/`else` at handler level | `match` on a value or operation outcome |
| Define a reusable function | `fn …`, `def …` | A plugin (§10), or copy the pattern |
| Mutate a variable | `x = x + 1`, `x += 1` | Re-bind via `bind NAME / from EXPR` |
| Loop with a counter | `for i in range`, `while` | `for_each_row(r) of result` or SQL itself |
| String-interpolate into SQL | `{{request.tbl}}` as a table name | `{{identifier request.tbl}}` (validated) |
| Read an env var at request time | `env.FOO` | Declare it as a server `{{secret …}}` (boot-time) |
| Build a response from a literal Go template syntax | `{{.User.Name}}` dot-form | Declare an input + use `{{name}}` |

When you reach for any of these, stop and re-read §2.

---

## §4. The six declaration kinds

Each declaration is a top-level block. They can appear in any order;
the loader resolves references at the end. Repeat any kind as many
times as needed.

### 4.1 `storage`

```capy
storage main
    kind     sqlite
    location "./app.db"

storage analytics
    kind     postgres
    location "{{secret analytics_dsn}}"
    pool_max 20

storage legacy
    kind     mysql
    location "user:pass@tcp(host:3306)/db"
```

Routes refer to one with `on main do sql`. Multiple coexist freely.

### 4.2 `connection`

```capy
connection events
    kind           server_sent_events
    subscribe_path "/events"
    buffer_size    256

connection cursors
    kind           websocket
    subscribe_path "/ws/cursors"
    buffer_size    256

connection notify
    kind           server_sent_events
    subscribe_path "/notify"
    buffer_size    64
    partition_by   auth.user.id     # per-user channel
```

`subscribe_path` is auto-registered as a GET route — **do not declare
a route for it yourself**.

### 4.3 `plugin`

```capy
plugin worker                              # one-shot subprocess
    kind    subprocess
    command "python3" "worker.py"

plugin llm                                 # persistent process
    kind    long_lived_subprocess
    command "./llm_server"
    environment
        MODEL_PATH "./model.bin"

plugin slack                               # remote HTTP
    kind         http
    endpoint_url "https://hooks.slack.com/services/T0/B0/XXX"
    headers
        Content-Type "application/json"
```

Routes invoke with `on <name> do call_plugin / trigger "..." /
pass_input NAME / from EXPR`.

### 4.4 `authentication`

```capy
authentication user                       # session cookie
    kind             session_cookie
    cookie_name      "session"
    session_lifetime 24h

authentication api                        # API key in header
    kind        header_lookup
    header_name "X-Api-Key"
    validation_sql `
        SELECT user_id FROM api_keys WHERE key = {{request.key}} LIMIT 1
    `

authentication google                     # OAuth
    kind          oauth
    provider      google
    client_id     "{{secret google_client_id}}"
    client_secret "{{secret google_client_secret}}"
    callback_path "/auth/google/callback"
    scopes        "openid", "email", "profile"
```

Routes opt in with `requires_authentication user`. See §11.

### 4.5 `migration`

```capy
migration "001_initial" on main
    description "users + index"
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
```

Run automatically at boot in declaration order. Full reference in
[`wave-capy-migrations.md`](../../docs/design/wave-capy-migrations.md);
summarised in §9.

### 4.6 Triggers — `route`, `every`, `at`

Triggers are what causes the `do` block to execute.

```capy
route "/items"                           # HTTP request
    methods GET
    request
    do
        ...

every 5s as poll_prices                  # interval
    on prices_api do call_plugin
        trigger "get"
    ...

at "04:30" as nightly_cleanup            # daily at local time
    on main do sql `DELETE FROM sessions WHERE expires < {{now}}`
```

All three accept the same operations and `match` vocabulary. The only
difference is **when** they fire.

---

## §5. Building blocks

This section is what you actually write inside a route. Read every
subsection at least once.

### 5.1 Route skeleton

```capy
route "/path/with/{param}"
    methods POST                          # required — one or more
    [requires_authentication SCHEME]
    [rate_limit / per "1m" / max 60]
    [timeout 5s]

    request                               # optional inputs
        [content_type "MIME"]             # required iff body input declared
        [path_parameter ...]
        [query_parameter ...]
        [header ...]
        [cookie ...]
        [body_field ...]                  # JSON / form / multipart fields
        [body NAME / type json_array|json_object]
        [body_file NAME]                  # multipart only
        [body_raw NAME]                   # any MIME, raw bytes

    do                                    # required — the handler
        # statements, operations, matches, responses

    [handle]                              # optional — override failure cases
```

### 5.2 Every input source

| Source | When to use | Reference as |
|---|---|---|
| `path_parameter NAME` | URL segment matching `{NAME}` | `request.NAME` |
| `query_parameter NAME` | URL `?NAME=…` | `request.NAME` |
| `header NAME` | HTTP request header | `request.name_snake_cased` |
| `cookie NAME` | Cookie value | `request.NAME` |
| `body_field NAME` | JSON property / form field / multipart text field | `request.NAME` |
| `body NAME / type json_array \| json_object` | Whole body as one structured value | `request.NAME` |
| `body_file NAME` | Multipart file part | `request.NAME.bytes` / `.name` / `.content_type` |
| `body_raw NAME` | Raw bytes (signed webhooks) | `request.NAME` |

**Types**: `text`, `integer`, `decimal`, `true_or_false`, `email`,
`uuid`, `file`, `bytes`, `json_array`, `json_object`.

**Modifiers**: `required`, `minimum`/`maximum` (numeric),
`min_length`/`max_length` (text), `pattern` (regex), `max_bytes`
(file/raw), `default VALUE`.

A failed validator → `400` + `case invalid_input(field, reason)`.
Override in `handle`.

### 5.3 Templates — SQL & response, the two rules

**SQL templates** (inside `\`...\`` after `do sql`):

Every `{{ }}` interpolation binds as a `?` parameter. Helpers you'll
use most:

| Helper | Binds |
|---|---|
| `{{request.x}}` | a declared input |
| `{{auth.user.id}}` | authenticated principal |
| `{{now}}` / `{{add_days N}}` / `{{add_hours N}}` / `{{add_minutes N}}` | timestamp |
| `{{like_contains x}}` / `{{like_prefix x}}` / `{{like_suffix x}}` | LIKE pattern (wildcards in the bound value) |
| `{{new_token}}` / `{{new_uuid}}` | random token / UUID |
| `{{hash_password x}}` | argon2id hash |
| `{{secret "k"}}` | configured secret |
| `{{client_ip}}` | request client IP |
| `{{identifier x}}` | validated identifier (the ONLY value→skeleton path) |
| `{{if has_value "x"}}…{{end}}` | conditional clause when input present |
| `{{each "x" as row, i}}…{{end}}` | iterate a `json_array` input |

**Response templates** (inside `body \`...\``):

Bindings come from named values (route inputs, prior operation
results, `case` bindings). Auto-escape per `content_type`.

| Helper | What it does |
|---|---|
| `{{to_json x}}` | JSON-encode a row, list, or map |
| `{{raw x}}` | bypass auto-escape (opt-in only) |
| `{{join "," xs}}` | join a list of strings |
| `{{format_time "layout" t}}` | format a timestamp |
| `{{escape x}}` | force-escape for the current content type |

The full helper catalogue lives in
[`wave-capy-templating.md`](../../docs/design/wave-capy-templating.md).
Look there when you need anything beyond the table above.

### 5.4 Operations

Inside `do`, every action is `[NAME =] on TARGET do OPERATION`.

| Operation | Form |
|---|---|
| SQL | `result = on STORAGE do sql \`…\`` (optionally `bind KEY / from PATH`) |
| Plugin call | `out = on PLUGIN do call_plugin / trigger "T" [/ streaming true] / pass_input KEY / from PATH` |
| Broadcast | `on CONNECTION do broadcast / event_name "E" / data \`…\` [/ partition_value EXPR]` |
| Forward / proxy | `on PROXY do forward / base_url "…" / preserve_path true` |
| Background block | `task = background / …` |
| Iterate rows | `for_each_row(NAME) of QUERY_RESULT / …` |
| Iterate emissions | `for_each_emission(NAME) of PLUGIN_STREAM / …` |
| Auth: login | `match on AUTH do authenticate_with_password / using FIELD / from request.x` |
| Auth: end session | `on AUTH do end_session` |
| Auth: magic-link issue | `on AUTH do send_magic_link / using email / from request.email` |
| Auth: magic-link consume | `match on AUTH do consume_magic_link / using token / from request.token` |
| Auth: OAuth begin | `on AUTH do begin_oauth_flow` |
| Auth: OAuth complete | `match on AUTH do complete_oauth_flow / using code / from request.code / using state / from request.state` |

### 5.5 Match & pipelines

Outcomes are pattern-matched. Names in `case`'s parens **bind for the
duration of the case body**.

```capy
match result
    case error(err)
        # operation failed; err is the message
    case empty
        # SELECT 0 rows / DML 0 rows affected / plugin empty
    case found(data)
        # SELECT returned 1+ rows
        # LIMIT 1 → data is a single map (use data.field)
        # otherwise → data is a list (use {{to_json data}})
    case success(info)
        # any non-error outcome
        # info.last_insert_id, info.rows_affected
```

`match` also works on **values**:

```capy
match auth.user.roles
    case contains("admin")
        # ...
    case _
        # default
```

Available value matchers: `contains("x")`, `equals(y)`,
`matches("regex")`, integer literals, `_` (wildcard).

**Pipelines** are nested `match` blocks. Inner operations only run
inside the `case` branch above them — that's how dependency on the
outer result is expressed:

```capy
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

**`bind KEY / from EXPR`** is how an inner SQL reads a value from the
enclosing `case` binding. Always prefer `bind` over relying on
implicit scope — it makes the dependency visible.

### 5.6 Responses, headers, cookies

```capy
response
    status       200                                  # required
    content_type "application/json"                   # required
    body         `{{to_json data}}`                   # body string OR body_bytes

    # Optional modifiers (any combination):
    set_header X-Total-Count
        value count
    set_cookie session
        value     session_token
        lifetime  24h
        http_only true
        secure    true
        same_site lax
    clear_cookie csrf
```

Binary download:

```capy
response
    status                         200
    body_bytes                     file.blob
    content_type_from_extension_of file.name
    set_header Content-Disposition
        value `attachment; filename="{{file.name}}"`
```

Redirect:

```capy
response
    status       302
    content_type "text/plain; charset=utf-8"
    body         ``
    set_header Location
        value "/dashboard"
```

204 / 304:

```capy
response
    status       204
    content_type "application/json"
    body         ``
```

### 5.7 Failure cases & `handle`

Every pre-handler stage emits a default JSON response on failure.
Override per route:

```capy
route "/items"
    methods POST
    ...
    handle
        case invalid_input(field, reason)
            response
                status       422
                content_type "application/json"
                body         `{"error":"invalid","field":"{{field}}","reason":"{{reason}}"}`
```

The 11 cases (all default to JSON; override as needed):

| Case | When | Default status |
|---|---|---|
| `method_not_allowed(allowed)` | wrong method | 405 |
| `wrong_content_type(expected, got)` | body MIME mismatch | 415 |
| `body_too_large(max_bytes)` | body exceeds `max_body_bytes` | 413 |
| `invalid_body(detail)` | body parse failed | 400 |
| `invalid_input(field, reason)` | input validator failed | 400 |
| `unauthenticated` | auth required, none given | 401 |
| `invalid_credentials` | password / token wrong | 401 |
| `forbidden(reason)` | RBAC predicate failed | 403 |
| `rate_limited(retry_after)` | over the limit | 429 |
| `timeout(elapsed)` | exceeded route `timeout` | 504 |
| `server_error(err)` | unhandled / panic / DB error | 500 |

Server-wide defaults go in a top-level `defaults / handle` block (see
[`wave-capy-complete.md` §5](../../docs/design/wave-capy-complete.md)).

---

## §6. Walkthrough 1 — hello → todo API

Goal: a JSON API for a todo list. Storage, validation, CRUD, search.
Time: 45 minutes of typing.

### Step 1 — empty server

`server.capy`:

```capy
route "/healthz"
    methods GET, HEAD
    request
    do
        response
            status       200
            content_type "text/plain; charset=utf-8"
            body         `ok`
```

Run: `wave serve server.capy --watch`. Hit `http://localhost:8080/healthz`.

### Step 2 — add storage + first migration

```capy
storage main
    kind     sqlite
    location "./todos.db"

migration "001_initial" on main
    up `
        CREATE TABLE todos (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            title      TEXT NOT NULL,
            done       INTEGER NOT NULL DEFAULT 0,
            created_at TEXT NOT NULL
        );
        CREATE INDEX todos_done ON todos(done);
    `
    down `
        DROP INDEX IF EXISTS todos_done;
        DROP TABLE  IF EXISTS todos;
    `

route "/healthz"
    methods GET, HEAD
    request
    do
        response
            status       200
            content_type "text/plain; charset=utf-8"
            body         `ok`
```

Restart (or let `--watch` pick it up). Migration runs at boot; the
table appears in `./todos.db`.

### Step 3 — list and create

```capy
route "/todos"
    methods GET
    request
        query_parameter q
            type       text
            max_length 100
        query_parameter limit
            type    integer
            minimum 1
            maximum 200
            default 50
    do
        result = on main do sql `
            SELECT id, title, done, created_at FROM todos
             WHERE 1=1
            {{if has_value "q"}} AND title LIKE {{like_contains request.q}} {{end}}
             ORDER BY id DESC
             LIMIT {{request.limit}}
        `
        match result
            case success(rows)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json rows}}`

route "/todos"
    methods POST
    request
        content_type "application/json"
        body_field title
            type       text
            required   true
            min_length 1
            max_length 200
    do
        result = on main do sql `
            INSERT INTO todos (title, done, created_at) VALUES ({{request.title}}, 0, {{now}})
        `
        match result
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}}}`
```

`curl localhost:8080/todos -d '{"title":"buy milk"}' -H 'content-type: application/json'`
→ `{"id":1}`.

`curl localhost:8080/todos?q=milk` → `[{"id":1,"title":"buy milk","done":0,"created_at":"…"}]`.

### Step 4 — get, update, delete

```capy
route "/todos/{id}"
    methods GET
    request
        path_parameter id
            type     integer
            required true
    do
        lookup = on main do sql `
            SELECT id, title, done, created_at FROM todos WHERE id = {{request.id}} LIMIT 1
        `
        match lookup
            case empty
                response
                    status       404
                    content_type "application/json"
                    body         `{"error":"not found"}`
            case found(t)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json t}}`

route "/todos/{id}"
    methods PATCH
    request
        path_parameter id
            type     integer
            required true
        content_type "application/json"
        body_field done
            type     true_or_false
            required true
    do
        result = on main do sql `UPDATE todos SET done = {{request.done}} WHERE id = {{request.id}}`
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

route "/todos/{id}"
    methods DELETE
    request
        path_parameter id
            type     integer
            required true
    do
        result = on main do sql `DELETE FROM todos WHERE id = {{request.id}}`
        match result
            case success(info)
                response
                    status       204
                    content_type "application/json"
                    body         ``
```

### Step 5 — verify

```bash
wave check server.capy             # parse + validate
wave test  server.test.capy        # write a test file (next subsection)
```

Optional `server.test.capy`:

```capy
import "server.capy"

test "list empty"
    request GET "/todos"
    expect
        status 200
        json   []

test "create returns id"
    request POST "/todos"
        content_type "application/json"
        json { "title": "ship it" }
    expect
        status 201
        json   { "id": "*" }
    capture id from json.id

test "get by id"
    request GET "/todos/{{id}}"
    expect
        status 200
        json   { "id": id, "title": "ship it", "done": 0 }

test "404 on missing"
    request GET "/todos/999999"
    expect
        status 404
        json   { "error": "not found" }
```

That's a complete REST resource in ~120 lines of capy.

---

## §7. Walkthrough 2 — auth + per-user data

Goal: add password auth, per-user todos. Time: 60 minutes.

### Step 1 — add users + auth

Append to the previous file:

```capy
migration "002_users" on main
    up `
        CREATE TABLE users (
            id            INTEGER PRIMARY KEY AUTOINCREMENT,
            email         TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at    TEXT NOT NULL
        );
        ALTER TABLE todos ADD COLUMN owner_id INTEGER REFERENCES users(id);
        CREATE INDEX todos_owner ON todos(owner_id);
    `
    down `
        DROP INDEX IF EXISTS todos_owner;
        ALTER TABLE todos DROP COLUMN owner_id;
        DROP TABLE  IF EXISTS users;
    `

authentication user
    kind             session_cookie
    cookie_name      "session"
    session_lifetime 7d
    same_site        lax
    secure           true
    http_only        true
```

### Step 2 — signup + login + logout

```capy
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
        result = on main do sql `
            INSERT INTO users (email, password_hash, created_at)
            VALUES ({{request.email}}, {{hash_password request.password}}, {{now}})
        `
        match result
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
                        lifetime  7d
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
```

### Step 3 — gate the existing routes; scope by owner

Replace the four todo routes:

```capy
route "/todos"
    methods GET
    requires_authentication user
    request
        query_parameter q
            type       text
            max_length 100
        query_parameter limit
            type    integer
            minimum 1
            maximum 200
            default 50
    do
        result = on main do sql `
            SELECT id, title, done, created_at FROM todos
             WHERE owner_id = {{auth.user.id}}
            {{if has_value "q"}} AND title LIKE {{like_contains request.q}} {{end}}
             ORDER BY id DESC
             LIMIT {{request.limit}}
        `
        match result
            case success(rows)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json rows}}`

route "/todos"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field title
            type       text
            required   true
            min_length 1
            max_length 200
    do
        result = on main do sql `
            INSERT INTO todos (title, done, created_at, owner_id)
            VALUES ({{request.title}}, 0, {{now}}, {{auth.user.id}})
        `
        match result
            case success(info)
                response
                    status       201
                    content_type "application/json"
                    body         `{"id":{{info.last_insert_id}}}`

route "/todos/{id}"
    methods GET
    requires_authentication user
    request
        path_parameter id
            type     integer
            required true
    do
        # 404 hides "exists but not yours" — that's the right behavior
        lookup = on main do sql `
            SELECT id, title, done, created_at FROM todos
             WHERE id = {{request.id}} AND owner_id = {{auth.user.id}}
             LIMIT 1
        `
        match lookup
            case empty
                response
                    status       404
                    content_type "application/json"
                    body         `{"error":"not found"}`
            case found(t)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json t}}`

route "/todos/{id}"
    methods PATCH
    requires_authentication user
    request
        path_parameter id
            type     integer
            required true
        content_type "application/json"
        body_field done
            type     true_or_false
            required true
    do
        result = on main do sql `
            UPDATE todos SET done = {{request.done}}
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
                            status       200
                            content_type "application/json"
                            body         `{"updated":{{info.rows_affected}}}`

route "/todos/{id}"
    methods DELETE
    requires_authentication user
    request
        path_parameter id
            type     integer
            required true
    do
        result = on main do sql `
            DELETE FROM todos WHERE id = {{request.id}} AND owner_id = {{auth.user.id}}
        `
        match result
            case success(info)
                response
                    status       204
                    content_type "application/json"
                    body         ``
```

**Key pattern**: combine the auth check with the SQL predicate. A
missing row and a non-owned row are indistinguishable → 404. Don't
split them — that leaks information.

### Step 4 — verify

```bash
wave check server.capy             # migrations + routes parse + validate
wave migrate status                # see what would run
wave serve server.capy --watch
```

```bash
curl -X POST localhost:8080/signup \
  -d '{"email":"a@b.c","password":"hunter22"}' \
  -H 'content-type: application/json'
# {"id":1}

curl -c jar -X POST localhost:8080/login \
  -d '{"email":"a@b.c","password":"hunter22"}' \
  -H 'content-type: application/json'
# {"ok":true}

curl -b jar -X POST localhost:8080/todos \
  -d '{"title":"private todo"}' \
  -H 'content-type: application/json'
# {"id":1}

curl localhost:8080/todos                       # no cookie
# {"error":"authentication_required"}  (401)
```

---

## §8. Walkthrough 3 — realtime + background + plugin

Goal: a chat-with-LLM app. User sends a prompt; we hand it to a
plugin that streams chunks; chunks fan out over SSE per-user; the
full transcript persists. Time: 90 minutes.

### Step 1 — declarations

```capy
storage main
    kind     sqlite
    location "./chat.db"

connection chat_feed
    kind           server_sent_events
    subscribe_path "/chat/stream"
    buffer_size    64
    partition_by   auth.user.id

plugin llm
    kind        long_lived_subprocess
    command     "node" "./plugins/llm/server.js"
    environment
        OPENAI_API_KEY "{{secret openai_api_key}}"

authentication user
    kind             session_cookie
    cookie_name      "session"
    session_lifetime 7d

migration "001_chat" on main
    up `
        CREATE TABLE users (
            id            INTEGER PRIMARY KEY AUTOINCREMENT,
            email         TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at    TEXT NOT NULL
        );
        CREATE TABLE chat_log (
            id      INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL REFERENCES users(id),
            role    TEXT NOT NULL,
            content TEXT NOT NULL,
            at      TEXT NOT NULL
        );
        CREATE INDEX chat_log_user_at ON chat_log(user_id, at DESC);
    `
    down `
        DROP INDEX IF EXISTS chat_log_user_at;
        DROP TABLE  IF EXISTS chat_log;
        DROP TABLE  IF EXISTS users;
    `
```

### Step 2 — login (reuse from §7)

Add `/signup`, `/login`, `/logout` from Walkthrough 2 verbatim.

### Step 3 — the chat route

```capy
route "/chat"
    methods POST
    requires_authentication user
    request
        content_type "application/json"
        body_field prompt
            type       text
            required   true
            min_length 1
            max_length 4000
    do
        # Persist the user message synchronously so it's in the log
        # even if the streaming plugin fails partway through.
        on main do sql `
            INSERT INTO chat_log (user_id, role, content, at)
            VALUES ({{auth.user.id}}, 'user', {{request.prompt}}, {{now}})
        `

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

            on chat_feed do broadcast
                event_name      "done"
                data            `{}`
                partition_value auth.user.id

        response
            status       202
            content_type "application/json"
            body         `{"task_id":"{{task.id}}"}`
```

### Step 4 — the client

The client opens `EventSource("/chat/stream")` (which requires the
session cookie because `partition_by auth.user.id` does). Each
`message` it receives is `{role: "assistant", content: "...partial..."}`.

`POST /chat` returns 202 immediately with a `task_id`. The plugin
streams chunks; each one is broadcast and persisted in the same loop.

### Step 5 — verify

`wave check server.capy` and `wave test server.test.capy`. For a
quick demo without writing a frontend, use `curl` for the SSE stream
in one terminal and the POST in another:

```bash
# Terminal 1 — subscribe
curl -b jar -N localhost:8080/chat/stream

# Terminal 2 — send
curl -b jar -X POST localhost:8080/chat \
  -d '{"prompt":"hello"}' \
  -H 'content-type: application/json'
# {"task_id":"…"}
# Watch chunks arrive in terminal 1
```

The plugin contract for the `llm` subprocess is the
[long-lived plugin protocol](../../docs-site/reference/plugin-contract.md);
JSON-RPC LSP-framed over stdio. The plugin's job is to read the
`chat` trigger, call OpenAI / Anthropic / Ollama, and emit one
`{content: "..."}` frame per token chunk.

---

## §9. Schema migrations

Migrations are first-class. Declare each as `migration ID on STORAGE`
with `up` (required) and `down` (recommended). They apply at boot in
declaration order, are tracked in a wave-owned `__wave_migrations`
ledger per storage, and refuse to boot on checksum drift.

```capy
migration "001_initial" on main
    description "users + index"
    up `
        CREATE TABLE users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            email TEXT UNIQUE NOT NULL
        );
    `
    down `DROP TABLE users`

migration "002_concurrent_index" on analytics
    description "build index without locking"
    transactional false                     # required for CONCURRENTLY
    up   `CREATE INDEX CONCURRENTLY events_type ON events(event_type)`
    down `DROP INDEX CONCURRENTLY IF EXISTS events_type`

migration "099_seed_dev_admin" on main
    tag dev, staging                        # never runs in prod
    up `
        INSERT OR IGNORE INTO users (email) VALUES ('admin@local')
    `
```

CLI:

```bash
wave migrate status     # what's applied / pending
wave migrate up         # apply pending without booting the server
wave migrate down       # roll back the most recent
wave migrate new "name" # scaffold the next sequential id
wave migrate verify     # re-checksum all applied; fail on drift
```

Rules:
- **Never edit a committed migration.** Add a new one.
- **Always provide `down`** unless genuinely one-way (seeds).
- **Long backfills go in a scheduled job**, not a migration — don't
  block boot on a million-row UPDATE.

Full reference:
[`wave-capy-migrations.md`](../../docs/design/wave-capy-migrations.md).

---

## §10. Plugins

When capy can't express it — talking to a third-party API, running
ML inference, parsing PDFs, streaming LLM tokens — write a plugin.

### When to plugin

| You need | Use |
|---|---|
| Any HTTP API call from a route | `plugin / kind http` |
| A persistent worker holding state (model, connection pool) | `plugin / kind long_lived_subprocess` |
| One-shot CLI invocations (ffmpeg, image-magick) | `plugin / kind subprocess` |
| Custom storage backend (Clickhouse, DynamoDB) | storage-kind plugin |
| Custom auth flow (SAML, custom SSO) | auth-kind plugin |
| Custom secret resolver (Vault, AWS SM) | secrets-kind plugin |
| Custom metrics/log exporter (Datadog, Honeycomb) | exporter-kind plugin |

### The contract

Wave sends JSON in, gets JSON out:

```jsonc
// In
{
  "trigger_key": "chat",
  "metadata":    {"remote_ip":"...", "request_id":"..."},
  "headers":     {...},
  "query":       {...},
  "body":        {/* arbitrary JSON */}
}

// Out
{ "status": 200, "headers": {...}, "body": {/* anything */} }
```

Three transports speak the same envelope:

- **`subprocess`** — one process per call. JSON on stdin, JSON on
  stdout. Simplest; right for short ops.
- **`long_lived_subprocess`** — one process for the server's lifetime.
  JSON-RPC 2.0 LSP-framed over stdio. Right for boot-cost-heavy
  workers and streaming.
- **`http`** — POST to `<endpoint>/<method>`. Right when the plugin
  already runs as a service.

### Walkthrough — Go plugin in 30 lines

```go
package main

import (
    "encoding/json"
    "io"
    "os"
)

type Req struct {
    Trigger string                 `json:"trigger_key"`
    Body    map[string]any         `json:"body"`
}
type Resp struct {
    Status int            `json:"status"`
    Body   map[string]any `json:"body"`
}

func main() {
    raw, _ := io.ReadAll(os.Stdin)
    var req Req
    _ = json.Unmarshal(raw, &req)

    body := map[string]any{
        "echo":       req.Body,
        "trigger":    req.Trigger,
    }
    out, _ := json.Marshal(Resp{Status: 200, Body: body})
    os.Stdout.Write(out)
}
```

`go build -o echo ./...` then declare:

```capy
plugin echo
    kind    subprocess
    command "./echo"
```

Use:

```capy
route "/echo"
    methods POST
    request
        content_type "application/json"
        body_field message
            type text
    do
        result = on echo do call_plugin
            trigger "echo"
            pass_input message
                from request.message
        match result
            case success(data)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json data}}`
```

Full plugin protocol with framing details, all kinds, and the
JSON-RPC envelope for long-lived plugins:
[`docs-site/reference/plugin-contract.md`](../reference/plugin-contract.md).

Multi-language examples (Python, Node, Rust, Bash):
[`docs-site/cookbook/build-plugin.md`](../cookbook/build-plugin.md).

---

## §11. Auth — pick by use case

| Use case | Scheme |
|---|---|
| Standard web app (browser, cookies) | `session_cookie` |
| Email-only sign-in (no passwords) | `magic_link_email` |
| "Sign in with Google" / Workspace gated | `oauth / provider google` |
| GitHub / Apple / generic OIDC | `oauth / provider oidc` |
| API consumers (bots, mobile, CLI) | `jwt` |
| Internal services with API keys | `header_lookup` (validated against a SQL table) |
| Multi-factor with TOTP | combine `session_cookie` + TOTP routes (see [magic-link-plus-totp demo](../../examples/apps/magic-link-plus-totp)) |
| SAML / custom SSO | auth-kind plugin |

Each scheme exposes the same `auth.user` shape (`id`, `email`,
`roles`, `claims.X`) in routes. Mix and match — a route can require
any scheme; different routes can require different schemes.

### Auth-failure response shape (`redirect_on_failure`)

When auth fails, Wave returns either a `302` redirect to
`redirect_on_failure` or a JSON `401`. The rule:

> **302 redirect** iff the request method is `GET` **and** the
> `Accept` header contains `text/html`. Anything else → JSON `401`
> with `{success:false, error:..., code:"unauthorized"}`.

This is the post-fix semantic. Practical effects an agent should
internalise:

- Top-level browser navigation to a protected page → 302 to login.
- SPA `fetch('/api/...')` (any method, `Accept: application/json` or
  `*/*`) → JSON 401. Handle it client-side; don't expect a redirect.
- Browser `POST`/`PUT`/`DELETE` → JSON 401 even with
  `Accept: text/html`. A 302 here would strand the form payload.
- `curl` / API consumers → JSON 401.
- No `redirect_on_failure` configured → JSON 401 always.

If you need a redirect for a non-GET-html request (rare), write an
explicit route that handles `case unauthenticated` and emits the
redirect yourself. Don't rely on `redirect_on_failure` to do it.

Per-scheme config:
[`wave-capy-complete.md` §4](../../docs/design/wave-capy-complete.md).

---

## §12. Observability, secrets, deployment

### Health

`GET /healthz` (liveness) and `GET /readyz` (readiness) are
auto-registered. Don't redeclare them.

### Metrics

`GET /metrics` exposes Prometheus exposition format. Counters and
histograms are emitted automatically: `wave_requests_total`,
`wave_request_duration_seconds`, `wave_storage_query_duration_seconds`,
`wave_plugin_call_duration_seconds`, `wave_outbox_pending`,
`wave_sse_subscribers`, etc.

### Traces & logs

```capy
observability
    otel
        endpoint     "{{secret otel_endpoint}}"
        service_name "my-app"
        sample_rate  0.1
    logs
        format "json"
        level  "info"
```

JSON logs include `request_id` (same value as the `X-Request-ID`
response header), method, path, status, duration, user, IP.

### Secrets

`{{secret name}}` in capy reads, in order:

1. `WAVE_SECRET_NAME` env var (uppercased).
2. A configured secrets plugin (e.g. Vault).
3. `secrets.env` file (dev only).

CLI: `wave secrets set NAME`, `wave secrets list`,
`wave secrets generate NAME --length 64`.

### Deployment

Single binary + `server.capy` + persistent dir for SQLite. The
[official Docker image](../guide/deploy-docker) is distroless nonroot.
Fly, Railway, Render, or a plain VPS — all work. See:

- [`docs-site/guide/deploy-docker.md`](../guide/deploy-docker.md)
- [`docs-site/guide/deploy-fly.md`](../guide/deploy-fly.md)
- [`docs-site/guide/deploy-checklist.md`](../guide/deploy-checklist.md)

---

## §13. Testing

`wave test` runs capy test suites in-process against a real boot of
your config — no port conflicts, no Docker, no flaky ports.

```capy
# server.test.capy
import "server.capy"

# Override storage to in-memory for fast tests
storage main
    kind     sqlite
    location ":memory:"

test "list empty"
    request GET "/todos"
    expect
        status 200
        json   []

test "create + read back"
    request POST "/todos"
        content_type "application/json"
        json { "title": "x" }
    expect
        status 201
        json   { "id": "*" }
    capture id from json.id

    request GET "/todos/{{id}}"
    expect
        status 200
        json   { "id": id, "title": "x", "done": 0 }
```

Patterns:

- `"*"` in expected JSON = "any value of any type".
- `capture NAME from json.path` extracts a value to use in later
  tests.
- `expect status 4xx / 5xx` accepts a range.
- `expect json` is a partial match — extra keys in the response are
  allowed.
- `expect json_strict` is exact-match.
- `migrations_applied STORAGE, [IDS...]` asserts the migration ledger.

Run:

```bash
wave test server.test.capy             # all tests
wave test server.test.capy --only "create"   # by name substring
wave test server.test.capy --format json     # for CI
```

Full reference:
[`docs-site/cookbook/testing.md`](../cookbook/testing.md).

---

## §14. Common mistakes & how to detect them

### Mistake 1 — referencing an undeclared input

```capy
# WRONG
route "/users/{id}"
    methods GET
    request
        # forgot to declare id
    do
        result = on main do sql `SELECT * FROM users WHERE id = {{request.id}}`
```

**Detect**: `wave check` reports
`undeclared input "id" referenced in route "/users/{id}"`.

**Fix**: declare it.

```capy
request
    path_parameter id
        type     integer
        required true
```

### Mistake 2 — using session user as a request input

```capy
# WRONG — user_id is the logged-in user, not a request value
request
    body_field user_id
        type     integer
        required true
do
    result = on main do sql `... WHERE user_id = {{request.user_id}}`
```

The client could send any value. **Fix**: use `{{auth.user.id}}`
and `requires_authentication`.

```capy
route "/notes"
    methods GET
    requires_authentication user
    request
    do
        result = on main do sql `SELECT * FROM notes WHERE owner_id = {{auth.user.id}}`
```

### Mistake 3 — `case found(row)` on a multi-row query

```capy
# WRONG — no LIMIT 1, but body uses row.field
result = on main do sql `SELECT id, name FROM users ORDER BY id`
match result
    case found(row)
        body `{"name":"{{row.name}}"}`     # row is a list, not a map → undefined
```

**Detect**: response body renders `null` or errors.

**Fix**: either add `LIMIT 1` (it's a single-row query) or iterate /
`to_json` (it's a multi-row query).

```capy
result = on main do sql `SELECT id, name FROM users ORDER BY id`
match result
    case success(rows)
        body `{{to_json rows}}`
```

### Mistake 4 — missing `content_type`

```capy
# WRONG
request
    body_field title
        type text
        required true
# error: content_type required when body input declared
```

**Detect**: `wave check` reports
`content_type missing on request that reads a body`.

**Fix**: add `content_type "application/json"` (or the right MIME).

### Mistake 5 — `INSERT` then expecting `case found`

```capy
# WRONG
result = on main do sql `INSERT INTO items (name) VALUES ({{request.name}})`
match result
    case found(row)
        body `{"id":{{row.id}}}`     # INSERT returns no rows
```

**Detect**: response is empty.

**Fix**: use `case success(info)` and `info.last_insert_id`.

```capy
match result
    case success(info)
        body `{"id":{{info.last_insert_id}}}`
```

### Mistake 6 — declaring a route for a connection's `subscribe_path`

```capy
# WRONG — Wave already auto-registers this
connection events
    kind           server_sent_events
    subscribe_path "/events"

route "/events"
    methods GET
    ...
# error: route conflict at "/events"
```

**Fix**: just declare the `connection`. The subscribe route is free.

### Mistake 7 — editing a migration that's already applied

```capy
migration "001_initial" on main
    up `CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT)`
# Later: someone adds "NOT NULL" to email here
```

**Detect**: boot fails with
`migration_checksum_mismatch (001_initial: expected sha256:…, got sha256:…)`.

**Fix**: revert the edit and add a new migration that ALTERs the
column. Never silently rewrite history.

### Mistake 8 — mixing tabs and spaces

**Detect**: `wave check` reports a parse error with a column number.

**Fix**: configure your editor to use 4 spaces. `wave fmt` can fix
existing files.

---

## §15. Self-verification checklist

After every significant edit, run this list:

1. **`wave check server.capy`** — must succeed. If it fails, the
   error is precise. Read it.
2. **Every `{{request.x}}` in SQL has a matching input** in the
   route's `request` block (or is `{{auth.user.id}}`, `{{now}}`,
   etc., from the helper table).
3. **Every `response` block has `status`, `content_type`, and
   `body`** (or `body_bytes`).
4. **Every route that reads a body declares `content_type`** in its
   `request`.
5. **Every SQL `LIMIT 1` is matched by `case found(row)` access
   like `row.field`**; every non-LIMIT-1 SELECT uses
   `{{to_json rows}}` or iterates.
6. **`INSERT` / `UPDATE` / `DELETE` use `case success(info)`** and
   read `info.last_insert_id` / `info.rows_affected`.
7. **Per-resource ownership** combines the auth check into the SQL
   predicate (one query, not two).
8. **Auth-gated routes have `requires_authentication`**, and the
   gated logic doesn't trust client-sent user ids.
9. **Schema changes are in a `migration` block**, not in a scheduled
   job or hand-edited DB.
10. **`wave test server.test.capy`** — must pass.

If all ten are green, you're production-ready for the slice of
behavior you've built.

---

## §16. Where to look next (every other doc, with purpose)

When you need depth past this handbook, look here. Each link includes
why you'd open it.

### Reference (canonical)

- [`wave-capy-complete.md`](../../docs/design/wave-capy-complete.md) —
  Every declaration shape, every field, every modifier in catalogue
  form. Look when you need a field you don't recognise.
- [`wave-capy-templating.md`](../../docs/design/wave-capy-templating.md) —
  Why SQL binding and response escaping work the way they do. Read
  when you're confused about a `{{ }}` not behaving as expected.
- [`wave-capy-route.md`](../../docs/design/wave-capy-route.md) — The
  precise route lifecycle and every failure case with timing.
- [`wave-capy-migrations.md`](../../docs/design/wave-capy-migrations.md) —
  Migration ledger, checksum guard, requires graph, CLI, round-trip
  testing. Read when you need anything beyond the basics in §9.
- [`docs-site/reference/plugin-contract.md`](../reference/plugin-contract.md) —
  The wire-format spec for every plugin kind (handler, storage,
  auth, secrets, exporter) over every transport (subprocess,
  longlived, http).
- [`docs-site/reference/features.md`](../reference/features.md) — The
  exhaustive feature inventory. Read when you're not sure if Wave
  *has* a feature.

### Practical

- [`wave-from-zero-to-complex.md`](../../docs/design/wave-from-zero-to-complex.md) —
  Human-facing end-to-end build-up. The same shape as this handbook
  but written for a developer, not an agent.
- [`wave-capy-cookbook.md`](../../docs/design/wave-capy-cookbook.md)
  & [`docs-site/cookbook/`](../cookbook/) — Recipe per task: JSON
  API, magic-link, OAuth, file uploads, multi-tenant, rate-limit,
  SSE, scheduled jobs, Stripe webhooks, etc. Look when you have a
  specific task in mind.
- [`wave-capy-patterns.md`](../../docs/design/wave-capy-patterns.md) —
  Larger composed patterns (event sourcing, sagas, idempotency
  envelopes).
- [`wave-capy-projects.md`](../../docs/design/wave-capy-projects.md) —
  25 worked apps, full source. Read when you want a full reference
  app to crib from.
- [`examples/apps/INDEX.md`](../../examples/apps/INDEX.md) — 64
  runnable demos. Each is a complete `server.capy` you can `wave
  serve`.

### Tooling

- [`wave-cli.md`](../../docs/design/wave-cli.md) — Every CLI
  subcommand (`serve`, `check`, `test`, `migrate`, `fmt`,
  `describe`, `export`, `new`, `doctor`, `secrets`).
- [`wave-capy-tooling.md`](../../docs/design/wave-capy-tooling.md) —
  Editor integration (LSP), `wave fmt` semantics, `wave doctor`
  diagnostics.
- [`wave-capy-describe.md`](../../docs/design/wave-capy-describe.md) —
  `wave describe` outputs (Markdown / JSON / OpenAPI / Postman).
- [`wave-export.md`](../../docs/design/wave-export.md) — `wave export`
  generates typed clients in TypeScript / Go / Python from your
  routes.

### Deployment & operations

- [`docs-site/guide/deploy-docker.md`](../guide/deploy-docker.md) —
  Official image and run commands.
- [`docs-site/guide/deploy-fly.md`](../guide/deploy-fly.md) — Fly.io
  walkthrough.
- [`docs-site/guide/deploy-checklist.md`](../guide/deploy-checklist.md) —
  Production sanity check.
- [`docs-site/guide/concepts-observability.md`](../guide/concepts-observability.md) —
  Metrics, traces, logs.
- [`docs-site/cookbook/testing.md`](../cookbook/testing.md) — Test
  patterns + CI integration.

---

## §17. Agent-specific tips

These are the patterns that get agents in trouble building Wave
backends, and how to avoid them.

### Tip 1 — write declarations first, routes second

Always start by listing storage, auth, plugins, connections,
migrations at the top of the file. Routes come last. This forces
you to think about what the route depends on before naming it.

### Tip 2 — read a worked demo before generating

If the task is "X", search `examples/apps/INDEX.md` for the closest
existing demo. Read its `server.capy` end-to-end (under 200 lines
typically). You'll catch subtleties (cookie names, status codes,
input modifiers) you wouldn't invent.

### Tip 3 — never reach for `{{.x}}` dot-form

If you find yourself writing `{{.Data.x}}` or `{{.User.Name}}`,
**stop**. That's old Go-template thinking. Capy templates only use
`{{name}}` where `name` is a declared input or `case` binding.

### Tip 4 — `LIMIT 1` is a signal, not a typo

`LIMIT 1` tells Wave the result is single-row → `case found(row)`
binds a map. Without it, the result is a list. Always be deliberate.

### Tip 5 — use `bind` over implicit scope in pipelines

```capy
# Less clear
match user_lookup
    case found(user)
        on main do sql `SELECT * FROM orders WHERE user_id = {{user.id}}`

# Clearer — declares the dependency
match user_lookup
    case found(user)
        on main do sql `SELECT * FROM orders WHERE user_id = {{uid}}`
            bind uid
                from user.id
```

Both work; `bind` is preferred when the dependency might be
non-obvious to a future reader.

### Tip 6 — auth-gated SQL combines, never splits

When ownership matters, write **one** SQL with both the lookup and
the ownership predicate, not a lookup followed by a separate auth
check. It's faster and it doesn't leak existence.

### Tip 7 — fail-fast on unrecognized status

If you can't decide the right status, default to the route lifecycle
table in §5.7. Don't invent. `404` for missing; `403` for forbidden;
`401` for unauthenticated; `422` for validation; `409` for conflict;
`429` for rate-limited; `5xx` for server-side problems.

### Tip 8 — when in doubt, run `wave check`

It is the cheapest source of truth. Run it after every block you
add. `wave check` understands the full grammar; you don't have to.

### Tip 9 — prefer a migration over CREATE TABLE IF NOT EXISTS

Capy has first-class migrations (§9). Use them. Don't hide schema
work inside a scheduled job — you lose ordering, drift detection,
and the round-trip story.

### Tip 10 — every output should pass through declared inputs

If you find yourself building a response with values that didn't
come from `request`, `auth`, an operation result (`case` binding),
or a helper (`{{now}}`, `{{client_ip}}`, `{{secret …}}`), pause.
You're probably about to leak something or hardcode something. The
discipline of "every value declared" pays off in audit-ability.

---

## Appendix — quick reference

### Operations one-liner table

```
[name =] on STORAGE     do sql `…`     [bind KEY / from PATH]
[name =] on PLUGIN      do call_plugin / trigger "T" / pass_input K / from PATH
         on CONNECTION  do broadcast   / event_name "E" / data `…` [/ partition_value EXPR]
         on PROXY       do forward     / base_url "…" / preserve_path true
         on AUTH        do authenticate_with_password / using FIELD / from request.x
         on AUTH        do end_session
         on AUTH        do send_magic_link / using email / from request.email
match …  on AUTH        do consume_magic_link / using token / from request.token
         on AUTH        do begin_oauth_flow
match …  on AUTH        do complete_oauth_flow / using code / from request.code / using state / from request.state
task   = background     / …
for_each_row(R)        of RESULT
for_each_emission(E)   of STREAM
```

### Case-binding reference

```
case error(err)         err  = error message string
case empty              (nothing)
case found(data)        data = single map (LIMIT 1) OR list of maps
case success(info)      info.last_insert_id, info.rows_affected
case contains("x")      (value match on list)
case equals(y)          (value match)
case matches("regex")   (value match on string)
case _                  default / wildcard
```

### Decision tree — which thing do I declare?

```
need to persist data?              → storage + migration
need to talk to anything else?     → plugin
need real-time push to client?     → connection
need to gate routes by identity?   → authentication
need work that isn't HTTP-triggered? → every / at
```

That's the model. Build accordingly.

---

*This handbook is the canonical agent-facing Wave guide. If anything
here contradicts what you observe at `wave check` time, **trust
`wave check`** and file an issue with the capy snippet that confused
you.*
