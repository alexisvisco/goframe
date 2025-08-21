package genscaffold

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

func (s *ScaffoldGenerator) generateCreateRoute(name string, fields []Field) error {
	resourcePath := "/api/v1/" + str.ToSnakeCase(s.pluralize.Plural(name))
	ignoredFields := []string{"id", "created_at", "updated_at"}
	handlerName := str.ToPascalCase(name)
	routeName := fmt.Sprintf("Create%s", str.ToPascalCase(name))

	imports := map[string]bool{
		filepath.Join(s.Gen.GoModuleName, "internal/types"): true,
	}

	requestStructBody := strings.Builder{}
	for _, field := range fields {
		if slices.Contains(ignoredFields, field.Name) {
			continue
		}

		requestStructBody.WriteString(fmt.Sprintf("%s %s `json:\"%s\"`\n", str.ToPascalCase(field.Name), field.GoType, field.Name))
		imp := s.importFromType(field)
		if imp != "" {
			imports[imp] = true
		}
	}

	responseStructBody := fmt.Sprintf("Data *types.%s", str.ToPascalCase(name))

	routeBody := strings.Builder{}
	routeBody.WriteString(fmt.Sprintf("res, err := h.%sService.Create(r.Context(), &types.%s{\n", str.ToCamelCase(name), str.ToPascalCase(name)))
	for _, field := range fields {
		if slices.Contains(ignoredFields, field.Name) {
			continue
		}

		routeBody.WriteString(fmt.Sprintf("\t%s: req.%s,\n", str.ToPascalCase(field.Name), str.ToPascalCase(field.Name)))
	}
	routeBody.WriteString("})\n")
	routeBody.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")

	routeResponse := strings.Builder{}
	routeResponse.WriteString(fmt.Sprintf("return httpx.JSON.Ok(%sResponse{\n", str.ToPascalCase(routeName)))
	routeResponse.WriteString("\tData: res,\n")
	routeResponse.WriteString("}), nil")

	return s.HTTPGen.GenerateRoute(genhttp.RouteConfig{
		Handler:       handlerName,
		Name:          routeName,
		Method:        "POST",
		Path:          resourcePath,
		RequestBody:   requestStructBody.String(),
		ResponseBody:  responseStructBody,
		RouteBody:     routeBody.String(),
		RouteResponse: routeResponse.String(),
		NewFile:       true,
		NoMiddleware:  true,
		Imports:       imports,
	})
}
