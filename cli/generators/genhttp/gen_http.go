package genhttp

import (
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

var DefaultBasePath = "internal/v1handler"

// HTTPGenerator handles http related files and handlers.
type HTTPGenerator struct {
	// BasePath path is the package that contains the handlers
	BasePath string

	// RootPath is the package that contains the router and registry files.
	// for instance if you have handlers in internal/v1handler/dashboard and registry and router in internal/v1handler,
	// the base path will be internal/v1handler.
	// If RootPath is empty, it will use BasePath as the base path.
	RootPath string

	Gen *generators.Generator
}

func (p *HTTPGenerator) GetBasePath() string {
	if p.BasePath == "" {
		return DefaultBasePath
	}
	return p.BasePath
}

func (p *HTTPGenerator) GetBasePkgName() string {
	if p.BasePath == "" {
		return "v1handler"
	}
	return filepath.Base(p.BasePath)
}

func (p *HTTPGenerator) GetRootPath() string {
	if p.RootPath == "" {
		return p.GetBasePath()
	}

	return p.RootPath
}

func (p *HTTPGenerator) GetRootPkgName() string {
	if p.RootPath == "" {
		return p.GetBasePkgName()
	}
	return filepath.Base(p.RootPath)
}

//go:embed templates
var fs embed.FS

// RouteConfig contains all configuration for generating a route
type RouteConfig struct {
	Handler       string
	Name          string
	Method        string
	Path          string
	NewFile       bool
	NoMiddleware  bool
	RequestBody   string
	ResponseBody  string
	RouteBody     string
	RouteResponse string
	Imports       map[string]bool
}

func (p *HTTPGenerator) Generate() error {
	files := []generators.FileConfig{
		p.createProvider("internal/provide/provide_http.go"),
		p.createRouter(path.Join(p.GetRootPath(), "router.go")),
		p.createOrUpdateRegistry(path.Join(p.GetRootPath(), "registry.go")),
	}

	if err := p.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if err := p.Update(); err != nil {
		return err
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
		Skip: p.Gen.SkipFileIfExists(path),
	}
}

func (p *HTTPGenerator) createRouter(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/router.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("pkgname", p.GetRootPkgName())
		},
		Skip: p.Gen.SkipFileIfExists(path),
	}
}

func (p *HTTPGenerator) createHandler(name string, services []string) generators.FileConfig {
	path := filepath.Join(p.GetBasePath(), fmt.Sprintf("handler_%s.go", str.ToSnakeCase(name)))

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
				if !strings.HasSuffix(p, "Service") {
					p += "Service"
				}
				svcs = append(svcs, dep{ServiceName: p, VarName: str.ToCamelCase(p)})
			}
			g.WithVar("services", svcs).WithImport("go.uber.org/fx", "fx")
			g.WithVar("pkgname", p.GetBasePkgName())
			if len(svcs) > 0 {
				g.WithImport(filepath.Join(p.Gen.GoModuleName, "internal/types"), "types")
			}
		},
	}
}

func (p *HTTPGenerator) createOrUpdateRegistry(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/registry.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			handlerInfos, _ := p.listHandlers()
			var handlers []string
			for _, h := range handlerInfos {
				handler := "New" + h.Name
				if h.ImportPath != filepath.Join(p.Gen.GoModuleName, p.GetRootPath()) {
					pkg := filepath.Base(h.ImportPath)
					g.WithImport(h.ImportPath, pkg)
					handler = fmt.Sprintf("%s.%s", filepath.Base(pkg), handler)
				}

				handlers = append(handlers, handler)
			}
			g.WithVar("handlers", handlers)
			g.WithVar("pkgname", p.GetRootPkgName())
		},
	}
}

func (p *HTTPGenerator) updateAppModule() error {
	x := "internal/app/module.go"
	gf, err := genhelper.LoadGoFile(x)
	if err != nil {
		return nil
	}
	gf.AddNamedImport("", filepath.Join(p.Gen.GoModuleName, p.GetRootPath()))
	gf.AddLineAfterString("return []fx.Option{", fmt.Sprintf("\tfx.Provide(%s.Dependencies...),", p.GetRootPkgName()))
	return gf.Save()
}

