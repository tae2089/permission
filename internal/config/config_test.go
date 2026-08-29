package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{name: "wildcard host", address: ":8080"},
		{name: "hostname", address: "localhost:8080"},
		{name: "ipv4", address: "127.0.0.1:8080"},
		{name: "ipv6", address: "[::1]:8080"},
		{name: "ephemeral port", address: ":0"},
		{name: "maximum port", address: ":65535"},
		{name: "empty", address: "", wantErr: true},
		{name: "missing port", address: "localhost", wantErr: true},
		{name: "non-numeric port", address: ":http", wantErr: true},
		{name: "negative port", address: ":-1", wantErr: true},
		{name: "out of range port", address: ":65536", wantErr: true},
		{name: "too many colons", address: "localhost:8080:extra", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Serve:    Serve{Address: tt.address},
				Database: Database{Driver: DefaultDatabaseDriver, DSN: DefaultDatabaseDSN},
				Instance: Instance{AdminKey: "test-instance-admin-key"},
			}

			err := cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("Validate() error = nil, want error")
				}
				if !strings.Contains(err.Error(), "serve.address") {
					t.Errorf("Validate() error = %q, want serve.address context", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() error = %v", err)
			}
		})
	}
}

func TestConfigValidateRejectsEmptyDatabaseDSN(t *testing.T) {
	cfg := Config{
		Serve:    Serve{Address: DefaultServeAddress},
		Database: Database{Driver: DefaultDatabaseDriver, DSN: "  "},
		Instance: Instance{AdminKey: "test-instance-admin-key"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want database.dsn error")
	}
	if !strings.Contains(err.Error(), "database.dsn") {
		t.Errorf("Validate() error = %q, want database.dsn context", err)
	}
}

func TestConfigValidateRejectsUnsupportedDatabaseDriver(t *testing.T) {
	tests := []struct {
		name   string
		driver string
	}{
		{name: "empty"},
		{name: "unsupported", driver: "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Serve: Serve{Address: DefaultServeAddress},
				Database: Database{
					Driver: tt.driver,
					DSN:    DefaultDatabaseDSN,
				},
				Instance: Instance{AdminKey: "test-instance-admin-key"},
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() error = nil, want database.driver error")
			}
			if !strings.Contains(err.Error(), "database.driver") {
				t.Errorf("Validate() error = %q, want database.driver context", err)
			}
		})
	}
}

func TestConfigValidateRejectsMissingInstanceAdminKey(t *testing.T) {
	cfg := Config{
		Serve:    Serve{Address: DefaultServeAddress},
		Database: Database{Driver: DefaultDatabaseDriver, DSN: DefaultDatabaseDSN},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want instance.admin_key error")
	}
	if !strings.Contains(err.Error(), "instance.admin_key") {
		t.Errorf("Validate() error = %q, want instance.admin_key context", err)
	}
}

func TestLoadInstanceAdminKeyUsesDocumentedPrecedence(t *testing.T) {
	const envKey = "GO_TEMPLATE_INSTANCE_ADMIN_KEY"

	tests := []struct {
		name      string
		overrides map[string]any
		env       string
		file      string
		kv        map[string]any
		want      string
	}{
		{
			name:      "explicit set overrides every source",
			overrides: map[string]any{"instance.admin_key": "set-key"},
			env:       "env-key",
			file:      "file-key",
			kv:        instanceConfig("kv-key"),
			want:      "set-key",
		},
		{
			name: "environment overrides file and lower sources",
			env:  "env-key",
			file: "file-key",
			kv:   instanceConfig("kv-key"),
			want: "env-key",
		},
		{
			name: "config file overrides kv store and default",
			file: "file-key",
			kv:   instanceConfig("kv-key"),
			want: "file-key",
		},
		{
			name: "kv store overrides default",
			kv:   instanceConfig("kv-key"),
			want: "kv-key",
		},
		{
			name: "empty when no source provides a value",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetEnv(t, envKey)
			if tt.env != "" {
				t.Setenv(envKey, tt.env)
			}

			opts := Options{
				KV:        tt.kv,
				Overrides: tt.overrides,
			}
			if tt.file != "" {
				opts.ConfigFile = writeInstanceConfig(t, tt.file)
			}

			got, err := Load(opts)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.Instance.AdminKey != tt.want {
				t.Errorf("Instance.AdminKey did not resolve from the expected source")
			}
		})
	}
}

