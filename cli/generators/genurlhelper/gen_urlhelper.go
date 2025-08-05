package genurlhelper

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

type URLHelperGenerator struct {
	Gen *generators.Generator
}

//go:embed templates
var templatesFS embed.FS

// NamespaceTemplateData represents data for the namespace.go.tmpl template
type NamespaceTemplateData struct {
	Pkg    string   // Package name (e.g., dashboard_urlhelper)
	Name   string   // Namespace struct name (e.g., "UserURLs")
	Routes []string // Array of rendered route method strings
}

// RootTemplateData represents data for the root.go.tmpl template
type RootTemplateData struct {
	Pkg      string
	Imports  string            // Import statements as a string
	Handlers []HandlerAccessor // List of handler accessors in the main struct
}

// HandlerAccessor represents a field in the root URLs struct
type HandlerAccessor struct {
	Name string // Field name (e.g., "User")
	Type string // Type name (e.g., "UserURLs")
}

// RouteTemplateData represents data for the route.go.tmpl template
type RouteTemplateData struct {
	RouteName      string       // Method name (e.g., "GetUserProfile")
	ParamsName     string       // Name of the parameter (e.g., "GetUserProfile" or "DashboardGetUserProfile" if it's in a subfolder)
	NamespaceName  string       // Receiver type name (e.g., "UserURLs")
	RoutePath      string       // URL path template (e.g., "/users/{id}")
	HasParams      bool         // Whether this route has any parameters
	HasParamsPath  bool         // Whether this route has path parameters
	HasParamsQuery bool         // Whether this route has query parameters
	Fields         []ParamField // All parameters (path + query)
	ParamsPath     []PathParam  // Path parameters only
	ParamsQuery    []QueryParam // Query parameters only
}

// ParamField represents a parameter in the Params struct
type ParamField struct {
	Name string // Parameter name (e.g., "ID", "Filter")
	Type string // Go type (e.g., "string", "*bool")
}

// PathParam represents a path parameter that needs replacement
type PathParam struct {
	Name     string // Go field name (e.g., "ID")
	PathName string // Template placeholder (e.g., "{id}")
}

// QueryParam represents a query parameter
type QueryParam struct {
	Name      string // Parameter name (e.g., "filter")
	QueryName string // Query name (e.g., "filter")
	Optional  bool   // Whether this parameter is optional (pointer type)
}

type HandlerTemplateData struct {
	File                  string
	RouteTemplateData     []RouteTemplateData
	NamespaceTemplateData *NamespaceTemplateData
}

type RootHandlerPackage struct {
	RootTemplateData RootTemplateData
	Namespaces       map[string]*HandlerTemplateData // struct name -> HandlerTemplateData
}

