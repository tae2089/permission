# Development Conventions

This document is the canonical source for Go code style and implementation
conventions in this repository.

## Naming

- Package names are short, lowercase, single words that describe one concept.
  Avoid concatenated layer or protocol names and generic packages such as
  `util`, `common`, `base`, or `helpers`.
- Name reusable capability packages after the behavior they own. Prefer names
  such as `token`, `password`, or `seal` over names that merely say the code is
  shared or cryptographic.
- A package name matches its leaf directory. Parent directories may group
  related packages without containing Go files, for example
  `internal/http/errors` and `internal/http/middleware`.
- Split distinct responsibilities into scoped leaf packages instead of
  inventing a compound package name.
- File names are lowercase snake case and describe their primary responsibility,
  for example `serve.go` and `serve_test.go`.
- Exported names communicate repository concepts. Do not expose a name only to
  make a test possible.
- Initialisms remain consistently capitalized: `ID`, `HTTP`, `URL`, `API`.
- Boolean names describe a positive fact such as `Enabled`, `Ready`, or
  `Exists`.
- Constructors use `New<Type>`. Cobra command constructors use
  `New<Name>Command`.
- Receiver names are short and consistent for a type. Do not use `this` or
  `self`.
- Sentinel errors use `Err<Name>`. Custom error types use `<Name>Error`.
- Avoid repeating the package name in exported identifiers.

## Formatting and Imports

- Run `gofmt` on every changed Go file.
- Imports are grouped in this order, with one blank line between groups:

```go
import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tae2089/go-template/internal/config"
)
```

1. Go standard library
2. Third-party modules
3. Packages from this repository

- Keep imports lexicographically sorted within each group.
- Do not use dot imports. Use blank imports only when registration by side effect
  is the documented contract of a dependency.
- Prefer the standard library before adding a dependency.

## Functions and Types

- Keep the common path readable without knowing rare-case machinery.
- Accept `context.Context` as the first parameter of context-aware functions.
  Do not store a context in a struct.
- Constructors receive dependencies explicitly. Return concrete types by
  default; a feature's `NewService` and `NewRepository` return their deliberate
  layer contracts.
- Each feature package exposes one cohesive `Service` interface containing the
  use cases its Handlers consume. Its unexported `service` struct implements
  that contract, and Handlers receive the interface.
- Each persisted feature exposes one cohesive `Repository` interface containing
  the persistence operations its Service consumes. Its unexported `repository`
  struct implements that contract and owns GORM.
- Name an interface for the role or behavior the consumer needs and include only
  the methods it calls. Do not mirror every method of the concrete provider.
- Keep configuration structs, plain values, and stable infrastructure types
  such as `*slog.Logger` and `*gorm.DB` concrete. Do not create an interface that
  merely forwards another concrete type.
- Prefer concrete types inside a module. Add another interface only for a proven
  variation or external-system boundary.
- Avoid package-level mutable state and hidden dependency construction.
- Inject nondeterministic operations such as time, random values, and generated
  identifiers when deterministic tests need control. Prefer a function
  dependency when one operation is sufficient.
- Return observable values where practical instead of scattering side effects.

```go
type Repository interface {
	Create(context.Context, User) error
}

type repository struct {
	db *gorm.DB
}

type Service interface {
	Create(context.Context, CreateInput) (User, error)
}

type service struct {
	repository Repository
	newID      func() (uuid.UUID, error)
}

type Handler struct {
	service Service
}
```

The exported `Service` and `Repository` interfaces are the feature's use-case
and persistence contracts. Their unexported structs own implementation details;
only `repository` imports GORM. UUID generation is injected as one focused
function instead of requiring monkey patching or a broad factory interface.

## Error Handling

- Check every returned error.
- Return errors instead of panicking. Panic is reserved for an unrecoverable
  programmer error during process initialization.
- Avoid named return values. Keep operation and cleanup errors in explicit local
  variables and preserve both with `errors.Join` when both can fail.
- Use `github.com/tae2089/trace/v3` only to record where an error originated and
  which returning boundaries added meaningful operation context.
- Define application error meanings in `internal/apperr`. HTTP status, public
  code, and response-message policy remain in `internal/http/errors`.
- Create an application semantic error where its meaning becomes known. Every
  returning function that contributes useful operation context then uses Trace
  while preserving that cause. This applies equally to repositories, services,
  token/password/sealing packages, encoders, clients, and other adapters:

```go
func (r *repository) Find(ctx context.Context, id string) (*User, error) {
	user, err := r.find(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.New(apperr.KindNotFound, "user not found")
	}
	if err != nil {
		return nil, trace.Wrap(err, "query user")
	}
	return user, nil
}

func (s *service) Get(ctx context.Context, id string) (*User, error) {
	user, err := s.repository.Find(ctx, id)
	if err != nil {
		return nil, trace.Wrap(err, "get user")
	}
	return user, nil
}
```

- Wrap once at each boundary that contributes useful operation meaning; do not
  add generic wrappers that only repeat a function or package name.
- A capability package translates a dependency-specific error to `apperr` only
  when it knows the stable application meaning. Otherwise it returns
  `trace.Wrap(err, "<operation>")` and leaves classification to an informed
  caller.
- Error strings are lowercase and have no trailing punctuation.
- Use `trace.Wrap` for propagation context and `apperr.Wrap` when assigning an
  application meaning while preserving an underlying cause.
- Use sentinel or typed errors only when callers need `errors.Is` or `errors.As`
  branching.
- Do not both log and return the same error. Return it until the boundary that
  can make the final retry, response, or process-exit decision.