func TestLoadUsesDocumentedPrecedence(t *testing.T) {
	const envKey = "GO_TEMPLATE_SERVE_ADDRESS"

	tests := []struct {
		name      string
		overrides map[string]any
		flag      string
		env       string
		file      string
		kv        map[string]any
		want      string
	}{
		{
			name:      "explicit set overrides every source",
			overrides: map[string]any{"serve.address": ":9005"},
			flag:      ":9004",
			env:       ":9003",
			file:      ":9002",
			kv:        addressConfig(":9001"),
			want:      ":9005",
		},
		{
			name: "flag overrides environment and lower sources",
			flag: ":9004",
			env:  ":9003",
			file: ":9002",
			kv:   addressConfig(":9001"),
			want: ":9004",
		},
		{
			name: "environment overrides file and lower sources",
			env:  ":9003",
			file: ":9002",
			kv:   addressConfig(":9001"),
			want: ":9003",
		},
		{
			name: "config file overrides kv store and default",
			file: ":9002",
			kv:   addressConfig(":9001"),
			want: ":9002",
		},
		{
			name: "kv store overrides default",
			kv:   addressConfig(":9001"),
			want: ":9001",
		},
		{
			name: "default is used when no source provides a value",
			want: DefaultServeAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetEnv(t, envKey)
			if tt.env != "" {
				t.Setenv(envKey, tt.env)
			}

			flags := pflag.NewFlagSet("serve", pflag.ContinueOnError)
			flags.String("address", "", "listen address")
			if tt.flag != "" {
				if err := flags.Parse([]string{"--address", tt.flag}); err != nil {
					t.Fatalf("parse flags: %v", err)
				}
			}

			opts := Options{
				FlagSet:   flags,
				KV:        tt.kv,
				Overrides: tt.overrides,
			}
			if tt.file != "" {
				opts.ConfigFile = writeConfig(t, tt.file)
			}

			got, err := Load(opts)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.Serve.Address != tt.want {
				t.Errorf("Serve.Address = %q, want %q", got.Serve.Address, tt.want)
			}
		})
	}
}

func TestLoadDatabaseDSNUsesDocumentedPrecedence(t *testing.T) {
	const envKey = "GO_TEMPLATE_DATABASE_DSN"

	tests := []struct {
		name      string
		overrides map[string]any
		flag      string
		env       string
		file      string
		kv        map[string]any
		want      string
	}{
		{
			name:      "explicit set overrides every source",
			overrides: map[string]any{"database.dsn": "set.db"},
			flag:      "flag.db",
			env:       "env.db",
			file:      "file.db",
			kv:        databaseConfig("kv.db"),
			want:      "set.db",
		},
		{
			name: "flag overrides environment and lower sources",
			flag: "flag.db",
			env:  "env.db",
			file: "file.db",
			kv:   databaseConfig("kv.db"),
			want: "flag.db",
		},
		{
			name: "environment overrides file and lower sources",
			env:  "env.db",
			file: "file.db",
			kv:   databaseConfig("kv.db"),
			want: "env.db",
		},
		{
			name: "config file overrides kv store and default",
			file: "file.db",
			kv:   databaseConfig("kv.db"),
			want: "file.db",
		},
		{
			name: "kv store overrides default",
			kv:   databaseConfig("kv.db"),
			want: "kv.db",
		},
		{
			name: "default is used when no source provides a value",
			want: DefaultDatabaseDSN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetEnv(t, envKey)
			if tt.env != "" {
				t.Setenv(envKey, tt.env)
			}

			flags := pflag.NewFlagSet("serve", pflag.ContinueOnError)
			flags.String("database-dsn", "", "database dsn")
			if tt.flag != "" {
				if err := flags.Parse([]string{"--database-dsn", tt.flag}); err != nil {
					t.Fatalf("parse flags: %v", err)
				}
			}

			opts := Options{
				FlagSet:   flags,
				KV:        tt.kv,
				Overrides: tt.overrides,
			}
			if tt.file != "" {
				opts.ConfigFile = writeDatabaseConfig(t, tt.file)
			}

			got, err := Load(opts)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.Database.DSN != tt.want {
				t.Errorf("Database.DSN = %q, want %q", got.Database.DSN, tt.want)
			}
		})
	}
}

