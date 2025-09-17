package introspect

import (
	"fmt"
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func (ctx *ParseContext) ParseEnums(pkg *packages.Package) {
	// Check if we already parsed enums for this package to avoid redundant work
	if ctx.EnumsParsed[pkg.PkgPath] {
		return
	}

	// Group constants by their type
	typeConstants := make(map[string][]*types.Const)

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}

		// Only process exported constants
		if !obj.Exported() {
			continue
		}

		// Check if it's a constant
		if constObj, ok := obj.(*types.Const); ok {
			// Get the type name
			var typeName string
			if named, ok := constObj.Type().(*types.Named); ok {
				typeName = fmt.Sprintf("%s.%s", named.Obj().Pkg().Path(), named.Obj().Name())
			} else {
				// Skip basic types that aren't named
				continue
			}

			typeConstants[typeName] = append(typeConstants[typeName], constObj)
		}
	}

	// Process each type that has constants
	for typeName, constants := range typeConstants {
		// Only consider it an enum if there are at least 2 constants
		if len(constants) < 2 {
			continue
		}

		// Create enum based on the underlying type
		enum := &FieldTypeEnum{
			TypeName: typeName,
		}

		// Determine if it's string or int based enum
		firstConstant := constants[0]
		if firstConstant.Val().Kind() == constant.String {
			enum.KeyValuesString = make(map[string]string)
			for _, constObj := range constants {
				if constObj.Val().Kind() == constant.String {
					enum.KeyValuesString[constObj.Name()] = constant.StringVal(constObj.Val())
				}
			}
		} else if firstConstant.Val().Kind() == constant.Int {
			enum.KeyValuesInt = make(map[string]int)
			for _, constObj := range constants {
				if constObj.Val().Kind() == constant.Int {
					if val, ok := constant.Int64Val(constObj.Val()); ok {
						enum.KeyValuesInt[constObj.Name()] = int(val)
					}
				}
			}
		} else {
			// Skip non-string, non-int Enums
			continue
		}

		ctx.Enums[typeName] = enum
	}

	// Mark this package as having had enums parsed
	ctx.EnumsParsed[pkg.PkgPath] = true
}

func (ctx *ParseContext) detectEnum(pkg *packages.Package, named *types.Named) *FieldTypeEnum {
	enumKey := fmt.Sprintf("%s.%s", named.Obj().Pkg().Path(), named.Obj().Name())

	// Check if we already parsed this enum
	if enum, exists := ctx.Enums[enumKey]; exists {
		return enum
	}

	// Always try to parse enums for the package where this type is defined
	// even if it's the same package, to ensure we don't miss any
	targetPkgPath := named.Obj().Pkg().Path()

	// Load the package where the type is defined
	typePkg, err := ctx.LoadPackage(targetPkgPath)
	if err != nil {
		// If we can't load the package, try with the current package
		// in case it's a local type we haven't processed yet
		if targetPkgPath != pkg.PkgPath {
			return nil
		}
		typePkg = pkg
	}

	// Parse enums for this package - this is safe to call multiple times
	ctx.ParseEnums(typePkg)

	// Check again after parsing
	if enum, exists := ctx.Enums[enumKey]; exists {
		return enum
	}

	return nil
}