- HTTP handlers attach the wrapped error to Gin. The global error middleware
  inspects the complete chain and is the only component that renders a failed
  HTTP response.
- Never discard an error with `_` unless the ignored outcome is explicitly safe
  and explained.
- Follow [HTTP API Contract](http.md) for public error messages and response
  mapping.

### Security-Sensitive Functions

- Never include a token, password, key, nonce, plaintext, ciphertext, signature,
  or credential-derived value in an error message or diagnostic log.
- Token signing and parsing functions wrap operations with stable context such
  as `sign token` or `verify token`; they never include the token or claims in
  that context.
- Invalid, expired, revoked, or otherwise unusable credentials are
  unauthenticated failures. A valid identity that lacks permission is an access
  denied failure. Do not collapse both into one `apperr.Kind`.
- Password comparison should distinguish an expected mismatch from an
  operational hashing failure, allowing the authentication caller to return
  unauthenticated without exposing the library error.
- Encryption and decryption helpers usually cannot classify their own failure:
  invalid client input may be a bad parameter, while unreadable persisted data,
  invalid server keys, and random-source failures are internal failures. The
  caller that knows the data source chooses the semantic type.

The application owns distinct authentication and authorization meanings in
`internal/apperr`. Use `apperr.New` or `apperr.Wrap` where that meaning becomes
known, then add outer repository, service, capability, and handler context with
`trace.Wrap`. The global HTTP error handler maps these chains to 401 and 403.
Trace v3 does not know either protocol status.

## Logging

- Use the standard library `log/slog` package for structured logging.
- Inject `*slog.Logger` into components that need configurable logging; do not
  use mutable global loggers. `cmd/app/main.go` creates the single process JSON
  logger and injects it into `internal/server.Run`, which passes it to its
  router.
- Use stable, lowercase attribute keys such as `trace_id`, `span_id`, `command`,
  and `duration`.
- Pass context to `InfoContext`, `ErrorContext`, and other context-aware methods
  when one is available.
- Log operational facts at the boundary that makes the final handling decision.
- Record an error once at the boundary that handles it:

```go
logger.ErrorContext(ctx, "serve command failed",
	slog.String("error", err.Error()),
	slog.String("trace", fmt.Sprintf("%+v", err)),
)
```

- Never log secrets, credentials, authorization headers, or unbounded request
  and response bodies.

`internal/apperr` represents semantic application errors. Trace v3 records
origin and propagation diagnostics, while OpenTelemetry carries distributed
trace and span context. Alias `go.opentelemetry.io/otel/trace` as `oteltrace`
when both trace concepts appear nearby.

## Concurrency and Lifecycle

- Start a goroutine only for a clear concurrency requirement.
- Every goroutine has an owner, a cancellation path, and a bounded lifetime.
- Make channel ownership and closing responsibility explicit.
- Prefer `context.Context` cancellation over custom stop channels.
- Do not use sleeps for synchronization in production code or tests.
- The `serve` runner orchestrates startup, readiness, graceful shutdown, and
  cleanup; Cobra command code only passes its context and typed configuration.
- `internal/server.ListenAndServe` owns TCP listen and HTTP shutdown. It uses a
  fresh bounded context for `http.Server.Shutdown`, because the signal context
  that initiated shutdown is already canceled.
- The HTTP server bounds header reads to 5 seconds, complete request reads and
  response writes to 60 seconds, and idle keep-alive connections to 120 seconds.
  Changing these limits requires a server test and an update to the HTTP
  contract.
- `internal/server.Run` owns `telemetry.Provider.Shutdown`. The provider is
  required even when export is disabled because HTTP correlation still depends
  on valid trace and span IDs. Use a fresh bounded context for this cleanup too.

## Testing

- Use Red → Green → Refactor for behavior changes and defect fixes.
- Name tests after observable behavior, for example
  `TestLoadUsesDocumentedPrecedence`.
- Use a named `[]struct` table and `t.Run(test.name, ...)` when multiple cases
  share one setup and assertion flow.
- Keep table cases deterministic and independent. Prefer a slice; use a map only
  when deliberately varying execution order is part of the test.
- Do not force a single behavior into a table when a direct test is clearer.
- Structure complex cases so setup, invocation, and assertions remain visibly
  separable as given, when, and then.
- Test through the same exported surface used by callers. Avoid assertions about
  private implementation steps.
- Prefer real in-process behavior. Fake only external boundaries; do not mock
  concrete internal collaborators merely to isolate every function.
- Inject time, randomness, and generated identifiers rather than monkey
  patching them.
- Use `t.Helper()` in helpers and `t.TempDir()` for filesystem fixtures.
- Use `t.Setenv()` for environment-dependent behavior and do not run such tests
  in parallel.
- Avoid time-based synchronization. Use contexts, channels, or explicit hooks.
- Assert semantic errors with `errors.Is` or `errors.As` when callers rely on
  them.
- A bug fix begins with a test that reproduces the failure.
- Never weaken, skip, or delete an assertion to make a change pass.

## Required Verification

Run the narrowest affected test first, then:

```sh
gofmt -w <changed-go-files>
make verify
```

`make verify` checks formatting without rewriting files, runs the architecture
checker, executes uncached tests, and runs `go vet`. These checks should
eventually run in CI; until that increment exists, `make verify` is the minimum
local final gate.

## References

- [Prefer table driven tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
  by Dave Cheney
- [Go Best Practice in Banksalad](https://blog.banksalad.com/tech/go-best-practice-in-banksalad/)
  by Banksalad
