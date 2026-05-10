# multi-plugin-stack — production-shaped reference

A small issue tracker (3 routes) that exercises the full plugin
contract surface in one config:

| Kind     | Plugin              | Used for                              |
| -------- | ------------------- | ------------------------------------- |
| storage  | `postgres-storage`  | `issues` table reads/writes           |
| auth     | `saml-auth`         | corporate SSO (`/login/saml`)         |
| secrets  | `vault-secrets`     | JWT signing key, Postgres DSN         |
| exporter | `otel-exporter`     | OTLP push of metrics/traces           |

This is what a "real" Wave deployment actually looks like — secrets
are never in YAML, identity comes from the corporate IdP, durable data
lives in Postgres, observability fans out to OTel.

## Setup

1. Build all four plugin binaries (paths must match the YAML):

   ```sh
   cd examples/plugins/postgres-storage && go build -o /tmp/wave-postgres . && cd -
   cd examples/plugins/saml-auth        && go build -o /tmp/wave-saml     . && cd -
   cd examples/plugins/vault-secrets    && go build -o /tmp/wave-vault    . && cd -
   cd examples/plugins/otel-exporter    && go build -o /tmp/wave-otel     . && cd -
   ```

2. External services:

   - **Postgres**: create an `issues` table:
     ```sql
     CREATE TABLE issues (
       id     SERIAL PRIMARY KEY,
       title  TEXT NOT NULL,
       status TEXT NOT NULL DEFAULT 'open'
     );
     ```
   - **Vault** (dev mode is fine): write the secrets the config references:
     ```sh
     vault kv put secret/wave \
       jwt_secret=$(openssl rand -hex 32) \
       pg_dsn="postgres://user:pass@localhost:5432/wave?sslmode=disable"
     ```
   - **SAML IdP**: any IdP whose metadata URL you can reach. Register
     `http://127.0.0.1:8705/login/saml/callback` as the ACS.
   - **OTLP collector**: Jaeger all-in-one or otel-collector on `:4317`.
     See `examples/apps/otel-tracing-demo/README.md` for a compose snippet.

3. Export env:

   ```sh
   export VAULT_ADDR=http://127.0.0.1:8200
   export VAULT_TOKEN=devroot
   export SAML_IDP_METADATA_URL=https://samltest.id/saml/idp
   export SAML_SP_CERT_PATH=/tmp/wave-sp.crt
   export SAML_SP_KEY_PATH=/tmp/wave-sp.key
   export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
   ```

## Run it

```sh
wave serve examples/apps/multi-plugin-stack/server.yaml --port 8705
```

## Try it

```sh
# Browser flow: /login → SSO → /issues
open http://127.0.0.1:8705/login

# Once you have the corp_session cookie:
curl -b cookies.txt localhost:8705/issues
curl -b cookies.txt -XPOST localhost:8705/issues \
  -d '{"title":"first bug"}' -H 'content-type: application/json'
curl -b cookies.txt -XPOST localhost:8705/issues/1/close
```

## What to look at

- **Plugin ordering at boot.** Secrets-kind plugins start first so
  that `${PLUGIN:vault:...}` markers in *other* plugin env (`pg.env.PG_DSN`)
  resolve before those plugins boot.
- **Single source of identity.** The route's `auth: ["corp_sso"]` and
  the SAML plugin's `Claims.Roles` together give you RBAC without any
  per-route policy logic.
- **Telemetry side-channel.** `observability.exporters: [otel]` adds a
  push subscriber; `/metrics` (Prometheus pull) keeps working.

## Caveats

Boot fails fast if any plugin binary is missing, any `${PLUGIN:...}`
marker is unresolved, or any external dep (Vault, Postgres, IdP
metadata, OTLP receiver) is unreachable. That is by design — the
config is meant to be a deployment contract, not a best-effort.
