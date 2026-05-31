# Wave Capy — schema migrations

First-class schema migrations declared in capy, applied at boot in
deterministic order, tracked per storage in a wave-owned table. No
external tool, no Go code, no plugin required. Same language for the
schema as for everything else.

> Why first-class? Until now, Wave users either ran their migrations
> outside the server (sqlc / goose / atlas / `migrate`) or hand-rolled
> `CREATE TABLE IF NOT EXISTS` in a scheduled job. Both work but neither
> is the *capy way* — declarative, versioned, deterministic, auditable.
> This doc adds `migration` as a top-level declaration alongside
> `storage`, `plugin`, `connection`, `authentication`, and `every`.

---

## TL;DR

```capy
storage app
    kind     sqlite
    location "./app.db"

migration "001_initial" on app
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

migration "002_add_roles" on app
    up `
        ALTER TABLE users ADD COLUMN roles TEXT NOT NULL DEFAULT '[]';
    `
    down `
        ALTER TABLE users DROP COLUMN roles;
    `
```

Boot order:

1. Open the storage.
2. Ensure the migration ledger table exists (`__wave_migrations`).
3. Read which migration ids are already applied.
4. Apply pending ones in declaration order, each inside a transaction.
5. Refuse to boot if any `up` fails. Log + exit non-zero.
6. Start the server.

That's the whole feature. Everything below is detail.

---

## 1. Declaration shape

```capy
migration ID on STORAGE
    [description "text"]
    [requires_id1, id2, …]              # optional dependency on prior migrations
    [transactional true|false]          # default true; some DDL needs false
    [tag dev|staging|prod, …]           # optional environment gating
    up `
        # SQL statements, multi-statement supported
    `
    [down `
        # reverse SQL — used by `wave migrate down`
    `]
```

| Field | Required | Meaning |
|---|---|---|
| `ID` | yes | Unique identifier (string). Convention: `NNN_snake_case` (e.g. `001_initial`). Sort order = declaration order, NOT id order — but matching the two prevents surprises. |
| `on STORAGE` | yes | Name of a declared `storage` block. Each storage has its own ledger. |
| `description` | no | Human-readable summary; surfaced in `wave migrate status`. |
| `requires` | no | Comma-separated list of prior migration ids that must be applied first. Defaults to "everything declared earlier in the file". Useful when migrations are split across multiple `.capy` files. |
| `transactional` | no | `true` (default) → `BEGIN; up; COMMIT`. Some DDL (Postgres `CREATE INDEX CONCURRENTLY`, SQLite vacuum) cannot run inside a transaction — set to `false`. |
| `tag` | no | Comma-separated environment tags. If the running server has `--env prod`, only migrations tagged `prod` (or untagged) apply. |
| `up` | yes | The forward SQL. Multi-statement, `;`-separated. |
| `down` | no | The reverse SQL. If absent, `wave migrate down` refuses to roll back past this migration. |

---

## 2. The ledger table

Every storage with at least one migration gets a wave-owned ledger:

```sql
-- created automatically at boot
CREATE TABLE IF NOT EXISTS __wave_migrations (
    id           TEXT PRIMARY KEY,        -- the migration id
    applied_at   TEXT NOT NULL,           -- ISO-8601 UTC
    duration_ms  INTEGER NOT NULL,        -- how long the `up` took
    checksum     TEXT NOT NULL,           -- SHA-256 of the `up` body, normalized
    description  TEXT
);
```

**Checksum guard**. If a migration's `up` body changes after it's been
applied (someone edits a committed migration), boot fails with a
`migration_checksum_mismatch` error. You either:

- Revert the edit (recommended).
- Add a new migration that adjusts the schema forward.
- Force-rewrite the ledger row with `wave migrate force-update <id>`
  (manual escape hatch; logs a warning at every subsequent boot until
  cleared).

This is the single biggest source of "works on my laptop, broken in
prod" — Wave makes it impossible to silently diverge.

---

## 3. Boot lifecycle

```
wave serve server.capy
  ├── parse file
  ├── for each declared storage:
  │     open connection
  │     ensure __wave_migrations exists
  │     read applied set
  │     compute pending = declared - applied (in declaration order)
  │     for each pending:
  │       check requires:* satisfied
  │       check env tag matches running env
  │       open transaction (unless transactional false)
  │       run up SQL
  │       insert into __wave_migrations
  │       commit
  │     fail-fast on any error → process exits non-zero
  ├── wire routes, schedules, connections
  └── start HTTP listener
```

