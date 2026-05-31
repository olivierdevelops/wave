# Functional testing with `wave test`

Wave ships a built-in test runner. You write **capy test suites** that
boot your `server.capy` in-process (no port binding) and assert
request/response behavior. Same execution path as production —
including auth, validation, CORS, rate limits.

::: tip
This is functional / integration testing. For unit-testing your Go
plugins, use standard `go test`.
:::

## The shape

A `server.test.capy` file lives next to your `server.capy`. It imports
the config under test and declares a sequence of cases:

```capy
# server.test.capy
import "server.capy"

# Optional: secrets/env exported for the run, restored after.
env
    JWT_SECRET            "test-only-secret"
    STRIPE_WEBHOOK_SECRET "whsec_test"

# Optional: runs before tests; if any fails, tests are skipped.
setup "seed first user"
    request POST "/auth/request"
        json { "email": "ada@example.com" }
    expect
        status 200

# The actual assertions
test "built-in /healthz returns ok"
    request GET "/healthz"
    expect
        status 200
        body   `ok`

test "POST /items creates a row"
    request POST "/items"
        header Authorization "Bearer test-token"
        json { "name": "laptop", "price": 999 }
    expect
        status 201
        json   { "id": "*" }            # any present value
    capture created_id from json.id     # save for later cases

test "GET /items/{id} returns what we just created"
    request GET "/items/{{created_id}}"
    expect
        status 200
        json   { "name": "laptop", "price": 999 }

test "GET nonexistent returns framework 404 envelope"
    request GET "/items/9999"
    expect
        status 404
        json   { "error": "not found" }

# Optional: runs regardless of test outcome.
teardown "cleanup test user"
    request DELETE "/admin/users/ada@example.com"
        header Authorization "Bearer admin-test-token"
    expect
        status_in 200, 404               # any of these is fine
```

Run it:

```sh
wave test server.test.capy
#   PASS  built-in /healthz returns ok (200, 1ms)
#   PASS  POST /items creates a row (201, 3ms)
#   PASS  GET /items/{id} returns what we just created (200, 1ms)
#   PASS  GET nonexistent returns framework 404 envelope (404, 0s)
#
#   4 passed, 0 failed, 0.01s
```

By default, server boot logs + per-request access logs are silenced so
the report is clean. Add `--verbose` (or `-v`) to see them.

In CI:

```sh
wave test server.test.capy --format json   # machine-readable
echo $?                                     # 0 all-pass, 1 any failure, 2 bad invocation
```

