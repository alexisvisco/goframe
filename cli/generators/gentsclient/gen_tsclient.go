package gentsclient

import (
	"embed"
	"fmt"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

type TypescriptClientGenerator struct {
	schemas map[string]string // schemaName -> Zod schema
	lookup  map[string]string // TypeName -> schemaName
}

//go:embed templates
var fs embed.FS

func NewTypescriptClientGenerator() *TypescriptClientGenerator {
	t := &TypescriptClientGenerator{
		schemas: make(map[string]string),
		lookup:  make(map[string]string),
	}

	t.createErrorSchema()

	return t
}

func (gen *TypescriptClientGenerator) AddSchema(prefix string, objectTypes ...introspect.ObjectType) {
	for _, objectType := range objectTypes {
		if _, ok := gen.lookup[objectType.TypeName]; ok {
			continue // schema already generated
		}

		schemaName := objectType.TypeName[strings.LastIndex(objectType.TypeName, ".")+1:]
		if prefix != "" {
			schemaName = prefix + "_" + schemaName
		}
		schemaName = str.ToCamelCase(schemaName) + "Schema"

		gen.lookup[objectType.TypeName] = schemaName
		gen.schemas[schemaName] = gen.generateZodSchema(schemaName, objectType)
	}
}

func (gen *TypescriptClientGenerator) generateZodSchema(schemaName string, obj introspect.ObjectType) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("export const %s = z.object({\n", schemaName))
	for _, field := range obj.Fields {
		zodType := gen.zodFieldType(field.Type)
		if field.Optional {
			zodType = fmt.Sprintf("%s.optional()", zodType)
		}

		sb.WriteString(fmt.Sprintf("  %s: %s,\n", field.ExposedName(), zodType))
	}
	sb.WriteString("});\n")
	return sb.String()
}

func (gen *TypescriptClientGenerator) zodFieldType(ft introspect.FieldType) string {
	zodFieldStr := strings.Builder{}
	excludedObjectPrimitive := []introspect.FieldTypePrimitive{
		introspect.FieldTypePrimitiveFile,
		introspect.FieldTypePrimitiveTime,
	}
	if ft.Array != nil {
		zodFieldStr.WriteString(fmt.Sprintf("z.array(%s)", gen.zodFieldType(ft.Array.ItemType)))
	} else if ft.Map != nil {
		zodFieldStr.WriteString(fmt.Sprintf("z.record(%s, %s)", gen.zodFieldType(ft.Map.Key), gen.zodFieldType(ft.Map.Value)))
	} else if ft.Enum != nil {
		gen.createEnumSchema("", *ft.Enum)
		refSchema := gen.lookup[ft.Enum.TypeName]
		zodFieldStr.WriteString(fmt.Sprintf(refSchema))
	} else if ft.Object != nil && !slices.Contains(excludedObjectPrimitive, ft.Primitive) {
		gen.AddSchema("", *ft.Object)
		refSchema := gen.lookup[ft.Object.TypeName]
		zodFieldStr.WriteString(refSchema)
	} else if ft.Primitive != "" {
		switch ft.Primitive {
		case introspect.FieldTypePrimitiveString:
			zodFieldStr.WriteString("z.string()")
		case introspect.FieldTypePrimitiveInt, introspect.FieldTypePrimitiveFloat:
			zodFieldStr.WriteString("z.number()")
		case introspect.FieldTypePrimitiveBool:
			zodFieldStr.WriteString("z.boolean()")
		case introspect.FieldTypePrimitiveAny:
			zodFieldStr.WriteString("z.any()")
		case introspect.FieldTypePrimitiveFile:
			zodFieldStr.WriteString("z.instanceof(File)")
		case introspect.FieldTypePrimitiveTime:
			gen.createDateSchema()
			zodFieldStr.WriteString("dateSchema")
		case introspect.FieldTypePrimitiveDuration:
			gen.createDurationSchema()
			zodFieldStr.WriteString("durationSchema")
		default:
			zodFieldStr.WriteString("z.unknown()")
		}
	}

	return zodFieldStr.String()
}