`wave check server.capy` does the same up to the "open transaction"
step in a **read-only dry run** — it parses every migration and reports
what would apply, without changing the database. Use in CI.

---

## 4. CLI commands

| Command | What it does |
|---|---|
| `wave migrate status [server.capy]` | Show applied vs pending per storage |
| `wave migrate up [server.capy]` | Apply all pending without booting the server |
| `wave migrate up --to ID` | Apply pending up to and including `ID` |
| `wave migrate down [server.capy]` | Roll back the most recently applied migration (uses `down` body) |
| `wave migrate down --to ID` | Roll back to and including `ID` (exclusive — `ID` itself stays applied) |
| `wave migrate new "name"` | Scaffold a new `migration` block with the next sequential id |
| `wave migrate force-update ID` | Rewrite the checksum in the ledger to match the current `up` body (escape hatch) |
| `wave migrate verify` | Re-compute every applied migration's checksum and compare to the ledger; fails on drift |

All commands accept `--storage NAME` to scope to one storage when
multiple are declared. `--dry-run` prints SQL without executing.

---

## 5. End-to-end example

```capy
storage app
    kind     sqlite
    location "./app.db"

# --- schema ---

migration "001_initial" on app
    description "users, things, audit"
    up `
        CREATE TABLE users (
            id            INTEGER PRIMARY KEY AUTOINCREMENT,
            email         TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            roles         TEXT NOT NULL DEFAULT '[]',
            created_at    TEXT NOT NULL
        );
        CREATE INDEX users_email ON users(email);

        CREATE TABLE things (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            owner_id   INTEGER NOT NULL REFERENCES users(id),
            name       TEXT NOT NULL,
            created_at TEXT NOT NULL
        );
        CREATE INDEX things_owner ON things(owner_id);

        CREATE TABLE audit_log (
            id     INTEGER PRIMARY KEY AUTOINCREMENT,
            actor  INTEGER,
            action TEXT NOT NULL,
            target TEXT,
            ip     TEXT,
            at     TEXT NOT NULL
        );
    `
    down `
        DROP TABLE IF EXISTS audit_log;
        DROP TABLE IF EXISTS things;
        DROP TABLE IF EXISTS users;
    `

migration "002_add_things_archived" on app
    description "soft-delete flag for things"
    up `
        ALTER TABLE things ADD COLUMN archived_at TEXT;
        CREATE INDEX things_archived ON things(archived_at) WHERE archived_at IS NULL;
    `
    down `
        DROP INDEX IF EXISTS things_archived;
        ALTER TABLE things DROP COLUMN archived_at;
    `

migration "003_seed_default_admin" on app
    description "create the bootstrap admin row"
    tag dev, staging
    up `
        INSERT OR IGNORE INTO users (email, password_hash, roles, created_at)
        VALUES ('admin@local', '!unset', '["admin"]', strftime('%Y-%m-%dT%H:%M:%SZ','now'));
    `
    # no down — seed data isn't reversible

# --- routes use the migrated schema ---

route "/things"
    methods GET
    requires_authentication user
    request
    do
        result = on app do sql `
            SELECT id, name, created_at FROM things
             WHERE owner_id = {{auth.user.id}} AND archived_at IS NULL
             ORDER BY id DESC
        `
        match result
            case success(rows)
                response
                    status       200
                    content_type "application/json"
                    body         `{{to_json rows}}`
```

Run it:

```bash
$ wave migrate status server.capy
storage: app
  ✓ applied:  (none)
  ▸ pending: 001_initial          users, things, audit
              002_add_things_archived  soft-delete flag for things
              003_seed_default_admin   create the bootstrap admin row   [tag: dev,staging]

$ wave serve server.capy --env dev
applying 001_initial on app ... ok (4ms)
applying 002_add_things_archived on app ... ok (1ms)
applying 003_seed_default_admin on app ... ok (1ms)
listening on :8080
```

Same flow in prod, but the seed migration is skipped:

```bash
$ wave serve server.capy --env prod
applying 001_initial on app ... ok (4ms)
applying 002_add_things_archived on app ... ok (1ms)
skipping  003_seed_default_admin (tag mismatch: needs dev|staging; running prod)
listening on :8080
```

