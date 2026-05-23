# Functional testing with `wave test`

Wave ships a built-in test runner. You write **YAML test suites**
that boot your `server.yaml` in-process (no port binding) and
assert request/response behavior. Same execution path as production
— including auth, validation, CORS, rate limits.

::: tip
This is functional / integration testing. For unit-testing your
Go plugins, use standard `go test`.
:::

## The shape

A `server.test.yaml` file lives next to your `server.yaml`. It
imports the config under test and declares a sequence of cases:

```yaml
# server.test.yaml
import: server.yaml

# Optional: env vars exported for the run, so server.yaml's
# ${env:NAME} interpolation resolves them. Restored after.
env:
  JWT_SECRET: test-only-secret
  STRIPE_WEBHOOK_SECRET: whsec_test

# Optional: run before tests; if any fails, tests are skipped
setup:
  - name: seed first user
    request:
      method: POST
      path: /signup
      json: { email: ada@example.com }
    expect:
      status: 201

# The actual assertions
tests:
  - name: built-in /healthz returns ok
    request:
      method: GET
      path: /healthz
    expect:
      status: 200
      json:
        status: ok

  - name: POST /items creates a row
    request:
      method: POST
      path: /items
      headers:
        Authorization: "Bearer test-token"
      json:
        name: laptop
        price: 999
    expect:
      status: 200
      json:
        id: "*"                  # any present value
    capture:
      created_id: json.id        # save for later cases

  - name: GET /items/{id} returns what we just created
    request:
      method: GET
      path: /items/{{.created_id}}
    expect:
      status: 200
      json:
        name: laptop
        price: 999

  - name: GET nonexistent returns framework 404 envelope
    request:
      method: GET
      path: /items/9999
    expect:
      status: 404
      json:
        error: not found

# Optional: runs regardless of test outcome; failures reported but
# don't change the suite's pass/fail.
teardown:
  - name: cleanup test user
    request:
      method: DELETE
      path: /admin/users/ada@example.com
      headers: { Authorization: "Bearer admin-test-token" }
    expect:
      status: [200, 404]         # any of these is fine
```

Run it:

```sh
wave test server.test.yaml
#   PASS [test] built-in /healthz returns ok (200, 1ms)
#   PASS [test] POST /items creates a row (200, 3ms)
#   PASS [test] GET /items/{id} returns what we just created (200, 1ms)
#   PASS [test] GET nonexistent returns framework 404 envelope (404, 0s)
#
#   4 passed, 0 failed, 0.01s
```

In CI:

```sh
wave test server.test.yaml --json    # machine-readable, exits non-zero on failure
```

## Request shape

Inside `request:`:

| Field | What |
|---|---|
| `method` | GET / POST / PUT / PATCH / DELETE / OPTIONS (default GET) |
| `path` | URL path. Path templating supported via `{{.var}}` |
| `headers` | Map of header name → value. Templated. |
| `query` | Map of query param → value. Templated. |
| `body` | Raw string body. Templated. |
| `json` | Object — marshaled to JSON, sets `Content-Type: application/json`. Templated through every string leaf. |
| `form` | Map — encoded as `application/x-www-form-urlencoded`. Templated. |

Only one of `body` / `json` / `form` should be set per request.

## Expect shape

Inside `expect:`:

