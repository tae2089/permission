# Database

## Selected Stack

The template uses GORM with the official SQLite driver:

- `gorm.io/gorm`
- `gorm.io/driver/sqlite`

SQLite is the only supported driver. Driver selection is explicit so another
implemented GORM dialector can be added without inferring a driver from its DSN.
PostgreSQL and MySQL are not accepted until their dependencies and integration
tests exist.

The official SQLite driver uses `github.com/mattn/go-sqlite3`, which requires
CGO and a C compiler. Build and CI environments must enable CGO.

## Configuration

`config.Config.Database.Driver` selects the database implementation and
`config.Config.Database.DSN` contains that driver's connection string. Their
complete source mapping and precedence contract live in
[Configuration](configuration.md). The defaults select SQLite and create
`go-template.db` relative to the process working directory when opened.

Do not log the DSN. SQLite DSNs are normally paths, but treating DSNs as
sensitive avoids exposing credentials if another driver is introduced later.

## Connection Ownership

`internal/database.Open(ctx, opts)`:

1. selects the GORM dialector from `opts.Driver`;
2. rejects an unsupported driver or empty DSN;
3. opens the database through GORM;
4. obtains the underlying `*sql.DB`;
5. verifies connectivity with `PingContext`;
6. returns a `database.Connection` that owns close.

GORM opens with `TranslateError` enabled and its internal logger disabled.
Feature repositories may therefore inspect portable errors such as
`gorm.ErrDuplicatedKey`, but must verify each semantic translation with the
selected real database driver. Database errors flow to the application boundary
instead of being logged by GORM, preventing interpolated SQL values from
bypassing the application's structured logging and sensitive-data policy.

`internal/server.Run` opens and closes the connection as part of HTTP runtime
startup and cleanup:

```go
connection, err := database.Open(ctx, database.Options{
	Driver: database.Driver(cfg.Database.Driver),
	DSN:    cfg.Database.DSN,
})
if err != nil {
	return err
}

db := connection.DB()
// Construct repositories and run the server with db.

if err := connection.Close(); err != nil {
	return err
}
```

Repositories may receive the returned `*gorm.DB`. HTTP handlers and application
behavior must not open connections or read `config.Config` directly.

`database.Open` is the driver factory. It switches to an implemented GORM
dialector and does not define a parallel custom driver interface. To add a
driver, add its GORM dependency, factory case, accepted configuration value, and
real integration test in the same change.

## Migration Execution

Schema changes are explicit:

```sh
go run ./cmd/app migrate --config config.example.yaml
```

`internal/migration.Run` opens the configured database, applies every registered
feature migration, and closes the connection. The `migrate` command validates
database configuration only. `serve` never calls `AutoMigrate` and never changes
the schema.

The example User migration is `user.Migrate(ctx, db)`. It is idempotent and
creates the `users` table with:

| Column | Contract |
| --- | --- |
| `id` | UUID string primary key |
| `name` | non-null normalized User name |
| `email` | non-null normalized User email with a unique index |

The Project bootstrap migration is `project.Migrate(ctx, db)` and creates the
`projects` table. The shared management-audit migration is
`audit.Migrate(ctx, db)` and creates `audit_events`. Both are idempotent:

| Table | Important columns | Contract |
| --- | --- | --- |
| `projects` | `id`, `name`, `created_at` | UUID primary key, non-null name and creation timestamp |
| `audit_events` | `id`, `occurred_at`, `action`, `actor`, `project_id`, `target_id` | immutable management-audit row; `project_id` is indexed and `target_id` identifies the changed resource |
| `api_keys` | `id`, `project_id`, `kind`, `status`, `secret_hash`, `created_at`, `revoked_at` | opaque Project-scoped key; the hash is unique and no raw secret is persisted |

`audit.Writer` owns the common event-to-record mapping. A feature Repository
receives the concrete Writer and calls `Append` with its active transaction.
`project.Repository.Create` therefore persists the Project and its audit event
in one GORM transaction. If either insert fails, the transaction rolls back.
The Service defines the paired-write requirement; persistence adapters own the
GORM transaction so GORM remains outside application code.

A new feature with persisted records must expose an idempotent migration and
register it explicitly in `internal/migration.Run`. Do not use package `init`,
automatic discovery, or migration as a side effect of Repository construction.

## Repository Rules

- A feature exposes a `Repository` interface consumed by its Service.
- `NewRepository` returns that contract; an unexported `repository`
  implementation receives `*gorm.DB` and owns GORM.
- Services must not import GORM.
- Every query uses `db.WithContext(ctx)`.
- Persistence records remain private to the feature Repository and are mapped
  to and from feature entities.
- Select columns and mutation conditions explicitly when partial updates are
  introduced. Do not use `Save` or `FirstOrCreate` as a default.
- Enforce uniqueness with a database constraint, not a check-then-insert race.
- Translate a known duplicate constraint to
  `apperr.Wrap(apperr.KindAlreadyExists, err, "user already exists")`; wrap
  unknown database failures with `trace.Wrap` operation context.
- Do not log an error that is returned to the Service.
- Do not put domain behavior in GORM hooks or callbacks.

The example Create User use case performs one insert and requires no custom
transaction seam. A multi-write use case must define its atomicity contract at
the Service/use-case level while keeping `*gorm.DB` inside persistence adapters.
Do not smuggle a transaction through `context.Context`.

## Deliberately Undecided

The template does not yet define:

- connection-pool tuning;
- database readiness checks.

`GET /healthz` remains dependency-free liveness. Add a separate readiness
contract before making deployment health depend on SQLite.
