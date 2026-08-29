package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	cobraImport = "github.com/spf13/cobra"
	ginImport   = "github.com/gin-gonic/gin"
	pflagImport = "github.com/spf13/pflag"
	viperImport = "github.com/spf13/viper"
	gormPrefix  = "gorm.io/"
)

type report struct {
	Files      int
	Features   int
	Routes     int
	Migrations int
	Violations []violation
}

type violation struct {
	File   string
	Rule   string
	Detail string
}

type sourceFile struct {
	RelativePath string
	Directory    string
	Base         string
	Syntax       *ast.File
	Imports      []string
}

func main() {
	root, err := findModuleRoot(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "architecture inspection failed:", err)
		os.Exit(2)
	}

	result, err := inspect(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "architecture inspection failed:", err)
		os.Exit(2)
	}
	if len(result.Violations) != 0 {
		fmt.Fprintln(os.Stderr, "architecture check failed:")
		for _, item := range result.Violations {
			fmt.Fprintf(os.Stderr, "- %s [%s]: %s\n", item.File, item.Rule, item.Detail)
		}
		os.Exit(1)
	}

	fmt.Printf(
		"architecture check passed: files=%d features=%d routes=%d migrations=%d\n",
		result.Files,
		result.Features,
		result.Routes,
		result.Migrations,
	)
}

func inspect(root string) (report, error) {
	modulePath, err := readModulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return report{}, err
	}

	files, err := parseProductionFiles(root)
	if err != nil {
		return report{}, err
	}

	featureDirectories := findFeatureDirectories(files)
	routePackages := findFeaturePackages(files, modulePath, "routes.go")
	migrationPackages := findFeaturePackages(files, modulePath, "migration.go")
	violations := inspectImports(files, modulePath, featureDirectories)

	routeCalls := findRegistrationCalls(
		files,
		"internal/server/router.go",
		"RegisterRoutes",
	)
	for packagePath, relativeDirectory := range routePackages {
		if !routeCalls[packagePath] {
			violations = append(violations, violation{
				File:   relativeDirectory + "/routes.go",
				Rule:   "route-registration",
				Detail: "feature RegisterRoutes is not called from internal/server/router.go",
			})
		}
	}

	migrationCalls := findRegistrationCalls(
		files,
		"internal/migration/migration.go",
		"Migrate",
	)
	for packagePath, relativeDirectory := range migrationPackages {
		if !migrationCalls[packagePath] {
			violations = append(violations, violation{
				File:   relativeDirectory + "/migration.go",
				Rule:   "migration-registration",
				Detail: "feature Migrate is not called from internal/migration/migration.go",
			})
		}
	}

	slices.SortFunc(violations, func(left, right violation) int {
		if compared := strings.Compare(left.File, right.File); compared != 0 {
			return compared
		}
		if compared := strings.Compare(left.Rule, right.Rule); compared != 0 {
			return compared
		}
		return strings.Compare(left.Detail, right.Detail)
	})

	return report{
		Files:      len(files),
		Features:   len(featureDirectories),
		Routes:     len(routePackages),
		Migrations: len(migrationPackages),
		Violations: violations,
	}, nil
}

func findModuleRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve start directory: %w", err)
	}
	info, err := os.Stat(current)
	if err != nil {
		return "", fmt.Errorf("inspect start directory: %w", err)
	}
	if !info.IsDir() {
		current = filepath.Dir(current)
	}

	for {
		moduleFile := filepath.Join(current, "go.mod")
		if _, err := os.Stat(moduleFile); err == nil {
			return current, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect go.mod: %w", err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("go.mod not found")
		}
		current = parent
	}
}

func readModulePath(moduleFile string) (string, error) {
	content, err := os.ReadFile(moduleFile)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "module ") {
			continue
		}
		modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module "))
		if modulePath == "" {
			return "", errors.New("go.mod module path is empty")
		}
		return modulePath, nil
	}
	return "", errors.New("go.mod module declaration not found")
}

func parseProductionFiles(root string) ([]sourceFile, error) {
	var files []sourceFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("resolve relative source path: %w", err)
		}
		relativePath = filepath.ToSlash(relativePath)
		syntax, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", relativePath, err)
		}
		imports, err := importedPaths(syntax)
		if err != nil {
			return fmt.Errorf("parse imports in %s: %w", relativePath, err)
		}
		files = append(files, sourceFile{
			RelativePath: relativePath,
			Directory:    filepath.ToSlash(filepath.Dir(relativePath)),
			Base:         filepath.Base(relativePath),
			Syntax:       syntax,
			Imports:      imports,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk production source: %w", err)
	}
	slices.SortFunc(files, func(left, right sourceFile) int {
		return strings.Compare(left.RelativePath, right.RelativePath)
	})
	return files, nil
}

func ignoredDirectory(name string) bool {
	return name == ".git" ||
		name == "_workspace" ||
		name == "vendor" ||
		strings.HasPrefix(name, ".")
}

func importedPaths(file *ast.File) ([]string, error) {
	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("unquote import %s: %w", spec.Path.Value, err)
		}
		imports = append(imports, importPath)
	}
	slices.Sort(imports)
	return imports, nil
}

