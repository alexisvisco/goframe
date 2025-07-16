package genurlhelper

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/http/apidoc"
)

type URLHelperGenerator struct {
	Gen *generators.Generator
}

func (g *URLHelperGenerator) Generate(routes []*apidoc.Route) error {
	return g.Gen.GenerateFile(generators.FileConfig{
		Path:     "internal/v1handler/urlhelper/urlhelper.go",
		Template: g.buildFileTemplate(routes),
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithImport(filepath.Join(g.Gen.GoModuleName, "config"), "config")
		},
	})
}

func (g *URLHelperGenerator) buildFileTemplate(routes []*apidoc.Route) []byte {
	namespaces := map[string]map[string]string{}

	for _, r := range routes {
		ns := "root"
		if r.ParentStructName != nil {
			ns = strings.TrimSuffix(*r.ParentStructName, "Handler")
		}
		ns = str.ToCamelCase(ns)
		if _, ok := namespaces[ns]; !ok {
			namespaces[ns] = map[string]string{}
		}

		for path, methods := range r.Paths {
			for _, method := range methods {
				baseName := r.Name
				if r.NamedRoutes != nil {
					if mm, ok := r.NamedRoutes[path]; ok {
						if nr, ok2 := mm[method]; ok2 && nr != "" {
							baseName = nr
						}
					}
				}
				fnName := str.ToPascalCase(baseName)
				if _, exists := namespaces[ns][fnName]; exists {
					fnName = str.ToPascalCase(strings.ToLower(method) + "_" + baseName)
				}
				if _, exists := namespaces[ns][fnName]; exists {
					pathPart := strings.ReplaceAll(strings.ReplaceAll(path, "/", "_"), "{", "")
					pathPart = strings.ReplaceAll(pathPart, "}", "")
					pathPart = strings.Trim(pathPart, "_")
					fnName = str.ToPascalCase(strings.ToLower(method) + "_" + pathPart + "_" + baseName)
				}
				namespaces[ns][fnName] = g.buildRouteFunction(r, path, fnName)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("package urlhelper\n\n")
	sb.WriteString("import (\n\t\"fmt\"\n\t\"net/url\"\n\t\"strings\"\n\t{{ .imports}}\n)\n\n")

	// URLs struct with baseURL
	sb.WriteString("type URLs struct {\n\tbaseURL string\n}\n\n")

	// NewURLs constructor
	sb.WriteString("func NewURLs(c *config.Config) URLs {\n\treturn URLs{\n\t\tbaseURL: c.GetServer().URL,\n\t}\n}\n\n")

	// Sort namespaces for consistent output
	nsNames := make([]string, 0, len(namespaces))
	for k := range namespaces {
		nsNames = append(nsNames, k)
	}
	slices.Sort(nsNames)

	// Generate namespace accessor methods on URLs struct
	for _, ns := range nsNames {
		if len(namespaces[ns]) > 0 { // Only create struct if there are methods
			structName := str.ToPascalCase(ns) + "URL"
			sb.WriteString(fmt.Sprintf("func (u URLs) %s() %s {\n\treturn %s{\n\t\tbaseURL: u.baseURL,\n\t}\n}\n", str.ToPascalCase(ns), structName, structName))
		}
	}
	sb.WriteString("\n")

	// Generate namespace structs and their methods
	for _, ns := range nsNames {
		if len(namespaces[ns]) == 0 {
			continue // Skip empty namespaces
		}

		structName := str.ToPascalCase(ns) + "URL"
		sb.WriteString(fmt.Sprintf("type %s struct {\n\tbaseURL string\n}\n\n", structName))

		// Sort method names for consistent output
		methodNames := make([]string, 0, len(namespaces[ns]))
		for k := range namespaces[ns] {
			methodNames = append(methodNames, k)
		}
		slices.Sort(methodNames)

		for _, methodName := range methodNames {
			// Add method to the namespace struct
			method := namespaces[ns][methodName]
			// Replace function name with receiver method and update return to include baseURL
			lines := strings.Split(method, "\n")
			if len(lines) > 0 {
				// Replace "func functionName(" with "func (u StructName) functionName("
				firstLine := lines[0]
				if strings.HasPrefix(firstLine, "func ") {
					firstLine = strings.Replace(firstLine, "func ", fmt.Sprintf("func (u %s) ", structName), 1)
					lines[0] = firstLine
				}
			}
			// Update return statement to include baseURL
			for i, line := range lines {
				if strings.Contains(line, "return path") {
					lines[i] = "\treturn u.baseURL + path"
				}
			}
			sb.WriteString(strings.Join(lines, "\n"))
			sb.WriteString("\n")
		}
	}

	return []byte(sb.String())
}

func (g *URLHelperGenerator) buildRouteFunction(route *apidoc.Route, path, fnName string) string {
	var pathFields []introspect.Field
	var queryFields []introspect.Field
	if route.Request != nil {
		for _, f := range route.Request.Fields {
			for _, t := range f.Tags {
				if t.Key == introspect.FieldKindPath {
					pathFields = append(pathFields, f)
				}
				if t.Key == introspect.FieldKindQuery {
					queryFields = append(queryFields, f)
				}
			}
		}
	}

	var params []string
	for _, f := range pathFields {
		params = append(params, fmt.Sprintf("%s string", str.ToCamelCase(f.ExposedName())))
	}
	for _, f := range queryFields {
		params = append(params, fmt.Sprintf("%s string", str.ToCamelCase(f.ExposedName())))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("func %s(%s) string {\n", fnName, strings.Join(params, ", ")))
	sb.WriteString(fmt.Sprintf("\tpath := \"%s\"\n", path))
	for _, f := range pathFields {
		name := str.ToCamelCase(f.ExposedName())
		sb.WriteString(fmt.Sprintf("\tpath = strings.ReplaceAll(path, \"{%s}\", url.PathEscape(fmt.Sprint(%s)))\n", f.ExposedName(), name))
	}
	if len(queryFields) > 0 {
		sb.WriteString("\tq := url.Values{}\n")
		for _, f := range queryFields {
			name := str.ToCamelCase(f.ExposedName())
			sb.WriteString(fmt.Sprintf("\tif %s != \"\" { q.Set(\"%s\", fmt.Sprint(%s)) }\n", name, f.ExposedName(), name))
		}
		sb.WriteString("\tif enc := q.Encode(); enc != \"\" { path += \"?\" + enc }\n")
	}
	sb.WriteString("\treturn path\n")
	sb.WriteString("}\n")

	return sb.String()
}
