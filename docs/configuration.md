# Configuration

## Resolution Contract

Each configuration key is resolved independently with this precedence, from
highest to lowest:

1. Explicit `Set` override
2. Command-line flag
3. Environment variable
4. YAML configuration file
5. KV store value
6. Default

Viper exists only inside `internal/config`. `config.Load` resolves all sources
once and returns `config.Config`. Command runners and application packages use
typed fields and must not query Viper directly.

## Why Viper

The executable needs one typed configuration assembled from defaults, remote KV
values, a local file, environment variables, command-line flags, and controlled
programmatic overrides. Viper provides this multi-source resolution with a
documented
[configuration precedence](https://pkg.go.dev/github.com/spf13/viper#section-readme).

The alternatives considered were:

- pflag only is simpler, but file, environment, and KV precedence would become
  custom infrastructure.
- Global Viper access is convenient initially, but it hides dependencies,
  spreads string keys, and makes isolated tests harder.
- A local Viper instance plus typed decoding keeps resolution in one testable
  boundary while application code remains explicit.

This choice makes Viper a deliberate infrastructure dependency. Every new
setting must keep its code, tests, and documentation synchronized. A concrete
remote KV adapter is still selected only when a provider is required.

## Canonical Key Mapping

| Canonical key | Go field | Flag | Environment | YAML / KV path | Default |
| --- | --- | --- | --- | --- | --- |
| `serve.address` | `Config.Serve.Address` | `--address` | `GO_TEMPLATE_SERVE_ADDRESS` | `serve.address` | `:8080` |
| `database.driver` | `Config.Database.Driver` | `--database-driver` | `GO_TEMPLATE_DATABASE_DRIVER` | `database.driver` | `sqlite` |
| `database.dsn` | `Config.Database.DSN` | `--database-dsn` | `GO_TEMPLATE_DATABASE_DSN` | `database.dsn` | `go-template.db` |
| `instance.admin_key` | `Config.Instance.AdminKey` | — | `GO_TEMPLATE_INSTANCE_ADMIN_KEY` | `instance.admin_key` | none (required for `serve`) |

The dotted canonical key is the source of truth. Environment names use the
`GO_TEMPLATE_` prefix and replace dots or hyphens with underscores.

## Validation

`cmd/<binary>/options.Serve.Complete` calls `config.Load` to resolve sources and
decode typed values. `Serve.Validate` calls `Config.Validate` to check their
command-level meaning. `Serve.Run` invokes the injected runner only after both
stages succeed.

`options.Migrate` uses the same completion and runner sequence but validates
only `Config.Database`. An invalid or unused `serve.address` must not prevent an
explicit database migration. `instance.admin_key` is likewise not required for
`migrate`, because migration does not expose an HTTP administrator endpoint.

`serve.address` must use a TCP `host:port` form with a decimal port from `0` to
`65535`:

```text
:8080
localhost:8080
127.0.0.1:8080
[::1]:8080
```

An empty host and port `0` are valid. IPv6 addresses require brackets. Validation
does not perform DNS lookup. Missing, named, negative, and out-of-range ports are
rejected with an error that identifies `serve.address`.

`database.driver` must identify an implemented driver. The template currently
accepts only `sqlite`; PostgreSQL or MySQL becomes a valid value only when its
GORM driver and integration tests are added.

`database.dsn` must not be empty or whitespace-only. It is interpreted as a
DSN for the selected driver by `internal/database`. The SQLite default is a
database file relative to the process working directory. A DSN may contain
sensitive data; never log it.

`instance.admin_key` must not be empty or whitespace-only when running `serve`.
It authenticates the bootstrap instance administrator. Treat it as a secret:
prefer the environment or a secret-managed configuration file, and never log,
return, or commit its value.

## Supported Inputs

### YAML Configuration File

YAML is the canonical file format. Pass its path with `--config`:

```yaml
serve:
  address: ":8080"
database:
  driver: "sqlite"
  dsn: "go-template.db"
```

Malformed files return an error. A provided file path that cannot be read is
also an error.

### Environment

```sh
GO_TEMPLATE_SERVE_ADDRESS=:8081 app serve
GO_TEMPLATE_DATABASE_DRIVER=sqlite app serve
GO_TEMPLATE_DATABASE_DSN=local.db app serve
GO_TEMPLATE_INSTANCE_ADMIN_KEY=<secret> app serve
```

### Flags

```sh
app serve --address :8082
app serve --database-driver sqlite
app serve --database-dsn local.db
app migrate --database-driver sqlite --database-dsn local.db
```

The flag's pflag default is intentionally empty. Setting a real default on the
flag would incorrectly mask environment, file, KV, and default values.

### KV Store

`config.Options.KV` accepts the decoded nested shape:

```go
map[string]any{
	"serve": map[string]any{
		"address": ":8083",
	},
	"database": map[string]any{
		"driver": "sqlite",
		"dsn":    "local.db",
	},
	"instance": map[string]any{
		"admin_key": "provided-by-secret-management",
	},
}
```

A future Consul, etcd, or other adapter is responsible for fetching and decoding
provider data into this shape. The template does not select or import a remote
KV client yet.

### Explicit Override

Composition code may apply the highest-priority override:

```go
config.Options{
	Overrides: map[string]any{
		"serve.address": ":8084",
		"database.driver": "sqlite",
		"database.dsn":    "local.db",
		"instance.admin_key": "provided-by-secret-management",
	},
}
```

Use overrides for controlled programmatic decisions, not as a second flag or
environment parsing mechanism.

## Adding a Setting

1. Choose one dotted canonical key.
2. Add its typed field and `mapstructure` tag to `config.Config`.
3. Add its default in `config.Load`, or document why it has no default.
4. Bind any flag to the canonical key and define its environment mapping.
5. Add the YAML/KV shape and every public source name to the mapping table.
6. Extend precedence tests so every supported source participates.
7. Extend `Config.Validate` and its valid and invalid test cases when the setting
   has semantic constraints.

Do not add `viper.Get...` calls outside `internal/config`.
