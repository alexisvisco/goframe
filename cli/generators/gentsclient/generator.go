package gentsclient

import (
	"embed"
	"fmt"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"golang.org/x/exp/maps"
)

type TypescriptClientGenerator struct {
	schemaCode     map[string]string                // schemaName -> Zod schema
	schemaOrder    []string                         // Keep insertion order of schemas
	routeCode      map[string]map[string]string     // namespace -> routeName -> function code
	lookup         map[string]string                // TypeName -> schemaName
	objects        map[string]introspect.ObjectType // schemaName -> object
	isRequest      map[string]bool                  // schemaName -> true if request
	rootImportPath string                           // import path of the root handler package
	typeNamePrefix map[string]string                // TypeName -> prefix to apply when exporting
}

const indentStr = "  "

//go:embed templates
var fs embed.FS

func NewTypescriptClientGenerator(rootImportPath string, typeNamePrefix map[string]string) *TypescriptClientGenerator {
	t := &TypescriptClientGenerator{
		schemaCode:     make(map[string]string),
		schemaOrder:    []string{},
		routeCode:      make(map[string]map[string]string),
		lookup:         make(map[string]string),
		objects:        make(map[string]introspect.ObjectType),
		isRequest:      make(map[string]bool),
		rootImportPath: rootImportPath,
		typeNamePrefix: typeNamePrefix,
	}

	t.createErrorSchema()

	return t
}

func (gen *TypescriptClientGenerator) indent(n int) string {
	return strings.Repeat(indentStr, n)
}

func (gen *TypescriptClientGenerator) File() string {
	var sb strings.Builder
	sb.WriteString("import { z, ZodSchema } from 'zod';\n\n")
	sb.WriteString("export type ValueOf<T> = T[keyof T];\n\n")
	for _, key := range gen.schemaOrder {
		sb.WriteString(gen.schemaCode[key])
		sb.WriteString("\n")
	}

	sb.WriteString(gen.createInterfaces())
	sb.WriteString("\n")

	b, _ := fs.ReadFile("templates/fetcher.ts.tmpl")
	sb.WriteString(string(b))
	sb.WriteString("\n")

	namespaces := maps.Keys(gen.routeCode)
	slices.Sort(namespaces)
	for _, ns := range namespaces {
		sb.WriteString(fmt.Sprintf("export namespace %sClient {\n", str.ToPascalCase(ns)))
		routeIdentifiers := maps.Keys(gen.routeCode[ns])
		slices.Sort(routeIdentifiers)
		for _, key := range routeIdentifiers {
			sb.WriteString(gen.addIndent(gen.routeCode[ns][key], 1))
			sb.WriteString("\n")
		}
		sb.WriteString("}\n")
	}

	return sb.String()
}
