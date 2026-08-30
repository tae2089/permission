# Permission Service Foundation

An opinionated Go foundation for a multi-project authorization service. It lets
instance administrators create Project isolation boundaries, manage
Project-scoped API keys, and record management changes in a durable audit log.

This is an early authorization slice, not a complete permission system. It does
not yet define Tenants, Principals, Roles, Policies, or permission-decision
evaluation, and it does not provide end-user login, password, or OIDC
authentication. The User vertical slice remains an implementation example.

The service uses a Cobra command tree with `serve` and `migrate` subcommands.
Configuration is resolved by Viper into typed Go values before application code
runs.

## Implementation Principles

- Thin executable boundary
- Constructor-based Cobra commands
- Typed configuration boundary
- Gin HTTP routing with feature-owned registration
- Application-owned semantic errors, Trace v3 diagnostics, and safe HTTP responses
- Global Gin error handling for every registered route
- OpenTelemetry trace correlation with export disabled by default
- Structured `slog` request completion logging
- Explicit route composition without global registries
- GORM with an explicitly owned SQLite connection
- User creation vertical slice as an implementation example
- Project bootstrap with instance-administrator Bearer authentication and
  transactional audit records
- Explicit database migrations
- AST-enforced architecture boundaries
- No global mutable Cobra or Viper state
- Test-enforced configuration precedence

See [Feature development](docs/feature-development.md),
[Architecture](docs/architecture.md), [Configuration](docs/configuration.md),
[HTTP API Contract](docs/http.md), and [Database](docs/database.md) before
extending the service.

## Documentation

| Document | Canonical responsibility |
| --- | --- |
| [Agent guide](AGENTS.md) | Required reading order and non-negotiable repository rules |
| [Feature development](docs/feature-development.md) | Contract-first vertical-slice implementation workflow and completion gates |
| [Development conventions](docs/development.md) | Naming, formatting, imports, errors, logging, concurrency, and tests |
| [Architecture](docs/architecture.md) | Module boundaries, dependency direction, and data flow |
| [HTTP API contract](docs/http.md) | Handler errors, response envelope, trace context, recovery, and request logging |
| [Database](docs/database.md) | GORM/SQLite selection, migrations, Repository rules, and connection ownership |
| [Domain contract](docs/domain.md) | Terms, states, transitions, invariants, allowed and forbidden behavior |
| [Configuration](docs/configuration.md) | Source precedence, key mapping, formats, and extension procedure |

## Repository Layout

```text
cmd/                  executable composition roots
cmd/app/options/      command-specific serve and migrate lifecycle
internal/command/     CLI parsing, configuration loading, and dispatch
internal/config/      configuration resolution and typed values
internal/database/    GORM SQLite connection creation and lifecycle
internal/health/      liveness HTTP handler and route registration
internal/http/errors/ HTTP error classification and safe response types
internal/http/input/  bounded JSON, XML, and form request decoding
internal/http/middleware/ Global errors, recovery, tracing, and logging
internal/audit/       shared management-audit event persistence
internal/migration/   explicit feature schema migration runtime
internal/server/      Gin engine, middleware, and top-level route composition
internal/telemetry/   OpenTelemetry provider, propagation, and lifecycle
internal/user/        example Handler-Service-Repository vertical slice
internal/project/     Project bootstrap and atomic audit integration
internal/apikey/      Project-scoped API-key issuance, rotation, revocation, and audit integration
scripts/architecture/ dependency and explicit-registration architecture checker
docs/                 development, architecture, domain, and configuration guidance
```

`cmd/app/main.go` creates the signal context and the process logger, then
executes the command. `internal/server.Run` receives the injected logger,
creates the database connection, telemetry provider, health handler, and
router, then starts the HTTP server. SIGINT and SIGTERM
trigger graceful HTTP shutdown.

## User Example Slice

`POST /users` accepts:

```json
{"name":"Alice","email":"alice@example.com"}
```

It returns `201 Created`, `Location: /users/<uuid>`, and no body. Name and email
normalization, duplicate handling, and allowed behavior are defined in
[Domain Rules](docs/domain.md).

## Health Check

`GET /healthz` returns HTTP 200 with:

```json
{"status":"ok"}
```

This endpoint reports process liveness only. It does not check databases,
queues, or other external dependencies.

## Project Bootstrap

`POST /v1/projects` creates a Project when called with the instance
administrator credential. `GET /v1/projects` lists them. Both endpoints require
`Authorization: Bearer <instance administrator key>`. Project creation writes a
`project.created` audit event in the same database transaction.

```json
{"name":"billing-api"}
```

Names must match `^[a-z0-9-]{1,63}$`.

## Project API Keys

An instance administrator can issue an initial `project_admin` key. That key
then manages `project_admin` and `decision` keys only for its own Project.
Raw secrets are returned once, while only verification hashes are stored.

## Configuration

The canonical file format is YAML:

```yaml
serve:
  address: ":8080"
database:
  driver: "sqlite"
  dsn: "go-template.db"
```

Copy [config.example.yaml](config.example.yaml) when creating a local
configuration file. The complete source mapping and precedence contract live in
[docs/configuration.md](docs/configuration.md).

Apply the schema, then run the service locally:

```sh
go run ./cmd/app migrate --config config.example.yaml
GO_TEMPLATE_INSTANCE_ADMIN_KEY=<secret> go run ./cmd/app serve --config config.example.yaml
```

## Verification

Go 1.25.13 or newer is required. The module pins `govulncheck` as a Go tool
dependency, so no separate global installation is needed.

```sh
make verify
```

The target checks formatting without rewriting files, runs the architecture
checker, executes uncached tests with and without the race detector, runs
`go vet`, and rejects reachable known vulnerabilities.

The module path currently retains its template origin:
`github.com/tae2089/go-template`. Update it when this repository adopts its
final module path.
