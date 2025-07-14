package introspect

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Field struct {
	Name     string     `json:"name"`
	Tags     []FieldTag `json:"tags"` // Changed from Tag to Tags (list)
	Type     FieldType  `json:"type"`
	Optional bool       `json:"optional"` // New field for optional detection
}

func (f Field) ExposedName() string {
	// if json tag is present, use that as the name
	// otherwise use query, path, header, etc. tags until the Name

	for _, tag := range f.Tags {
		if tag.Key == FieldKindJSON && tag.Value != "" {
			return tag.Value
		}
		if tag.Key == FieldKindQuery && tag.Value != "" {
			return tag.Value
		}
		if tag.Key == FieldKindPath && tag.Value != "" {
			return tag.Value
		}
		if tag.Key == FieldKindHeader && tag.Value != "" {
			return tag.Value
		}
		if tag.Key == FieldKindForm && tag.Value != "" {
			return tag.Value
		}
		if tag.Key == FieldKindCookie && tag.Value != "" {
			return tag.Value
		}
		if tag.Key == FieldKindFile && tag.Value != "" {
			return tag.Value
		}
		if tag.Key == FieldKindFiles && tag.Value != "" {
			return tag.Value
		}
	}

	return f.Name
}

// IsNotSerializable Field is not serializable if it has json:"-" tag and no other field kind tags
func (f Field) IsNotSerializable() bool {
	var hasJSONTag bool
	var hasOtherTags bool

	for _, tag := range f.Tags {
		if tag.Key == FieldKindJSON && tag.Value == "-" {
			hasJSONTag = true
		} else if _, exists := tags[tag.Key]; exists {
			hasOtherTags = true
		}
	}

	return hasJSONTag && !hasOtherTags
}

type FieldKind = string

const (
	FieldKindJSON     FieldKind = "json"
	FieldKindQuery    FieldKind = "query"
	FieldKindHeader   FieldKind = "header"
	FieldKindForm     FieldKind = "form"
	FieldKindPath     FieldKind = "path"
	FieldKindCookie   FieldKind = "cookie"
	FieldKindFile     FieldKind = "file"
	FieldKindFiles    FieldKind = "files"
	FieldKindOptional FieldKind = "optional"
)

var tags = map[FieldKind]struct{}{
	FieldKindJSON:     {},
	FieldKindQuery:    {},
	FieldKindHeader:   {},
	FieldKindForm:     {},
	FieldKindPath:     {},
	FieldKindCookie:   {},
	FieldKindFile:     {},
	FieldKindFiles:    {},
	FieldKindOptional: {},
}

type FieldTag struct {
	Key     FieldKind `json:"key"`     // The tag key (json, query, header, etc.)
	Value   string    `json:"value"`   // The raw tag value
	Options []string  `json:"options"` // Additional options like "omitempty", "required", etc.
}

type ObjectType struct {
	TypeName string  `json:"type_name"`
	Fields   []Field `json:"fields"`
}

// Generic helper method to check for any field kind
func (o ObjectType) hasFieldKind(kinds ...string) bool {
	for _, field := range o.Fields {
		for _, tag := range field.Tags {
			has := slices.Contains(kinds, tag.Key)
			if has {
				return true
			}
		}
	}
	return false
}

func (o ObjectType) HasSearchParams() bool {
	return o.hasFieldKind(FieldKindQuery)
}

func (o ObjectType) HasPathParams() bool {
	return o.hasFieldKind(FieldKindPath)
}

func (o ObjectType) HasHeaders() bool {
	return o.hasFieldKind(FieldKindHeader)
}

func (o ObjectType) HasCookies() bool {
	return o.hasFieldKind(FieldKindCookie)
}

func (o ObjectType) HasBody() bool {
	return o.hasFieldKind(FieldKindJSON, FieldKindForm, FieldKindFile, FieldKindFiles)
}

func (o ObjectType) HasJSONBody() bool {
	return o.hasFieldKind(FieldKindJSON)
}

