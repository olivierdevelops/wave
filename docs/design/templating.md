# Templating unification — design proposal

Status: **proposal / not yet implemented**

Wave today has at least four distinct templating dialects mixed
across YAML configs. This document captures the actual state,
explains why every divergence exists, and proposes a unified model
that preserves the safety properties while removing the cognitive
load.

This is a planning document. No code changes yet. Implementation
will land in phases (Phase 1 is additive, Phase 2 deprecates the
old forms, Phase 3 removes them at v1.0).

---

## 1. The actual current state

What you can write inside a single `server.yaml` today:

| Syntax | Example | Where it works | Evaluator | When |
|---|---|---|---|---|
| `${ENV:NAME}` | `secret: "${ENV:STRIPE_WEBHOOK_SECRET}"` | Any string value | `infra/secrets` | **Boot-time** |
| `${ENV:NAME:default}` | `port: "${ENV:PORT:8080}"` | Any string value | `infra/secrets` | Boot-time |
| `${FILE:/path}` | `key: "${FILE:/etc/wave/jwt.pem}"` | Any string value | `infra/secrets` | Boot-time |
| `${PLUGIN:name:uri}` | `dsn: "${PLUGIN:vault:secret/data/pg#dsn}"` | Any string value | `infra/secrets/plugin` | Boot-time |
| `{name}` | `path: /users/{id}` | Route `path:` and forward URLs | Go 1.22 ServeMux + `strings.ReplaceAll` | Request-time |
| `{{name}}` (function call) | `execute: "SELECT … WHERE id={{user_id}}"` | SQL `execute:` | `infra/sqlite` funcMap | Request-time, **binds to `?` (parameterised)** |
| `{{.name}}` (dot-form) | `output_template: '{"id":{{.Data.id}}}'` | `output_template` (storage-access, fetch, run-process), error templates | `infra/render.Render` | Request-time, **literal string interpolation** |
| `{{varname}}` (simple) | `body: '{"city":"{{city}}"}'` in `requests:` | HTTP client request bodies + URLs | `infra/httpclient.substituteVars` | Request-time, no functions |
| `{{call .X | .Y}}` | (Go template builtins — `call`, `range`, `if`, `with`) | Wherever `infra/render` is used | Go `text/template` | Request-time |
| (literal — no templating) | `content.body`, `redirect.redirect_url`, `headers` values | Several route types | none | n/a |

That's **at minimum five evaluators** with three different timings
and inconsistent helper sets.

---

## 2. Concrete inconsistencies

Found by exhaustive scan of `infra/render`, `infra/sqlite`,
`infra/httpclient`, `infra/secrets`, and every YAML field that
takes a templatable value.

### 2.1 Dot-form is required in some contexts, **forbidden** in others

The framework's biggest cognitive load:

```yaml
# SQL: function-form ONLY. {{.user_id}} would interpolate the value as a literal —
# SQL injection. CLAUDE.md says NEVER use dot-form here.
storage-access:
  execute: "SELECT * FROM users WHERE id = {{user_id}}"

# output_template: dot-form is the idiomatic way to access result fields.
# function-form {{Data}} also works (every map key becomes a zero-arg
# function), but the convention is dot-form.
storage-access:
  output_template: '{"id":{{.Data.id}}, "name":"{{.Data.name}}"}'
```

**Why it's like this:** SQL templates use the strict-scope DataLoader
as the template context — DataLoader has no exported fields, so
`{{.anything}}` evaluates to nil. This is a safety property by
*construction*: the only way to access a value is through the
function-form (`{{user_id}}`), which goes through the binding path
(`?` + params). Removing the dot-form from SQL is impossible because
Go's `text/template` always allows it; we rely on "there's nothing
to access".

For `output_template`, the renderer passes a real `map[string]any`,
so both forms work. The dot-form became idiomatic because users
need to access **nested** result fields (`.Data.id`, `.Data.rows.0.name`)
and the function-form can't traverse nesting.

### 2.2 Two SQL aliases for the same operation

```yaml
execute: "INSERT INTO t(x) VALUES ({{name}})"            # function-call form
execute: "INSERT INTO t(x) VALUES ({{value \"name\"}})"  # explicit alias
```

Both bind a `?`. Two ways to write the same thing.

### 2.3 Path variables: three syntaxes for "a value from the URL path"

