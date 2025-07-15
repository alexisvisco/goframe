package gentsclient

import (
	"fmt"
	"strings"

	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/http/apidoc"
)

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

	if hasRequest {
		sb.WriteString(fmt.Sprintf("%sconst parseResult = %s.safeParse(request);\n", gen.indent(1), gen.lookup[route.Request.TypeName]))
		sb.WriteString(fmt.Sprintf("%sif (!parseResult.success) {\n", gen.indent(1)))
		sb.WriteString(fmt.Sprintf("%sthrow new RequestParseError(parseResult.error);\n", gen.indent(2)))
		sb.WriteString(fmt.Sprintf("%s}\n", gen.indent(1)))
		sb.WriteString(fmt.Sprintf("%sconst safeRequest = parseResult.data;\n", gen.indent(1)))
	}

	sb.WriteString(fmt.Sprintf("%slet options : FetcherOptions = {\n", gen.indent(1)))
	sb.WriteString(fmt.Sprintf("%spath: '%s',\n", gen.indent(2), path))
	sb.WriteString(fmt.Sprintf("%smethod: '%s',\n", gen.indent(2), method))
	sb.WriteString(fmt.Sprintf("%s}\n\n", gen.indent(1)))

	if hasRequest {
		if route.Request.HasPathParams() {
			sb.WriteString(fmt.Sprintf("%ssetPathParams(options, safeRequest.pathParams);\n", gen.indent(1)))
		}
		if route.Request.HasSearchParams() {
			sb.WriteString(fmt.Sprintf("%ssetSearchParams(options, safeRequest.searchParams);\n", gen.indent(1)))
		}
		if route.Request.HasHeaders() {
			sb.WriteString(fmt.Sprintf("%ssetHeaders(options, safeRequest.headers);\n", gen.indent(1)))
		}
		if route.Request.HasCookies() {
			sb.WriteString(fmt.Sprintf("%ssetCookies(options, safeRequest.cookies);\n", gen.indent(1)))
		}
		if route.Request.HasBody() {
			sb.WriteString(fmt.Sprintf("%ssetRequestBody(options, safeRequest.body);\n", gen.indent(1)))
		}
	}

	sb.WriteString(fmt.Sprintf("\n%sconst statusesAllowedToSchema: { pattern: RegExp, schema: ZodSchema<any>, raw?: boolean }[] = [%s];\n", gen.indent(1), gen.getAllowedStatusCodesToSchema(route.StatusToResponse)))

	constCall := `try {
    const response = await fetcher(options);
    return handleResponse(response, statusesAllowedToSchema);
  } catch (error) {
    if (error instanceof ErrorResponse || error instanceof RequestParseError || error instanceof ResponseParseError) {
      throw error;
    } else {
      throw new FetchError(error as Error);
    }
  }`
	sb.WriteString(fmt.Sprintf("%s%s\n", gen.indent(1), constCall))
	sb.WriteString("}")

	gen.routeCode[route.Name] = sb.String()
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
	for path, methods := range paths {
		if len(methods) > 0 {
			return path
		}
	}
	return ""
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
		pattern := fmt.Sprintf("/%s/", response.StatusPattern.String())
		var schema string
		if response.IsRedirect {
			schema = "z.any()"
		} else if response.Response != nil {
			schemaName := gen.lookup[response.Response.TypeName]
			if schemaName != "" {
				schema = schemaName
			} else {
				schema = "z.any()"
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