// Generate is the main entry point. It orchestrates the process.
// Will collect all root handlers and generate a URL helper file for each root handler.
// Image you have :
// - internal/v1handler/dashboard root -> {UserHandler, AuthHandler }
// - internal/v1handler/dashboard/taxes -> {TaxesHandler}
// - internal/v1handler/dashboard/users -> {UserHandler}
// - internal/v1handler root -> {JobHandler }
// - internal/v1handler/user -> {UserHandler}
// - internal/v1handler/user/company -> {CompanyHandler}
// - internal/v1handler/profile -> {ProfileHandler}
// - internal/v2handler root -> {UserHandler}
// It will generate:
// - internal/v1handler/dashboard_urlhelper/ -> With a file per handler -> {user.go, users_user.go, auth.go, taxes.go, root.go}
// - internal/v1handler_urlhelper/ -> With a file per handler -> {job.go, user.go, company.go, profile.go, root.go}
// - internal/v2handler_urlhelper/ -> With a file per handler -> {user.go, root.go}
// In case of a handler with the same name in different root handlers, it will generate a file with the pkg as prefix.
//
// Each file will contain a struct with methods to build URLs for each handler.
func (g *URLHelperGenerator) Generate() error {
	packages, err := genhelper.CollectRootHandlerPackages(g.Gen.WorkDir)
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		paths := []string{pkg.Path}
		paths = append(paths, pkg.Subfolders...)
		documentation, err := genhelper.CollectRoutesDocumentation(g.Gen.WorkDir, paths)
		structNameToNamespaceData := make(map[string]*HandlerTemplateData)
		if err != nil {
			return fmt.Errorf("failed to collect routes documentation for package %s: %w", pkg.Path, err)
		}

		rootHandlerPackage := RootHandlerPackage{
			Namespaces: make(map[string]*HandlerTemplateData),
		}

		for _, route := range documentation {
			if route.ParentStructName == nil {
				continue // Skip if no parent struct name for now
			}

			var namespaceData *HandlerTemplateData

			if data, exists := structNameToNamespaceData[fmt.Sprintf("%s.%s", route.PackagePath, *route.ParentStructName)]; exists {
				namespaceData = data // Reuse existing namespace data
			} else {
				namespaceName := str.ToSnakeCase(strings.TrimSuffix(*route.ParentStructName, "Handler")) // Remove "Handler" suffix for URL helper generation
				structNamespaceName := fmt.Sprintf("%sURLs", str.ToPascalCase(namespaceName))
				if _, exists := rootHandlerPackage.Namespaces[structNamespaceName]; exists { // prefix with pkg name if a collision occurs
					namespaceName = str.ToSnakeCase(fmt.Sprintf("%s_%s", str.ToPascalCase(filepath.Base(route.PackagePath)), namespaceName))
					structNamespaceName = fmt.Sprintf("%sURLs", str.ToPascalCase(namespaceName))
				}
				file := filepath.Join(g.Gen.WorkDir, pkg.Path, filepath.Base(pkg.Path)+"_urlhelper", str.ToSnakeCase(namespaceName)+".go")

				nd, exists := rootHandlerPackage.Namespaces[structNamespaceName]
				if !exists {
					nd = &HandlerTemplateData{
						File:              file, // e.g., "internal/v1handler/dashboard_urlhelper/user.go"
						RouteTemplateData: []RouteTemplateData{},
						NamespaceTemplateData: &NamespaceTemplateData{
							Pkg:  fmt.Sprintf("%s_urlhelper", filepath.Base(pkg.Path)), // e.g., "dashboard_urlhelper"
							Name: structNamespaceName,                                  // e.g., "UserURLs"
						},
					}
					rootHandlerPackage.Namespaces[structNamespaceName] = nd
					structNameToNamespaceData[fmt.Sprintf("%s.%s", route.PackagePath, *route.ParentStructName)] = nd // Store for reuse
				}

				namespaceData = nd
			}

			alreadyExistRouteName := map[string]struct{}{}
			for path, methods := range route.Paths {
				for _, method := range methods {
					routeName := str.ToPascalCase(route.Name) // Convert to PascalCase for method name
					nameForMethod, existNameForMethod := route.NamedRoutes[path]
					if existNameForMethod {
						value, exists := nameForMethod[method]
						if exists {
							routeName = str.ToPascalCase(value) // Use the named route if it exists
						}
					}

					if _, exists := alreadyExistRouteName[routeName]; exists {
						slog.Warn("Duplicate route name %s as %s in package %s and struct, skipping.", route.Name, pkg.Path, route.ParentStructName)
						continue // Skip if route name already exists in this namespace
					}

					paramName := routeName
					if pkg.Path != route.PackagePath {
						paramName = fmt.Sprintf("%s%s", str.ToPascalCase(filepath.Base(route.PackagePath)), routeName)
					}

					routeData := RouteTemplateData{
						RouteName:      routeName, // e.g., "GetUserProfile"
						ParamsName:     paramName,
						NamespaceName:  namespaceData.NamespaceTemplateData.Name, // e.g., "UserURLs"
						RoutePath:      path,
						HasParams:      route.Request.HasSearchParams() || route.Request.HasPathParams(),
						HasParamsPath:  route.Request.HasPathParams(),
						HasParamsQuery: route.Request.HasSearchParams(),
						Fields:         g.buildFields(route.Request),
						ParamsPath:     g.buildPathParams(route.Request),
						ParamsQuery:    g.buildQueryParams(route.Request),
					}

					// Add the route data to the namespace
					namespaceData.RouteTemplateData = append(namespaceData.RouteTemplateData, routeData)
				}
			}
		}

		err = g.generateRootFile(pkg, structNameToNamespaceData)
		if err != nil {
			return err
		}

		err = g.generateNamespaceFile(pkg, rootHandlerPackage)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *URLHelperGenerator) generateNamespaceFile(pkg genhelper.RootHandlerPackage, rootHandlerPackage RootHandlerPackage) error {
	// Generate a file for each namespace
	for structName, data := range rootHandlerPackage.Namespaces {
		// need to execute template for each data.NamespaceTemplateData.Routes
		for _, routeData := range data.RouteTemplateData {
			builder := strings.Builder{}
			tmpl, err := template.New("route.go.tmpl").ParseFS(templatesFS, "templates/route.go.tmpl")
			if err != nil {
				return fmt.Errorf("failed to parse route template for package %s: %w", pkg.Path, err)
			}

			err = tmpl.Execute(&builder, routeData)
			if err != nil {
				return fmt.Errorf("failed to execute route template for package %s: %w", pkg.Path, err)
			}

			data.NamespaceTemplateData.Routes = append(data.NamespaceTemplateData.Routes, builder.String())
		}

		// now create the file
		file, err := os.OpenFile(data.File, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to create URL helper file for package %s and struct %s: %w", pkg.Path, structName, err)
		}
		defer file.Close()
		tmpl, err := template.New("namespace.go.tmpl").ParseFS(templatesFS, "templates/namespace.go.tmpl")
		if err != nil {
			return fmt.Errorf("failed to parse namespace template for package %s and struct %s: %w", pkg.Path, structName, err)
		}

		err = tmpl.Execute(file, data.NamespaceTemplateData)
		if err != nil {
			return fmt.Errorf("failed to execute namespace template for package %s and struct %s: %w", pkg.Path, structName, err)
		}
	}
	return nil
}

