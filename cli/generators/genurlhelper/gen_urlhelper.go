package genurlhelper

import (
	"bytes"
	"embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/alexisvisco/goframe/cli/generators"
)

// URLHelperGenerator generates URL helper functions based on router and handler definitions.
type URLHelperGenerator struct{ Gen *generators.Generator }

//go:embed templates
var fsTemplates embed.FS

// Route represents a router entry.
type Route struct {
	Method        string
	Path          string
	HandlerStruct string
	HandlerMethod string
}

type Param struct {
	FieldName string
	TagName   string
	Type      string
	Comment   string
	Source    string
}

type helperData struct {
	Name        string
	Path        string
	Params      []Param
	PathParams  []Param
	QueryParams []Param
}

// Generate generates the url helper file.
func (u *URLHelperGenerator) Generate() error {
	routes, err := parseRoutes("internal/v1handler/router.go")
	if err != nil {
		return fmt.Errorf("parse router: %w", err)
	}
	handlers, err := parseHandlers("internal/v1handler")
	if err != nil {
		return fmt.Errorf("parse handlers: %w", err)
	}
	var helpers []helperData
	for _, r := range routes {
		key := r.HandlerStruct + "#" + r.HandlerMethod
		h, ok := handlers[key]
		if !ok {
			continue
		}
		path := r.Path
		if h.PathOverride != "" {
			path = h.PathOverride
		}
		helpers = append(helpers, helperData{
			Name:        h.HelperName,
			Path:        path,
			Params:      append(append([]Param(nil), h.PathParams...), h.QueryParams...),
			PathParams:  h.PathParams,
			QueryParams: h.QueryParams,
		})
	}
	tmplBytes, err := fs.ReadFile(fsTemplates, "templates/url_helpers.go.tmpl")
	if err != nil {
		return fmt.Errorf("read template: %w", err)
	}
	tmpl, err := template.New("urlhelper").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, map[string]any{"Helpers": helpers}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	if err := os.MkdirAll("internal/urlhelper", 0o755); err != nil {
		return err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("format source: %w", err)
	}
	return os.WriteFile("internal/urlhelper/url_helpers.go", formatted, 0o644)
}

type handlerInfo struct {
	HelperName   string
	PathOverride string
	PathParams   []Param
	QueryParams  []Param
}

func parseHandlers(dir string) (map[string]handlerInfo, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	// first pass: collect all struct definitions across the directory
	allStructs := make(map[string]*ast.StructType)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, f.Name())
		fs := token.NewFileSet()
		node, err := parser.ParseFile(fs, path, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, d := range node.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if st, ok := ts.Type.(*ast.StructType); ok {
					allStructs[ts.Name.Name] = st
				}
			}
		}
	}

	// second pass: parse handler methods and associate request structs
	handlers := make(map[string]handlerInfo)
	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "handler_") || !strings.HasSuffix(f.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, f.Name())
		fileSet := token.NewFileSet()
		node, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		for _, d := range node.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil {
				continue
			}
			recvType := receiverName(fd)
			if recvType == "" {
				continue
			}
			tags := parseTags(fd.Doc)
			reqName := fd.Name.Name + "Request"
			if v, ok := tags["req"]; ok {
				reqName = v
			}
			helperName := fd.Name.Name
			if v, ok := tags["as"]; ok {
				helperName = v
			}
			pathOverride := tags["path"]
			st := allStructs[reqName]
			var pathParams, queryParams []Param
			if st != nil {
				pathParams, queryParams = extractParams(st)
			}
			key := recvType + "#" + fd.Name.Name
			handlers[key] = handlerInfo{
				HelperName:   helperName,
				PathOverride: pathOverride,
				PathParams:   pathParams,
				QueryParams:  queryParams,
			}
		}
	}

	return handlers, nil
}

func receiverName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	switch t := fd.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func parseTags(cg *ast.CommentGroup) map[string]string {
	out := map[string]string{}
	if cg == nil {
		return out
	}
	for _, c := range cg.List {
		text := strings.TrimPrefix(strings.TrimSpace(c.Text), "//")
		if strings.HasPrefix(text, "goframe:") {
			parts := strings.SplitN(strings.TrimPrefix(text, "goframe:"), "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.Trim(parts[1], "\"")
				out[key] = val
			}
		}
	}
	return out
}

func extractParams(st *ast.StructType) (pathParams, queryParams []Param) {
	for _, f := range st.Fields.List {
		if f.Tag == nil || len(f.Names) == 0 {
			continue
		}
		tag := reflect.StructTag(strings.Trim(f.Tag.Value, "`"))
		name := f.Names[0].Name
		typ := exprString(f.Type)
		if v, ok := tag.Lookup("path"); ok {
			pathParams = append(pathParams, Param{FieldName: name, TagName: v, Type: typ, Comment: "path parameter: " + v, Source: "path"})
		}
		if v, ok := tag.Lookup("query"); ok {
			queryParams = append(queryParams, Param{FieldName: name, TagName: v, Type: typ, Comment: "query parameter: " + v, Source: "query"})
		}
	}
	return
}

func exprString(e ast.Expr) string {
	var buf bytes.Buffer
	_ = format.Node(&buf, token.NewFileSet(), e)
	return buf.String()
}

func parseRoutes(path string) ([]Route, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var routes []Route
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "HandleFunc" {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		if lit, ok := call.Args[0].(*ast.BasicLit); ok {
			pattern := strings.Trim(lit.Value, "\"")
			route := parseRoutePattern(pattern)
			if handlerCall, ok := call.Args[1].(*ast.CallExpr); ok {
				if sel2, ok := handlerCall.Fun.(*ast.SelectorExpr); ok {
					if x, ok := sel2.X.(*ast.SelectorExpr); ok {
						route.HandlerStruct = x.Sel.Name
						route.HandlerMethod = sel2.Sel.Name
					}
				}
			}
			routes = append(routes, route)
		}
		return true
	})
	return routes, nil
}

func parseRoutePattern(pattern string) Route {
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) == 2 {
		return Route{Method: parts[0], Path: parts[1]}
	}
	return Route{Path: pattern}
}
