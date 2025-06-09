package generators

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
)

// CoreGenerator generates go files for the core of the application
type CoreGenerator struct {
	g *Generator
}

func (g *CoreGenerator) Generate() error {
	// Create go.mod file
	if err := g.GenerateGoMod(); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	files := []FileConfig{
		g.createAppMain("cmd/app/main.go"),
		g.createCliMain("cmd/cli/main.go"),
		g.createBinGoFrame("bin/goframe"),
		g.createInternalAppModule("internal/app/module.go"),
	}

	for _, file := range files {
		if err := g.g.GenerateFile(file); err != nil {
			return fmt.Errorf("failed to create main file %s: %w", file.Path, err)
		}
	}

	return nil
}

// GenerateGoMod initializes a Go module and adds dependencies
func (g *CoreGenerator) GenerateGoMod() error {
	out := bytes.NewBuffer(nil)
	cmd := exec.Command("go", "mod", "init", g.g.GoModuleName)
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create go.mod file: %s: %w", out.String(), err)
	}

	// Open file for appending dependencies
	file, err := os.OpenFile("go.mod", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open go.mod file: %w", err)
	}
	defer file.Close()

	// Add framework dependencies
	goframeDeps := []string{
		"github.com/alexisvisco/goframe/core",
		"github.com/alexisvisco/goframe/http",
		"github.com/alexisvisco/goframe/db",
		"github.com/alexisvisco/goframe/cli",
		"github.com/alexisvisco/goframe/storage",
	}

	dependencies := goframeDeps

	// Add ORM dependencies
	if g.g.ORMType == "gorm" {
		dependencies = append(dependencies, "gorm.io/gorm")
		switch g.g.DatabaseType {
		case "postgres":
			dependencies = append(dependencies, "gorm.io/driver/postgres")
		case "sqlite":
			dependencies = append(dependencies, "gorm.io/driver/sqlite")
		}
	}

	// Write dependencies
	for _, dep := range dependencies {
		_, err := file.WriteString(fmt.Sprintf(`require %s latest`, dep))
		if err != nil {
			return fmt.Errorf("failed to write dependency %s: %w", dep, err)
		}
		file.WriteString("\n")
	}

	// Add replace directives for local development if maintainer mode
	if g.g.Maintainer {
		for _, dependency := range goframeDeps {
			mod := filepath.Base(dependency)
			_, err := file.WriteString(fmt.Sprintf(`replace %s => ../../goframe/%s`, dependency, mod))
			if err != nil {
				return fmt.Errorf("failed to write replace directive for %s: %w", dependency, err)
			}
			file.WriteString("\n")
		}
	}

	g.g.TrackFile("go.mod", false, CategoryGo)

	// Run go mod tidy
	return g.RunGoModTidy()
}

// RunGoModTidy runs go mod tidy to update dependencies
func (g *CoreGenerator) RunGoModTidy() error {
	cmd := exec.Command("go", "mod", "tidy")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run go mod tidy: %s: %w", out.String(), err)
	}

	return nil
}

func (g *CoreGenerator) createBinGoFrame(path string) FileConfig {
	return FileConfig{
		Path:       path,
		Template:   templates.InternalBinGoframe,
		Category:   CategoryCore,
		Condition:  true,
		Executable: true,
	}
}

func (g *CoreGenerator) createInternalAppModule(path string) FileConfig {
	return FileConfig{
		Path:      path,
		Template:  templates.InternalAppModuleGo,
		Condition: true,
		Category:  CategoryCore,
		Gen: func(genh *genhelper.GenHelper) {
			providers := []string{
				"providers.DB(true)",
				"fxutil.As(storage.NewRepository, new(contracts.StorageRepository))",
				"providers.Storage",
				"providers.Worker",
			}

			if g.g.WebFiles {
				providers = append(providers, "providers.HTTP")
			}

			if g.g.ExampleWebFiles {
				providers = append(providers, "fxutil.As(repository.NewNoteRepository, new(types.NoteRepository))")
				providers = append(providers, "fxutil.As(service.NewNoteService, new(types.NoteService))")
				providers = append(providers, "v1handler.NewNoteHandler")

				genh.WithImport(filepath.Join(g.g.GoModuleName, "internal/types"), "types").
					WithImport(filepath.Join(g.g.GoModuleName, "internal/repository"), "repository").
					WithImport(filepath.Join(g.g.GoModuleName, "internal/service"), "service").
					WithImport(filepath.Join(g.g.GoModuleName, "internal/v1handler"), "v1handler")
			}

			genh.
				WithImport("github.com/alexisvisco/goframe/storage", "storage").
				WithImport("github.com/alexisvisco/goframe/core/contracts", "contracts").
				WithImport(filepath.Join(g.g.GoModuleName, "config"), "config").
				WithImport(filepath.Join(g.g.GoModuleName, "internal/providers"), "providers")

			genh.WithVar("provides", providers)
		},
	}
}

// createAppMain creates the FileConfig for the app main.go file
func (g *CoreGenerator) createAppMain(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.CmdAppMainGo,
		Gen: func(genh *genhelper.GenHelper) {
			var invokes []string

			if g.g.WebFiles {
				invokes = append(invokes, "v1handler.Router")
			}

			genh.WithImport(filepath.Join(g.g.GoModuleName, "internal/app"), "app").
				WithVar("invokes", invokes)
		},
		Category:  CategoryCore,
		Condition: true,
	}
}

// createCliMain creates the FileConfig for the CLI main.go file
func (g *CoreGenerator) createCliMain(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.CmdCLIMainGo,
		Gen: func(genh *genhelper.GenHelper) {
			genh.WithImport(filepath.Join(g.g.GoModuleName, "config"), "config").
				WithImport(filepath.Join(g.g.GoModuleName, "internal/app"), "app").
				WithImport(filepath.Join(g.g.GoModuleName, "internal/providers"), "providers").
				WithImport(filepath.Join(g.g.GoModuleName, "db"), "db")
		},
		Category:  CategoryCore,
		Condition: true,
	}
}
