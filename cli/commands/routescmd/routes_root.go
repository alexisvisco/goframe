package routescmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewCmdRoutes() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:     "routes",
		Aliases: []string{"r"},
		Short:   "Show routes",
		Long:    "Show the routes of the application, including their methods, paths, and handlers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			readFile, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			routes := parseRoutes(string(readFile))
			if len(routes) == 0 {
				fmt.Println("No routes found.")
				return nil
			}

			printRoutingTable(routes)

			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "internal/v1handler/router.go", "Path to the routes file")

	return cmd
}

func parseRoutes(sourceCode string) []Route {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", sourceCode, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	var routes []Route

	ast.Inspect(node, func(n ast.Node) bool {
		// Look for function calls
		if call, ok := n.(*ast.CallExpr); ok {
			// Check if it's a HandleFunc call
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "HandleFunc" {
				if len(call.Args) >= 2 {
					// Extract route pattern from first argument
					if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						routePattern := strings.Trim(lit.Value, `"`)
						route := parseRoutePattern(routePattern)

						// Extract handler from second argument
						if handlerCall, ok := call.Args[1].(*ast.CallExpr); ok {
							if sel, ok := handlerCall.Fun.(*ast.SelectorExpr); ok {
								if x, ok := sel.X.(*ast.SelectorExpr); ok {
									handlerName := x.Sel.Name + "#" + sel.Sel.Name
									route.Handler = handlerName
								}
							}
						}

						routes = append(routes, route)
					}
				}
			}
		}
		return true
	})

	return routes
}

func parseRoutePattern(pattern string) Route {
	// Pattern format: "METHOD /path"
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) == 2 {
		return Route{
			Method: parts[0],
			Path:   parts[1],
		}
	}
	return Route{Path: pattern}
}

type Route struct {
	Method  string
	Path    string
	Handler string
}

func printRoutingTable(routes []Route) {
	for _, route := range routes {
		slog.Info("", "method", route.Method, "path", route.Path, "handler", route.Handler)
	}
}