func TestLoadDatabaseDriverUsesDocumentedPrecedence(t *testing.T) {
	const envKey = "GO_TEMPLATE_DATABASE_DRIVER"

	tests := []struct {
		name      string
		overrides map[string]any
		flag      string
		env       string
		file      string
		kv        map[string]any
		want      string
	}{
		{
			name:      "explicit set overrides every source",
			overrides: map[string]any{"database.driver": "set"},
			flag:      "flag",
			env:       "env",
			file:      "file",
			kv:        databaseDriverConfig("kv"),
			want:      "set",
		},
		{
			name: "flag overrides environment and lower sources",
			flag: "flag",
			env:  "env",
			file: "file",
			kv:   databaseDriverConfig("kv"),
			want: "flag",
		},
		{
			name: "environment overrides file and lower sources",
			env:  "env",
			file: "file",
			kv:   databaseDriverConfig("kv"),
			want: "env",
		},
		{
			name: "config file overrides kv store and default",
			file: "file",
			kv:   databaseDriverConfig("kv"),
			want: "file",
		},
		{
			name: "kv store overrides default",
			kv:   databaseDriverConfig("kv"),
			want: "kv",
		},
		{
			name: "default is used when no source provides a value",
			want: DefaultDatabaseDriver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsetEnv(t, envKey)
			if tt.env != "" {
				t.Setenv(envKey, tt.env)
			}

			flags := pflag.NewFlagSet("serve", pflag.ContinueOnError)
			flags.String("database-driver", "", "database driver")
			if tt.flag != "" {
				if err := flags.Parse([]string{"--database-driver", tt.flag}); err != nil {
					t.Fatalf("parse flags: %v", err)
				}
			}

			opts := Options{
				FlagSet:   flags,
				KV:        tt.kv,
				Overrides: tt.overrides,
			}
			if tt.file != "" {
				opts.ConfigFile = writeDatabaseDriverConfig(t, tt.file)
			}

			got, err := Load(opts)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.Database.Driver != tt.want {
				t.Errorf("Database.Driver = %q, want %q", got.Database.Driver, tt.want)
			}
		})
	}
}

func TestLoadRejectsMalformedConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("serve: ["), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(Options{ConfigFile: path})
	if err == nil {
		t.Fatal("Load() error = nil, want malformed config error")
	}
}

func TestExampleConfigMatchesTypedConfiguration(t *testing.T) {
	path := filepath.Join("..", "..", "config.example.yaml")

	got, err := Load(Options{ConfigFile: path})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Serve.Address != DefaultServeAddress {
		t.Errorf("Serve.Address = %q, want %q", got.Serve.Address, DefaultServeAddress)
	}
	if got.Database.DSN != DefaultDatabaseDSN {
		t.Errorf("Database.DSN = %q, want %q", got.Database.DSN, DefaultDatabaseDSN)
	}
	if got.Database.Driver != DefaultDatabaseDriver {
		t.Errorf("Database.Driver = %q, want %q", got.Database.Driver, DefaultDatabaseDriver)
	}
}

func addressConfig(address string) map[string]any {
	return map[string]any{
		"serve": map[string]any{
			"address": address,
		},
	}
}

func databaseConfig(dsn string) map[string]any {
	return map[string]any{
		"database": map[string]any{
			"dsn": dsn,
		},
	}
}

func databaseDriverConfig(driver string) map[string]any {
	return map[string]any{
		"database": map[string]any{
			"driver": driver,
		},
	}
}

func instanceConfig(adminKey string) map[string]any {
	return map[string]any{
		"instance": map[string]any{
			"admin_key": adminKey,
		},
	}
}

func writeConfig(t *testing.T, address string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("serve:\n  address: " + address + "\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func writeDatabaseConfig(t *testing.T, dsn string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("database:\n  dsn: " + dsn + "\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func writeDatabaseDriverConfig(t *testing.T, driver string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("database:\n  driver: " + driver + "\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func writeInstanceConfig(t *testing.T, adminKey string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("instance:\n  admin_key: " + adminKey + "\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if ok {
		t.Cleanup(func() {
			if err := os.Setenv(key, value); err != nil {
				t.Errorf("restore environment: %v", err)
			}
		})
	}
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset environment: %v", err)
	}
}
