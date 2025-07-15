package gentsclient

import (
	"embed"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"golang.org/x/exp/maps"
)

type TypescriptClientGenerator struct {
	schemaCode map[string]string                // schemaName -> Zod schema
	routeCode  map[string]string                // routeName -> function code
	lookup     map[string]string                // TypeName -> schemaName
	objects    map[string]introspect.ObjectType // schemaName -> object
	isRequest  map[string]bool                  // schemaName -> true if request
}

const indentStr = "  "

//go:embed templates
var fs embed.FS

func NewTypescriptClientGenerator() *TypescriptClientGenerator {
	t := &TypescriptClientGenerator{
		schemaCode: make(map[string]string),
		routeCode:  make(map[string]string),
		lookup:     make(map[string]string),
		objects:    make(map[string]introspect.ObjectType),
		isRequest:  make(map[string]bool),
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

	identifiers := maps.Keys(gen.schemaCode)
	slices.Sort(identifiers)
	for _, key := range identifiers {
		sb.WriteString(gen.schemaCode[key])
		sb.WriteString("\n")
	}

	sb.WriteString(gen.createInterfaces())
	sb.WriteString("\n")

	b, _ := fs.ReadFile("templates/fetcher.ts.tmpl")
	sb.WriteString(string(b))
	sb.WriteString("\n")

	routeIdentifiers := maps.Keys(gen.routeCode)
	slices.Sort(routeIdentifiers)
	for _, key := range routeIdentifiers {
		sb.WriteString(gen.routeCode[key])
		sb.WriteString("\n")
	}

	return sb.String()
}
