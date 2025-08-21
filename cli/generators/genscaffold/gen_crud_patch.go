package genscaffold

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

func (s *ScaffoldGenerator) generatePatchRoute(name string, fields []Field) error {
	resourcePath := "/api/v1/" + str.ToSnakeCase(s.pluralize.Plural(name)) + "/{id}"
	ignoredFields := []string{"id", "created_at", "updated_at"}
	handlerName := str.ToPascalCase(name)
	routeName := fmt.Sprintf("Update%s", str.ToPascalCase(name))

	imports := map[string]bool{
		filepath.Join(s.Gen.GoModuleName, "internal/types"): true,
	}

	requestStructBody := strings.Builder{}
	requestStructBody.WriteString("ID string `path:\"id\"`\n")
	for _, field := range fields {
		if slices.Contains(ignoredFields, field.Name) {
			continue
		}

		requestStructBody.WriteString(fmt.Sprintf("%s *%s `json:\"%s\"`\n", str.ToPascalCase(field.Name), field.GoType, field.Name))
		imp := s.importFromType(field)
		if imp != "" {
			imports[imp] = true
		}
	}

	responseStructBody := fmt.Sprintf("Data *types.%s", str.ToPascalCase(name))

	routeBody := strings.Builder{}
	routeBody.WriteString(fmt.Sprintf("entity := &types.%s{\n", str.ToPascalCase(name)))
	routeBody.WriteString("\t\tID: req.ID,\n")
	routeBody.WriteString("\t}\n\n")
	routeBody.WriteString("\tvar columnsToUpdate []string\n\n")

	for _, field := range fields {
		if slices.Contains(ignoredFields, field.Name) {
			continue
		}
		routeBody.WriteString(fmt.Sprintf("\tif req.%s != nil {\n", str.ToPascalCase(field.Name)))
		routeBody.WriteString(fmt.Sprintf("\t\tentity.%s = *req.%s\n", str.ToPascalCase(field.Name), str.ToPascalCase(field.Name)))
		routeBody.WriteString(fmt.Sprintf("\t\tcolumnsToUpdate = append(columnsToUpdate, \"%s\")\n", field.Name))
		routeBody.WriteString("\t}\n\n")
	}

	routeBody.WriteString(fmt.Sprintf("\tres, err := h.%sService.Update(r.Context(), entity, columnsToUpdate)\n", str.ToCamelCase(name)))
	routeBody.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}")

	routeResponse := strings.Builder{}
	routeResponse.WriteString(fmt.Sprintf("return httpx.JSON.Ok(%sResponse{\n", str.ToPascalCase(routeName)))
	routeResponse.WriteString("\tData: res,\n")
	routeResponse.WriteString("}), nil")

	return s.HTTPGen.GenerateRoute(genhttp.RouteConfig{
		Handler:       handlerName,
		Name:          routeName,
		Method:        "PATCH",
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
