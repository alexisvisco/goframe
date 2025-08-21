package genscaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

func (s *ScaffoldGenerator) generateFindAllRoute(name string) error {
	resourcePath := "/api/v1/" + str.ToSnakeCase(s.pluralize.Plural(name))
	handlerName := str.ToPascalCase(name)
	routeName := fmt.Sprintf("FindAll%s", str.ToPascalCase(s.pluralize.Plural(name)))

	imports := map[string]bool{
		filepath.Join(s.Gen.GoModuleName, "internal/types"): true,
		"github.com/alexisvisco/goframe/db/pagination":      true,
	}

	requestStructBody := "pagination.Params"
	responseStructBody := fmt.Sprintf("*pagination.Paginated[*types.%s]", str.ToPascalCase(name))

	routeBody := strings.Builder{}
	routeBody.WriteString(fmt.Sprintf("res, err := h.%sService.FindAllPaginated(r.Context(), req.Params)\n", str.ToCamelCase(name)))
	routeBody.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")

	routeResponse := strings.Builder{}
	routeResponse.WriteString(fmt.Sprintf("return httpx.JSON.Ok(&%sResponse{\n", str.ToPascalCase(routeName)))
	routeResponse.WriteString("\t\t\tPaginated: res,\n")
	routeResponse.WriteString("\t\t}), nil")

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