func (o ObjectType) HasFormBody() bool {
	return o.hasFieldKind(FieldKindForm, FieldKindFile, FieldKindFiles)
}

type FieldType struct {
	Primitive FieldTypePrimitive `json:"primitive,omitempty"` // The primitive type (string, int, etc.)

	Map    *FieldTypeMap   `json:"map,omitempty"`    // For map types
	Array  *FieldTypeArray `json:"array,omitempty"`  // For array/slice types
	Object *ObjectType     `json:"object,omitempty"` // For struct types
	Enum   *FieldTypeEnum  `json:"enum,omitempty"`   // For enum types
}

type FieldTypeMap struct {
	Key   FieldType `json:"key"`
	Value FieldType `json:"value"`
}

type FieldTypeArray struct {
	ItemType FieldType `json:"item_type"` // The type of items in the array/slice
}

type FieldTypeEnum struct {
	TypeName string `json:"type_name"` // The type name of the enum

	KeyValuesString map[string]string `json:"key_values_string,omitempty"`
	KeyValuesInt    map[string]int    `json:"key_values_int,omitempty"`
}

type FieldTypePrimitive string

const (
	FieldTypePrimitiveString   FieldTypePrimitive = "string"
	FieldTypePrimitiveInt      FieldTypePrimitive = "int"
	FieldTypePrimitiveFloat    FieldTypePrimitive = "float"
	FieldTypePrimitiveBool     FieldTypePrimitive = "bool"
	FieldTypePrimitiveTime     FieldTypePrimitive = "time"
	FieldTypePrimitiveDuration FieldTypePrimitive = "duration"
	FieldTypePrimitiveArray    FieldTypePrimitive = "array"
	FieldTypePrimitiveMap      FieldTypePrimitive = "map"
	FieldTypePrimitiveObject   FieldTypePrimitive = "object"
	FieldTypePrimitiveEnum     FieldTypePrimitive = "enum"
	FieldTypePrimitiveAny      FieldTypePrimitive = "any"
	FieldTypePrimitiveFile     FieldTypePrimitive = "file"
)

// ParseContext holds the parsing state to prevent circular references
type ParseContext struct {
	Visited  map[string]*ObjectType    // key: package.TypeName
	Enums    map[string]*FieldTypeEnum // key: package.TypeName
	Packages map[string]*packages.Package
	RootPath string
}

func ParseStruct(rootPath, relPkgPath, structName string) (*ObjectType, error) {
	ctx := &ParseContext{
		Visited:  make(map[string]*ObjectType),
		Enums:    make(map[string]*FieldTypeEnum),
		Packages: make(map[string]*packages.Package),
		RootPath: rootPath,
	}

	// Load the target package
	pkg, err := ctx.LoadPackage(relPkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %s: %w", relPkgPath, err)
	}

	// Find the struct type
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		return nil, fmt.Errorf("struct %s not found in package %s", structName, relPkgPath)
	}

	namedType, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%s is not a named type", structName)
	}

	structType, ok := namedType.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct type", structName)
	}

	// Parse Enums for this package
	ctx.ParseEnums(pkg)

	// Parse the struct
	return ctx.parseStruct(pkg, structType, namedType)
}

func (ctx *ParseContext) LoadPackage(relPkgPath string) (*packages.Package, error) {
	if pkg, exists := ctx.Packages[relPkgPath]; exists {
		return pkg, nil
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Dir: ctx.RootPath,
	}

	pkgs, err := packages.Load(cfg, relPkgPath)
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("package not found: %s", relPkgPath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package has errors: %v", pkg.Errors)
	}

	ctx.Packages[relPkgPath] = pkg
	return pkg, nil
}

