package generators

import (
	"fmt"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
)

type WebGenerator struct {
	g *Generator
}

func (p *WebGenerator) Generate() error {
	files := []FileConfig{
		p.createHTTPProvider("internal/providers/http.go"),
		p.createRouter("internal/v1handler/router.go"),
		p.createExampleV1handler("internal/v1handler/handler_note.go"),
		p.createExampleTypes("internal/types/note.go"),
		p.createExampleRepository("internal/repository/repository_note.go"),
		p.createExampleService("internal/service/service_note.go"),
	}

	for _, file := range files {
		if err := p.g.GenerateFile(file); err != nil {
			return fmt.Errorf("failed to create web file %s: %w", file.Path, err)
		}
	}

	return nil
}

// createHTTPProvider creates the FileConfig for the HTTP provider
func (p *WebGenerator) createHTTPProvider(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.ProvidersProvideHTTPServerGo,
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.g.GoModuleName, "config"), "config")
		},
		Category:  CategoryWeb,
		Condition: p.g.WebFiles,
	}
}

func (p *WebGenerator) createRouter(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.InternalV1HandlerRouterGo,
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("example", p.g.ExampleWebFiles)
		},
		Category:  CategoryWeb,
		Condition: p.g.WebFiles,
	}
}

// createExampleRepository creates the FileConfig for the example repository
func (p *WebGenerator) createExampleRepository(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.InternalRepositoryExampleGo,
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.g.GoModuleName, "internal/types"), "types")
		},
		Category:  CategoryWeb,
		Condition: p.g.ExampleWebFiles,
	}
}

// createExampleTypes creates the FileConfig for the example types
func (p *WebGenerator) createExampleTypes(path string) FileConfig {
	return FileConfig{
		Path:      path,
		Template:  templates.InternalTypesExampleGo,
		Category:  CategoryWeb,
		Condition: p.g.ExampleWebFiles,
	}
}

// createExampleTypesFile creates the FileConfig for the example types file
func (p *WebGenerator) createExampleService(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.InternalServiceExampleGo,
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.g.GoModuleName, "internal/types"), "types")
		},
		Category:  CategoryWeb,
		Condition: p.g.ExampleWebFiles,
	}
}

func (p *WebGenerator) createExampleV1handler(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.InternalV1HandlerExampleGo,
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.g.GoModuleName, "internal/types"), "types")
		},
		Category:  CategoryWeb,
		Condition: p.g.ExampleWebFiles,
	}
}
