package genhelper

import (
	"fmt"
	"io"
	"strings"
	"text/template"
)

type GenHelper struct {
	// Name is the name of the generator.
	Name     string
	registry *importRegistry
	content  []byte
	data     map[string]interface{}
}

// New creates a new GenHelper instance with the given name.
func New(name string, content []byte) *GenHelper {
	return &GenHelper{
		Name:     name,
		registry: newImportRegistry(),
		content:  content,
	}
}

// WithImport adds an import path to the generator.
// If the same import path is added again, it will not be added again.
// If the import is different but with the same alias, we will create an incremental alias.
func (g *GenHelper) WithImport(importPath, importVariable string, optionalAlias ...string) *GenHelper {
	optionalAliasUsed := importVariable
	if len(optionalAlias) > 0 {
		optionalAliasUsed = optionalAlias[0]
	}
	finalAlias := g.registry.addImport(importPath, optionalAliasUsed)
	return g.WithVar(importVariable, finalAlias)
}

func (g *GenHelper) WithVar(key string, value interface{}) *GenHelper {
	if g.data == nil {
		g.data = make(map[string]interface{})
	}
	g.data[key] = value
	return g
}

func (g *GenHelper) Generate(at io.Writer) error {
	if g.content == nil {
		return fmt.Errorf("content is not set")
	}

	imports := make([]string, 0, len(g.registry.imports))
	for importPath, alias := range g.registry.alias {
		if alias != "" {
			imports = append(imports, fmt.Sprintf("%s \"%s\"", alias, importPath))
		} else {
			imports = append(imports, fmt.Sprintf("\"%s\"", importPath))
		}
	}

	importsTemplate := strings.Join(imports, "\n	")

	g.WithVar("imports", importsTemplate)

	// Parse the template content and execute it
	tmpl, err := template.New(g.Name).Funcs(template.FuncMap{
		// todo: add more functions if needed
	}).Parse(string(g.content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	return tmpl.Execute(at, g.data)
}

type importRegistry struct {
	imports      map[string]bool
	alias        map[string]string
	aliasCounter map[string]int
}

func newImportRegistry() *importRegistry {
	return &importRegistry{
		imports:      make(map[string]bool),
		alias:        make(map[string]string),
		aliasCounter: make(map[string]int),
	}
}

// addImport adds an import path to the registry.
// if the same import path is added again, it will not be added again.
// if the import is different but with same alias, we will create an incremental alias.
func (r *importRegistry) addImport(importPath string, alias ...string) string {
	// Check if this import path already exists
	if r.imports[importPath] {
		// Return the existing alias for this import path
		return r.alias[importPath]
	}

	var desiredAlias string
	if len(alias) > 0 && alias[0] != "" {
		desiredAlias = alias[0]
	} else {
		// Extract package name from import path as default alias
		parts := strings.Split(importPath, "/")
		desiredAlias = parts[len(parts)-1]
	}

	// Find a unique alias
	finalAlias := desiredAlias
	counter := r.aliasCounter[desiredAlias]

	// Check if this alias is already used by a different import
	for existingImport := range r.imports {
		if r.alias[existingImport] == finalAlias {
			// Alias conflict - increment counter and try again
			counter++
			finalAlias = fmt.Sprintf("%s%d", desiredAlias, counter)
		}
	}

	// Update counters and maps
	r.aliasCounter[desiredAlias] = counter
	r.imports[importPath] = true
	r.alias[importPath] = finalAlias

	return finalAlias
}
