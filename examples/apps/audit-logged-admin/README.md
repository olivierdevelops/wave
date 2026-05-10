# audit-logged-admin

Admin routes gated by RBAC. Every request that hits `/admin/...` emits
events through `infra/audit` — webhook signature checks, dashboard
views, and the built-in admin panel all use the same sink.

## What it shows off

- `require_roles: [admin]` claim-based RBAC on mutating routes.
- Built-in `/admin/` dashboard auto-registered by the orchestrator.
- `infra/audit` automatic event emission for sensitive operations.
- `inputs:` validation on POST/DELETE bodies and path params.

## Run

```sh
SECRET_KEY=devsecret \
  wave serve examples/apps/audit-logged-admin/server.yaml --port 8603
# Default sink writes JSON-line events to stderr. Tail them:
#   wave serve ... 2>&1 | grep '"action"'
```

Visit `http://127.0.0.1:8603/admin/` — the dashboard hit emits an
`admin.view` audit event.

## Caveats

- The default audit sink is a stderr WriterSink. To persist, swap in a
  FileSink (or your own MultiSink) in main.go before `Server.Start`.
- The session must carry a `roles` (or `groups`) claim containing
  `admin`. A real OIDC bridge populates that — the dev JWT won't.
