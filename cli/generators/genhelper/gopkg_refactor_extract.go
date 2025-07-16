package genhelper

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ExtractMethod extracts a method from a type and moves or appends it to a new file in the same package.
func (g *GoPkg) ExtractMethod(typeName, methodName, fromFile, newFileName string) error {
	pkgFile, ok := g.Files[fromFile]
	if !ok {
		return fmt.Errorf("file %s not found in package", fromFile)
	}

	var methodDecl *ast.FuncDecl
	var otherDecls []ast.Decl

	for _, decl := range pkgFile.File.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != methodName {
			otherDecls = append(otherDecls, decl)
			continue
		}

		if len(fn.Recv.List) > 0 {
			switch recv := fn.Recv.List[0].Type.(type) {
			case *ast.StarExpr:
				if ident, ok := recv.X.(*ast.Ident); ok && ident.Name == typeName {
					methodDecl = fn
					continue
				}
			case *ast.Ident:
				if recv.Name == typeName {
					methodDecl = fn
					continue
				}
			}
		}
		otherDecls = append(otherDecls, decl)
	}

	if methodDecl == nil {
		return fmt.Errorf("method %s.%s not found", typeName, methodName)
	}

	pkgFile.File.Decls = otherDecls

	// Extract comments related to the method so they can be moved to the new file
	var methodComments []*ast.CommentGroup
	var remainingComments []*ast.CommentGroup
	start, end := methodDecl.Pos(), methodDecl.End()

	if methodDecl.Doc != nil && methodDecl.Doc.Pos() < start {
		start = methodDecl.Doc.Pos()
	}

	for _, cg := range pkgFile.File.Comments {
		if cg.Pos() >= start && cg.End() <= end {
			methodComments = append(methodComments, cg)
		} else {
			remainingComments = append(remainingComments, cg)
		}
	}
	pkgFile.File.Comments = remainingComments

	// Clone the method so we can remove its documentation from the original file
	extractedMethod := *methodDecl
	methodDecl.Doc = nil

	usedImports := collectUsedImportsFromFunc(pkgFile.File, methodDecl)
	newPath := filepath.Join(g.rootPath, filepath.Dir(fromFile), newFileName)

	mergedFile, err := mergeIntoExistingFile(g.fset, newPath, &extractedMethod, usedImports, pkgFile.PackageName, methodComments)
	if err != nil {
		return err
	}

	if err := writeAstToFile(g.fset, mergedFile, newPath); err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}

	oldPath := filepath.Join(g.rootPath, fromFile)
	if err := writeAstToFile(g.fset, pkgFile.File, oldPath); err != nil {
		return fmt.Errorf("failed to update original file: %w", err)
	}

	return nil
}

// ExtractStruct extracts a struct from a type file and moves or appends it to a new file in the same package.
func (g *GoPkg) ExtractStruct(structName, fromFile, newFileName string) error {
	pkgFile, ok := g.Files[fromFile]
	if !ok {
		return fmt.Errorf("file %s not found in package", fromFile)
	}

	var structDecl ast.Decl
	var otherDecls []ast.Decl

	for _, decl := range pkgFile.File.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			otherDecls = append(otherDecls, decl)
			continue
		}

		newSpecs := []ast.Spec{}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != structName {
				newSpecs = append(newSpecs, spec)
				continue
			}

			if _, ok := typeSpec.Type.(*ast.StructType); ok {
				structDecl = &ast.GenDecl{
					Tok:    genDecl.Tok,
					Specs:  []ast.Spec{typeSpec},
					Doc:    genDecl.Doc,
					Lparen: genDecl.Lparen,
					Rparen: genDecl.Rparen,
				}
			} else {
				newSpecs = append(newSpecs, spec)
			}
		}

		if len(newSpecs) > 0 {
			genDecl.Specs = newSpecs
			otherDecls = append(otherDecls, genDecl)
		}
	}

	if structDecl == nil {
		return fmt.Errorf("struct %s not found", structName)
	}

	pkgFile.File.Decls = otherDecls

	usedImports := collectUsedImportsFromStruct(pkgFile.File, structDecl)
	newPath := filepath.Join(g.rootPath, filepath.Dir(fromFile), newFileName)

	mergedFile, err := mergeIntoExistingFile(g.fset, newPath, structDecl, usedImports, pkgFile.PackageName, nil)
	if err != nil {
		return err
	}

	if err := writeAstToFile(g.fset, mergedFile, newPath); err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}

	oldPath := filepath.Join(g.rootPath, fromFile)
	if err := writeAstToFile(g.fset, pkgFile.File, oldPath); err != nil {
		return fmt.Errorf("failed to update original file: %w", err)
	}

	forStruct := g.FindAllMethodsForStruct(structName, fromFile)
	for _, method := range forStruct {
		if err := g.ExtractMethod(structName, method, fromFile, newFileName); err != nil {
			return fmt.Errorf("failed to extract method %s: %w", method, err)
		}
	}

	return nil
}

