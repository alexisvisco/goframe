package genhttp

import (
	"embed"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
	"github.com/alexisvisco/goframe/http/apidoc"
)

// HTTPGenerator handles http related files and handlers.
type HTTPGenerator struct{ Gen *generators.Generator }

//go:embed templates
var fs embed.FS

func (p *HTTPGenerator) Generate() error {
	files := []generators.FileConfig{
		p.createProvider("internal/provide/provide_http.go"),
		p.createRouter("internal/v1handler/router.go"),
		p.createOrUpdateRegistry(),
	}

	if err := p.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if p.Gen.HTTPServer {
		if err := p.Update(); err != nil {
			return err
		}
	}
	return nil
}

func (p *HTTPGenerator) createProvider(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/provide_http.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.Gen.GoModuleName, "config"), "config")
		},
		Skip: !p.Gen.HTTPServer,
	}
}

func (p *HTTPGenerator) createRouter(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/router.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("example", p.Gen.ExampleHTTPFiles)
		},
		Skip: !p.Gen.HTTPServer,
	}
}

func (p *HTTPGenerator) createHandler(name string, services []string) generators.FileConfig {
	path := filepath.Join("internal/v1handler", fmt.Sprintf("handler_%s.go", str.ToSnakeCase(name)))

	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/new_handler.go.tmpl")),
		Skip:     p.Gen.SkipDirectoryIfExists(path),
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
				if !strings.HasSuffix(p, "CreateService") {
					p += "CreateService"
				}
				svcs = append(svcs, dep{ServiceName: p, VarName: str.ToCamelCase(p)})
			}
			g.WithVar("services", svcs).WithImport("go.uber.org/fx", "fx")
			if len(svcs) > 0 {
				g.WithImport(filepath.Join(p.Gen.GoModuleName, "internal/types"), "types")
			}
		},
	}
}

func (p *HTTPGenerator) createOrUpdateRegistry() generators.FileConfig {
	path := "internal/v1handler/registry.go"
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/registry.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			handlers, _ := p.listHandlers()
			g.WithVar("handlers", handlers)
		},
	}
}

func (p *HTTPGenerator) updateAppModule() error {
	path := "internal/app/module.go"
	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return nil
	}
	gf.AddNamedImport("", filepath.Join(p.Gen.GoModuleName, "internal/v1handler"))
	gf.AddLineAfterString("return []fx.Option{", "\tfx.Provide(v1handler.Dependencies...),")
	return gf.Save()
}

func (p *HTTPGenerator) Update() error {
	err := p.Gen.GenerateFile(p.createOrUpdateRegistry())
	if err != nil {
		return fmt.Errorf("failed to generate registry: %w", err)
	}
	err = p.updateAppModule()
	if err != nil {
		return fmt.Errorf("failed to update app module: %w", err)
	}

	if err := p.GenerateRoutes(nil); err != nil {
		return fmt.Errorf("failed to generate routes: %w", err)
	}

	return nil
}

func (p *HTTPGenerator) GenerateHandler(name string, services []string) error {
	files := []generators.FileConfig{
		p.createHandler(name, services),
		p.createOrUpdateRegistry(),
	}

	if err := p.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if err := p.UpdateRouter(str.ToPascalCase(name) + "Handler"); err != nil {
		return fmt.Errorf("failed to update router: %w", err)
	}
	return p.updateAppModule()
}

func (p *HTTPGenerator) UpdateRouter(handlerType string) error {
	path := "internal/v1handler/router.go"
	gofile, err := genhelper.LoadGoFile(path)
	if err != nil {
		return fmt.Errorf("failed to load router file: %w", err)
	}
	gofile.AddLineAfterRegex(`Mux\s+\*http.ServeMux`, fmt.Sprintf("\t%s *%s", handlerType, handlerType))
	return gofile.Save()
}