func (p *HTTPGenerator) Update() error {
	err := p.Gen.GenerateFile(p.createOrUpdateRegistry(path.Join(p.GetRootPath(), "registry.go")))
	if err != nil {
		return fmt.Errorf("failed to generate registry: %w", err)
	}
	err = p.updateAppModule()
	if err != nil {
		return fmt.Errorf("failed to update app module: %w", err)
	}

	if err := p.GenerateRoutes(); err != nil {
		return fmt.Errorf("failed to generate routes: %w", err)
	}

	return nil
}

func (p *HTTPGenerator) GenerateHandler(name string, services []string) error {
	files := []generators.FileConfig{
		p.createHandler(name, services),
		p.createOrUpdateRegistry(path.Join(p.GetRootPath(), "registry.go")),
	}

	if err := p.Gen.GenerateFiles(files); err != nil {
		return err
	}

	var mayImport []string
	if p.GetRootPath() != p.GetBasePath() {
		mayImport = append(mayImport, filepath.Join(p.Gen.GoModuleName, p.GetBasePath()))
	}

	if err := p.UpdateRouter(str.ToPascalCase(name)+"Handler", mayImport...); err != nil {
		return fmt.Errorf("failed to update router: %w", err)
	}
	return p.updateAppModule()
}

func (p *HTTPGenerator) UpdateRouter(handlerType string, externalImport ...string) error {
	x := filepath.Join(p.GetRootPath(), "router.go")
	gofile, err := genhelper.LoadGoFile(x)
	if err != nil {
		return fmt.Errorf("failed to load router file: %w", err)
	}

	fields := gofile.GetFieldsFromStruct("RouterParams")
	qualifiedType, packageName := handlerType, ""

	// Handle external imports
	if len(externalImport) > 0 {
		// Early return if field already exists
		for _, f := range fields {
			if f.TypeName == handlerType && f.PackagePath == externalImport[0] {
				return nil
			}
		}

		// Add imports and set up qualified type
		for _, imp := range externalImport {
			gofile.AddNamedImport("", imp)
		}
		packageName = filepath.Base(externalImport[0])
		qualifiedType = fmt.Sprintf("%s.%s", packageName, handlerType)
	}

	// Build existing field names set for O(1) lookups
	existingFields := make(map[string]bool, len(fields))
	for _, f := range fields {
		existingFields[f.FieldName] = true
	}

	// Generate unique field name using priority order
	candidates := []string{handlerType}
	if packageName != "" {
		candidates = append(candidates, str.ToPascalCase(packageName+handlerType))
	}

	fieldName := ""
	for _, candidate := range candidates {
		if !existingFields[candidate] {
			fieldName = candidate
			break
		}
	}

	// Fall back to numeric suffixes if needed
	if fieldName == "" {
		for i := 1; ; i++ {
			candidate := fmt.Sprintf("%s%d", handlerType, i)
			if !existingFields[candidate] {
				fieldName = candidate
				break
			}
		}
	}

	gofile.AddLineAfterRegex(`Mux\s+\*http.ServeMux`, fmt.Sprintf("\t%s *%s", fieldName, qualifiedType))
	return gofile.Save()
}

