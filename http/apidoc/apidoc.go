package apidoc

import (
	"regexp"
	"strings"
	"unicode"
)

type RouteDefinition struct {
	Path   string
	Method string
	Name   string // optional name for this specific route
}

type FromDoc struct {
	Routes          []RouteDefinition
	Requests        string
	Responses       []string
	RequiredHeaders []string
	StatusResponses []FromDocStatusToResponse
}

type FromDocStatusToResponse struct {
	StatusPattern *regexp.Regexp
	Response      string
}

func ParseAPIDocRoute(lines []string) *FromDoc {
	route := &FromDoc{}

	for _, line := range lines {
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "goframe:http_route") {
			continue
		}

		content := strings.TrimSpace(line[len("goframe:http_route"):])
		pairs := parseKeyValuePairs(content)

		// Handle path-method pairs
		if path, hasPath := pairs["path"]; hasPath {
			methods := []string{"GET"} // default method
			if method, hasMethod := pairs["method"]; hasMethod {
				methods = parseList(method)
			}

			routeName := ""
			if name, hasName := pairs["name"]; hasName {
				routeName = name
			}

			// Create a route definition for each method
			for _, method := range methods {
				routeDef := RouteDefinition{
					Path:   path,
					Method: method,
					Name:   routeName, // same name applies to all methods in this line
				}
				route.Routes = append(route.Routes, routeDef)
			}
		}

		// Handle other attributes
		if request, ok := pairs["request"]; ok {
			route.Requests = request
		}
		if response, ok := pairs["response"]; ok {
			if statusResponse := parseStatusResponse(response); statusResponse != nil {
				route.StatusResponses = append(route.StatusResponses, *statusResponse)
			} else {
				route.Responses = append(route.Responses, parseList(response)...)
			}
		}
		if header, ok := pairs["required_header"]; ok {
			route.RequiredHeaders = append(route.RequiredHeaders, header)
		}
	}

	return route
}

func parseKeyValuePairs(content string) map[string]string {
	pairs := make(map[string]string)
	i := 0
	for i < len(content) {
		// Skip whitespace
		for i < len(content) && unicode.IsSpace(rune(content[i])) {
			i++
		}
		if i >= len(content) {
			break
		}

		// Find key
		keyStart := i
		for i < len(content) && content[i] != '=' {
			i++
		}
		if i >= len(content) {
			break
		}
		key := strings.TrimSpace(content[keyStart:i])
		i++ // skip '='

		// Find value
		valueStart := i
		if i < len(content) && content[i] == '[' {
			// Handle bracketed values
			i++
			for i < len(content) && content[i] != ']' {
				i++
			}
			if i < len(content) {
				i++ // skip ']'
			}
		} else {
			// Handle regular values (until space or end)
			for i < len(content) && !unicode.IsSpace(rune(content[i])) {
				i++
			}
		}
		value := strings.TrimSpace(content[valueStart:i])
		pairs[key] = value
	}
	return pairs
}

func parseList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	// Handle bracketed lists: [item1, item2, item3]
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = value[1 : len(value)-1]
		if value == "" {
			return nil
		}
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
		return result
	}

	// Single value
	return []string{value}
}

func parseStatusResponse(value string) *FromDocStatusToResponse {
	value = strings.TrimSpace(value)
	colonIndex := strings.Index(value, ":")
	if colonIndex == -1 {
		return nil
	}

	statusPart := strings.TrimSpace(value[:colonIndex])
	responsePart := strings.TrimSpace(value[colonIndex+1:])

	statusRegex := convertStatusToRegex(statusPart)
	if statusRegex == nil {
		return nil
	}

	return &FromDocStatusToResponse{
		StatusPattern: statusRegex,
		Response:      responsePart,
	}
}

func convertStatusToRegex(status string) *regexp.Regexp {
	var pattern string

	switch {
	case strings.Contains(status, "-"):
		// Handle ranges like "200-299"
		parts := strings.Split(status, "-")
		if len(parts) == 2 {
			start := strings.TrimSpace(parts[0])
			end := strings.TrimSpace(parts[1])
			if len(start) == 3 && len(end) == 3 {
				startPrefix := start[:2]
				endPrefix := end[:2]
				if startPrefix == endPrefix {
					pattern = startPrefix + "[" + start[2:] + "-" + end[2:] + "]"
				}
			}
		}
	case strings.Contains(status, "x") || strings.Contains(status, "X"):
		// Handle wildcards like "2xx", "4XX"
		pattern = strings.ReplaceAll(strings.ReplaceAll(status, "x", "\\d"), "X", "\\d")
	default:
		// Exact match like "200", "404"
		pattern = "^" + regexp.QuoteMeta(status) + "$"
	}

	if pattern == "" {
		return nil
	}

	if !strings.HasPrefix(pattern, "^") {
		pattern = "^" + pattern
	}
	if !strings.HasSuffix(pattern, "$") {
		pattern = pattern + "$"
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	return regex
}
