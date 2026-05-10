# multi-tenant-saas

Tenants are derived from the request subdomain (`acme.localhost`), so
storage queries scope by `tenant_id` without app code.

## What it shows off

- `inputs:` pulling Host header with a regex `pattern` to validate.
- `type: storage-access` parameterised SQL with `{{tenant}}` substitution.
- Inline `storage.app.tables.projects.columns` schema declaration.

## Run

```sh
wave serve examples/apps/multi-tenant-saas/server.yaml --port 8602
```

## Test

Either add to `/etc/hosts`:
```
127.0.0.1 acme.localhost beta.localhost
```
or use curl `--resolve`:
```sh
curl --resolve acme.localhost:8602:127.0.0.1 -X POST \
     http://acme.localhost:8602/projects \
     -H 'Content-Type: application/json' -d '{"name":"alpha"}'

curl --resolve beta.localhost:8602:127.0.0.1 \
     http://beta.localhost:8602/projects
```

`acme` and `beta` see only their own rows.

## Caveats

- The substring trick assumes the host shape `<tenant>.localhost(:port)`.
  In production, terminate TLS at a proxy and forward `X-Tenant`.
