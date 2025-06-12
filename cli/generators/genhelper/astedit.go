package genhelper

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// GoFile helps to modify go source files using AST.
type GoFile struct {
	File *ast.File
	fset *token.FileSet
	path string
}

// LoadGoFile parses a Go file.
func LoadGoFile(path string) (*GoFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return &GoFile{File: file, fset: fset, path: path}, nil
}

// Save writes the modified file back to disk.
func (g *GoFile) Save(path ...string) error {
	var p string
	if len(path) > 0 {
		p = path[0]
	} else {
		p = g.path
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, g.fset, g.File); err != nil {
		return err
	}
	return os.WriteFile(p, buf.Bytes(), 0644)
}

// HasImport checks if the file already imports the given path.
func (g *GoFile) HasImport(path string) bool {
	for _, imp := range g.File.Imports {
		if strings.Trim(imp.Path.Value, "\"") == path {
			return true
		}
	}
	return false
}

// AddNamedImport adds an import with an optional alias.
func (g *GoFile) AddNamedImport(name, path string) {
	if g.HasImport(path) {
		return
	}
	astutil.AddNamedImport(g.fset, g.File, name, path)
}

// exprToString returns the formatted string representation of an expression.
func (g *GoFile) exprToString(e ast.Expr) string {
	var buf bytes.Buffer
	format.Node(&buf, g.fset, e)
	return buf.String()
}

// AddArgToFuncCall appends an argument to a specific function call inside the given function.
// call should be of the form pkg.Func.
func (g *GoFile) AddArgToFuncCall(funcName, call, arg string) error {
	parts := strings.Split(call, ".")
	if len(parts) != 2 {
		return fmt.Errorf("call should be pkg.Func")
	}
	pkg, fun := parts[0], parts[1]

	expr, err := parser.ParseExpr(arg)
	if err != nil {
		return err
	}

	ast.Inspect(g.File, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName {
			return true
		}
		ast.Inspect(fd.Body, func(nn ast.Node) bool {
			ce, ok := nn.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := ce.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if ident.Name == pkg && sel.Sel.Name == fun {
				// check if arg already exists
				for _, a := range ce.Args {
					if g.exprToString(a) == g.exprToString(expr) {
						return false
					}
				}
				ce.Args = append(ce.Args, expr)
				return false
			}
			return true
		})
		return false
	})
	return nil
}

// AddReturnCompositeElement adds an element to the composite literal returned in the given function.
func (g *GoFile) AddReturnCompositeElement(funcName, element string) error {
	expr, err := parser.ParseExpr(element)
	if err != nil {
		return err
	}

	ast.Inspect(g.File, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != funcName {
			return true
		}
		ast.Inspect(fd.Body, func(nn ast.Node) bool {
			ret, ok := nn.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, res := range ret.Results {
				cl, ok := res.(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, el := range cl.Elts {
					if g.exprToString(el) == g.exprToString(expr) {
						return false
					}
				}
				cl.Elts = append([]ast.Expr{expr}, cl.Elts...)
				return false
			}
			return true
		})
		return false
	})
	return nil
}

func (g *GoFile) HasFunc(name string) bool {
	for _, decl := range g.File.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Recv == nil && fn.Name.Name == name {
				return true
			}
		}
	}
	return false
}

func (g *GoFile) HasMethod(structName, methodName string) bool {
	for _, decl := range g.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != methodName || fn.Recv == nil {
			continue
		}
		if len(fn.Recv.List) == 0 {
			continue
		}
		t := fn.Recv.List[0].Type
		switch rt := t.(type) {
		case *ast.StarExpr:
			if ident, ok := rt.X.(*ast.Ident); ok && ident.Name == structName {
				return true
			}
		case *ast.Ident:
			if rt.Name == structName {
				return true
			}
		}
	}
	return false
}

func addLine(path, pattern, line string, before, regex bool) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")

	var r *regexp.Regexp
	if regex {
		r, err = regexp.Compile(pattern)
		if err != nil {
			return err
		}
	}

	for i, l := range lines {
		var match bool
		if regex {
			match = r.MatchString(l)
		} else {
			match = strings.Contains(l, pattern)
		}
		if match {
			if before {
				lines = append(lines[:i], append([]string{line}, lines[i:]...)...)
			} else {
				lines = append(lines[:i+1], append([]string{line}, lines[i+1:]...)...)
			}
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		}
	}
	return fmt.Errorf("pattern not found")
}

func (g *GoFile) AddLineBeforeString(pattern, line string) error {
	return addLine(g.path, pattern, line, true, false)
}

func (g *GoFile) AddLineAfterString(pattern, line string) error {
	return addLine(g.path, pattern, line, false, false)
}

func (g *GoFile) AddLineBeforeRegex(pattern, line string) error {
	return addLine(g.path, pattern, line, true, true)
}

func (g *GoFile) AddLineAfterRegex(pattern, line string) error {
	return addLine(g.path, pattern, line, false, true)
}
