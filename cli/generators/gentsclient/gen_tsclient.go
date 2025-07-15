package gentsclient

import (
	"embed"
	"fmt"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/http/apidoc"
	"golang.org/x/exp/maps"
)

type TypescriptClientGenerator struct {
	schemaCode map[string]string // schemaName -> Zod schema
	routeCode  map[string]string // routeName -> function code
	lookup     map[string]string // TypeName -> schemaName
}

const indentStr = "  "

//go:embed templates
var fs embed.FS

func NewTypescriptClientGenerator() *TypescriptClientGenerator {
	t := &TypescriptClientGenerator{
		schemaCode: make(map[string]string),
		lookup:     make(map[string]string),
		routeCode:  make(map[string]string),
	}

	t.createErrorSchema()

	return t
}

func (gen *TypescriptClientGenerator) indent(n int) string {
	return strings.Repeat(indentStr, n)
}

func (gen *TypescriptClientGenerator) AddSchema(prefix string, isRequestOrResponse bool, objectTypes ...introspect.ObjectType) {
	for _, objectType := range objectTypes {
		if _, ok := gen.lookup[objectType.TypeName]; ok {
			continue // schema already generated
		}

		// Check if this is a request type with no serializable fields
		if isRequestOrResponse && gen.hasNoSerializableFields(objectType) {
			continue // Skip generating schema for empty request types
		}

		schemaName := objectType.TypeName[strings.LastIndex(objectType.TypeName, ".")+1:]
		if prefix != "" {
			schemaName = prefix + "_" + schemaName
		}
		schemaName = str.ToCamelCase(schemaName) + "Schema"

		gen.lookup[objectType.TypeName] = schemaName
		gen.schemaCode[schemaName] = gen.generateZodSchema(schemaName, objectType, isRequestOrResponse)
	}
}

func (gen *TypescriptClientGenerator) hasNoSerializableFields(objectType introspect.ObjectType) bool {
	for _, field := range objectType.Fields {
		if !field.IsNotSerializable() {
			return false
		}
	}
	return true
}

func (gen *TypescriptClientGenerator) hasRequestFields(route apidoc.Route) bool {
	objectType := route.Request
	for _, field := range objectType.Fields {
		if !field.IsNotSerializable() {
			return true
		}
	}
	return false
}

