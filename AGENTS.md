# Repository Guide for Coding Agents

This repository is an opinionated Go HTTP service template. Preserve its
explicit command, configuration, and HTTP routing boundaries and its single
example User domain without inventing additional product behavior.

## Read Before Changing Code

1. Read [docs/development.md](docs/development.md) for Go and implementation
   conventions.
2. Read [docs/architecture.md](docs/architecture.md) for module boundaries,
   dependency direction, and data flow.
3. Read [docs/feature-development.md](docs/feature-development.md) before adding
   or expanding an HTTP feature.
4. Read [docs/http.md](docs/http.md) before changing handlers, middleware,
   request logging, recovery, or error responses.
5. Read [docs/domain.md](docs/domain.md) only when a change introduces product
   behavior.
6. Read [docs/configuration.md](docs/configuration.md) before adding or changing
   configuration.
7. Read [docs/database.md](docs/database.md) before adding persistence behavior.
8. Read nearby production code and tests before proposing a new pattern.

## Architecture Rules

- Keep `cmd/<binary>/main.go` as an executable boundary. It may create a signal
  context, construct the single process logger, execute the root command, and
  choose an exit code. Put no other application behavior or server dependency
  construction there.
- Keep Cobra command construction and pflag definitions in `internal/command`.
  `internal/config` may accept a prepared `*pflag.FlagSet` only to bind flags
  into its local Viper instance.
- Keep command-specific `Complete`, `Validate`, and `Run` state in
  `cmd/<binary>/options`; it receives prepared `config.Options` and must not
  import Cobra or pflag.
- Keep Viper usage in `internal/config`. Decode once into `config.Config`;
  application packages must not import Viper or query string keys.
- Construct commands with `New...Command` functions. Do not register commands
  through package `init` functions or package-level mutable state.
- Command code owns parsing and invokes a consumer-owned command lifecycle.
  `cmd/<binary>/options` owns configuration completion, validation, and runner
  dispatch; `internal/server.Run` owns server behavior and
  `internal/migration.Run` owns explicit schema migration.
- Keep complete HTTP runtime startup, Gin engine construction, global middleware,
  top-level route paths, and runtime cleanup in `internal/server`.
- Keep OpenTelemetry SDK provider construction, W3C propagation, trace-context
  access, and shutdown in `internal/telemetry`. Do not import Gin there.
- `internal/server.Run` constructs one `telemetry.Provider`, injects it into the
  router, and closes it with `Shutdown`. Export remains disabled unless an
  exporter is explicitly designed and configured.
- Keep application-error-to-HTTP classification and safe response types in
  `internal/http/errors`.
- Keep bounded JSON, XML, URL-encoded form, and multipart form decoding in
  `internal/http/input`. Request DTOs remain private to feature handlers.
- Keep global error responses, recovery, OpenTelemetry error recording, and
  request completion logging in `internal/http/middleware`.
- Install middleware in this order: `otelgin`, request logging, error handling,
  recovery, handlers.
- Let each feature package own its HTTP handlers and `RegisterRoutes` functions.
  Route registration receives a `*gin.RouterGroup` and registers relative paths.
- Register feature handlers directly as Gin handlers. A handler owns its
  success response; on failure it attaches the trace error with `c.Error`,
  aborts, and returns without logging or writing an error response.
- Keep `*gin.Context` inside HTTP handlers. Pass typed values and
  `context.Context` into application behavior.
- Use Handler -> Service -> Repository as the default feature request flow.
  Omit a layer only when the feature has no corresponding responsibility.
- Compose feature routes explicitly. Do not use package `init`, global
  registries, or a common module interface without a demonstrated need.
- Constructors receive dependencies explicitly. Return concrete types by
  default; a feature's `NewService` and `NewRepository` return their deliberate
  layer contracts.
- Let each feature package expose one cohesive `Service` interface containing
  the use cases its Handlers consume. Keep the implementing `service` struct
  unexported and inject the interface into Handlers.
- Let each persisted feature expose one cohesive `Repository` interface
  containing the persistence operations its Service consumes. Keep the GORM
  implementation as unexported `repository` and inject the interface into
  `service`.
- Keep configuration, plain values, loggers, GORM handles, telemetry providers,
  and other stable infrastructure types concrete unless demonstrated variation
  requires a seam. Do not add pass-through interfaces.