func (ctx *ParseContext) parseStruct(pkg *packages.Package, structType *types.Struct, namedType *types.Named) (*ObjectType, error) {
	// Check if already Visited (circular reference protection)
	// Use the actual package where the type is defined, not the package being processed
	typeKey := fmt.Sprintf("%s.%s", namedType.Obj().Pkg().Path(), namedType.Obj().Name())
	if obj, visited := ctx.Visited[typeKey]; visited {
		return obj, nil
	}

	obj := &ObjectType{
		TypeName: typeKey, // This will now correctly reflect the package where the struct is actually defined
		Fields:   []Field{},
	}
	ctx.Visited[typeKey] = obj

	// Find the AST node for this struct to get struct tags
	// Note: We need to find the AST in the correct package (the one where the type is defined)
	actualPkg := pkg
	if namedType.Obj().Pkg().Path() != pkg.PkgPath {
		// The type is defined in a different package, load it
		var err error
		actualPkg, err = ctx.LoadPackage(namedType.Obj().Pkg().Path())
		if err != nil {
			return nil, fmt.Errorf("failed to load package for type %s: %w", typeKey, err)
		}
	}

	var astStruct *ast.StructType
	for _, file := range actualPkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == namedType.Obj().Name() {
				if st, ok := ts.Type.(*ast.StructType); ok {
					astStruct = st
					return false
				}
			}
			return true
		})
		if astStruct != nil {
			break
		}
	}

	// Parse each field
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		structTag := structType.Tag(i)

		if !field.Exported() {
			continue // Skip unexported fields
		}

		// Get AST field for additional info
		var astField *ast.Field
		if astStruct != nil && i < len(astStruct.Fields.List) {
			astField = astStruct.Fields.List[i]
		}

		parsedField, err := ctx.parseField(actualPkg, field, structTag, astField)
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %s: %w", field.Name(), err)
		}

		obj.Fields = append(obj.Fields, *parsedField)
	}

	return obj, nil
}

func (ctx *ParseContext) parseField(pkg *packages.Package, field *types.Var, structTag string, astField *ast.Field) (*Field, error) {
	fieldName := field.Name()

	// Handle embedded fields
	if field.Embedded() {
		// For embedded fields, use the type name as field name
		if named, ok := field.Type().(*types.Named); ok {
			fieldName = named.Obj().Name()
		}
	}

	// Parse struct tags to get list of FieldTags
	fieldTags := ctx.parseFieldTags(structTag)

	// Parse field type
	fieldType, err := ctx.parseType(pkg, field.Type())
	if err != nil {
		return nil, err
	}

	// Check if field type is a pointer (for optional detection)
	isPointer := false
	if _, ok := field.Type().(*types.Pointer); ok {
		isPointer = true
	}

	// Override primitive for file types based on tags
	for _, tag := range fieldTags {
		if tag.Key == FieldKindFile || tag.Key == FieldKindFiles {
			fieldType.Primitive = FieldTypePrimitiveFile
			break
		}
	}

	// Determine if field is optional
	optional := ctx.isFieldOptional(fieldTags, isPointer)

	return &Field{
		Name:     fieldName,
		Tags:     fieldTags,
		Type:     *fieldType,
		Optional: optional,
	}, nil
}

func (ctx *ParseContext) parseFieldTags(structTag string) []FieldTag {
	var fieldTags []FieldTag

	if structTag == "" {
		// Return default json tag if no tags present
		return []FieldTag{{
			Key:     FieldKindJSON,
			Value:   "",
			Options: []string{},
		}}
	}

	tag := reflect.StructTag(structTag)

	// Check for all possible tag types
	tagKeys := []struct {
		key  string
		kind FieldKind
	}{
		{"json", FieldKindJSON},
		{"query", FieldKindQuery},
		{"header", FieldKindHeader},
		{"form", FieldKindForm},
		{"path", FieldKindPath},
		{"cookie", FieldKindCookie},
		{"file", FieldKindFile},
		{"files", FieldKindFiles},
		{"optional", FieldKindOptional},
	}

	// Process each tag type that exists
	for _, tagKey := range tagKeys {
		if value := tag.Get(tagKey.key); value != "" {
			// Parse the tag value and options
			parts := strings.Split(value, ",")
			tagValue := parts[0]
			var options []string
			if len(parts) > 1 {
				options = parts[1:]
			}

			fieldTags = append(fieldTags, FieldTag{
				Key:     tagKey.kind,
				Value:   tagValue,
				Options: options,
			})
		}
	}

	// If no tags were found, add default json tag
	if len(fieldTags) == 0 {
		fieldTags = append(fieldTags, FieldTag{
			Key:     FieldKindJSON,
			Value:   "",
			Options: []string{},
		})
	}

	return fieldTags
}