func findFeatureDirectories(files []sourceFile) map[string]bool {
	features := make(map[string]bool)
	for _, file := range files {
		if !isFeatureDirectory(file.Directory) {
			continue
		}
		if roleFile(file.Base, "handler") ||
			roleFile(file.Base, "service") ||
			file.Base == "routes.go" {
			features[file.Directory] = true
		}
	}
	return features
}

func findFeaturePackages(
	files []sourceFile,
	modulePath string,
	fileName string,
) map[string]string {
	packages := make(map[string]string)
	for _, file := range files {
		if file.Base != fileName || !isFeatureDirectory(file.Directory) {
			continue
		}
		packages[modulePath+"/"+file.Directory] = file.Directory
	}
	return packages
}

func isFeatureDirectory(directory string) bool {
	if !strings.HasPrefix(directory, "internal/") {
		return false
	}
	for _, infrastructure := range []string{
		"internal/command",
		"internal/config",
		"internal/database",
		"internal/http",
		"internal/migration",
		"internal/server",
		"internal/telemetry",
	} {
		if within(directory, infrastructure) {
			return false
		}
	}
	return true
}

func inspectImports(
	files []sourceFile,
	modulePath string,
	featureDirectories map[string]bool,
) []violation {
	var violations []violation
	for _, file := range files {
		for _, importPath := range file.Imports {
			if importPath == viperImport && !within(file.Directory, "internal/config") {
				violations = append(violations, importViolation(
					file,
					"viper-owner",
					importPath,
					"Viper is owned by internal/config",
				))
			}
			if importPath == cobraImport && !within(file.Directory, "internal/command") {
				violations = append(violations, importViolation(
					file,
					"cobra-owner",
					importPath,
					"Cobra is owned by internal/command",
				))
			}
			if importPath == pflagImport &&
				!within(file.Directory, "internal/command") &&
				!within(file.Directory, "internal/config") {
				violations = append(violations, importViolation(
					file,
					"pflag-owner",
					importPath,
					"pflag is owned by internal/command and internal/config",
				))
			}

			if roleFile(file.Base, "service") &&
				(importPath == ginImport ||
					importPath == viperImport ||
					strings.HasPrefix(importPath, gormPrefix)) {
				violations = append(violations, importViolation(
					file,
					"service-import",
					importPath,
					"Service must not import Gin, GORM, or Viper",
				))
			}
			if roleFile(file.Base, "handler") &&
				(strings.HasPrefix(importPath, gormPrefix) ||
					importPath == viperImport ||
					importWithin(importPath, modulePath+"/internal/config") ||
					importWithin(importPath, modulePath+"/internal/database")) {
				violations = append(violations, importViolation(
					file,
					"handler-import",
					importPath,
					"Handler must not import persistence or configuration packages",
				))
			}
			if roleFile(file.Base, "repository") &&
				(importPath == ginImport ||
					importPath == viperImport ||
					importWithin(importPath, modulePath+"/internal/config") ||
					importWithin(importPath, modulePath+"/internal/server")) {
				violations = append(violations, importViolation(
					file,
					"repository-import",
					importPath,
					"Repository must not import transport, configuration, or Server packages",
				))
			}
			if featureDirectories[file.Directory] &&
				importWithin(importPath, modulePath+"/internal/server") {
				violations = append(violations, importViolation(
					file,
					"feature-server",
					importPath,
					"feature packages must not import internal/server",
				))
			}
			if within(file.Directory, "internal/telemetry") && importPath == ginImport {
				violations = append(violations, importViolation(
					file,
					"telemetry-gin",
					importPath,
					"internal/telemetry must not import Gin",
				))
			}
			if within(file.Directory, "internal/database") &&
				importWithin(importPath, modulePath+"/internal/config") {
				violations = append(violations, importViolation(
					file,
					"database-config",
					importPath,
					"internal/database must not import internal/config",
				))
			}
		}
	}
	return violations
}

func importViolation(
	file sourceFile,
	rule string,
	importPath string,
	message string,
) violation {
	return violation{
		File:   file.RelativePath,
		Rule:   rule,
		Detail: fmt.Sprintf("%s: imports %s", message, importPath),
	}
}

func roleFile(base string, role string) bool {
	return base == role+".go" || strings.HasPrefix(base, role+"_")
}

func within(path string, directory string) bool {
	return path == directory || strings.HasPrefix(path, directory+"/")
}

func importWithin(importPath string, packagePath string) bool {
	return importPath == packagePath || strings.HasPrefix(importPath, packagePath+"/")
}

func findRegistrationCalls(
	files []sourceFile,
	relativePath string,
	functionName string,
) map[string]bool {
	for _, file := range files {
		if file.RelativePath != relativePath {
			continue
		}
		return registrationCalls(file.Syntax, functionName)
	}
	return map[string]bool{}
}

func registrationCalls(file *ast.File, functionName string) map[string]bool {
	imports := make(map[string]string)
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		alias := filepath.Base(importPath)
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		if alias == "." || alias == "_" {
			continue
		}
		imports[alias] = importPath
	}

	calls := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != functionName {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if importPath, ok := imports[identifier.Name]; ok {
			calls[importPath] = true
		}
		return true
	})
	return calls
}