- Add other interfaces only at a proven variation or external-system boundary.
  Prefer concrete types inside a module.
- Preserve feature-oriented modular-monolith packaging. Treat Handler, Service,
  Repository contracts, persistence implementations, and composition as
  ports-and-adapters-compatible seams; do not reorganize the repository into
  global `domain`, `application`, `ports`, or `adapters` trees solely to claim
  Hexagonal Architecture.
- When a feature gains a real broker, external API, clock, or alternative
  persistence dependency, define only the narrow consumer-owned port its
  Service needs, implement the adapter at that boundary, and wire it in the
  composition root.
- Do not scaffold Unit of Work, Outbox, or Saga packages before an explicit
  consistency contract exists. Outbox requires atomic domain and event
  persistence; Saga requires durable state, idempotent steps, compensation,
  retry, and timeout rules.
- Inject nondeterministic operations such as time, random values, and generated
  identifiers when deterministic behavior is required in tests. Prefer a
  focused function dependency when one operation is sufficient.
- Do not create generic `util`, `common`, or `helpers` packages. Put reusable
  behavior in capability packages named for what they own, such as token,
  password, or sealing behavior.
- Use `github.com/tae2089/trace/v3` at every returning function boundary that adds
  meaningful operation context, including repositories, services, token
  handling, cryptography, encoding, and external adapters. Do not wrap solely
  to repeat a function or package name.
- Define application error meanings in `internal/apperr` where the meaning is
  known. Keep HTTP status, response code, and public-message mapping in
  `internal/http/errors`; use Trace only for origin and propagation diagnostics.
- Never include tokens, passwords, keys, plaintext, ciphertext, signatures, or
  other credential material in error messages or diagnostic logs.
- Avoid named return values. Preserve multiple operation and cleanup failures
  with explicit local variables and `errors.Join`.
- Before adding authentication, define separate application meanings for
  invalid or expired credentials and for authorization failure. Map them to 401
  and 403 respectively only in `internal/http/errors`.
- Keep GORM and the SQLite driver in `internal/database` and feature persistence
  adapters. `internal/server.Run` owns `database.Connection.Close`.
- Apply schema changes only through `app migrate`. `serve`, Repository
  constructors, and package initialization must never migrate implicitly.
- Register every idempotent feature migration explicitly in
  `internal/migration.Run`; do not use automatic discovery.
- Accept a `database.driver` value only when its GORM dialector and integration
  test are included; do not infer the driver from `database.dsn`.
- Do not open a database from HTTP handlers or application behavior. Inject
  persistence dependencies through constructors.
- Do not add models, migrations, transaction policy, or database readiness
  checks without an explicit contract and tests.
- Do not add feature-specific interfaces, domain behavior, models, migrations,
  or transaction policy before their requirements are known.

## Domain Rules

- [docs/domain.md](docs/domain.md) is the canonical source for terms, states,
  transitions, invariants, allowed behavior, and forbidden behavior.
- The example User creation slice is the only selected product domain. Do not
  infer update, deletion, authentication, authorization, or additional User
  behavior.
- Domain documentation does not select a package structure or server
  architecture.
- If a required rule is absent or contradictory, request a product decision
  before implementing the behavior.
- Domain rules require tests for valid behavior and every documented rejection.

## Configuration Rules

- The required precedence is `Set > flag > environment > config file > KV store
  > default`.
- YAML is the canonical configuration-file format.
- Every setting has one dotted canonical key. Its YAML path, environment
  variable, flag, typed Go field, default, and KV shape must stay aligned with
  [docs/configuration.md](docs/configuration.md).
- A flag that participates in multi-source resolution must use an empty CLI
  default; the actual default belongs to `internal/config`.
- The `serve` command must call `Config.Validate` after loading and before
  invoking its runner.
- The `migrate` command validates database configuration only; it must not
  require a valid or relevant server address.
- A new setting is incomplete until its precedence tests and configuration
  validation tests and documentation are updated.

## Development Rules

- Follow naming, formatting, import, error, logging, concurrency, and test
  conventions in [docs/development.md](docs/development.md).
- Use Red → Green → Refactor for behavior changes and defect fixes.
- Keep command tests focused on parsing and dispatch. Test configuration
  precedence and validation in `internal/config`, and server dispatch through
  the runner seam.
- Run `gofmt` on changed Go files and `make verify` before declaring work
  complete.
- Keep structural and behavioral changes separate when committing.