func (ctx *ParseContext) isFieldOptional(tags []FieldTag, isPointer bool) bool {
	// Check for optional tag
	for _, tag := range tags {
		if tag.Key == FieldKindOptional {
			return true
		}

		// Check for omitempty or omitzero options in any tag
		for _, option := range tag.Options {
			if option == "omitempty" || option == "omitzero" {
				return true
			}
		}
	}

	return false
}

func (ctx *ParseContext) parseFieldKind(structTag string) FieldKind {
	if structTag == "" {
		return FieldKindJSON // default
	}

	tag := reflect.StructTag(structTag)

	// Check for various tag types in order of preference
	tagKeys := []struct {
		key  string
		kind FieldKind
	}{
		{"json", FieldKindJSON},
		{"query", FieldKindQuery},
		{"header", FieldKindHeader},
		{"form", FieldKindForm},
		{"path", FieldKindPath},
		{"cookie", FieldKindCookie},
		{"file", FieldKindFile},
		{"files", FieldKindFiles},
		{"optional", FieldKindOptional}, // Optional can override others
	}

	for _, tagKey := range tagKeys {
		if value := tag.Get(tagKey.key); value != "" && value != "-" {
			return tagKey.kind
		}
	}

	return FieldKindJSON // default
}

func (ctx *ParseContext) parseType(pkg *packages.Package, t types.Type) (*FieldType, error) {
	switch typ := t.(type) {
	case *types.Basic:
		return ctx.parseBasicType(typ)
	case *types.Named:
		return ctx.parseNamedType(pkg, typ)
	case *types.Pointer:
		return ctx.parseType(pkg, typ.Elem())
	case *types.Slice:
		return ctx.parseSliceType(pkg, typ)
	case *types.Array:
		return ctx.parseArrayType(pkg, typ)
	case *types.Map:
		return ctx.parseMapType(pkg, typ)
	case *types.Struct:
		return ctx.parseStructType(pkg, typ)
	case *types.Interface:
		// Handle interface{} as any
		if typ.Empty() {
			return &FieldType{Primitive: FieldTypePrimitiveAny}, nil
		}
		return &FieldType{Primitive: FieldTypePrimitiveAny}, nil
	default:
		return &FieldType{Primitive: FieldTypePrimitiveAny}, nil
	}
}

func (ctx *ParseContext) parseBasicType(basic *types.Basic) (*FieldType, error) {
	switch basic.Kind() {
	case types.String:
		return &FieldType{Primitive: FieldTypePrimitiveString}, nil
	case types.Bool:
		return &FieldType{Primitive: FieldTypePrimitiveBool}, nil
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		return &FieldType{Primitive: FieldTypePrimitiveInt}, nil
	case types.Float32, types.Float64:
		return &FieldType{Primitive: FieldTypePrimitiveFloat}, nil
	default:
		return &FieldType{Primitive: FieldTypePrimitiveAny}, nil
	}
}