The JSON envelope goes to stdout; stderr is reserved for real errors
(missing file, parse error, suite couldn't boot). So
`wave test … --format json 2>/dev/null | jq` is safe in pipelines.

## Request shape

Inside `request METHOD "PATH"`:

| Line | What |
|---|---|
| `request METHOD "/path"` | GET / POST / PUT / PATCH / DELETE / OPTIONS. Path supports `{{var}}`. |
| `header NAME "value"` | a request header (templated) |
| `query NAME "value"` | a query parameter (templated) |
| `body \`…\`` | a raw string body (templated) |
| `json { … }` | a JSON object — sets `Content-Type: application/json`, templated through every string leaf |
| `form { … }` | encoded as `application/x-www-form-urlencoded` |

Use only one of `body` / `json` / `form` per request.

## Expect shape

Inside `expect`:

| Line | Behavior |
|---|---|
| `status N` | exact HTTP status |
| `status_in A, B` | status is one of the listed codes |
| `body \`…\`` | exact body match (after trim) |
| `body_contains "…"` | substring check |
| `header NAME "value"` | exact header value (case-insensitive name) |
| `json { … }` | **strict subset** match against the response parsed as JSON |

### JSON subset semantics

`expect / json { … }` is matched against the response body parsed as
JSON.

- **Objects**: every key in `expect` must exist in the response with a
  matching value. Extra keys are fine.
- **Arrays**: same length, element-wise subset by index.
- **Scalars**: equal (numeric types coerce).
- **Wildcard**: the literal `"*"` matches any present value — use for
  "field exists, don't care about the value."

```capy
expect
    json {
        "id":   "*",
        "name": "ada",
        "todos": [ { "text": "first" }, { "text": "second" } ]
    }
```

## Capture and reuse

`capture NAME from PATH` stores a value from a response so later cases
can use it. Path syntax:

- `json.field` — drill into the parsed JSON body
- `json.items.0.id` — into an array element
- `header:X-Request-Id` — read a response header
- `status` — the HTTP status code

Interpolate captured values via `{{name}}` in `path`, `body`,
`header`, `query`, or any string leaf inside `json`.

```capy
test "create"
    request POST "/items"
        json { "name": "laptop" }
    capture id from json.id

test "read it back"
    request GET "/items/{{id}}"
    expect
        status 200
```

## What's wired automatically

When your suite runs, the in-process Wave server gets:

- Every part of the request pipeline your `server.capy` declares
  (auth, CSRF, rate limit, CORS, …)
- Built-in routes (`/healthz`, `/readyz`, `/metrics`, the JSON 404
  envelope)
- The default secure-headers middleware
- All schedule jobs (heads up: an `every 1s` job fires during the run
  — use a test-only config or guard it)
- All connections (SSE brokers) — publish to them from cases and probe
  the subscribe path to verify

It does **not** get HTTPS termination, a real port binding, or the
background outbox worker (disabled in test mode).

## CI integration

The CI gate is two shell commands; wire them into whatever runner you
use (GitHub Actions, GitLab CI, Buildkite, Drone — all the same):

```sh
go install github.com/olivierdevelops/wave/orchestrator@latest
wave check server.capy
wave test  server.test.capy --format json
```

`wave check` fails the build on any parse or load error; `wave test`
exits non-zero on the first failed test. The JSON output is
machine-readable for downstream summaries.

The `--format json` envelope:

```jsonc
{
  "suite": "/path/to/server.test.capy",
  "results": [
    { "name": "built-in /healthz returns ok", "phase": "test",
      "passed": true, "status": 200, "duration_ns": 1234567 }
  ],
  "passed": 4, "failed": 0, "duration_seconds": 0.012, "ok": true
}
```

Pre-commit hook:

```sh
# .git/hooks/pre-commit
wave check server.capy || exit 1
wave fmt   server.capy --check || exit 1
wave test  server.test.capy --format json > /dev/null || {
  echo "wave tests failed; run \`wave test server.test.capy -v\` for detail"
  exit 1
}
```

## Calling the runner from Go

```go
import (
  "context"
  "testing"
  "github.com/olivierdevelops/wave/infra/wavetest"
)

func TestSuite(t *testing.T) {
  s, err := wavetest.RunFile(context.Background(), "server.test.capy")
  if err != nil { t.Fatal(err) }
  if !s.OK {
    for _, r := range s.Results {
      if !r.Passed { t.Errorf("%s [%s]: %v", r.Name, r.Phase, r.Errors) }
    }
  }
}
```

For a quieter run: `wavetest.RunFileWithOptions(ctx, "server.test.capy",
wavetest.Options{Quiet: true})`.

## Patterns

### Test the unhappy paths

The input validators run before your handler. Confirm they reject what
you'd expect:

```capy
test "empty name → 400"
    request POST "/items"
        json { "name": "" }
    expect
        status 400

test "oversized payload → 413"
    request POST "/items"
        body `{{big_string}}`
    expect
        status 413
```

### Test auth

Use `setup` to acquire a session cookie, then reuse via `capture`:

```capy
setup "login"
    request POST "/auth/login"
        form { "email": "ada@test", "password": "secret" }
    capture session_cookie from header:Set-Cookie

test "protected route with session"
    request GET "/me"
        header Cookie "{{session_cookie}}"
    expect
        status 200
```

### Test the framework 404 envelope

```capy
test "unmatched paths return JSON"
    request GET "/nope/anywhere"
    expect
        status 404
        json   { "error": "not found", "path": "/nope/anywhere" }
```

## Gotcha: fixture state between runs

The runner boots your **real** `server.capy`. A file-backed SQLite DB
persists between runs, so a "create row" test can fail on the second
run (duplicate key). Options:

1. **Idempotent tests** — clean up in `setup`/`teardown`.
2. **Per-run database** — point `location` at a secret/env you set in
   the suite, and delete the file in CI.
3. **In-memory SQLite** — `location ":memory:"` (best for CI; fresh DB
   each run, no cleanup).

## When wavetest isn't the right tool

- **Load testing** — use vegeta or k6.
- **Plugin unit tests** — `go test` (or pytest) in the plugin's repo.
- **Browser-level tests** — Playwright / Cypress.
- **Production monitoring** — Checkly / Datadog / Pingdom.

## See also

- Runnable: [`examples/apps/url-shortener/server.test.capy`](https://github.com/olivierdevelops/wave/blob/main/examples/apps/url-shortener/server.test.capy)
- [`infra/wavetest`](https://github.com/olivierdevelops/wave/tree/main/infra/wavetest) — embed the runner in your Go tests
- [Production checklist](/guide/deploy-checklist) — wavetest in pre-deploy gates
