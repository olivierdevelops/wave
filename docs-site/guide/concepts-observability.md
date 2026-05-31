# Observability

Wave ships with the four pillars wired in: metrics, traces, logs, and
an audit channel. They're on by default; you opt out, not in.

## Health endpoints

Always registered, no config required:

| Endpoint | Returns |
|---|---|
| `GET /healthz` | 200 once boot is complete |
| `GET /readyz` | 503 until all storages, plugins, and connections are reachable |
| `GET /version` | binary version + commit (set via ldflags) |

Use these as liveness/readiness probes. In Kubernetes, point
`livenessProbe` at `/healthz` and `readinessProbe` at `/readyz` on the
container port — both `httpGet` checks, no extra config needed.

## Prometheus metrics

`GET /metrics` exposes the Prometheus exposition format. Counters
and histograms emitted out of the box:

| Metric | Labels | What it counts |
|---|---|---|
| `wave_requests_total` | route, method, status | per-route request count |
| `wave_request_duration_seconds` | route, method | latency histogram |
| `wave_rate_limit_rejects_total` | route | requests dropped by limiter |
| `wave_circuit_open_total` | route | requests rejected by open circuit |
| `wave_storage_query_duration_seconds` | source, op | SQL latency |
| `wave_plugin_call_duration_seconds` | plugin, trigger | plugin RPC latency |
| `wave_outbox_pending` | — | live outbox queue size |
| `wave_outbox_dlq_total` | — | dead-letter count |
| `wave_sse_subscribers` | broker | connected SSE clients |

Scrape it from Prometheus by adding a `wave` job that targets the
container's port 8080. Static target, no auth, no extras —
`/metrics` is open on the same listener as the rest of the server.

## OpenTelemetry traces

Spans emitted automatically per route, including downstream HTTP
calls and SQL queries. Configure the OTLP exporter in the top-level
`observability` block:

```capy
# server.capy
observability
    otel
        endpoint     "{{secret otel_endpoint}}"   # e.g. http://otel-collector:4318
        service_name "my-wave-app"
        sample_rate  0.1                            # 10% sampling
```

See [`otel-tracing-demo`](https://github.com/olivierdevelops/wave/tree/main/examples/apps/otel-tracing-demo)
for a full pipeline with Jaeger.

## Structured logs

Logs are JSON-formatted by default. Each request emits:

```jsonc
{
  "ts": "2026-05-23T09:00:00.123Z",
  "level": "info",
  "msg": "request",
  "request_id": "abcd1234",
  "method": "POST",
  "path": "/items",
  "status": 201,
  "duration_ms": 12.4,
  "user_id": "ada",
  "ip": "192.168.1.42"
}
```

`request_id` is the same value the framework returns in the
`X-Request-ID` response header — easy correlation between log search
and a specific user-reported error.

Set `LOG_LEVEL=debug` for verbose handler-level traces. Set
`LOG_FORMAT=text` for human-readable colored output (dev only).

## Audit log

Distinct from regular logs — this is your durable, queryable
record of state-changing events. Write the audit row in the same SQL
statement as the mutation — one transaction, no chance of drift:

```capy
route "/admin/users/{id}"
    methods DELETE
    requires_authentication primary
    request
        path_parameter id
            type     integer
            required true
    do
        result = on app do sql `
            INSERT INTO audit_log (actor, action, target, ip, at)
            VALUES ({{auth.user.id}}, 'user.delete', 'user:' || {{request.id}}, {{client_ip}}, {{now}});
            DELETE FROM users WHERE id = {{request.id}}
        `
        match result
            case success(info)
                response
                    status       200
                    content_type "application/json"
                    body         `{"deleted":{{info.rows_affected}}}`
```

The audit row records the authenticated user (`{{auth.user.id}}`), the
client IP (`{{client_ip}}`), a timestamp, the action, and the target —
all bound parameters. See the [audit log recipe](/cookbook/audit-log).

## Plugin-based exporter fanout

If your observability stack doesn't speak OTLP or Prometheus,
write a plugin that exports to it. Wave's observability fanout
broadcasts each event to every registered exporter plugin
concurrently — Prometheus AND your Datadog/Honeycomb/SaaS at the
same time.

See [`exporter-plugins.md`](https://github.com/olivierdevelops/wave/blob/main/docs/exporter-plugins.md).

## Production checklist

- [ ] Configure Prometheus scrape on `/metrics`
- [ ] OTLP endpoint set, sample rate tuned (10-25% for high volume)
- [ ] Log shipping pipeline aggregates JSON logs to your SIEM
- [ ] Audit table replicates to long-term storage (or an outbox to
      a SIEM)
- [ ] Alert on `wave_outbox_dlq_total > 0` and on `/readyz` failing
- [ ] Dashboards for: p99 request latency by route, error rate by
      status code, rate-limit rejects per minute

## See also

- Demo: [`otel-tracing-demo`](https://github.com/olivierdevelops/wave/tree/main/examples/apps/otel-tracing-demo)
- [Audit log recipe](/cookbook/audit-log)
- [Production checklist](/guide/deploy-checklist)
