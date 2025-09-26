package gentsclient

import (
	"fmt"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

// hasOnlyCtxTags returns true if the field has only ctx tags and no other serializable tags
func hasOnlyCtxTags(field introspect.Field) bool {
	hasCtxTag := false
	hasOtherSerializableTags := false

	for _, tag := range field.Tags {
		if tag.Key == "ctx" {
			hasCtxTag = true
		} else if tag.Key == "json" || tag.Key == "query" || tag.Key == "header" || tag.Key == "form" || tag.Key == "path" || tag.Key == "cookie" || tag.Key == "file" || tag.Key == "files" {
			hasOtherSerializableTags = true
		}
	}

	return hasCtxTag && !hasOtherSerializableTags
}

func (gen *TypescriptClientGenerator) AddSchema(prefix string, isRequest bool, objectTypes ...introspect.ObjectType) {
	for _, objectType := range objectTypes {
		var schemaName string
		var lookupKey string

		effectivePrefix := prefix
		if !objectType.IsAnonymous {
			if p, ok := gen.typeNamePrefix[objectType.TypeName]; ok {
				effectivePrefix = p
			}
		}

		if objectType.IsAnonymous {
			// For anonymous structs, create a unique name based on context
			if effectivePrefix != "" {
				schemaName = str.ToCamelCase(effectivePrefix) + "Schema"
			} else {
				schemaName = "AnonymousSchema"
			}
			// Use a unique key for anonymous structs based on their structure
			lookupKey = fmt.Sprintf("anonymous_%s", schemaName)
		} else {
			// For named structs, use the existing logic
			lookupKey = objectType.TypeName
			if _, ok := gen.lookup[lookupKey]; ok {
				continue
			}

			schemaName = objectType.TypeName[strings.LastIndex(objectType.TypeName, ".")+1:]
			if effectivePrefix != "" {
				schemaName = effectivePrefix + "_" + schemaName
			}
			schemaName = str.ToCamelCase(schemaName) + "Schema"
		}

		if _, ok := gen.lookup[lookupKey]; ok {
			continue
		}

		if isRequest && gen.hasNoSerializableFields(objectType) {
			continue
		}

		gen.lookup[lookupKey] = schemaName
		gen.schemaCode[schemaName] = gen.generateZodSchema(schemaName, objectType, isRequest)
		gen.objects[schemaName] = objectType
		gen.isRequest[schemaName] = isRequest
		gen.schemaOrder = append(gen.schemaOrder, schemaName)
	}
}

func (gen *TypescriptClientGenerator) hasNoSerializableFields(objectType introspect.ObjectType) bool {
	for _, field := range objectType.Fields {
		if !field.IsNotSerializable() && !hasOnlyCtxTags(field) {
			return false
		}
	}
	return true
}

func (gen *TypescriptClientGenerator) generateZodSchema(schemaName string, obj introspect.ObjectType, isRequest bool) string {
	var sb strings.Builder

	// Check if this schema contains recursive references
	if gen.hasRecursiveReference(obj, obj.TypeName) {
		// Use z.lazy() for recursive schemas
		sb.WriteString(fmt.Sprintf("export const %s: z.ZodType<%s> = z.lazy(() =>\n", schemaName, gen.schemaNameToExportedType(schemaName)))
		sb.WriteString(fmt.Sprintf("%sz.object({\n", gen.indent(1)))
	} else {
		sb.WriteString(fmt.Sprintf("export const %s = z.object({\n", schemaName))
	}
	fields := map[string]*strings.Builder{}

	for _, field := range obj.Fields {
		if field.IsNotSerializable() || field.IsCtx() {
			continue
		}
		zodType := gen.zodFieldType(field.Type, obj.TypeName, field.Name)
		if field.Optional {
			zodType = fmt.Sprintf("%s.optional()", zodType)
		}

		if isRequest {
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

	if isRequest {
		hasBodyForm := fields["bodyForm"] != nil && fields["bodyForm"].Len() > 0
		hasBodyJson := fields["bodyJson"] != nil && fields["bodyJson"].Len() > 0

		if hasBodyForm && hasBodyJson {
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

		for key, value := range fields {
			if key != "bodyForm" && key != "bodyJson" && value.Len() > 0 {
				sb.WriteString(fmt.Sprintf("%s%s: z.object({\n", gen.indent(1), key))
				sb.WriteString(gen.addIndent(value.String(), 2))
				sb.WriteString(gen.indent(1) + "}),\n")
			}
		}
	}

	// Close the schema appropriately
	if gen.hasRecursiveReference(obj, obj.TypeName) {
		sb.WriteString(fmt.Sprintf("%s}).passthrough(),\n", gen.indent(1)))
		sb.WriteString(");\n")
	} else {
		sb.WriteString("}).passthrough();\n")
	}
	return sb.String()
}

// hasRecursiveReference checks if an object type contains a recursive reference to itself
func (gen *TypescriptClientGenerator) hasRecursiveReference(obj introspect.ObjectType, targetTypeName string) bool {
	return gen.checkFieldsForRecursion(obj.Fields, targetTypeName, make(map[string]bool))
}

// checkFieldsForRecursion recursively checks fields for circular references
func (gen *TypescriptClientGenerator) checkFieldsForRecursion(fields []introspect.Field, targetTypeName string, visited map[string]bool) bool {
	for _, field := range fields {
		if field.IsNotSerializable() || field.IsCtx() {
			continue
		}
		if gen.checkFieldTypeForRecursion(field.Type, targetTypeName, visited) {
			return true
		}
	}
	return false
}

// checkFieldTypeForRecursion checks if a field type contains a recursive reference
func (gen *TypescriptClientGenerator) checkFieldTypeForRecursion(ft introspect.FieldType, targetTypeName string, visited map[string]bool) bool {
	if ft.Array != nil {
		return gen.checkFieldTypeForRecursion(ft.Array.ItemType, targetTypeName, visited)
	} else if ft.Map != nil {
		return gen.checkFieldTypeForRecursion(ft.Map.Key, targetTypeName, visited) ||
			gen.checkFieldTypeForRecursion(ft.Map.Value, targetTypeName, visited)
	} else if ft.Object != nil {
		// Direct self-reference
		if ft.Object.TypeName == targetTypeName {
			return true
		}

		// Avoid infinite recursion when checking nested objects
		if visited[ft.Object.TypeName] {
			return false
		}
		visited[ft.Object.TypeName] = true
		defer func() { delete(visited, ft.Object.TypeName) }()

		// Check nested object fields for indirect recursion
		return gen.checkFieldsForRecursion(ft.Object.Fields, targetTypeName, visited)
	}
	return false
}

func (gen *TypescriptClientGenerator) zodFieldType(ft introspect.FieldType, parentTypeName, fieldName string) string {
	zodFieldStr := strings.Builder{}
	excludedObjectPrimitive := []introspect.FieldTypePrimitive{
		introspect.FieldTypePrimitiveFile,
		introspect.FieldTypePrimitiveTime,
	}
	if ft.Array != nil {
		zodFieldStr.WriteString(fmt.Sprintf("z.array(%s)", gen.zodFieldType(ft.Array.ItemType, parentTypeName, fieldName)))
	} else if ft.Map != nil {
		zodFieldStr.WriteString(fmt.Sprintf("z.record(%s, %s)", gen.zodFieldType(ft.Map.Key, parentTypeName, fieldName), gen.zodFieldType(ft.Map.Value, parentTypeName, fieldName)))
	} else if ft.Enum != nil {
		gen.createEnumSchema("", *ft.Enum)
		refSchema := gen.lookup[ft.Enum.TypeName]
		zodFieldStr.WriteString(refSchema)
	} else if ft.Object != nil && !slices.Contains(excludedObjectPrimitive, ft.Primitive) {
		if ft.Object.IsAnonymous {
			// Generate name for anonymous struct based on parent type and field name
			var prefix string
			if parentTypeName != "" {
				// Extract just the type name without package path
				typeName := parentTypeName[strings.LastIndex(parentTypeName, ".")+1:]
				prefix = str.ToPascalCase(typeName) + str.ToPascalCase(fieldName)
			} else {
				prefix = str.ToPascalCase(fieldName)
			}
			gen.AddSchema(prefix, false, *ft.Object)
			lookupKey := fmt.Sprintf("anonymous_%sSchema", str.ToCamelCase(prefix))
			refSchema := gen.lookup[lookupKey]
			zodFieldStr.WriteString(refSchema)
		} else {
			gen.AddSchema("", false, *ft.Object)
			refSchema := gen.lookup[ft.Object.TypeName]
			zodFieldStr.WriteString(refSchema)
		}
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
		return
	}
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
	for k, value := range enum.KeyValuesString {
		key := strings.TrimPrefix(k, typeName)
		if key == "" {
			key = k
		}
		sb.WriteString(fmt.Sprintf("%s%s: '%s',\n", gen.indent(1), strings.ToUpper(str.ToSnakeCase(key)), value))
	}
	for k, value := range enum.KeyValuesInt {
		key := strings.TrimPrefix(k, typeName)
		if key == "" {
			key = k
		}
		sb.WriteString(fmt.Sprintf("%s%s: %d,\n", gen.indent(1), strings.ToUpper(str.ToSnakeCase(key)), value))
	}
	sb.WriteString("} as const;\n")
	sb.WriteString(fmt.Sprintf("export type %s = ValueOf<typeof %s>;\n", enumName, enumName))

	sb.WriteString(fmt.Sprintf("export const %s = z.union([", enumSchemaName))
	first := true
	for _, value := range enum.KeyValuesString {
		if !first {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("z.literal('%s')", value))
		first = false
	}
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
	gen.objects[enumSchemaName] = introspect.ObjectType{}
	gen.schemaOrder = append(gen.schemaOrder, enumSchemaName)
}

func (gen *TypescriptClientGenerator) createDurationSchema() {
	if _, ok := gen.lookup["durationSchema"]; ok {
		return
	}
	sb := strings.Builder{}
	b, _ := fs.ReadFile("templates/duration.ts.tmpl")
	sb.Write(b)
	sb.WriteString("\n")
	sb.WriteString("const durationSchema = z.number().int().nonnegative().transform((value) => new Duration(value))\n")
	gen.lookup["durationSchema"] = "durationSchema"
	gen.schemaCode["durationSchema"] = sb.String()
	gen.objects["durationSchema"] = introspect.ObjectType{}
	gen.schemaOrder = append(gen.schemaOrder, "durationSchema")
}

func (gen *TypescriptClientGenerator) createDateSchema() {
	if _, ok := gen.lookup["dateSchema"]; ok {
		return
	}
	gen.lookup["dateSchema"] = "dateSchema"
	gen.schemaCode["dateSchema"] = "const dateSchema = z.string().datetime({ offset: true }).transform((value) => new Date(value));\n"
	gen.objects["dateSchema"] = introspect.ObjectType{}
	gen.schemaOrder = append(gen.schemaOrder, "dateSchema")
}

func (gen *TypescriptClientGenerator) createErrorSchema() {
	if _, ok := gen.lookup["errorSchema"]; ok {
		return
	}
	b, _ := fs.ReadFile("templates/error.ts.tmpl")
	gen.lookup["errorSchema"] = "errorSchema"
	gen.schemaCode["errorSchema"] = string(b) + "\n"
	gen.objects["errorSchema"] = introspect.ObjectType{}
	gen.schemaOrder = append(gen.schemaOrder, "errorSchema")
}

func (gen *TypescriptClientGenerator) addIndent(str string, n int) string {
	indentStr := gen.indent(n)
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
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

func (gen *TypescriptClientGenerator) createInterfaces() string {
	excludedSchemas := []string{"errorSchema", "dateSchema", "durationSchema"}
	var sb strings.Builder
	for schemaName, obj := range gen.objects {
		if slices.Contains(excludedSchemas, schemaName) {
			continue
		}
		if strings.HasSuffix(schemaName, "EnumSchema") {
			continue
		}
		interfaceName := gen.schemaNameToExportedType(schemaName)
		// Generate an ISO TypeScript interface mirroring the schema structure
		sb.WriteString(fmt.Sprintf("export interface %s {\n", interfaceName))

		if gen.isRequest[schemaName] {
			fieldKindToInterfaceField := map[string]string{
				introspect.FieldKindForm:   "bodyForm",
				introspect.FieldKindFiles:  "bodyForm",
				introspect.FieldKindFile:   "bodyForm",
				introspect.FieldKindJSON:   "bodyJson",
				introspect.FieldKindPath:   "pathParams",
				introspect.FieldKindQuery:  "searchParams",
				introspect.FieldKindHeader: "headers",
				introspect.FieldKindCookie: "cookies",
			}

			fields := map[string]*strings.Builder{}
			usedNames := map[string]map[string]bool{}
			for _, field := range obj.Fields {
				if field.IsNotSerializable() || hasOnlyCtxTags(field) {
					continue
				}
				tsType := gen.tsFieldType(field.Type, obj.TypeName, field.Name)
				optional := ""
				if field.Optional {
					optional = "?"
				}
				for _, t := range field.Tags {
					if fieldKind, ok := fieldKindToInterfaceField[t.Key]; ok {
						if _, exists := fields[fieldKind]; !exists {
							fields[fieldKind] = &strings.Builder{}
							usedNames[fieldKind] = map[string]bool{}
						}
						name := field.ExposedName()
						if usedNames[fieldKind][name] {
							continue
						}
						usedNames[fieldKind][name] = true
						fields[fieldKind].WriteString(fmt.Sprintf("%s%s: %s;\n", name, optional, tsType))
					}
				}
			}

			hasBodyForm := fields["bodyForm"] != nil && fields["bodyForm"].Len() > 0
			hasBodyJson := fields["bodyJson"] != nil && fields["bodyJson"].Len() > 0

			if hasBodyForm && hasBodyJson {
				sb.WriteString(gen.indent(1) + "body: (\n")
				sb.WriteString(gen.indent(2) + "{\n")
				sb.WriteString(gen.indent(3) + "formData: {\n")
				sb.WriteString(gen.addIndent(fields["bodyForm"].String(), 4))
				sb.WriteString(gen.indent(3) + "}\n")
				sb.WriteString(gen.indent(2) + "} | {\n")
				sb.WriteString(gen.indent(3) + "json: {\n")
				sb.WriteString(gen.addIndent(fields["bodyJson"].String(), 4))
				sb.WriteString(gen.indent(3) + "}\n")
				sb.WriteString(gen.indent(2) + "}\n")
				sb.WriteString(gen.indent(1) + ");\n")
			} else if hasBodyForm {
				sb.WriteString(gen.indent(1) + "body: {\n")
				sb.WriteString(gen.indent(2) + "formData: {\n")
				sb.WriteString(gen.addIndent(fields["bodyForm"].String(), 3))
				sb.WriteString(gen.indent(2) + "}\n")
				sb.WriteString(gen.indent(1) + "};\n")
			} else if hasBodyJson {
				sb.WriteString(gen.indent(1) + "body: {\n")
				sb.WriteString(gen.indent(2) + "json: {\n")
				sb.WriteString(gen.addIndent(fields["bodyJson"].String(), 3))
				sb.WriteString(gen.indent(2) + "}\n")
				sb.WriteString(gen.indent(1) + "};\n")
			}

			order := []string{"pathParams", "searchParams", "headers", "cookies"}
			for _, key := range order {
				if fields[key] != nil && fields[key].Len() > 0 {
					sb.WriteString(fmt.Sprintf("%s%s: {\n", gen.indent(1), key))
					sb.WriteString(gen.addIndent(fields[key].String(), 2))
					sb.WriteString(gen.indent(1) + "};\n")
				}
			}
		} else {
			for _, field := range obj.Fields {
				if field.IsNotSerializable() || hasOnlyCtxTags(field) {
					continue
				}
				tsType := gen.tsFieldType(field.Type, obj.TypeName, field.Name)
				optional := ""
				if field.Optional {
					optional = "?"
				}
				sb.WriteString(fmt.Sprintf("%s%s%s: %s;\n", gen.indent(1), field.ExposedName(), optional, tsType))
			}
		}

		sb.WriteString("}\n")
	}
	return sb.String()
}

func (gen *TypescriptClientGenerator) tsFieldType(ft introspect.FieldType, parentTypeName, fieldName string) string {
	if ft.Array != nil {
		return fmt.Sprintf("Array<%s>", gen.tsFieldType(ft.Array.ItemType, parentTypeName, fieldName))
	} else if ft.Map != nil {
		return fmt.Sprintf("Record<%s, %s>", gen.tsFieldType(ft.Map.Key, parentTypeName, fieldName), gen.tsFieldType(ft.Map.Value, parentTypeName, fieldName))
	} else if ft.Enum != nil {
		enumSchema := gen.lookup[ft.Enum.TypeName]
		return gen.schemaNameToExportedType(enumSchema)
	} else if ft.Object != nil && ft.Primitive != introspect.FieldTypePrimitiveFile && ft.Primitive != introspect.FieldTypePrimitiveTime {
		if ft.Object.IsAnonymous {
			// Generate name for anonymous struct based on parent type and field name
			var prefix string
			if parentTypeName != "" {
				// Extract just the type name without package path
				typeName := parentTypeName[strings.LastIndex(parentTypeName, ".")+1:]
				prefix = str.ToPascalCase(typeName) + str.ToPascalCase(fieldName)
			} else {
				prefix = str.ToPascalCase(fieldName)
			}
			gen.AddSchema(prefix, false, *ft.Object)
			lookupKey := fmt.Sprintf("anonymous_%sSchema", str.ToCamelCase(prefix))
			refSchema := gen.lookup[lookupKey]
			return gen.schemaNameToExportedType(refSchema)
		} else {
			gen.AddSchema("", false, *ft.Object)
			refSchema := gen.lookup[ft.Object.TypeName]
			return gen.schemaNameToExportedType(refSchema)
		}
	} else if ft.Primitive != "" {
		switch ft.Primitive {
		case introspect.FieldTypePrimitiveString:
			return "string"
		case introspect.FieldTypePrimitiveInt, introspect.FieldTypePrimitiveFloat:
			return "number"
		case introspect.FieldTypePrimitiveBool:
			return "boolean"
		case introspect.FieldTypePrimitiveAny:
			return "any"
		case introspect.FieldTypePrimitiveFile:
			return "File"
		case introspect.FieldTypePrimitiveTime:
			return "Date"
		case introspect.FieldTypePrimitiveDuration:
			return "Duration"
		default:
			return "unknown"
		}
	}
	return "any"
}