---

## 6. Per-storage migrations (multi-DB apps)

Each storage has its own ledger. A single `server.capy` can migrate
multiple databases:

```capy
storage app
    kind     sqlite
    location "./app.db"

storage analytics
    kind     postgres
    location "{{secret analytics_dsn}}"

migration "001_app_initial" on app
    up `CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT UNIQUE NOT NULL)`
    down `DROP TABLE users`

migration "001_analytics_initial" on analytics
    up `
        CREATE TABLE events (
            id          BIGSERIAL PRIMARY KEY,
            user_id     BIGINT NOT NULL,
            event_type  TEXT NOT NULL,
            payload     JSONB NOT NULL,
            at          TIMESTAMPTZ NOT NULL DEFAULT now()
        );
        CREATE INDEX events_user_at ON events(user_id, at DESC);
    `
    down `DROP TABLE events`

migration "002_analytics_concurrent_index" on analytics
    description "index without locking the table"
    transactional false                     # required for CONCURRENTLY
    up `
        CREATE INDEX CONCURRENTLY events_type ON events(event_type);
    `
    down `
        DROP INDEX CONCURRENTLY IF EXISTS events_type;
    `
```

`wave migrate status` reports per storage:

```
storage: app
  ✓ applied:  001_app_initial          (2026-05-01T09:12:03Z)
  ▸ pending: (none)

storage: analytics
  ✓ applied:  001_analytics_initial    (2026-05-01T09:12:04Z)
  ▸ pending: 002_analytics_concurrent_index   index without locking the table
```

---

## 7. Migrations across multiple files

Large apps split migrations across files (`migrations/001_users.capy`,
`migrations/002_orders.capy`, …). Use `import` and explicit `requires`
to lock the dependency graph:

```capy
# server.capy
import "migrations/001_users.capy"
import "migrations/002_orders.capy"
import "migrations/003_billing.capy"

# migrations/002_orders.capy
migration "002_orders_initial" on app
    requires 001_users_initial             # explicit dependency
    up `
        CREATE TABLE orders (
            id      INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL REFERENCES users(id),
            total   DECIMAL NOT NULL,
            at      TEXT NOT NULL
        );
    `
    down `DROP TABLE orders`
```

`wave check` validates the requires-graph: missing dependency, cycle,
or duplicate id all fail at parse time.

---

## 8. Data migrations (not just DDL)

Migrations can run any SQL — including data fixes. Keep them
idempotent so re-running (e.g. after a `force-update`) is safe:

```capy
migration "007_backfill_roles" on app
    description "set default roles on existing users"
    up `
        UPDATE users
           SET roles = '["user"]'
         WHERE roles IS NULL OR roles = '';
    `
    # no down — this is a one-way data fix
```

For long-running backfills (millions of rows), prefer a **bounded
batched scheduler** over a single boot-time migration:

```capy
every 30s as backfill_user_tier
    pending = on app do sql `
        SELECT id FROM users WHERE tier IS NULL ORDER BY id LIMIT 1000
    `
    match pending
        case found(rows)
            for_each_row(row) of rows
                on app do sql `UPDATE users SET tier = 'free' WHERE id = {{id}}`
                    bind id
                        from row.id
        case empty
            # nothing left — log + carry on
```

Schema change goes in `migration`; long-running data work goes in a
scheduled job. Don't block boot on a million-row UPDATE.

---

## 9. Failure modes (named cases)

| case | when | exit |
|---|---|---|
| `migration_checksum_mismatch(id, expected, got)` | Applied migration's `up` was edited after the fact | non-zero; refuse to boot |
| `migration_missing_dependency(id, requires)` | `requires` lists an id that doesn't exist | non-zero; parse-time |
| `migration_cycle(ids)` | `requires` graph cycles | non-zero; parse-time |
| `migration_failed(id, err)` | `up` SQL raised an error | non-zero; rollback if transactional, half-applied if not |
| `migration_unknown_in_ledger(id)` | Ledger has an id no longer declared (deleted from source) | warn; do not auto-rollback. Tells you to run `wave migrate down --to PREV` or restore the file. |
| `migration_env_skipped(id, required, running)` | Tag mismatch — informational, not an error | continue boot |

Failure cases are JSON-logged in production. In dev, they're printed
with a hint:

```
✗ migration_checksum_mismatch
    id:       002_add_things_archived
    expected: sha256:7a3f...
    got:      sha256:e1b9...
  hint: someone edited a committed migration. revert the file, OR
        write a new migration that brings the schema forward, OR
        run `wave migrate force-update 002_add_things_archived`
        if you understand the consequences.
```

---

## 10. Testing migrations

In a `server.test.capy`:

```capy
import "server.capy"

# fresh in-memory storage for tests
storage app
    kind     sqlite
    location ":memory:"

test "migrations apply cleanly"
    expect
        migrations_applied app, ["001_initial", "002_add_things_archived", "003_seed_default_admin"]

test "things table exists with archived_at"
    request POST "/things"
        content_type "application/json"
        json { "name": "laptop" }
    expect
        status 201
```

`migrations_applied` is a `wave test` built-in that asserts the ledger
contents.

For round-trip testing of `down`:

```bash
wave test server.test.capy --roundtrip-migrations
```

This applies every migration, rolls them back via `down`, then
re-applies — failing if the final schema differs from the first.
Catches missing `down` statements and silent asymmetry.

---

## 11. Production discipline

A few non-negotiable rules:

1. **Never edit a committed migration.** Add a new one.
2. **Always provide `down`** unless you genuinely cannot. Seed data
   and data-migration-only steps are the only legitimate exceptions.
3. **Keep migrations small.** One topic per file/block. Reviewable in
   one screen.
4. **Test the round-trip locally** before merging
   (`wave test --roundtrip-migrations`).
5. **Tag environment-specific migrations.** Don't gate them with
   string comparison in SQL; use `tag prod` etc.
6. **For Postgres index builds, use `transactional false` and
   `CREATE INDEX CONCURRENTLY`.** Boot-time lock waits are how you
   wake up to a Sunday-morning page.
7. **`wave check` in CI.** It catches missing requires, cycles, and
   checksum drift before merge.

---

## 12. What this replaces

Today users either:

- Hand-rolled `CREATE TABLE IF NOT EXISTS` in a `every 1d` job
  (works, doesn't track versions, can't roll back).
- Ran an external migration tool before booting the server (`goose`,
  `migrate`, `atlas`) — extra binary, separate config, separate CI
  step.
- Wrote a one-off "ensure_schema" plugin.

`migration` blocks subsume all three. The schema lives next to the
routes that use it, the same language describes both, and `wave check`
verifies the whole thing in one pass.

---

## 13. Implementation sketch (for the curious)

For the Go side that ships this feature:

- **Parser**: one new top-level `migration` declaration in the capy
  grammar. Body identical in shape to existing SQL templates (so
  `{{secret …}}` and similar helpers work).
- **Loader**: at boot, the orchestrator iterates declared migrations
  per storage; each is realized as a `StorageRef.Execute` call on the
  parameterised SQL.
- **Ledger**: `infra/sqlite/migrations.go` (and `infra/postgres/`,
  `infra/mysql/`) owns the `__wave_migrations` table and the
  applied-set query. The interface is one function:
  `Migrator.Apply(ctx, []Migration) ([]Result, error)`.
- **Checksum**: SHA-256 of the `up` body after whitespace
  normalization (collapse runs of whitespace, trim, lowercase
  keywords? — actually no: just verbatim with leading/trailing
  whitespace trimmed; users want surprising-edits caught).
- **CLI**: `cmd/wave/migrate.go` wires `status`, `up`, `down`,
  `verify`, `new`, `force-update`.
- **Testing hooks**: `wave test --roundtrip-migrations` applies →
  rolls back → re-applies → schema-snapshot diff (via
  `PRAGMA table_info` / `information_schema`).

The feature touches only the `infra/` layer plus the orchestrator's
boot sequence — no changes to route handling, plugins, or auth.

---

## See also

- [`wave-capy-complete.md`](./wave-capy-complete.md) — every other
  declaration shape.
- [`wave-capy-templating.md`](./wave-capy-templating.md) — `{{secret …}}`
  and helpers work in `up`/`down` bodies too.
- [`wave-from-zero-to-complex.md`](./wave-from-zero-to-complex.md) —
  see the new "Schema migrations" section integrated into the build-up.
- [`wave-cli.md`](./wave-cli.md) — `wave migrate` subcommand surface.
