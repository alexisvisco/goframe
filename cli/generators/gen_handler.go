package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

type HandlerGenerator struct {
	g *Generator
}

func (h *HandlerGenerator) createHandlerFile(name string, services []string) error {
	path := filepath.Join("internal/v1handler", fmt.Sprintf("handler_%s.go", str.ToSnakeCase(name)))
	fileMustNotExist := true
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		fileMustNotExist = false
	}
	return h.g.GenerateFile(FileConfig{
		Path:     path,
		Template: templates.InternalV1HandlerNewGo,
		Category: CategoryWeb,
		Skip:     fileMustNotExist,
		Gen: func(g *genhelper.GenHelper) {
			pascal := str.ToPascalCase(name)
			g.WithVar("name_pascal", pascal)

			type dep struct {
				ServiceName string
				VarName     string
			}
			var svcs []dep
			for _, s := range services {
				p := str.ToPascalCase(s)
				if !strings.HasSuffix(p, "Service") {
					p += "Service"
				}
				svcs = append(svcs, dep{
					ServiceName: p,
					VarName:     str.ToCamelCase(p),
				})
			}
			g.WithVar("services", svcs)

			if len(svcs) > 0 {
				g.WithImport("go.uber.org/fx", "fx").
					WithImport(filepath.Join(h.g.GoModuleName, "internal/types"), "types")
			}
		},
	})
}

func (h *HandlerGenerator) listHandlers() ([]string, error) {
	entries, err := os.ReadDir("internal/v1handler")
	if err != nil {
		return nil, err
	}
	var handlers []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		if e.Name() == "registry.go" || e.Name() == "router.go" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		name = strings.TrimPrefix(name, "handler_")
		pascal := str.ToPascalCase(name)
		if !strings.HasSuffix(pascal, "Handler") {
			pascal += "Handler"
		}
		handlers = append(handlers, pascal)
	}
	return handlers, nil
}

func (h *HandlerGenerator) updateRegistry() error {
	handlers, err := h.listHandlers()
	if err != nil {
		return err
	}
	path := "internal/v1handler/registry.go"
	var file *os.File
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err = os.Create(path)
		if err != nil {
			return err
		}
	} else {
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	gh := genhelper.New("handler", templates.InternalV1HandlerRegistryGo)
	return gh.WithVar("handlers", handlers).WriteTo(file)
}

func (h *HandlerGenerator) updateAppModule() error {
	path := "internal/app/module.go"
	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return nil
	}
	gf.AddNamedImport("", filepath.Join(h.g.GoModuleName, "internal/v1handler"))
	gf.AddLineAfterString("return []fx.Option{", "\tfx.Provide(v1handler.Dependencies...),")
	return gf.Save()
}

func (h *HandlerGenerator) Update() error {
	if err := h.updateRegistry(); err != nil {
		return err
	}
	return h.updateAppModule()
}

func (h *HandlerGenerator) Create(name string, services []string) error {
	if err := h.createHandlerFile(name, services); err != nil {
		return err
	}
	if err := h.updateRegistry(); err != nil {
		return err
	}
	if err := h.updateRouter(str.ToPascalCase(name) + "Handler"); err != nil {
		return fmt.Errorf("failed to update router: %w", err)
	}
	return h.updateAppModule()
}

func (h *HandlerGenerator) updateRouter(handlerType string) error {
	path := "internal/v1handler/router.go"
	gofile, err := genhelper.LoadGoFile(path)
	if err != nil {
		return fmt.Errorf("failed to load router file: %w", err)
	}

	gofile.AddLineAfterRegex(`Mux\s+\*http.ServeMux`, fmt.Sprintf("\t%s *%s", handlerType, handlerType))

	if err := gofile.Save(); err != nil {
		return err
	}

	return nil
}