```yaml
- path: /users/{id}                       # Go mux pattern — required by stdlib
  inputs: [{ name: id, source: path, type: int, required: true }]
  storage-access:
    execute: "SELECT * FROM users WHERE id = {{id}}"   # accessed via declared input

# vs forward routes — uses raw {name} substitution in the URL, no
# input declaration required (and no validation):
- path: /proxy/{tenant}/api
  type: forward
  forward:
    forward_url: "http://{tenant}.upstream.local/api"     # raw {name}, no escaping

# vs redirect routes — no templating at all:
- path: /old/{id}
  type: redirect
  redirect:
    redirect_url: "https://new.example.com/{id}"          # literal — {id} NOT substituted
```

Same field shape (a URL), three different behaviors. Forward URLs
do raw substitution with no escaping. Redirect URLs do nothing.

### 2.4 `content.body` is literal but `output_template` is templated

```yaml
# Static — no way to interpolate values
- path: /
  type: content
  content:
    status_code: 200
    body: "Hello, world"          # literal; can't say {{getUser}} or {{.id}}

# Templated — full Go template renderer
- path: /
  type: storage-access
  storage-access:
    output_template: "Hello, {{getUser}}"
```

Users who want a templated response from a non-SQL route have to
fall through to `storage-access` even when there's no SQL to run.

### 2.5 HTTP client uses its own incompatible `{{...}}` syntax

```yaml
requests:
  weather_api:
    url:  "https://api.example.com/{{city}}"
    body: '{"city":"{{city}}"}'
```

Looks like Go template syntax. **It isn't.** `infra/httpclient/execute.go`
has its own simpler `substituteVars` that does flat string
replacement only — no functions, no `if`, no `range`, no nested
access. If you try `{{toJSON .city}}` it silently fails (the
literal text `{{toJSON .city}}` lands in the URL).

### 2.6 Helpers are scattered across contexts with no overlap

| Helper | Available in SQL? | Available in output_template? | Available in HTTP client? |
|---|:---:|:---:|:---:|
| `{{name}}` (input bind) | ✅ | ✅ (renders to string) | ⚠️ different impl |
| `{{getCurrentTime}}` | ✅ | ❌ | ❌ |
| `{{wrap}}` | ✅ | ❌ | ❌ |
| `{{jsonArray}}` | ✅ | ❌ | ❌ |
| `{{hasvalue}}` | ✅ | ❌ | ❌ |
| `{{getUser}}` | ✅ | ❌ | ❌ |
| `{{toJSON}}` | ❌ | ✅ | ❌ |
| `{{urlPathEscape}}` | ❌ | ✅ | ❌ |
| `{{escape}}` | ❌ | ✅ | ❌ |

Same `{{...}}` syntax, completely different callable sets in
different contexts. There's no way to use `{{toJSON x}}` in SQL
or `{{getCurrentTime}}` in `output_template`.

### 2.7 Headers, status codes, and content-types are static

```yaml
content:
  status_code: 200
  headers:
    - ["Content-Type", "application/json"]
    - ["Cache-Control", "no-store"]
  body: "..."
```

The `status_code`, `headers`, and `Content-Type` are all
config-time literals. There's no way to:
- Set status based on a query result (`if row.is_premium then 200 else 402`)
- Set a header from a captured value (`X-User-Id: {{getUser}}`)
- Vary content-type by `Accept` header

Some route types (`storage-access` with `if_empty_status: 404`)
work around this for the single specific case of "empty result".

### 2.8 Timing is mixed in the same value

```yaml
auth:
  app:
    secret: "${ENV:JWT_SECRET}"    # boot-time
    cookie_max_age_seconds: 86400  # literal

routes:
  - path: /me
    auth: [app]
    type: storage-access
    storage-access:
      execute: "SELECT * FROM users WHERE id = {{getUser}}"   # request-time
```

Three different evaluation phases in one route. The user has to
remember which syntax means "now (at boot)" vs "later (per
request)". `${ENV:NAME}` and `{{name}}` look totally different,
which is actually good — but the mental model is still triple.

---

## 3. Why this happened (charitably)

Each evaluator was added when it solved a real problem the others
couldn't:

- **`${ENV:…}`** — needed at boot to resolve secrets *before* anything else parses the config
- **`{{name}}` in SQL** — needed parameter binding (safety-critical)
- **`{{.x}}` in output_template** — needed nested traversal of query results
- **HTTP client's `substituteVars`** — added before Go templates were imported there; never refactored
- **`{name}` in forward URLs** — added when forward routes needed dynamic targets; the implementer chose path-var syntax to match the route pattern

Each individual choice was reasonable. The cumulative result is
five languages.

---

## 4. Proposed unification

