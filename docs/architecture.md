# Architecture

## Scope

This template defines the executable command boundary:

- construct a Cobra root command;
- parse the `serve` and `migrate` subcommands and their flags;
- resolve typed configuration;
- validate that configuration;
- invoke an injected function that starts the server.

It also selects Gin for HTTP routing and defines the first feature boundary:

- `internal/server` owns complete HTTP runtime startup and cleanup, constructs
  the Gin engine, and owns top-level paths;
- `internal/http/errors` owns failed-response classification and representation;
- `internal/http/middleware` owns global error handling, recovery, and request
  completion logging;
- `internal/telemetry` owns the OpenTelemetry SDK provider, W3C propagation,
  trace-context access, and shutdown;
- a feature package owns its handlers and relative route registration;
- route composition is explicit and constructor-driven;
- `GET /healthz` reports process liveness.
- `POST /users` demonstrates a complete Handler -> Service -> Repository slice.

GORM with SQLite is the selected database adapter. `internal/database` owns
connection creation, connectivity verification, and close. Features use
Handler -> Service -> Repository as their default request flow. The template
includes one User creation domain, explicit migrations, and no transaction
abstraction for its single database write.

## Execution Flow

```text
cmd/<binary>/main.go                  executable boundary
        |
        v
command.NewRootCommand
        |
        v
command.NewServeCommand              parses config, address, and database flags
        |
        v
options.Serve.Complete               resolves sources into config.Config
        |
        v
options.Serve.Validate               checks command-level configuration
        |
        v
options.Serve.Run                    invokes server.Run
        |
        v
server.Run                            constructs and cleans up the HTTP runtime
        |
        +--> database.Open           selects, opens, pings, and closes the database
        |
        +--> telemetry.New           creates the non-exporting SDK provider
        |
        v
server.New                           constructs Gin and top-level routes
        |
        v
server.ListenAndServe                owns TCP listen and graceful HTTP shutdown
        |
        v
HTTP middleware                      tracing, global errors, recovery, logging
        |
        v
health.RegisterRoutes                registers GET /healthz
user.RegisterRoutes                  registers POST /users
```

The executable creates a signal context for SIGINT and SIGTERM, then executes
the root command. The command initiates server startup by executing its injected
`options.Serve` lifecycle. `internal/server.Run` composes the concrete database,
telemetry provider, health handler, and router; it owns their cleanup after the
server stops. `internal/server` also owns TCP listen and graceful HTTP shutdown.
The independent `migrate` lifecycle resolves the same typed configuration,
validates only database settings, and invokes `internal/migration.Run`; it never
starts the HTTP runtime.

## Module Boundaries

| Module or seam | Owns | Must not own |
| --- | --- | --- |
| `cmd/<binary>` | signal context, root execution, exit code | flag definitions, configuration keys, server behavior or dependency construction |
| `internal/command` | Cobra tree, flags, prepared `config.Options`, lifecycle invocation | configuration resolution, server implementation |
| `internal/config` | source precedence, key mapping, defaults, typed decoding and validation | command execution, server startup |
| `cmd/<binary>/options` | command-specific completion, validation, and runner dispatch for serve and migrate | Cobra and pflag parsing, Gin middleware, persistence lifecycle |
| `internal/database` | GORM SQLite open, ping, handle access, connection close | models, migrations, repository or transaction policy |
| `internal/migration` | explicit database migration runtime and registered feature migrations | HTTP startup, automatic discovery |
| `internal/telemetry` | SDK tracer provider, service resource, W3C propagation, trace/span lookup, shutdown | Gin, HTTP response policy, exporter configuration |
| `internal/server` | runtime dependency construction and cleanup, Gin engine, global middleware, top-level paths, feature route composition | feature behavior, product rules |
| `internal/server.Run` | runtime startup and cleanup | command parsing, feature behavior, product rules |
| `internal/server.ListenAndServe` | TCP listen, HTTP server lifecycle, bounded graceful shutdown | feature behavior, persistence lifecycle |
| `internal/apperr` | application error meanings and cause preservation | HTTP status, response serialization, logging policy |
| `internal/http/errors` | application-error-to-HTTP classification and safe response types | Gin middleware, logging, feature behavior |
| `internal/http/input` | bounded JSON, XML, and form decoding | domain validation, success responses |
| `internal/http/middleware` | global error response, span error recording, recovery, completion logging | provider construction, feature behavior, product rules |
| `internal/health` | liveness handler and relative route registration | server lifecycle, external dependency readiness |
| `internal/user` | example User HTTP, use case, invariants, persistence, and migration | server lifecycle, configuration resolution |

## Dependency Direction

The command and configuration dependency is:

