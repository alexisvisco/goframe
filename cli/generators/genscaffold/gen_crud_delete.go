package genscaffold

import (
	"fmt"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

func (s *ScaffoldGenerator) generateDeleteRoute(name string) error {
	resourcePath := "/api/v1/" + str.ToSnakeCase(s.pluralize.Plural(name)) + "/{id}"
	handlerName := str.ToPascalCase(name)
	routeName := fmt.Sprintf("Delete%s", str.ToPascalCase(name))

	requestStructBody := "ID string `path:\"id\"`"
	responseStructBody := ""

	routeBody := strings.Builder{}
	routeBody.WriteString(fmt.Sprintf("err := h.%sService.Delete(r.Context(), req.ID)\n", str.ToCamelCase(name)))
	routeBody.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")

	routeResponse := "return httpx.JSON.NoContent(), nil"

	return s.HTTPGen.GenerateRoute(genhttp.RouteConfig{
		Handler:       handlerName,
		Name:          routeName,
		Method:        "DELETE",
		Path:          resourcePath,
		RequestBody:   requestStructBody,
		ResponseBody:  responseStructBody,
		RouteBody:     routeBody.String(),
		RouteResponse: routeResponse,
		NewFile:       true,
		NoMiddleware:  true,
	})
}