### 4.1 Principles

1. **One value-access syntax everywhere: `{{name}}`** (function-call form).
   - In SQL contexts: emits `?` + binds value as param. (Safety preserved.)
   - In every other context: renders the value to a string and interpolates.
2. **Dot-form (`{{.x.y}}`) is reserved for nested *result* traversal.**
   Declared inputs are always `{{name}}`. Result objects (`.data`,
   `.request`, `.response`) are accessed via dot-form. The two have
   non-overlapping namespaces — there is no `{{x}}` that's also
   accessible as `{{.x}}`.
3. **One base helper set, available in every templated context.**
   SQL adds binding semantics on top (the `{{name}}` function emits
   `?` instead of a string); everything else uses the same helpers.
4. **Templating is universal across string fields.**
   `content.body`, `redirect.redirect_url`, header values, status
   codes — anywhere a string is a config value, it can be templated.
   "Literal-only fields" are removed.
5. **`${...}` stays at boot, `{{...}}` runs at request.**
   This is the one good visual distinction we have. Keep it.
6. **Path variables become first-class declared inputs.**
   `{name}` in the route's `path:` is stdlib syntax (can't change),
   but accessing it anywhere else is `{{name}}` via an `inputs:`
   declaration. Forward URLs that today use raw `{name}` substitution
   migrate to `{{name}}`.
7. **One renderer, three modes.**
   - **render-mode** (default): interpolate values as strings; full helper set
   - **bind-mode** (SQL contexts): `{{name}}` emits `?` + binds; helpers
     that produce values (like `getCurrentTime`) bind their result;
     helpers that produce SQL fragments (like `jsonArray`) emit raw
   - **escape-mode** (HTML response bodies, error templates): values
     are HTML-escaped by default; opt out via `{{raw x}}`

### 4.2 Standardized data shape

Every templated context — `output_template`, `content.body`,
`redirect.redirect_url`, header values, request bodies, error
templates — gets the **same** template-context shape:

```go
// Available in every templated context:
{{<input_name>}}        // Every declared input — function-form, preferred
                        // In SQL: binds. Elsewhere: renders to string.

.inputs.<name>          // Same values via dot-form (for iteration / nested access)
.request.method         // "GET" / "POST" / …
.request.path           // Raw URL path
.request.query.<k>      // Query string values
.request.headers.<k>    // Request header values
.request.cookies.<k>    // Cookie values
.request.path_vars.<k>  // Path variables (same as declared path inputs)
.request.client_ip      // Best-effort IP
.request.id             // X-Request-ID

.user                   // Authenticated user, or nil
.user.id
.user.email
.user.roles
.user.claims.<k>

.data                   // Route-result data (route-type specific):
                        //   storage-access: query result rows / row
                        //   fetch / api:    { status, headers, json, text }
                        //   task:           { task_id }

.last_insert_id         // INSERT routes
.rows_affected          // UPDATE/DELETE routes
.column_names           // SELECT routes
```

Two access forms for the **same** values, by namespace:
- Declared inputs → `{{name}}` (function) preferred; `.inputs.name` for iteration
- Result data → `.data.field` (dot-form) — declared inputs never collide because they live in a separate namespace

This kills 2.1 (the "forbidden in SQL, required in output" tension):
in **every** context, declared inputs use `{{name}}` and nested
results use `{{.data.x}}`. The SQL safety property holds because
`{{name}}` always binds (never interpolates literally) in bind-mode.

### 4.3 Unified helper funcMap

One package — `infra/template` — owns the funcMap. Other packages
import it and add context-specific helpers (SQL adds nothing the
user calls explicitly; the binding semantics live in the
`{{name}}` evaluator).

**Universal helpers** (every context):

| Helper | What |
|---|---|
| `{{toJSON x}}` | JSON-encode any value |
| `{{toYAML x}}` | YAML-encode any value |
| `{{getCurrentTime}}` | UTC ISO-8601 timestamp |
| `{{getCurrentTimeLocal}}` | Local-tz ISO-8601 |
| `{{addDays N}}` | Now + N days |
| `{{formatTime "layout" t}}` | Go time format |
| `{{wrap "pattern" val}}` | Wrap with `%val%` etc. |
| `{{hasvalue "name"}}` | True if input present + non-empty |
| `{{hasvalues "a" "b" …}}` | AND |
| `{{hasanyvalue "a" "b" …}}` | OR |
| `{{iterlist "name"}}` | Iterate `type: array` input |
| `{{itermap "name"}}` | Iterate `type: object` input |
| `{{getindex "name" N}}` | Index into array input |
| `{{getUser}}` | Authenticated user id |
| `{{getClientIP}}` | Best-effort client IP |
| `{{escape s}}` | HTML escape |
| `{{urlPathEscape s}}` | URL path-segment escape |
| `{{urlQueryEscape s}}` | URL query-param escape |
| `{{error "msg"}}` | Abort with error |
| `{{default x defaultVal}}` | Fallback if x is empty |
| `{{coalesce x y z…}}` | First non-empty |