func (ctx *ParseContext) parseNamedType(pkg *packages.Package, named *types.Named) (*FieldType, error) {
	// Check for special time types
	pkgPath := named.Obj().Pkg().Path()
	typeName := named.Obj().Name()

	if pkgPath == "time" {
		switch typeName {
		case "Time":
			return &FieldType{Primitive: FieldTypePrimitiveTime}, nil
		case "Duration":
			return &FieldType{Primitive: FieldTypePrimitiveDuration}, nil
		}
	}

	// Check if this is an enum type
	enumKey := fmt.Sprintf("%s.%s", pkgPath, typeName)
	if enum, exists := ctx.Enums[enumKey]; exists {
		return &FieldType{
			Primitive: FieldTypePrimitiveEnum,
			Enum:      enum,
		}, nil
	}

	// Check underlying type
	switch underlying := named.Underlying().(type) {
	case *types.Basic:
		// This might be a type alias for a basic type or a potential enum
		basicType, err := ctx.parseBasicType(underlying)
		if err != nil {
			return nil, err
		}
		// Check if we have constants of this type (enum detection)
		if enum := ctx.detectEnum(pkg, named); enum != nil {
			return &FieldType{
				Primitive: FieldTypePrimitiveEnum,
				Enum:      enum,
			}, nil
		}
		return basicType, nil
	case *types.Struct:
		// This is a named struct type, parse it recursively
		obj, err := ctx.parseStruct(pkg, underlying, named)
		if err != nil {
			return nil, err
		}
		return &FieldType{
			Primitive: FieldTypePrimitiveObject,
			Object:    obj,
		}, nil
	default:
		return ctx.parseType(pkg, underlying)
	}
}

func (ctx *ParseContext) parseSliceType(pkg *packages.Package, slice *types.Slice) (*FieldType, error) {
	elemType, err := ctx.parseType(pkg, slice.Elem())
	if err != nil {
		return nil, err
	}
	return &FieldType{
		Primitive: FieldTypePrimitiveArray,
		Array:     &FieldTypeArray{ItemType: *elemType},
	}, nil
}

func (ctx *ParseContext) parseArrayType(pkg *packages.Package, array *types.Array) (*FieldType, error) {
	elemType, err := ctx.parseType(pkg, array.Elem())
	if err != nil {
		return nil, err
	}
	return &FieldType{
		Primitive: FieldTypePrimitiveArray,
		Array:     &FieldTypeArray{ItemType: *elemType},
	}, nil
}

func (ctx *ParseContext) parseMapType(pkg *packages.Package, mapType *types.Map) (*FieldType, error) {
	keyType, err := ctx.parseType(pkg, mapType.Key())
	if err != nil {
		return nil, err
	}
	valueType, err := ctx.parseType(pkg, mapType.Elem())
	if err != nil {
		return nil, err
	}
	return &FieldType{
		Primitive: FieldTypePrimitiveMap,
		Map:       &FieldTypeMap{Key: *keyType, Value: *valueType},
	}, nil
}

func (ctx *ParseContext) parseStructType(pkg *packages.Package, structType *types.Struct) (*FieldType, error) {
	// For anonymous structs, create a new ObjectType without a type name
	obj := &ObjectType{
		TypeName: "", // Anonymous struct has no type name
		Fields:   []Field{},
	}

	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		structTag := structType.Tag(i)

		if !field.Exported() {
			continue // Skip unexported fields
		}

		parsedField, err := ctx.parseField(pkg, field, structTag, nil)
		if err != nil {
			return nil, err
		}

		obj.Fields = append(obj.Fields, *parsedField)
	}

	return &FieldType{
		Primitive: FieldTypePrimitiveObject,
		Object:    obj,
	}, nil
}

// ParseStructByName parses a struct by package path and struct name using the existing context
func (ctx *ParseContext) ParseStructByName(relPkgPath, structName string) (*ObjectType, error) {
	// Load the target package
	pkg, err := ctx.LoadPackage(relPkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %s: %w", relPkgPath, err)
	}

	// Find the struct type
	obj := pkg.Types.Scope().Lookup(structName)
	if obj == nil {
		return nil, fmt.Errorf("struct %s not found in package %s", structName, relPkgPath)
	}

	namedType, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("%s is not a named type", structName)
	}

	structType, ok := namedType.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct type", structName)
	}

	// Parse enums for this package if not already done
	ctx.ParseEnums(pkg)

	// Parse the struct
	return ctx.parseStruct(pkg, structType, namedType)
}
