# otel-tracing-demo

Wave pushing metrics + traces through the `otel-exporter` plugin to an
OTLP gRPC collector. The Prometheus pull endpoint at `/metrics` keeps
working in parallel.

## Setup

1. Build the plugin:

   ```sh
   cd examples/plugins/otel-exporter && go build -o /tmp/wave-otel .
   ```

2. Stand up an OTLP receiver locally. Drop this in `docker-compose.yaml`:

   ```yaml
   services:
     jaeger:
       image: jaegertracing/all-in-one:1.57
       ports:
         - "16686:16686"   # UI
         - "4317:4317"     # OTLP gRPC
       environment:
         COLLECTOR_OTLP_ENABLED: "true"
   ```

   ```sh
   docker compose up -d
   export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
   ```

## Run it

```sh
wave serve examples/apps/otel-tracing-demo/server.yaml --port 8704
```

## Try it

Generate some load, then look in the Jaeger UI at
`http://localhost:16686`:

```sh
for i in $(seq 1 50); do curl -sXPOST "localhost:8704/hit?label=run$i" >/dev/null; done
curl localhost:8704/hits
curl localhost:8704/metrics | head   # built-in Prometheus still on
```

## What to look at

`observability.exporters: [otel]` selects which exporter-kind plugins
receive the fanout. Empty list = all of them. The drain goroutine is
non-blocking — slow plugins drop overflow rather than backing up the
request path.

## Caveats

If the collector at `OTEL_EXPORTER_OTLP_ENDPOINT` is unreachable the
plugin retries with backoff and emits warnings to stderr; the request
path keeps working.