func (p *HTTPGenerator) GenerateRoute(handler, method string, newFile, noMiddleware bool) error {
	handlerSnake := str.ToSnakeCase(handler)
	methodSnake := str.ToSnakeCase(method)
	methodPascal := str.ToPascalCase(method)
	handlerPascal := str.ToPascalCase(handler)

	tmpl := typeutil.Must(fs.ReadFile("templates/new_route.go.tmpl"))

	gen := func(g *genhelper.GenHelper) {
		g.WithVar("method_pascal", methodPascal).
			WithVar("handler_pascal", handlerPascal).
			WithVar("nomiddleware", noMiddleware).
			WithVar("path", "").
			WithVar("method", "").
			WithVar("new_file", newFile)
	}

	if newFile {
		path := filepath.Join("internal/v1handler", fmt.Sprintf("%s_handler_%s.go", handlerSnake, methodSnake))
		return p.Gen.GenerateFile(generators.FileConfig{Path: path, Template: tmpl, Gen: gen})
	}

	path := filepath.Join("internal/v1handler", fmt.Sprintf("handler_%s.go", handlerSnake))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return p.Gen.GenerateFile(generators.FileConfig{Path: path, Template: tmpl, Gen: gen})
	}

	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return err
	}
	gf.AddNamedImport("", "net/http")
	gf.AddNamedImport("", "github.com/alexisvisco/goframe/http/httpx")
	gf.AddNamedImport("", "github.com/alexisvisco/goframe/http/params")

	genHelper := genhelper.New("method", tmpl)
	genHelper.WithVar("method_pascal", methodPascal).
		WithVar("handler_pascal", handlerPascal).
		WithVar("nomiddleware", noMiddleware).
		WithVar("path", "").
		WithVar("method", "").
		WithVar("new_file", false)

	code, err := genHelper.Generate()
	if err != nil {
		return err
	}
	gf.AddContent("\n" + code)
	return gf.Save()
}

func (p *HTTPGenerator) listHandlers() ([]string, error) {
	gopkg, err := genhelper.LoadGoPkg("internal/v1handler")
	if err != nil {
		return nil, fmt.Errorf("failed to load v1handler package: %w", err)
	}

	var handlers []string
	structs := gopkg.FindAllStructRegexp(regexp.MustCompile(`(\w+)Handler$`))
	for _, info := range structs {
		handlers = append(handlers, info.Name)
	}

	return handlers, nil
}

func (p *HTTPGenerator) collectRoutes(workdir string) ([]*apidoc.Route, error) {
	var routes []*apidoc.Route

	gopkg, err := genhelper.LoadGoPkg(filepath.Join(workdir, "internal/v1handler"), true)
	if err != nil {
		return nil, err
	}

	for _, file := range gopkg.Files {
		ast.Inspect(file.File, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok || fd.Doc == nil || fd.Recv == nil {
				return true
			}
			hasRoute := false
			for _, c := range fd.Doc.List {
				if strings.Contains(c.Text, "goframe:http_route") {
					hasRoute = true
					break
				}
			}
			if !hasRoute {
				return true
			}

			var structName string
			switch t := fd.Recv.List[0].Type.(type) {
			case *ast.StarExpr:
				if ident, ok := t.X.(*ast.Ident); ok {
					structName = ident.Name
				}
			case *ast.Ident:
				structName = t.Name
			}
			if structName == "" {
				return true
			}

			r, err := apidoc.ParseRoute(workdir, file.ImportPath, structName, fd.Name.Name)
			if err == nil {
				routes = append(routes, r)
			}
			return true
		})
	}
	return routes, nil
}

func (p *HTTPGenerator) GenerateRoutes(routes []*apidoc.Route) error {
	if routes == nil {
		var err error
		routes, err = p.collectRoutes(".")
		if err != nil {
			return fmt.Errorf("failed to collect routes: %w", err)
		}
	}

	if len(routes) == 0 {
		return nil
	}

	gf, err := genhelper.LoadGoFile("internal/v1handler/router.go")
	if err != nil {
		return fmt.Errorf("failed to load router file: %w", err)
	}

	for _, r := range routes {
		if r.ParentStructName == nil {
			continue
		}
		for pathRoute, methods := range r.Paths {
			for _, m := range methods {
				line := fmt.Sprintf("\tp.Mux.HandleFunc(\"%s %s\", p.%s.%s())", m, pathRoute, *r.ParentStructName, r.Name)
				gf.AddLineAfterRegex(`func\s+Router\(p\s+RouterParams\)\s+{`, line)
			}
		}
	}

	return gf.Save()
}