func (p *HTTPGenerator) GenerateRoute(config RouteConfig) error {
	handlerSnake := str.ToSnakeCase(config.Handler)
	routeNameSnake := str.ToSnakeCase(config.Name)
	routeNamePascal := str.ToPascalCase(config.Name)
	handlerPascal := str.ToPascalCase(config.Handler)

	tmpl := typeutil.Must(fs.ReadFile("templates/new_route.go.tmpl"))

	// Common generator configuration
	configureGen := func(g *genhelper.GenHelper) {
		g.WithVar("route_name_pascal", routeNamePascal).
			WithVar("handler_pascal", handlerPascal).
			WithVar("nomiddleware", config.NoMiddleware).
			WithVar("path", config.Path).
			WithVar("method", config.Method).
			WithVar("pkgname", p.GetBasePkgName()).
			WithVar("new_file", config.NewFile).
			WithVar("request_body", config.RequestBody).
			WithVar("response_body", config.ResponseBody).
			WithVar("route_body", config.RouteBody).
			WithVar("route_response", config.RouteResponse)

		if config.Imports != nil {
			for pkg, _ := range config.Imports {
				g.WithImport(pkg, "")
			}
		}
	}

	if config.NewFile {
		gen := func(g *genhelper.GenHelper) {
			configureGen(g)
		}
		x := filepath.Join(p.GetBasePath(), fmt.Sprintf("%s_handler_%s.go", handlerSnake, routeNameSnake))
		return p.Gen.GenerateFile(generators.FileConfig{Path: x, Template: tmpl, Gen: gen})
	}

	x := filepath.Join(p.GetBasePath(), fmt.Sprintf("handler_%s.go", handlerSnake))
	if _, err := os.Stat(x); os.IsNotExist(err) {
		gen := func(g *genhelper.GenHelper) {
			configureGen(g)
		}
		return p.Gen.GenerateFile(generators.FileConfig{Path: x, Template: tmpl, Gen: gen})
	}

	gf, err := genhelper.LoadGoFile(x)
	if err != nil {
		return err
	}

	gf.AddNamedImport("", "net/http")
	gf.AddNamedImport("", "github.com/alexisvisco/goframe/http/httpx")
	gf.AddNamedImport("", "github.com/alexisvisco/goframe/http/params")

	genHelper := genhelper.New("method", tmpl)
	configureGen(genHelper)

	code, err := genHelper.Generate()
	if err != nil {
		return err
	}

	gf.AddContent("\n" + code)
	return gf.Save()
}

type handlerInfo struct {
	Name       string
	ImportPath string
}

func (p *HTTPGenerator) listHandlers() ([]handlerInfo, error) {
	packages, err := genhelper.CollectRootHandlerPackages(p.Gen.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect root handler packages: %w", err)
	}

	for _, pkg := range packages {
		if pkg.Path == p.GetRootPath() {
			paths := []string{pkg.Path}
			paths = append(paths, pkg.Subfolders...)

			var handlers []handlerInfo

			for _, subpkg := range paths {
				gopkg, err := genhelper.LoadGoPkg(subpkg, false)
				if err != nil {
					return nil, fmt.Errorf("failed to load %s package: %w", subpkg, err)
				}

				structs := gopkg.FindAllStructRegexp(regexp.MustCompile(`(\w+)Handler$`))
				for _, info := range structs {
					handlers = append(handlers, handlerInfo{
						Name:       info.Name,
						ImportPath: info.ImportPath,
					})
				}
			}

			return handlers, nil
		}
	}

	return nil, nil
}
func (p *HTTPGenerator) GenerateRoutes() error {
	packages, err := genhelper.CollectRootHandlerPackages(p.Gen.WorkDir)
	if err != nil {
		return nil
	}

	for _, pkg := range packages {
		paths := []string{pkg.Path}
		paths = append(paths, pkg.Subfolders...)

		routeDocs, err := genhelper.CollectRoutesDocumentation(p.Gen.WorkDir, paths)
		if err != nil {
			return fmt.Errorf("failed to collect routes documentation: %w", err)
		}

		gf, err := genhelper.LoadGoFile(filepath.Join(pkg.Path, "router.go"))
		if err != nil {
			return fmt.Errorf("failed to load go file %s: %w", pkg.Path, err)
		}

		// fields has .FieldName and .PackagePath
		fields := gf.GetFieldsFromStruct("RouterParams")

		for _, doc := range routeDocs {
			if doc.ParentStructName == nil {
				continue
			}

			// Add the route to the router
			for pathRoute, methods := range doc.Paths {
				for _, method := range methods {
					parentStructName := *doc.ParentStructName
					for _, routerParamsField := range fields {
						if routerParamsField.PackagePath == doc.PackagePath {
							parentStructName = routerParamsField.FieldName
						}
					}

					line := fmt.Sprintf("\tp.Mux.HandleFunc(\"%s %s\", p.%s.%s())", method, pathRoute, parentStructName, doc.Name)
					gf.AddLineAfterRegex(`func\s+Router\(p\s+RouterParams\)\s+{`, line)
				}
			}
		}

		if err := gf.Save(); err != nil {
			return err
		}

	}

	return nil
}