func (gen *TypescriptClientGenerator) AddRoute(route apidoc.Route) {
	sb := strings.Builder{}
	responseType := gen.createResponseType(route)
	hasRequest := gen.hasRequestFields(route)

	// function signature
	if hasRequest {
		sb.WriteString(fmt.Sprintf("export async function %s(fetcher: Fetcher, request: %s): Promise<{data: %s, status: number, headers: Headers}> {\n",
			str.ToCamelCase(route.Name),
			gen.schemaNameToExportedType(gen.lookup[route.Request.TypeName]),
			responseType,
		))
	} else {
		sb.WriteString(fmt.Sprintf("export async function %s(fetcher: Fetcher): Promise<{data: %s, status: number, headers: Headers}> {\n",
			str.ToCamelCase(route.Name),
			responseType,
		))
	}

	path := gen.getFirstRoutePath(route.Paths)
	method := route.Paths[path][0]

	// Only add request parsing if there are request fields
	if hasRequest {
		sb.WriteString(fmt.Sprintf("%sconst parseResult = %s.safeParse(request);\n", gen.indent(1), gen.lookup[route.Request.TypeName]))
		sb.WriteString(fmt.Sprintf("%sif (!parseResult.success) {\n", gen.indent(1)))
		sb.WriteString(fmt.Sprintf("%sthrow new RequestParseError(parseResult.error);\n", gen.indent(2)))
		sb.WriteString(fmt.Sprintf("%s}\n", gen.indent(1)))
		sb.WriteString(fmt.Sprintf("%sconst safeRequest = parseResult.data;\n", gen.indent(1)))
	}

	// declare the base fetcher options needed for the request
	sb.WriteString(fmt.Sprintf("%slet options : FetcherOptions = {\n", gen.indent(1)))
	sb.WriteString(fmt.Sprintf("%spath: '%s',\n", gen.indent(2), path))
	sb.WriteString(fmt.Sprintf("%smethod: '%s',\n", gen.indent(2), method))
	sb.WriteString(fmt.Sprintf("%s}\n\n", gen.indent(1)))

	// Only add request-specific logic if there are request fields
	if hasRequest {
		// replace pathParams in path
		if route.Request.HasPathParams() {
			sb.WriteString(fmt.Sprintf("%ssetPathParams(options, safeRequest.pathParams);\n", gen.indent(1)))
		}

		// add query parameters
		if route.Request.HasSearchParams() {
			sb.WriteString(fmt.Sprintf("%ssetSearchParams(options, safeRequest.searchParams);\n", gen.indent(1)))
		}

		// add headers
		if route.Request.HasHeaders() {
			sb.WriteString(fmt.Sprintf("%ssetHeaders(options, safeRequest.headers);\n", gen.indent(1)))
		}

		// add cookies
		if route.Request.HasCookies() {
			sb.WriteString(fmt.Sprintf("%ssetCookies(options, safeRequest.cookies);\n", gen.indent(1)))
		}

		if route.Request.HasBody() {
			sb.WriteString(fmt.Sprintf("%ssetRequestBody(options, safeRequest.body);\n", gen.indent(1)))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%sconst statusesAllowedToSchema: { pattern: RegExp, schema: ZodSchema<any>, raw?: boolean }[] = [%s];\n", gen.indent(1), gen.getAllowedStatusCodesToSchema(route.StatusToResponse)))

	const callFetcher = `try {
    const response = await fetcher(options);
    return handleResponse(response, statusesAllowedToSchema);
  } catch (error) {
    if (error instanceof ErrorResponse || error instanceof RequestParseError || error instanceof ResponseParseError) {
      throw error;
    } else {
      throw new FetchError(error as Error);
    }
  }`

	sb.WriteString(fmt.Sprintf("%s%s\n", gen.indent(1), callFetcher))

	// end
	sb.WriteString("}")

	gen.routeCode[route.Name] = sb.String()
}

func (gen *TypescriptClientGenerator) File() string {
	var sb strings.Builder
	sb.WriteString("import { z, ZodSchema } from 'zod';\n\n")

	// sort keys for consistent order
	identifiers := maps.Keys(gen.schemaCode)
	slices.Sort(identifiers)

	for _, key := range identifiers {
		schemaCode := gen.schemaCode[key]
		sb.WriteString(schemaCode)
		sb.WriteString("\n")
	}

	sb.WriteString(gen.createInferSchema())
	sb.WriteString("\n")

	// add fetcher
	b, _ := fs.ReadFile("templates/fetcher.ts.tmpl")
	sb.WriteString(string(b))
	sb.WriteString("\n")

	// Add route functions
	routeIdentifiers := maps.Keys(gen.routeCode)
	slices.Sort(routeIdentifiers)
	for _, key := range routeIdentifiers {
		routeCode := gen.routeCode[key]
		sb.WriteString(routeCode)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (gen *TypescriptClientGenerator) generateZodSchema(schemaName string, obj introspect.ObjectType, requestOrResponse bool) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("export const %s = z.object({\n", schemaName))
	fields := map[string]*strings.Builder{}

	for _, field := range obj.Fields {
		if field.IsNotSerializable() {
			continue
		}
		zodType := gen.zodFieldType(field.Type)

		if field.Optional {
			zodType = fmt.Sprintf("%s.optional()", zodType)
		}

		if requestOrResponse {
			fieldKindToSchemaField := map[string]string{
				introspect.FieldKindForm:   "bodyForm",
				introspect.FieldKindFiles:  "bodyForm",
				introspect.FieldKindFile:   "bodyForm",
				introspect.FieldKindJSON:   "bodyJson",
				introspect.FieldKindPath:   "pathParams",
				introspect.FieldKindQuery:  "searchParams",
				introspect.FieldKindHeader: "headers",
				introspect.FieldKindCookie: "cookies",
			}

			for _, t := range field.Tags {
				if fieldKind, ok := fieldKindToSchemaField[t.Key]; ok {
					if _, exists := fields[fieldKind]; !exists {
						fields[fieldKind] = &strings.Builder{}
					}
					fields[fieldKind].WriteString(fmt.Sprintf("%s: %s,\n", field.ExposedName(), zodType))
				}
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s%s: %s,\n", gen.indent(1), field.ExposedName(), zodType))
		}
	}

	// Write out the categories AFTER processing all fields
	if requestOrResponse {
		// Special handling for bodyForm and bodyJson - they are mutually exclusive
		hasBodyForm := fields["bodyForm"] != nil && fields["bodyForm"].Len() > 0
		hasBodyJson := fields["bodyJson"] != nil && fields["bodyJson"].Len() > 0

		if hasBodyForm && hasBodyJson {
			// Create a union type for mutually exclusive body types
			sb.WriteString(gen.indent(1) + "body: z.union([\n")
			sb.WriteString(gen.indent(2) + "z.object({\n")
			sb.WriteString(gen.indent(3) + "formData: z.object({\n")
			sb.WriteString(gen.addIndent(fields["bodyForm"].String(), 4))
			sb.WriteString(gen.indent(3) + "})\n")
			sb.WriteString(gen.indent(2) + "}),\n")
			sb.WriteString(gen.indent(2) + "z.object({\n")
			sb.WriteString(gen.indent(3) + "json: z.object({\n")
			sb.WriteString(gen.addIndent(fields["bodyJson"].String(), 4))
			sb.WriteString(gen.indent(3) + "})\n")
			sb.WriteString(gen.indent(2) + "})\n")
			sb.WriteString(gen.indent(1) + "]),\n")
		} else if hasBodyForm {
			sb.WriteString(gen.indent(1) + "body: z.object({\n")
			sb.WriteString(gen.indent(2) + "formData: z.object({\n")
			sb.WriteString(gen.addIndent(fields["bodyForm"].String(), 3))
			sb.WriteString(gen.indent(2) + "})\n")
			sb.WriteString(gen.indent(1) + "}),\n")
		} else if hasBodyJson {
			sb.WriteString(gen.indent(1) + "body: z.object({\n")
			sb.WriteString(gen.indent(2) + "json: z.object({\n")
			sb.WriteString(gen.addIndent(fields["bodyJson"].String(), 3))
			sb.WriteString(gen.indent(2) + "})\n")
			sb.WriteString(gen.indent(1) + "}),\n")
		}

		// Write out other categories normally
		for key, value := range fields {
			if key != "bodyForm" && key != "bodyJson" && value.Len() > 0 {
				sb.WriteString(fmt.Sprintf("%s%s: z.object({\n", gen.indent(1), key))
				sb.WriteString(gen.addIndent(value.String(), 2))
				sb.WriteString(gen.indent(1) + "}),\n")
			}
		}
	}

	sb.WriteString("}).passthrough();\n")
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
		gen.AddSchema("", false, *ft.Object)
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
		sb.WriteString(fmt.Sprintf("%s%s: '%s',\n", gen.indent(1), strings.ToUpper(str.ToSnakeCase(key)), value))
	}

	// Process integer values
	for k, value := range enum.KeyValuesInt {
		key := strings.TrimPrefix(k, typeName)
		if key == "" {
			key = k // fallback to original key if prefix removal results in empty string
		}
		sb.WriteString(fmt.Sprintf("%s%s: %d,\n", gen.indent(1), strings.ToUpper(str.ToSnakeCase(key)), value))
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
	gen.schemaCode[enumSchemaName] = sb.String()
}

func (gen *TypescriptClientGenerator) createDurationSchema() {
	if _, ok := gen.lookup["durationSchema"]; ok {
		return // Duration schema already generated
	}
	sb := strings.Builder{}
	b, _ := fs.ReadFile("templates/duration.tmpl")
	sb.Write(b)
	sb.WriteString("\n")
	sb.WriteString("const durationSchema = z.number().int().nonnegative().transform((value) => { new Duration(value) })")
	gen.lookup["durationSchema"] = "durationSchema"
	gen.schemaCode["durationSchema"] = sb.String()
}

func (gen *TypescriptClientGenerator) createDateSchema() {
	if _, ok := gen.lookup["dateSchema"]; ok {
		return // Date schema already generated
	}
	gen.lookup["dateSchema"] = "dateSchema"
	gen.schemaCode["dateSchema"] = "const dateSchema = z.string().datetime().transform((str) => new Date(str))"
}

func (gen *TypescriptClientGenerator) createErrorSchema() {
	if _, ok := gen.lookup["errorSchema"]; ok {
		return // Error schema already generated
	}
	b, _ := fs.ReadFile("templates/error.ts.tmpl")
	gen.lookup["errorSchema"] = "errorSchema"
	gen.schemaCode["errorSchema"] = string(b) + "\n"
}

func (gen *TypescriptClientGenerator) createInferSchema() string {
	excludedSchemas := []string{"errorSchema", "dateSchema", "durationSchema"}
	var sb strings.Builder
	for schemaName, _ := range gen.schemaCode {
		if slices.Contains(excludedSchemas, schemaName) {
			continue // Skip excluded schemaCode
		}
		if strings.HasSuffix(schemaName, "EnumSchema") {
			continue // Skip enum schemaCode
		}

		inferName := str.ToPascalCase(strings.TrimSuffix(schemaName, "Schema"))
		sb.WriteString(fmt.Sprintf("export type %s = z.infer<typeof %s>;\n", inferName, schemaName))
	}
	return sb.String()
}

func (gen *TypescriptClientGenerator) addIndent(str string, n int) string {
	indentStr := gen.indent(n)
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" { // Only add indent to non-empty lines
			lines[i] = indentStr + line
		}
	}
	return strings.Join(lines, "\n")
}

func (gen *TypescriptClientGenerator) schemaNameToExportedType(name string) string {
	if strings.HasSuffix(name, "Schema") {
		return str.ToPascalCase(strings.TrimSuffix(name, "Schema"))
	}
	return str.ToPascalCase(name)
}

func (gen *TypescriptClientGenerator) createResponseType(route apidoc.Route) string {
	var responses []apidoc.StatusToResponse
	for _, response := range route.StatusToResponse {
		if response.Response != nil || response.IsRedirect {
			responses = append(responses, response)
		}
	}

	if len(responses) == 0 {
		return "void"
	}

	var types []string
	for _, resp := range responses {
		if resp.IsRedirect {
			types = append(types, "any")
		} else {
			typ := gen.schemaNameToExportedType(gen.lookup[resp.Response.TypeName])
			types = append(types, typ)
		}
	}
	return strings.Join(types, " | ")
}

func (gen *TypescriptClientGenerator) getFirstRoutePath(paths map[string][]string) string {
	if len(paths) == 0 {
		return ""
	}

	// Get the first path
	for path := range paths {
		methods := paths[path]
		if len(methods) > 0 {
			return path // Return the first path found
		}
	}

	return "" // No paths found
}

func (gen *TypescriptClientGenerator) getAllowedStatusCodesToSchema(responses []apidoc.StatusToResponse) string {
	if len(responses) == 0 {
		return ""
	}

	var items []string

	for _, response := range responses {
		if response.IsError {
			continue
		}

		if response.StatusPattern == nil {
			continue
		}

		// Convert Go regexp to JavaScript regex string
		pattern := fmt.Sprintf("/%s/", response.StatusPattern.String())

		var schema string
		if response.IsRedirect {
			schema = "z.any()"
		} else if response.Response != nil {
			schemaName := gen.lookup[response.Response.TypeName]
			if schemaName != "" {
				schema = schemaName
			} else {
				schema = "z.any()" // Fallback
			}
		}

		raw := ""
		if schema == "z.any()" {
			raw = ", raw: true"
		}
		item := fmt.Sprintf("{ pattern: %s, schema: %s%s }", pattern, schema, raw)
		items = append(items, item)
	}

	return strings.Join(items, ",\n")
}