```text
cmd/<binary> -> internal/command -> internal/config
              \-> cmd/<binary>/options -> internal/config
```

The HTTP routing dependency is:

```text
internal/server -> internal/telemetry -> OpenTelemetry SDK
             \-> internal/database -> GORM -> SQLite
             \-> internal/user -> GORM
                              \-> internal/http/input
             \-> otelgin
             \-> internal/http/middleware
             \-> internal/health
internal/migration -> internal/database
                  \-> internal/user
internal/http/middleware -> internal/http/errors -> internal/apperr -> Trace v3
internal/http/middleware -> internal/telemetry -> OpenTelemetry API
```

The command package does not import server or migration implementations. Feature packages
do not import `internal/server`; the server composes them from the outside.
`internal/database` does not import `internal/config`; composition passes the
resolved DSN into `database.Open`.

## Data Flow

```text
process arguments
  -> Cobra and pflag parsing
  -> options.Serve.Complete
  -> options.Serve.Validate
  -> options.Serve.Run(ctx)
  -> returned error or clean exit
```

An error flows back to the executable boundary that decides the exit code.

The implemented HTTP data flow is:

```text
GET /healthz
  -> Gin engine
  -> HTTP middleware
  -> health.Handler.Check
  -> HTTP 200 {"status":"ok"}

POST /users
  -> strict JSON input
  -> user.Handler.Create
  -> user.Service.Create
  -> user.Repository.Create
  -> SQLite INSERT
  -> HTTP 201 + Location
```

The default feature request flow is:

```text
HTTP request -> Handler -> Service -> Repository -> external adapter
                                \-> capability package -> external library
```

Dependencies point in the same direction. Repositories must not import services
or handlers, and services must not import handlers. A feature may omit a layer
when it has no responsibility for that layer; the health liveness endpoint has
no application orchestration or persistence and therefore terminates at its
handler. Reusable token, password, sealing, encoding, or similar behavior lives
in a package named for that capability rather than a generic utility package.

## Constructor Injection

Constructors receive dependencies explicitly and return concrete types by
default. Each feature package deliberately exposes one cohesive `Service`
interface containing the application use cases its Handlers consume. A
persisted feature also exposes one cohesive `Repository` interface containing
the persistence operations its Service consumes. `NewService` and
`NewRepository` return those contracts; unexported `service` and `repository`
structs implement them. The composition root constructs implementations and
injects them into these seams.

Inject nondeterministic operations such as time, random values, and generated
identifiers when tests need deterministic behavior. Prefer a focused function
dependency when only one operation varies; the User Service receives its UUID
generator this way.

Configuration structs, plain values, `*slog.Logger`, `*gorm.DB`, and other stable
infrastructure types remain concrete unless a demonstrated alternative behavior
requires a seam. Do not wrap them in pass-through interfaces, and do not create
a common service, repository, dependency, or module interface.

## Ports and Adapters Evolution

The template remains a feature-oriented modular monolith. Its existing
boundaries are compatible with ports and adapters without imposing a global
Hexagonal Architecture directory tree:

| Role | Current seam |
| --- | --- |
| inbound adapter | feature HTTP Handler |
| application/use-case core | feature Service |
| outbound persistence port | feature Repository contract |
| persistence adapter | unexported GORM Repository implementation |
| composition | `internal/server.Run` and `internal/server.New` |

Keep these roles inside the owning feature. Do not create repository-wide
`domain`, `application`, `ports`, or `adapters` directories merely to mirror an
architecture diagram.

When a feature gains a demonstrated external boundary such as a message broker,
remote API, clock, or alternative persistence implementation:

1. define the narrow interface the consuming Service actually needs;
2. keep transport SDK types and errors inside the adapter;
3. inject the adapter through the composition root;
4. test Service behavior through the port and test the real adapter separately.

Do not add a common dependency container, service locator, or adapter registry.
An interface is justified by the external boundary or required variation, not
by the possibility that one might appear later.

### Transactions and Distributed Consistency

A same-database use case with multiple writes that must succeed or fail together
defines its transaction boundary at the Service/use-case level. Add a Unit of
Work or transaction port only when that concrete atomicity contract exists.
Keep transaction state out of `context.Context` and out of shared mutable
Repository or Unit of Work instances.

Outbox is introduced only when a committed domain change must reliably produce
an asynchronous message. The domain write and Outbox record must be persisted
in the same database transaction; dispatch, retry, idempotency, and delivery
semantics require explicit contracts and integration tests.

Saga is introduced only for a multi-system workflow that cannot share one local
transaction. Its application orchestrator requires durable workflow state,
idempotent steps, compensation behavior, retries, timeouts, and observable
terminal states. Participant calls remain behind consumer-owned ports.

