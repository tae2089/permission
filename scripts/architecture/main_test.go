package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestInspectRejectsArchitectureViolations(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		wantRules []string
		wantError bool
	}{
		{
			name: "viper outside config",
			files: map[string]string{
				"internal/order/entity.go": `package order
import _ "github.com/spf13/viper"
`,
			},
			wantRules: []string{"viper-owner"},
		},
		{
			name: "cobra outside command",
			files: map[string]string{
				"internal/order/entity.go": `package order
import _ "github.com/spf13/cobra"
`,
			},
			wantRules: []string{"cobra-owner"},
		},
		{
			name: "pflag outside command and config",
			files: map[string]string{
				"internal/order/entity.go": `package order
import _ "github.com/spf13/pflag"
`,
			},
			wantRules: []string{"pflag-owner"},
		},
		{
			name: "service imports gin and gorm",
			files: map[string]string{
				"internal/order/service.go": `package order
import (
	_ "github.com/gin-gonic/gin"
	_ "gorm.io/gorm"
)
`,
			},
			wantRules: []string{"service-import", "service-import"},
		},
		{
			name: "handler imports persistence and config",
			files: map[string]string{
				"internal/order/handler.go": `package order
import (
	_ "github.com/example/project/internal/config"
	_ "gorm.io/gorm"
)
`,
			},
			wantRules: []string{"handler-import", "handler-import"},
		},
		{
			name: "repository imports transport and config",
			files: map[string]string{
				"internal/order/repository.go": `package order
import (
	_ "github.com/example/project/internal/config"
	_ "github.com/gin-gonic/gin"
)
`,
			},
			wantRules: []string{"repository-import", "repository-import"},
		},
		{
			name: "feature imports server",
			files: map[string]string{
				"internal/order/service.go": `package order
import _ "github.com/example/project/internal/server"
`,
			},
			wantRules: []string{"feature-server"},
		},
		{
			name: "telemetry imports gin",
			files: map[string]string{
				"internal/telemetry/provider.go": `package telemetry
import _ "github.com/gin-gonic/gin"
`,
			},
			wantRules: []string{"telemetry-gin"},
		},
		{
			name: "database imports config",
			files: map[string]string{
				"internal/database/database.go": `package database
import _ "github.com/example/project/internal/config"
`,
			},
			wantRules: []string{"database-config"},
		},
		{
			name: "route is not composed",
			files: map[string]string{
				"internal/order/routes.go": `package order
func RegisterRoutes() {}
`,
			},
			wantRules: []string{"route-registration"},
		},
		{
			name: "migration is not registered",
			files: map[string]string{
				"internal/order/migration.go": `package order
func Migrate() {}
`,
			},
			wantRules: []string{"migration-registration"},
		},
		{
			name: "invalid go source",
			files: map[string]string{
				"internal/order/service.go": "package order\nfunc",
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := newFixture(t, test.files)

			report, err := inspect(root)

			if test.wantError {
				if err == nil {
					t.Fatal("inspect() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("inspect() error = %v", err)
			}
			gotRules := make([]string, 0, len(report.Violations))
			for _, violation := range report.Violations {
				gotRules = append(gotRules, violation.Rule)
			}
			if !slices.Equal(gotRules, test.wantRules) {
				t.Errorf("violation rules = %v, want %v", gotRules, test.wantRules)
			}
		})
	}
}

func TestInspectAcceptsExplicitComposition(t *testing.T) {
	root := newFixture(t, map[string]string{
		"internal/order/service.go": `package order
type Service interface{}
`,
		"internal/order/handler.go": `package order
import _ "github.com/gin-gonic/gin"
`,
		"internal/order/repository.go": `package order
import _ "gorm.io/gorm"
`,
		"internal/order/routes.go": `package order
func RegisterRoutes() {}
`,
		"internal/order/migration.go": `package order
func Migrate() {}
`,
		"internal/server/router.go": `package server
import "github.com/example/project/internal/order"
func routes() { order.RegisterRoutes() }
`,
		"internal/migration/migration.go": `package migration
import "github.com/example/project/internal/order"
func migrate() { order.Migrate() }
`,
	})

	report, err := inspect(root)

	if err != nil {
		t.Fatalf("inspect() error = %v", err)
	}
	if len(report.Violations) != 0 {
		t.Errorf("violations = %v, want none", report.Violations)
	}
	if report.Features != 1 {
		t.Errorf("features = %d, want 1", report.Features)
	}
	if report.Routes != 1 {
		t.Errorf("routes = %d, want 1", report.Routes)
	}
	if report.Migrations != 1 {
		t.Errorf("migrations = %d, want 1", report.Migrations)
	}
}

func TestInspectAcceptsCurrentRepository(t *testing.T) {
	root, err := findModuleRoot(".")
	if err != nil {
		t.Fatalf("find module root: %v", err)
	}

	report, err := inspect(root)

	if err != nil {
		t.Fatalf("inspect() error = %v", err)
	}
	if len(report.Violations) != 0 {
		t.Errorf("violations = %v, want none", report.Violations)
	}
}

func TestInspectRejectsMissingModule(t *testing.T) {
	_, err := inspect(t.TempDir())

	if err == nil {
		t.Fatal("inspect() error = nil, want missing module error")
	}
}

func newFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module github.com/example/project\n\ngo 1.24\n")
	for name, content := range files {
		writeFixtureFile(t, root, name, content)
	}
	return root
}

func writeFixtureFile(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
}
