package gencore

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

// CoreGenerator generates core application files.
type CoreGenerator struct{ Gen *generators.Generator }

//go:embed templates
var fs embed.FS

func (g *CoreGenerator) Generate() error {
	if err := g.generateGoMod(); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}
	files := []generators.FileConfig{
		g.createAppMain("cmd/app/main.go"),
		g.createCliMain("cmd/cli/main.go"),
		g.createBinGoframe("bin/goframe"),
		g.createBinMJML("bin/mjml"),
		g.createAppModule("internal/app/module.go"),
	}
	return g.Gen.GenerateFiles(files)
}

func (g *CoreGenerator) generateGoMod() error {
	out := bytes.NewBuffer(nil)
	cmd := exec.Command("go", "mod", "init", g.Gen.GoModuleName)
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
		"github.com/alexisvisco/goframe/mail",
		"github.com/alexisvisco/goframe/cache",
	}

	dependencies := goframeDeps

	// Add ORM dependencies
	if g.Gen.ORMType == "gorm" {
		dependencies = append(dependencies, "gorm.io/gorm")
		switch g.Gen.DatabaseType {
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
	if true {
		for _, dependency := range goframeDeps {
			mod := filepath.Base(dependency)
			_, err := file.WriteString(fmt.Sprintf(`replace %s => ../../goframe/%s`, dependency, mod))
			if err != nil {
				return fmt.Errorf("failed to write replace directive for %s: %w", dependency, err)
			}
			file.WriteString("\n")
		}
	}

	// Run go mod tidy
	return nil
}

func (g *CoreGenerator) createBinGoframe(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:       path,
		Template:   typeutil.Must(fs.ReadFile("templates/bin_goframe.tmpl")),
		Executable: true,
	}
}

func (g *CoreGenerator) createBinMJML(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:       path,
		Template:   typeutil.Must(fs.ReadFile("templates/bin_mjml.tmpl")),
		Executable: true,
	}
}

func (g *CoreGenerator) createAppModule(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/app_module.go.tmpl")),
		Gen: func(genh *genhelper.GenHelper) {
			providers := []string{
				"provide.Cache",
				"provide.DB(true)",
				"fxutil.As(storage.NewRepository, new(contracts.StorageRepository))",
				"provide.Storage",
				"provide.Worker",
			}
			if g.Gen.HTTPServer {
				providers = append(providers, "provide.HTTP")
			}
			genh.WithImport("github.com/alexisvisco/goframe/storage", "storage").
				WithImport("github.com/alexisvisco/goframe/core/contracts", "contracts").
				WithImport(filepath.Join(g.Gen.GoModuleName, "config"), "config").
				WithImport(filepath.Join(g.Gen.GoModuleName, "internal/provide"), "provide").
				WithVar("provides", providers)
		},
	}
}

func (g *CoreGenerator) createAppMain(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/app_main.go.tmpl")),
		Gen: func(genh *genhelper.GenHelper) {
			var invokes []string
			if g.Gen.HTTPServer {
				invokes = append(invokes, "v1handler.Router")
				genh.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/v1handler"), "v1handler")
			}
			genh.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/app"), "app").
				WithImport(filepath.Join(g.Gen.GoModuleName, "config"), "config").
				WithVar("invokes", invokes)
		},
	}
}

func (g *CoreGenerator) createCliMain(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/cli_main.go.tmpl")),
		Gen: func(genh *genhelper.GenHelper) {
			genh.WithImport(filepath.Join(g.Gen.GoModuleName, "config"), "config").
				WithImport(filepath.Join(g.Gen.GoModuleName, "internal/app"), "app").
				WithImport(filepath.Join(g.Gen.GoModuleName, "internal/provide"), "provide").
				WithImport(filepath.Join(g.Gen.GoModuleName, "db"), "db")
		},
	}
}
