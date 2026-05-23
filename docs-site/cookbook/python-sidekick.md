# Wave as a sidekick to a Python ML / data service

You have a Python service — FastAPI, Flask, an ML model server,
a data API. It does its job well but you don't want to bolt on
auth, rate limiting, audit logging, and webhook signature
verification in Python. Wave handles those; Python keeps doing
domain work.

## Architecture

```
                ┌──────────────────┐         ┌──────────────────┐
   client  ───► │  Wave            │   ───►  │  Python service  │
                │  - auth          │   HTTP  │  (FastAPI /      │
                │  - rate limit    │   or    │   Flask / model  │
                │  - audit         │   subp  │   server / DAG)  │
                │  - SSE bridge    │   roc   │  Unchanged.      │
                │  - 202 + tasks   │         │                  │
                └──────────────────┘         └──────────────────┘
                       │
                       ▼
                ┌──────────────┐
                │  SQLite for  │
                │  audit + req │
                │  counts      │
                └──────────────┘
```

## Two transport options

### Option A — Python is an HTTP service (`type: forward` / `type: api`)

Simplest. Python listens on `127.0.0.1:9001`; Wave proxies.

```yaml
plugins: {}        # not used in this variant

routes:
  - path: /api/predict
    methods: [POST]
    type: forward
    auth: [app]
    limits: [rate_100rpm]
    forward:
      forward_url: "http://127.0.0.1:9001/predict"
      include_headers:
        - ["X-User-Id", "{{getUser}}"]
```

When you need to *transform* the response (e.g., extract a nested
field), use `type: api` instead and shape with the response config.

### Option B — Python is a long-running subprocess (`type: plugin`)

If Python is best run as a worker (model loaded in memory, no HTTP
boilerplate), make it a Wave plugin. Wave manages its lifecycle.

```yaml
plugins:
  model:
    kind: longlived
    command: ["python3", "worker.py"]

routes:
  - path: /api/predict
    methods: [POST]
    type: plugin
    auth: [app]
    plugin:
      name: model
      trigger_key: predict
```

The worker reads framed JSON from stdin, writes framed JSON to
stdout. See [Plugins](/guide/concepts-plugins) for the contract.

## Turning a slow synchronous call into 202 + SSE progress

The big win: `type: task` lets you wrap any plugin call into a
"fire and watch" pattern. The client gets a `task_id` immediately;
progress events flow over SSE.

```yaml
connections:
  events:
    type: sse
    subscribe_path: /events/inference
    buffer_size: 256

routes:
  - path: /api/generate
    methods: [POST]
    type: task
    auth: [app]
    inputs:
      - { name: prompt, source: body, type: string, required: true }
    task:
      plugin: model
      trigger_key: generate
      streaming: true                       # plugin emits ndjson
      connection: events
      event_type: chunk
```

Python worker emits per-token output:

```python
# worker.py — long-lived
import sys, json

while True:
    line = sys.stdin.readline()
    if not line: break
    req = json.loads(line)
    body = json.loads(req['body'])

    # Stream tokens as ndjson — Wave forwards each line as one SSE event
    for token in model.generate(body['prompt']):
        print(json.dumps({"token": token}))
        sys.stdout.flush()
    # Finish with a sentinel
    print(json.dumps({"done": True}))
    sys.stdout.flush()
```

Frontend:

```js
const r = await fetch('/api/generate', { method: 'POST', body: JSON.stringify({prompt}) })
const { task_id } = await r.json()    // 202 Accepted

const es = new EventSource('/events/inference')
es.addEventListener('chunk', e => {
  const { token, done } = JSON.parse(e.data)
  if (done) { es.close(); return }
  appendToOutput(token)
})
```

## Full integration: auth + rate-limit + SSE + audit

```yaml
default:
  port: 8080

env:
  JWT_SECRET: { description: "session HMAC" }

auth:
  app:
    type: jwt
    secret: "${env:JWT_SECRET}"
    cookie_name: session

storage:
  app:
    type: sqlite
    path: ./data.db
    tables:
      audit_log:
        columns:
          - id     INTEGER PRIMARY KEY AUTOINCREMENT
          - actor  TEXT NOT NULL
          - action TEXT NOT NULL
          - at     TEXT NOT NULL DEFAULT (datetime('now'))

limits:
  inference_per_user:
    case: rate_limited
    rps: 1                  # 1 inference per second per user
    burst: 5
    key_claim: sub          # bucket per authenticated user
    on_fail:
      status_code: 429
      body: '{"error":"rate limited per user"}'

plugins:
  model:
    kind: longlived
    command: ["python3", "model_server.py"]

connections:
  inference:
    type: sse
    subscribe_path: /events/inference
    buffer_size: 256

routes:
  # Quick synchronous classification — no SSE
  - path: /api/classify
    methods: [POST]
    type: plugin
    auth: [app]
    limits: [inference_per_user]
    plugin:
      name: model
      trigger_key: classify

  # Long-running streaming generation
  - path: /api/generate
    methods: [POST]
    type: task
    auth: [app]
    limits: [inference_per_user]
    inputs: [{ name: prompt, source: body, type: string, required: true, max: 4000 }]
    task:
      plugin: model
      trigger_key: generate
      streaming: true
      connection: inference
      event_type: chunk
      store:                                # log every inference for audit
        source: app
        execute: "INSERT INTO audit_log(actor, action) VALUES ({{getUser}}, 'generate')"
```

The Python model server doesn't know any of this exists. It reads
JSON requests, writes JSON / ndjson responses. Wave handles the
rest.

## When Python should host the HTTP itself

- You're already invested in FastAPI middleware (custom auth, OPA,
  etc.) — keep it
- You need WebSockets where Python's `websockets`/`fastapi-ws` are
  better than what Wave's SSE gives you
- The Python service is consumed by clients that won't change for
  Wave (existing customers with hard-coded URLs)

In those cases, use Pattern 2 ([Wave in front of a Node service](/cookbook/node-gateway))
— the same pattern works for any HTTP backend regardless of
language.

## See also

- [Background tasks](/cookbook/background-tasks)
- [Stream events with SSE](/cookbook/sse)
- [Plugins](/guide/concepts-plugins)
- Demos:
  [`background-task-demo`](https://github.com/luowensheng/wave/tree/main/examples/apps/background-task-demo),
  [`queue-worker-demo`](https://github.com/luowensheng/wave/tree/main/examples/apps/queue-worker-demo),
  [`streaming-file-processor`](https://github.com/luowensheng/wave/tree/main/examples/apps/streaming-file-processor)