**Bind-mode-only** (SQL contexts only):

| Helper | What |
|---|---|
| `{{raw "name"}}` | Access raw input value (escape hatch — only inside other helpers) |
| `{{jsonArray (raw "name")}}` | SQL JSON literal for `json_each` |
| `{{bind expr}}` | Bind an in-template computed expression |

Removed (deprecated → removed at v1.0):
- `{{value "name"}}` — alias for `{{name}}`; just use `{{name}}`
- HTTP client's bespoke `substituteVars` — migrate to the unified renderer

### 4.4 Per-field migration table

| Field | Today | Tomorrow |
|---|---|---|
| `routes[].path` | Go mux `{name}` | unchanged — stdlib requirement |
| `storage-access.execute` | `{{name}}`, no `{{.x}}` | unchanged (already the design we want) |
| `storage-access.output_template` | `{{.Data.x}}` + helpers | `{{.data.x}}` (lowercase) + helpers; backward-compat alias for `{{.Data.x}}` |
| `content.body` | **literal** | **templated** (additive — no break) |
| `content.headers[].value` | literal | templated |
| `content.status_code` | literal int | accepts int OR `{{template}}` rendering to int |
| `redirect.redirect_url` | **literal** | **templated** (additive) |
| `forward.forward_url` | `{name}` (raw) | `{{name}}` (via inputs); `{name}` kept as deprecated alias |
| `forward.include_headers[].value` | literal | templated |
| `auth_login.error_template_str` | `infra/render` with limited helpers | unified renderer with full helper set |
| `magic-link-request.email_template` | `infra/render` | unified renderer |
| `requests.<name>.url` | HTTP client's `{{var}}` flat substitution | unified renderer |
| `requests.<name>.body` | HTTP client's `{{var}}` flat substitution | unified renderer |
| `requests.<name>.headers[].value` | literal | templated |
| `plugin.command[]` | literal | unchanged (process spawn, not templated) |
| `plugin.env.<k>` | `${ENV:…}` boot-time only | unchanged — boot-time substitution |

### 4.5 Specific fix for `type: content`

The user explicitly called this out. Today:

```yaml
- path: /
  type: content
  content:
    status_code: 200
    headers: [["Content-Type", "text/plain"]]
    body: "hello"           # literal
```

After Phase 1:

```yaml
- path: /
  type: content
  inputs: [{ name: name, source: query, type: string, default: "world" }]
  content:
    status_code: 200
    headers: [["Content-Type", "text/plain"]]
    body: "Hello, {{name}}!"     # templated — uses the same renderer as storage-access
```

Same template engine, same helpers, same data shape. Existing
`content.body` configs keep working because they have no `{{...}}`
markers; the renderer is a no-op for purely literal strings.

### 4.6 Migration phases

**Phase 1 — additive, zero breaking changes.**
- New package `infra/template` with the unified funcMap.
- Wire it into `content.body`, `redirect.redirect_url`,
  `forward.forward_url`, header values, error templates.
- Document the unified shape.
- All existing `{{.Data.x}}` configs keep working (renderer
  accepts both `Data` and `data`).
- All existing `{name}` forward URLs keep working.
- HTTP client gains the full funcMap; bespoke `substituteVars`
  becomes a thin wrapper.

**Phase 2 — deprecation warnings.**
- `wave validate` and `wave doctor` emit warnings on:
  - `{{value "name"}}` in SQL (use `{{name}}`)
  - `{name}` in forward URLs (use `{{name}}`)
  - `{{.Data.x}}` in output_template (use `{{.data.x}}`)
  - Any usage of HTTP client's old syntax that the unified
    renderer would handle differently
- CHANGELOG lists every deprecation with the rewrite.

**Phase 3 — removal at v1.0.**
- Remove the deprecated aliases.
- `wave fmt` auto-rewrites old → new during the migration window.

### 4.7 Implementation skeleton

