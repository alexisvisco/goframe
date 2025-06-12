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
	return astutil.UsesImport(g.File, path)
}

// AddNamedImport adds an import with an optional alias.
func (g *GoFile) AddNamedImport(name, path string) {
	if g.HasImport(path) {
		return
	}
	astutil.AddNamedImport(g.fset, g.File, name, path)
}

// exprToString returns the formatted string representation of an expression.

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