| Field | Behavior |
|---|---|
| `status` | Exact HTTP status (omit = don't check) |
| `body` | Exact body match (after TrimSpace) |
| `body_contains` | Substring check |
| `headers` | Map name → exact value (case-insensitive name lookup) |
| `json` | **Strict subset** match against the response parsed as JSON |

### JSON subset semantics

`expect.json` is matched against the response body parsed as JSON.

- **Maps**: every key in `expect` must exist in the response with
  a matching value. Extra keys in the response are **fine**.
- **Slices**: same length, element-wise subset by index.
- **Scalars**: equal. Numeric types coerce (YAML `1` matches JSON
  `1.0`).
- **Wildcard**: the literal string `"*"` matches any present value.
  Use for "field exists, I don't care about the value" assertions.

```yaml
expect:
  json:
    id: "*"            # any value, must be present
    name: ada
    todos:
      - text: first
      - text: second   # exact match by index
```

## Capture and reuse

`capture:` stores values from a response so later cases can use
them. Path syntax:

- `json.field` — drill into the parsed JSON body
- `json.items.0.id` — into a slice element
- `header:X-Request-Id` — read a response header
- `status` — the HTTP status code
- `json` — the whole body

Interpolate captured values via `{{.var_name}}` in `path`, `body`,
`headers`, `query`, or any string leaf inside `json:`.

```yaml
- name: create
  request: { method: POST, path: /items, json: { name: laptop } }
  capture: { id: json.id }

- name: read it back
  request: { method: GET, path: /items/{{.id}} }
  expect:  { status: 200 }

- name: update
  request:
    method: PATCH
    path: /items/{{.id}}
    json: { name: "{{.name}}-renamed" }     # if 'name' was captured earlier
```

## What's wired automatically

When your suite runs, the in-process Wave server gets:

- Every middleware your `server.yaml` declares (auth, CSRF,
  rate limit, CORS, audit, …)
- Built-in routes (`/healthz`, `/readyz`, `/metrics`, the JSON 404
  envelope for unmatched paths)
- The default secure-headers middleware
- All schedule jobs (heads up: a `schedule: { every: 1s }` will
  fire during the test run — use a test-only config or disable
  schedules via `env:`)
- All connections (SSE brokers) — you can publish to them from
  test cases and probe `/events/stream` to verify

It does **not** get:

- HTTPS termination (httptest is plain HTTP)
- A real port binding (httptest is in-memory)
- Background `outbox_db` worker (disabled in test mode unless
  the suite explicitly sets `outbox_db:` via `env:`)

## CI integration

GitHub Actions:

```yaml
# .github/workflows/wave-test.yml
name: wave test
on: [push, pull_request]
jobs:
  wave-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24.x' }
      - run: go install github.com/luowensheng/wave/orchestrator@latest
      - run: wave test server.test.yaml --json
```

The `--json` flag emits a machine-readable envelope:

```jsonc
{
  "suite": "/path/to/server.test.yaml",
  "results": [
    { "name": "built-in /healthz returns ok",
      "phase": "test",
      "passed": true,
      "status": 200,
      "duration_ns": 1234567 }
    // …
  ],
  "passed": 4,
  "failed": 0,
  "duration_seconds": 0.012,
  "ok": true
}
```

Pre-commit hook:

```sh
# .git/hooks/pre-commit
wave validate server.yaml || exit 1
wave fmt --check server.yaml || exit 1
wave test server.test.yaml --json > /dev/null || {
  echo "wave tests failed; run \`wave test server.test.yaml -v\` for detail"
  exit 1
}
```

## Calling the runner from Go

If you have a Go test suite already, drop into the runner directly:

```go
import "github.com/luowensheng/wave/infra/wavetest"

func TestSuite(t *testing.T) {
  s, err := wavetest.RunFile(context.Background(), "server.test.yaml")
  if err != nil { t.Fatal(err) }
  if !s.OK {
    for _, r := range s.Results {
      if !r.Passed {
        t.Errorf("%s [%s]: %v", r.Name, r.Phase, r.Errors)
      }
    }
  }
}
```

## Patterns

### Test the unhappy paths

Wave's input validators run before your handler. Confirm they
reject what you'd expect:

```yaml
- name: empty name → 400
  request: { method: POST, path: /items, json: { name: "" } }
  expect:  { status: 400 }

- name: oversized payload → 413
  request: { method: POST, path: /items, body: "{{.big_string}}" }
  expect:  { status: 413 }
```

### Test auth

Use `setup:` to acquire a session cookie, then reuse via `capture`:

```yaml
setup:
  - name: login
    request:
      method: POST
      path: /auth/login
      form: { email: "ada@test", password: "secret" }
    capture:
      session_cookie: "header:Set-Cookie"

tests:
  - name: protected route with session
    request:
      method: GET
      path: /me
      headers: { Cookie: "{{.session_cookie}}" }
    expect: { status: 200 }
```

### Test the framework 404 envelope

```yaml
- name: unmatched paths return JSON
  request: { method: GET, path: /nope/anywhere }
  expect:
    status: 404
    json:
      error: page not found
      path:  /nope/anywhere
```

## When wavetest isn't the right tool

- **Load testing**: use [vegeta](https://github.com/tsenart/vegeta)
  or [k6](https://k6.io). wavetest is functional, not
  performance.
- **Plugin unit tests**: standard `go test` (or pytest, etc.) in
  the plugin's own repo.
- **Browser-level integration tests**: Playwright / Cypress. They
  test the React+Wave stack together.
- **Production health monitoring**: use a real synthetic-monitoring
  tool (Checkly, Datadog, Pingdom) that runs from outside your
  infra.

## See also

- Runnable example: [`examples/apps/url-shortener/server.test.yaml`](https://github.com/luowensheng/wave/blob/main/examples/apps/url-shortener/server.test.yaml)
- [`infra/wavetest`](https://github.com/luowensheng/wave/tree/main/infra/wavetest) — Go package for embedding the runner in your own tests
- [Production checklist](/guide/deploy-checklist) — wavetest in
  pre-deploy gates
- [Token efficiency](/ai/token-efficiency) — and a YAML test suite
  is 5-10× shorter than the Go equivalent too