```
infra/template/
  template.go          — exposed Render(), RenderInto(), Compile()
  funcmap.go           — the one base funcMap
  context.go           — the standardized data shape (.request, .data, .user, .inputs)
  bind.go              — bind-mode shim (for SQL contexts)
  bind_test.go
  funcmap_test.go
  template_test.go

infra/sqlite/
  sqlite.go            — uses infra/template.Compile with bind-mode opt-in
  (delete its private funcMap)

infra/render/
  (delete; migrate callers to infra/template)

infra/httpclient/
  execute.go           — uses infra/template with render-mode (drop substituteVars)

infra/secrets/
  (unchanged — boot-time interpolation is a separate concern)
```

Estimated work:
- Phase 1: ~3-5 days of code + tests (the new package is small; the
  wiring touches many call sites).
- Phase 2: ~1 day to add the lint passes to `validate` / `doctor`.
- Phase 3: ~1 day at v1.0 to delete deprecated code paths.

---

## 5. Open questions

These are real choices that need a decision before Phase 1 lands:

1. **Should `{{name}}` in render-mode look up `inputs.name` OR fall through to any helper of the same name?**
   Today the SQL funcMap shadows everything. If we let user inputs
   shadow built-in helpers (e.g., declaring `input: getCurrentTime`),
   we risk silent breakage. Proposal: built-in helpers win; warn on
   shadowing input names.

2. **Should `content.body` becoming templated be opt-in via a flag?**
   I'd say no — literal strings stay literal because they contain
   no `{{...}}` markers. The renderer is transparent for them. Any
   string containing `{{` already would have been broken or
   invalid HTML anyway.

3. **Do we keep `{{value "name"}}` as a deprecated alias forever, or remove at v1.0?**
   Proposal: remove at v1.0. The migration tool can rewrite all
   uses automatically.

4. **How do we handle `{{.Data.x}}` (capital D) vs `{{.data.x}}` (lowercase) during migration?**
   Proposal: renderer accepts both during Phases 1-2. `wave fmt`
   rewrites to lowercase. v1.0 removes the capital form.

5. **Should we add `{{coalesce}}` and `{{default}}` to the bind-mode funcMap?**
   In SQL they'd need careful handling — `COALESCE(?, ?)` requires
   binding both args. Proposal: yes, with bind-mode treating
   `{{coalesce a b}}` as `COALESCE(?, ?)` + binding both values.

6. **Templated status codes — useful or footgun?**
   `status_code: "{{if hasvalue \"premium\"}}200{{else}}402{{end}}"`
   is expressive but error-prone (renders to a string; needs
   int-parsing). Proposal: keep status codes as literal int by
   default; add a `status_template:` field for the rare case.

7. **HTML auto-escape in render-mode for `text/html` responses?**
   Today nothing auto-escapes. We could detect `Content-Type:
   text/html` and switch the renderer to use `html/template`. Risk:
   silent behavior change. Proposal: add an explicit
   `escape_html: true` flag on `content:` and `output_template:`;
   default false.

---

## 6. What this fixes — concretely

After all three phases:

- ✅ One value-access syntax: `{{name}}` everywhere
- ✅ One data-shape: `.request`, `.data`, `.user`, `.inputs` available in every templated context
- ✅ One helper set: every helper works in every context (with bind-mode adapting the value evaluators in SQL)
- ✅ `content.body` is templated like everything else
- ✅ `redirect.redirect_url` is templated
- ✅ Headers can be templated
- ✅ Forward URLs use `{{name}}` not `{name}`
- ✅ HTTP client's `requests:` uses the same renderer
- ✅ SQL safety preserved by construction (bind-mode evaluator emits `?`, never a literal)
- ✅ `${ENV:…}` and `{{…}}` stay visually distinct — boot vs request
- ✅ Five evaluators → one base evaluator + one bind-mode shim

What stays the same:

- `${ENV:…}` / `${FILE:…}` / `${PLUGIN:…}` — boot-time only
- `{name}` in `routes[].path:` — Go stdlib requirement
- The strict-scope DataLoader safety property in SQL
- The `infra/secrets` package
- The data shape for `storage-access` query results (lowercased namespace)

---

## 7. Decision needed

Before any code lands, we need user/maintainer sign-off on:

- The principle that **`{{name}}` works in every context** (function-call for declared inputs)
- The principle that **`.data.x` (dot-form) is reserved for nested result traversal only**
- The migration phasing (Phase 1 additive → Phase 2 warn → Phase 3 remove)
- The 7 open questions in section 5

Once those are settled, Phase 1 is straightforward to implement.
The audit + this document is the spec.