func (gen *TypescriptClientGenerator) createEnumSchema(prefix string, enum introspect.FieldTypeEnum) {
	if _, ok := gen.lookup[enum.TypeName]; ok {
		return // enum already generated
	}

	// Handle case where TypeName might not contain a dot
	parts := strings.SplitN(enum.TypeName, ".", 2)
	var typeName string
	if len(parts) > 1 {
		typeName = parts[1]
	} else {
		typeName = parts[0]
	}

	name := typeName
	if prefix != "" {
		name = prefix + "_" + typeName
	}
	enumName := str.ToPascalCase(name) + "Enum"
	enumSchemaName := str.ToCamelCase(name) + "EnumSchema"

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("export const %s = {\n", enumName))

	// Process string values
	for k, value := range enum.KeyValuesString {
		key := strings.TrimPrefix(k, typeName)
		if key == "" {
			key = k // fallback to original key if prefix removal results in empty string
		}
		sb.WriteString(fmt.Sprintf("  %s: '%s',\n", strings.ToUpper(str.ToSnakeCase(key)), value))
	}

	// Process integer values
	for k, value := range enum.KeyValuesInt {
		key := strings.TrimPrefix(k, typeName)
		if key == "" {
			key = k // fallback to original key if prefix removal results in empty string
		}
		sb.WriteString(fmt.Sprintf("  %s: %d,\n", strings.ToUpper(str.ToSnakeCase(key)), value))
	}

	sb.WriteString("} as const;\n")

	// Create proper union type for zod schema
	sb.WriteString(fmt.Sprintf("export const %s = z.union([", enumSchemaName))

	// Add string literals
	first := true
	for _, value := range enum.KeyValuesString {
		if !first {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("z.literal('%s')", value))
		first = false
	}

	// Add numeric literals
	for _, value := range enum.KeyValuesInt {
		if !first {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("z.literal(%d)", value))
		first = false
	}

	sb.WriteString("]);\n")

	gen.lookup[enum.TypeName] = enumSchemaName
	gen.schemas[enumSchemaName] = sb.String()
}

func (gen *TypescriptClientGenerator) File() string {
	var sb strings.Builder
	sb.WriteString("import { z } from 'zod';\n\n")
	for _, schemaCode := range gen.schemas {
		sb.WriteString(schemaCode)
		sb.WriteString("\n")
	}
	sb.WriteString(gen.createInferSchema())
	return sb.String()
}

func (gen *TypescriptClientGenerator) createDurationSchema() {
	if _, ok := gen.lookup["durationSchema"]; ok {
		return // Duration schema already generated
	}
	sb := strings.Builder{}
	b, _ := fs.ReadFile("templates/duration.tmpl")
	sb.Write(b)
	sb.WriteString("\n")
	sb.WriteString("const durationSchema = z.number().int().nonnegative().transform((value) => { new Duration(value) }")
	gen.lookup["durationSchema"] = "durationSchema"
	gen.schemas["durationSchema"] = sb.String()
}

func (gen *TypescriptClientGenerator) createDateSchema() {
	if _, ok := gen.lookup["dateSchema"]; ok {
		return // Date schema already generated
	}
	gen.lookup["dateSchema"] = "dateSchema"
	gen.schemas["dateSchema"] = "const dateSchema = z.string().datetime().transform((str) => new Date(str))"
}

func (gen *TypescriptClientGenerator) createErrorSchema() {
	if _, ok := gen.lookup["errorSchema"]; ok {
		return // Error schema already generated
	}
	b, _ := fs.ReadFile("templates/error.ts.tmpl")
	gen.lookup["errorSchema"] = "errorSchema"
	gen.schemas["errorSchema"] = string(b) + "\n"
}

func (gen *TypescriptClientGenerator) createInferSchema() string {
	excludedSchemas := []string{"errorSchema", "dateSchema", "durationSchema"}
	var sb strings.Builder
	for schemaName, _ := range gen.schemas {
		if slices.Contains(excludedSchemas, schemaName) {
			continue // Skip excluded schemas
		}
		if strings.HasSuffix(schemaName, "EnumSchema") {
			continue // Skip enum schemas
		}

		inferName := str.ToPascalCase(strings.TrimSuffix(schemaName, "Schema"))
		sb.WriteString(fmt.Sprintf("export type %s = z.infer<typeof %s>;\n", inferName, schemaName))
	}
	sb.WriteString("export type ErrorResponse = z.infer<typeof errorSchema>;\n")
	return sb.String()
}