func (g *URLHelperGenerator) generateRootFile(pkg genhelper.RootHandlerPackage, structNameToNamespaceData map[string]*HandlerTemplateData) error {
	dir := filepath.Join(g.Gen.WorkDir, pkg.Path, filepath.Base(pkg.Path)+"_urlhelper")
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create URL helper directory for package %s: %w", pkg.Path, err)
	}

	// Generate the root handler file
	rootTemplateData := RootTemplateData{
		Pkg:     fmt.Sprintf("%s_urlhelper", filepath.Base(pkg.Path)), // e.g., "dashboard_urlhelper"
		Imports: fmt.Sprintf(`"%s"`, filepath.Join(g.Gen.GoModuleName, "config")),
	}

	// fill handlers accessors
	for _, data := range structNameToNamespaceData {
		rootTemplateData.Handlers = append(rootTemplateData.Handlers, HandlerAccessor{
			Name: data.NamespaceTemplateData.Name,
			Type: data.NamespaceTemplateData.Name,
		})
	}

	rootFile, err := os.OpenFile(filepath.Join(dir, "root.go"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create root URL helper file for package %s: %w", pkg.Path, err)
	}
	defer rootFile.Close()

	tmpl, err := template.New("root.go.tmpl").ParseFS(templatesFS, "templates/root.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse root template for package %s: %w", pkg.Path, err)
	}

	err = tmpl.Execute(rootFile, rootTemplateData)
	if err != nil {
		return fmt.Errorf("failed to execute root template for package %s: %w", pkg.Path, err)
	}

	return nil
}

func (g *URLHelperGenerator) buildFields(request *introspect.ObjectType) []ParamField {
	fields := make([]ParamField, 0, len(request.Fields))
	for _, field := range request.Fields {
		fields = append(fields, g.buildField(field))
	}
	return fields
}

func (g *URLHelperGenerator) buildPathParams(request *introspect.ObjectType) []PathParam {
	pathParams := make([]PathParam, 0, len(request.Fields))
	for _, field := range request.Fields {
		if tag, ok := field.PathParam(); ok {
			pathParams = append(pathParams, PathParam{
				Name:     field.Name,
				PathName: "{" + tag.Value + "}",
			})
		}
	}

	return pathParams
}

func (g *URLHelperGenerator) buildQueryParams(request *introspect.ObjectType) []QueryParam {
	queryParams := make([]QueryParam, 0, len(request.Fields))
	for _, field := range request.Fields {
		if tag, ok := field.QueryParam(); ok {
			queryParams = append(queryParams, QueryParam{
				Name:      field.Name,
				QueryName: tag.Value,
				Optional:  field.Optional,
			})
		}
	}

	return queryParams
}

func (g *URLHelperGenerator) buildField(field introspect.Field) ParamField {
	pf := ParamField{
		Name: field.Name,
	}

	switch field.Type.Primitive {
	case introspect.FieldTypePrimitiveInt:
		pf.Type = "int"
	case introspect.FieldTypePrimitiveFloat:
		pf.Type = "float64"
	case introspect.FieldTypePrimitiveBool:
		pf.Type = "bool"
	default:
		pf.Type = "string"
	}
	if field.Type.Array != nil {
		pf.Type = "[]" + pf.Type // Make it an array type
	}

	return pf
}