func collectUsedImportsFromFunc(file *ast.File, fn *ast.FuncDecl) []*ast.ImportSpec {
	usedIdents := map[string]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				usedIdents[ident.Name] = true
			}
		case *ast.Ident:
			usedIdents[x.Name] = true
		}
		return true
	})

	var usedImports []*ast.ImportSpec
	for _, spec := range file.Imports {
		var alias string
		if spec.Name != nil {
			alias = spec.Name.Name
		} else {
			path := strings.Trim(spec.Path.Value, `"`)
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}
		if usedIdents[alias] {
			usedImports = append(usedImports, spec)
		}
	}

	return usedImports
}

func collectUsedImportsFromStruct(file *ast.File, decl ast.Decl) []*ast.ImportSpec {
	used := map[string]bool{}

	ast.Inspect(decl, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				used[ident.Name] = true
			}
		case *ast.Ident:
			used[x.Name] = true
		}
		return true
	})

	var result []*ast.ImportSpec
	for _, spec := range file.Imports {
		var key string
		if spec.Name != nil {
			key = spec.Name.Name
		} else {
			path := strings.Trim(spec.Path.Value, `"`)
			parts := strings.Split(path, "/")
			key = parts[len(parts)-1]
		}
		if used[key] {
			result = append(result, spec)
		}
	}

	return result
}

func mergeIntoExistingFile(fset *token.FileSet, path string, newDecl ast.Decl, newImports []*ast.ImportSpec, pkgName string, comments []*ast.CommentGroup) (*ast.File, error) {
	if data, err := os.ReadFile(path); err == nil {
		file, err := parser.ParseFile(fset, path, data, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("failed to parse existing file: %w", err)
		}

		// Add the new declaration
		file.Decls = append(file.Decls, newDecl)
		file.Comments = append(file.Comments, comments...)

		// Merge imports properly
		mergedImports := upsertImports(file.Imports, newImports)
		updateFileImports(file, mergedImports)

		return file, nil
	}

	// Create new file
	newFile := &ast.File{
		Name:     ast.NewIdent(pkgName),
		Decls:    []ast.Decl{newDecl},
		Imports:  newImports,
		Comments: comments,
	}

	// Create import declaration if there are imports
	if len(newImports) > 0 {
		importDecl := &ast.GenDecl{
			Tok: token.IMPORT,
		}

		for _, imp := range newImports {
			importDecl.Specs = append(importDecl.Specs, imp)
		}

		// Insert import declaration at the beginning (after package declaration)
		newFile.Decls = append([]ast.Decl{importDecl}, newFile.Decls...)
	}

	return newFile, nil
}

func upsertImports(existing, newOnes []*ast.ImportSpec) []*ast.ImportSpec {
	existingMap := map[string]bool{}
	for _, imp := range existing {
		existingMap[imp.Path.Value] = true
	}

	result := make([]*ast.ImportSpec, len(existing))
	copy(result, existing)

	for _, imp := range newOnes {
		if !existingMap[imp.Path.Value] {
			result = append(result, imp)
		}
	}
	return result
}

func updateFileImports(file *ast.File, imports []*ast.ImportSpec) {
	// Update the file's import list
	file.Imports = imports

	// Find existing import declarations and update them
	var importDecls []*ast.GenDecl
	var otherDecls []ast.Decl

	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			importDecls = append(importDecls, genDecl)
		} else {
			otherDecls = append(otherDecls, decl)
		}
	}

	// If we have imports to add
	if len(imports) > 0 {
		if len(importDecls) > 0 {
			// Update the first import declaration with all imports
			importDecls[0].Specs = nil
			for _, imp := range imports {
				importDecls[0].Specs = append(importDecls[0].Specs, imp)
			}
			// Keep only the first import declaration
			file.Decls = append([]ast.Decl{importDecls[0]}, otherDecls...)
		} else {
			// Create new import declaration
			importDecl := &ast.GenDecl{
				Tok: token.IMPORT,
			}
			for _, imp := range imports {
				importDecl.Specs = append(importDecl.Specs, imp)
			}
			// Insert at the beginning
			file.Decls = append([]ast.Decl{importDecl}, otherDecls...)
		}
	} else {
		// No imports needed, remove import declarations
		file.Decls = otherDecls
	}
}

func writeAstToFile(fset *token.FileSet, file *ast.File, path string) error {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("failed to format node: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	return os.WriteFile(path, buf.Bytes(), 0644)
}
