package apidoc

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"regexp"
	"strings"

	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"golang.org/x/tools/go/packages"
)

type Route struct {
	Paths            map[string][]string // Maps path to methods
	Requests         *introspect.ObjectType
	StatusToResponse []StatusToResponse
	RequiredHeaders  []string
}

type StatusToResponse struct {
	StatusPattern *regexp.Regexp
	Response      *introspect.ObjectType // nil if IsError or IsRedirect is specified
	IsError       bool
	IsRedirect    bool
}

// ParseRoute parses a route by finding the method's godoc comments and extracting API documentation.
// It resolves request and response types, handling package prefixes using imports.
//
// Default behavior:
// - If no request type is specified in comments, looks for {methodName}Request struct
// - If no response type is specified in comments, looks for {methodName}Response struct
// - Default types are optional and won't cause errors if they don't exist
func ParseRoute(rootPath, relPkgPath, structName, method string) (*Route, error) {
	ctx := &introspect.ParseContext{
		Visited:  make(map[string]*introspect.ObjectType),
		Enums:    make(map[string]*introspect.FieldTypeEnum),
		Packages: make(map[string]*packages.Package),
		RootPath: rootPath,
	}

	// Load the package
	pkg, err := ctx.LoadPackage(relPkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load package %s: %w", relPkgPath, err)
	}

	// Parse enums for this package
	ctx.ParseEnums(pkg)

	// Find the method in AST and extract comments
	methodComments, err := findMethodComments(pkg, structName, method)
	if err != nil {
		return nil, fmt.Errorf("failed to find method %s in struct %s: %w", method, structName, err)
	}

	fromDoc := ParseAPIDocRoute(methodComments)

	jsonFromDoc, err := json.MarshalIndent(fromDoc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API doc: %w", err)
	}
	fmt.Println(string(jsonFromDoc))

	// Build import map for resolving types
	imports := buildImportMap(pkg)

	// Parse requests
	var requests *introspect.ObjectType
	if fromDoc.Requests != "" {
		requests, err = parseTypeReference(ctx, fromDoc.Requests, imports, relPkgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse request type %s: %w", fromDoc.Requests, err)
		}
	} else {
		// Try to find default request type: {methodName}Request
		defaultRequestType := method + "Request"
		requests, _ = parseTypeReference(ctx, defaultRequestType, imports, relPkgPath)
		// Ignore error if default type doesn't exist
	}

	// Parse status responses
	var statusResponses []StatusToResponse
	for _, statusResp := range fromDoc.StatusResponses {
		var responseObj *introspect.ObjectType
		var isError, isRedirect bool

		// Check for special response types
		switch statusResp.Response {
		case "TYPE_ERROR":
			isError = true
		case "TYPE_REDIRECT":
			isRedirect = true
		default:
			responseObj, err = parseTypeReference(ctx, statusResp.Response, imports, relPkgPath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse response type %s: %w", statusResp.Response, err)
			}
		}

		statusResponses = append(statusResponses, StatusToResponse{
			StatusPattern: statusResp.StatusPattern,
			Response:      responseObj,
			IsError:       isError,
			IsRedirect:    isRedirect,
		})
	}

	// Parse regular responses (if any)
	for _, respType := range fromDoc.Responses {
		var responseObj *introspect.ObjectType
		var isError, isRedirect bool

		// Check for special response types
		switch respType {
		case "Error":
			isError = true
		case "Redirect":
			isRedirect = true
		default:
			responseObj, err = parseTypeReference(ctx, respType, imports, relPkgPath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse response type %s: %w", respType, err)
			}
		}

		var pattern *regexp.Regexp
		if isError {
			pattern = regexp.MustCompile("^[45][0-9]{2}$")
		} else if isRedirect {
			pattern = regexp.MustCompile("^3[0-9]{2}$")
		} else {
			pattern = regexp.MustCompile("^2[0-9]{2}$")
		}
		statusResponses = append(statusResponses, StatusToResponse{
			StatusPattern: pattern,
			Response:      responseObj,
			IsError:       isError,
			IsRedirect:    isRedirect,
		})
	}

	// If no responses were specified, try to find default response type: {methodName}Response
	if len(fromDoc.Responses) == 0 && len(fromDoc.StatusResponses) == 0 {
		defaultResponseType := method + "Response"
		if defaultResponse, err := parseTypeReference(ctx, defaultResponseType, imports, relPkgPath); err == nil {
			// Add as 200 response by default
			pattern := regexp.MustCompile("^200$")
			statusResponses = append(statusResponses, StatusToResponse{
				StatusPattern: pattern,
				Response:      defaultResponse,
				IsError:       false,
				IsRedirect:    false,
			})
		}
		// Ignore error if default type doesn't exist
	}

	// Build paths map
	pathsMap := make(map[string][]string)
	for _, kv := range fromDoc.PathMethods {
		pathsMap[kv.Path] = kv.Methods
	}

	return &Route{
		Paths:            pathsMap,
		Requests:         requests,
		StatusToResponse: statusResponses,
		RequiredHeaders:  fromDoc.RequiredHeaders,
	}, nil
}

// findMethodComments extracts the godoc comments for a specific method in a struct
func findMethodComments(pkg *packages.Package, structName, methodName string) ([]string, error) {
	var comments []string
	found := false

	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if found {
				return false
			}

			if fd, ok := n.(*ast.FuncDecl); ok {
				// Check if this is a method
				if fd.Recv != nil && len(fd.Recv.List) > 0 {
					// Get receiver type
					var recvTypeName string
					switch typ := fd.Recv.List[0].Type.(type) {
					case *ast.StarExpr:
						if ident, ok := typ.X.(*ast.Ident); ok {
							recvTypeName = ident.Name
						}
					case *ast.Ident:
						recvTypeName = typ.Name
					}

					// Check if this is the method we're looking for
					if recvTypeName == structName && fd.Name.Name == methodName {
						// Extract comments
						if fd.Doc != nil {
							for _, comment := range fd.Doc.List {
								comments = append(comments, comment.Text)
							}
						}
						found = true
						return false
					}
				}
			}
			return true
		})

		if found {
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("method %s not found in struct %s", methodName, structName)
	}

	return comments, nil
}

// buildImportMap creates a map from package alias to full package path
func buildImportMap(pkg *packages.Package) map[string]string {
	imports := make(map[string]string)

	for _, file := range pkg.Syntax {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)

			var alias string
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				// Default alias is the last part of the path
				parts := strings.Split(path, "/")
				alias = parts[len(parts)-1]
			}

			imports[alias] = path
		}
	}

	return imports
}

// parseTypeReference resolves a type reference (which may have a package prefix) and parses it
func parseTypeReference(ctx *introspect.ParseContext, typeRef string, imports map[string]string, currentPkg string) (*introspect.ObjectType, error) {
	var pkgPath, typeName string

	if strings.Contains(typeRef, ".") {
		// Type has package prefix
		parts := strings.SplitN(typeRef, ".", 2)
		pkgAlias := parts[0]
		typeName = parts[1]

		if fullPath, exists := imports[pkgAlias]; exists {
			pkgPath = fullPath
		} else {
			return nil, fmt.Errorf("unknown package alias: %s", pkgAlias)
		}
	} else {
		// Type is in current package
		pkgPath = currentPkg
		typeName = typeRef
	}

	return ctx.ParseStructByName(pkgPath, typeName)
}
