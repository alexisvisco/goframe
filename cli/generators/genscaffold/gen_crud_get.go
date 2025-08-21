package genscaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

func (s *ScaffoldGenerator) generateGetByIDRoute(name string) error {
	resourcePath := "/api/v1/" + str.ToSnakeCase(s.pluralize.Plural(name)) + "/{id}"
	handlerName := str.ToPascalCase(name)
	routeName := fmt.Sprintf("Get%sByID", str.ToPascalCase(name))

	imports := map[string]bool{
		filepath.Join(s.Gen.GoModuleName, "internal/types"): true,
	}

	requestStructBody := "ID string `path:\"id\"`"
	responseStructBody := fmt.Sprintf("Data *types.%s", str.ToPascalCase(name))

	routeBody := strings.Builder{}
	routeBody.WriteString(fmt.Sprintf("res, err := h.%sService.FindByID(r.Context(), req.ID)\n", str.ToCamelCase(name)))
	routeBody.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")

	routeResponse := strings.Builder{}
	routeResponse.WriteString(fmt.Sprintf("return httpx.JSON.Ok(%sResponse{\n", str.ToPascalCase(routeName)))
	routeResponse.WriteString("\tData: res,\n")
	routeResponse.WriteString("}), nil")

	return s.HTTPGen.GenerateRoute(genhttp.RouteConfig{
		Handler:       handlerName,
		Name:          routeName,
		Method:        "GET",
		Path:          resourcePath,
		RequestBody:   requestStructBody,
		ResponseBody:  responseStructBody,
		RouteBody:     routeBody.String(),
		RouteResponse: routeResponse.String(),
		NewFile:       true,
		NoMiddleware:  true,
		Imports:       imports,
	})
}