A local Unit of Work cannot make database writes and remote API or broker calls
atomic. Do not hold a database transaction open across a remote call as a
substitute for Outbox, Saga, or another explicitly selected consistency model.

## HTTP Route Composition

`internal/server` owns the Gin engine, global middleware, and top-level route
paths. Each feature receives a `*gin.RouterGroup` and registers only its relative
paths:

```go
group.GET("", handler.Check)
```

Handlers receive dependencies through `New...` constructors. Registration is
explicit: do not use package `init` functions, global registries, automatic
discovery, or a common module interface unless a concrete requirement justifies
that additional seam. Feature packages do not import shared middleware to
register routes; the server installs global HTTP policy before composing
features.

Gin's `*gin.Context` stays inside HTTP handlers. Application behavior receives
typed inputs and `context.Context`.

Shared HTTP errors, trace correlation, logging, and recovery follow
[HTTP API Contract](http.md).

## Telemetry

`internal/server.Run` constructs one `telemetry.Provider`, passes it to
`server.New`, and calls `Shutdown` during server cleanup with a fresh bounded
context. The provider is an SDK tracer provider rather than the OpenTelemetry
no-op provider so every HTTP request receives a valid trace ID and span ID.

Export is disabled by default: the provider uses a never-sample policy and has
no span processor or exporter. It still extracts and propagates W3C
`traceparent`, `tracestate`, and baggage. Exporter protocol, endpoint,
credentials, retry, batching, and sampling configuration remain a separate
decision.

`internal/server` installs `otelgin` with the injected provider and propagator.
`internal/telemetry` does not import Gin or mutate OpenTelemetry globals.

## Health Check

`GET /healthz` is a liveness endpoint. It returns HTTP 200 and
`{"status":"ok"}` while the process can serve HTTP requests. It deliberately
does not check databases, queues, or other dependencies; add a separate
readiness contract if deployment requirements need one.

## Database Connection

`internal/database.Open(ctx, opts)` selects an implemented GORM dialector,
opens the database, retrieves the underlying standard-library connection pool,
and verifies it with `PingContext`. The returned concrete connection exposes
`DB()` for repository construction and owns `Close()`.

`internal/server.Run` maps `config.Config.Database.Driver` and
`config.Config.Database.DSN` into `database.Options`, passes it to `Open`, and
closes the result. HTTP handlers and application behavior do not open database
connections. `internal/migration.Run` opens its own bounded connection for the
explicit `migrate` command. User schema, Repository, and deferred transaction
policy are documented in [Database](database.md).

## Command Construction

Commands are created with constructors and receive dependencies explicitly:

```go
logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
serveOptions := options.NewServe(func(ctx context.Context, cfg config.Config) error {
	return server.Run(ctx, cfg, logger)
})
migrateOptions := options.NewMigrate(migration.Run)
root := command.NewRootCommand(
	server.ApplicationName,
	config.Options{},
	serveOptions,
	migrateOptions,
)
```

Do not use package `init` functions to register commands. Constructor-based
assembly keeps command tests isolated and startup dependencies visible.

## Adding a Subcommand

1. Define the command's user-visible contract and tests.
2. Create `internal/command/<name>.go` with a `New<Name>Command` constructor.
3. Keep argument parsing and command-level configuration checks at this boundary.
4. Inject the operation the command starts.
5. Register the command in `NewRootCommand`.

## Automated Boundary Check

Run the dependency-free AST checker from the module root:

```sh
go run ./scripts/architecture
```

It parses production Go files and rejects:

- Viper outside `internal/config`;
- Cobra outside `internal/command`;
- pflag outside `internal/command` and `internal/config`;
- Gin, GORM, or Viper imports from `service*.go`;
- persistence or configuration imports from `handler*.go`;
- transport, configuration, or Server imports from `repository*.go`;
- feature imports of `internal/server`;
- Gin imports from `internal/telemetry`;
- configuration imports from `internal/database`;
- feature `routes.go` without a matching `RegisterRoutes` call in
  `internal/server/router.go`;
- feature `migration.go` without a matching `Migrate` call in
  `internal/migration/migration.go`.

The checker excludes `_test.go` files so integration tests may cross production
boundaries. Role checks follow the repository's `service*.go`, `handler*.go`,
and `repository*.go` naming convention. This static check proves import
ownership and explicit registration only; it does not prove runtime data flow,
error semantics, authorization, or transaction correctness.

## Deliberately Undecided

The derived project chooses these only when requirements exist:

- additional feature services, repositories, models, and migrations;
- multi-write transaction mechanisms and consistency rules;
- Outbox delivery and Saga orchestration semantics;
- authentication, authorization, and non-recovery middleware;
- readiness checks beyond process liveness.
