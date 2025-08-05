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

// GoFile helps to modify go source files using AST for imports and lines for everything else.
type GoFile struct {
	File  *ast.File
	fset  *token.FileSet
	path  string
	lines []string // Keep track of actual file lines
}

// LoadGoFile parses a Go file using AST only - no type checking bullshit
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

	// Split content into lines for line-based operations
	lines := strings.Split(string(src), "\n")

	return &GoFile{
		File:  file,
		fset:  fset,
		path:  path,
		lines: lines,
	}, nil
}

// syncLinesFromAST updates the lines array when AST (imports) are modified
func (g *GoFile) syncLinesFromAST() {
	var buf bytes.Buffer
	_ = format.Node(&buf, g.fset, g.File)
	g.lines = strings.Split(buf.String(), "\n")
}

// Save writes the modified file back to disk.
func (g *GoFile) Save(path ...string) error {
	var p string
	if len(path) > 0 {
		p = path[0]
	} else {
		p = g.path
	}

	content := strings.Join(g.lines, "\n")
	return os.WriteFile(p, []byte(content), 0644)
}

// HasImport checks if the file already imports the given path.
func (g *GoFile) HasImport(path string) bool {
	return astutil.UsesImport(g.File, path)
}

// HasStruct checks if the file contains a struct with the given name.
func (g *GoFile) HasStruct(structName string) bool {
	found := false
	ast.Inspect(g.File, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if typeSpec.Name.Name == structName {
			_, isStruct := typeSpec.Type.(*ast.StructType)
			if isStruct {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// HasInterface checks if the file contains an interface with the given name.
func (g *GoFile) HasInterface(interfaceName string) bool {
	found := false
	ast.Inspect(g.File, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if typeSpec.Name.Name == interfaceName {
			_, isInterface := typeSpec.Type.(*ast.InterfaceType)
			if isInterface {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// AddNamedImport adds an import with an optional alias and syncs lines.
func (g *GoFile) AddNamedImport(name, path string) {
	if g.HasImport(path) {
		return
	}
	astutil.AddNamedImport(g.fset, g.File, name, path)
	g.syncLinesFromAST()
}

func (g *GoFile) HasFunc(name string) bool {
	for _, line := range g.lines {
		// Simple check in lines - could be made more robust
		if strings.Contains(line, "func "+name+"(") && !strings.Contains(line, "//") {
			return true
		}
	}
	return false
}

func (g *GoFile) HasMethod(structName, methodName string) bool {
	for _, line := range g.lines {
		// Simple check for method receiver - could be made more robust
		if strings.Contains(line, "func (") && strings.Contains(line, structName) && strings.Contains(line, methodName+"(") {
			return true
		}
	}
	return false
}

// findMultiLinePattern finds the position where a multi-line pattern matches
func (g *GoFile) findMultiLinePattern(pattern string, regex bool) int {
	patternLines := strings.Split(pattern, "\n")

	var patternRegexes []*regexp.Regexp
	if regex {
		patternRegexes = make([]*regexp.Regexp, len(patternLines))
		for i, patternLine := range patternLines {
			r, err := regexp.Compile(patternLine)
			if err != nil {
				panic("finding multi-line pattern: invalid regex pattern: " + err.Error())
			}
			patternRegexes[i] = r
		}
	}

	// Search for the pattern starting at each line
	for i := 0; i <= len(g.lines)-len(patternLines); i++ {
		match := true
		for j, patternLine := range patternLines {
			var lineMatch bool
			if regex {
				lineMatch = patternRegexes[j].MatchString(g.lines[i+j])
			} else {
				lineMatch = strings.Contains(g.lines[i+j], patternLine)
			}
			if !lineMatch {
				match = false
				break
			}
		}
		if match {
			// Return the position after the last matching line
			return i + len(patternLines) - 1
		}
	}
	return -1 // Pattern not found
}

func (g *GoFile) addLine(pattern, line string, before, regex bool) {
	// Check if pattern contains multiple lines
	patternLines := strings.Split(pattern, "\n")
	linesToAdd := strings.Split(line, "\n")

	if g.hasConsecutiveLines(linesToAdd) {
		return
	}

	if len(patternLines) > 1 {
		// Multi-line pattern matching
		matchPos := g.findMultiLinePattern(pattern, regex)
		if matchPos != -1 {
			if before {
				// Insert before the first line of the pattern
				insertPos := matchPos - len(patternLines) + 1
				g.lines = append(g.lines[:insertPos], append(linesToAdd, g.lines[insertPos:]...)...)
			} else {
				// Insert after the last line of the pattern
				insertPos := matchPos + 1
				g.lines = append(g.lines[:insertPos], append(linesToAdd, g.lines[insertPos:]...)...)
			}
		}
		return
	}

	// Single line pattern matching (original logic)
	var r *regexp.Regexp
	var err error
	if regex {
		r, err = regexp.Compile(pattern)
		if err != nil {
			panic("adding line in gofile: invalid regex pattern: " + err.Error())
		}
	}

	for i, l := range g.lines {
		var match bool
		if regex {
			match = r.MatchString(l)
		} else {
			match = strings.Contains(l, pattern)
		}
		if match {
			if before {
				g.lines = append(g.lines[:i], append(linesToAdd, g.lines[i:]...)...)
			} else {
				g.lines = append(g.lines[:i+1], append(linesToAdd, g.lines[i+1:]...)...)
			}
			return
		}
	}
}

// hasConsecutiveLines checks if the given lines already exist consecutively in the file
func (g *GoFile) hasConsecutiveLines(linesToCheck []string) bool {
	if len(linesToCheck) == 0 {
		return true
	}

	// Trim whitespace for comparison
	trimmedLinesToCheck := make([]string, len(linesToCheck))
	for i, line := range linesToCheck {
		trimmedLinesToCheck[i] = strings.TrimSpace(line)
	}

	// Look for consecutive matching lines
	for i := 0; i <= len(g.lines)-len(trimmedLinesToCheck); i++ {
		match := true
		for j, lineToCheck := range trimmedLinesToCheck {
			if strings.TrimSpace(g.lines[i+j]) != lineToCheck {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// FieldInfo contains field name and its type info
type FieldInfo struct {
	FieldName    string
	TypeName     string // Le nom du type (ex: "string", "User", "*User")
	PackagePath  string // Le path du package (ex: "time", "github.com/user/pkg") - vide si type local
	IsPointer    bool
	IsSlice      bool
	IsMap        bool
	MapKeyType   string // Si c'est une map, le type de la clé
	MapValueType string // Si c'est une map, le type de la valeur
}

// GetFieldsFromStruct parse les champs d'une struct depuis l'AST
func (g *GoFile) GetFieldsFromStruct(structName string) []FieldInfo {
	var fields []FieldInfo

	// Créer une map des imports pour résoudre les packages
	imports := g.getImportsMap()

	ast.Inspect(g.File, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if typeSpec.Name.Name != structName {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// Parser chaque champ
		for _, field := range structType.Fields.List {
			fieldInfos := g.parseField(field, imports)
			fields = append(fields, fieldInfos...)
		}

		return false // Trouvé la struct, stop
	})

	return fields
}

// getImportsMap crée une map alias -> package path
func (g *GoFile) getImportsMap() map[string]string {
	imports := make(map[string]string)

	for _, imp := range g.File.Imports {
		path := strings.Trim(imp.Path.Value, `"`)

		var alias string
		if imp.Name != nil {
			// Import avec alias explicite
			alias = imp.Name.Name
		} else {
			// Pas d'alias, utiliser le dernier segment du path
			parts := strings.Split(path, "/")
			alias = parts[len(parts)-1]
		}

		imports[alias] = path
	}

	return imports
}

// parseField parse un champ et retourne les infos
func (g *GoFile) parseField(field *ast.Field, imports map[string]string) []FieldInfo {
	var fieldInfos []FieldInfo

	// Récupérer les noms des champs
	var fieldNames []string
	if len(field.Names) > 0 {
		// Champs nommés
		for _, name := range field.Names {
			fieldNames = append(fieldNames, name.Name)
		}
	} else {
		// Champ embedded (anonymous)
		typeName := g.getTypeString(field.Type)
		fieldNames = append(fieldNames, typeName)
	}

	// Parser le type
	for _, fieldName := range fieldNames {
		fieldInfo := FieldInfo{
			FieldName: fieldName,
		}

		g.parseTypeInfo(field.Type, &fieldInfo, imports)
		fieldInfos = append(fieldInfos, fieldInfo)
	}

	return fieldInfos
}

// parseTypeInfo parse les infos de type récursivement
func (g *GoFile) parseTypeInfo(typeExpr ast.Expr, info *FieldInfo, imports map[string]string) {
	switch t := typeExpr.(type) {
	case *ast.Ident:
		// Type simple (int, string, User, etc.)
		info.TypeName = t.Name

	case *ast.StarExpr:
		// Pointeur
		info.IsPointer = true
		g.parseTypeInfo(t.X, info, imports)

	case *ast.ArrayType:
		if t.Len == nil {
			// Slice
			info.IsSlice = true
		}
		g.parseTypeInfo(t.Elt, info, imports)

	case *ast.MapType:
		// Map
		info.IsMap = true
		info.MapKeyType = g.getTypeString(t.Key)
		info.MapValueType = g.getTypeString(t.Value)
		info.TypeName = fmt.Sprintf("map[%s]%s", info.MapKeyType, info.MapValueType)

	case *ast.SelectorExpr:
		// Type avec package (ex: time.Time)
		if ident, ok := t.X.(*ast.Ident); ok {
			info.TypeName = t.Sel.Name

			// Résoudre le vrai path du package si possible
			if fullPath, exists := imports[ident.Name]; exists {
				info.PackagePath = fullPath
			} else {
				// Fallback sur le nom de l'alias si pas trouvé dans les imports
				info.PackagePath = ident.Name
			}
		}

	case *ast.InterfaceType:
		// Interface
		info.TypeName = "interface{}"

	case *ast.StructType:
		// Struct anonyme
		info.TypeName = "struct{}"

	case *ast.FuncType:
		// Function type
		info.TypeName = g.getTypeString(typeExpr)

	case *ast.ChanType:
		// Channel
		info.TypeName = g.getTypeString(typeExpr)

	default:
		// Fallback
		info.TypeName = g.getTypeString(typeExpr)
	}
}

// getTypeString convertit un ast.Expr en string
func (g *GoFile) getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name

	case *ast.SelectorExpr:
		return g.getTypeString(t.X) + "." + t.Sel.Name

	case *ast.StarExpr:
		return "*" + g.getTypeString(t.X)

	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + g.getTypeString(t.Elt)
		}
		return "[" + g.getTypeString(t.Len) + "]" + g.getTypeString(t.Elt)

	case *ast.MapType:
		return "map[" + g.getTypeString(t.Key) + "]" + g.getTypeString(t.Value)

	case *ast.InterfaceType:
		return "interface{}"

	case *ast.StructType:
		return "struct{}"

	case *ast.FuncType:
		// Pour les fonctions, c'est plus complexe mais on fait simple
		return "func(...)"

	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + g.getTypeString(t.Value)
		case ast.RECV:
			return "<-chan " + g.getTypeString(t.Value)
		default:
			return "chan " + g.getTypeString(t.Value)
		}

	case *ast.Ellipsis:
		return "..." + g.getTypeString(t.Elt)

	default:
		return "unknown"
	}
}

func (g *GoFile) AddLineBeforeString(pattern, line string) {
	g.addLine(pattern, line, true, false)
}

func (g *GoFile) AddLineAfterString(pattern, line string) {
	g.addLine(pattern, line, false, false)
}

func (g *GoFile) AddLineBeforeRegex(pattern, line string) {
	g.addLine(pattern, line, true, true)
}

func (g *GoFile) AddLineAfterRegex(pattern, line string) {
	g.addLine(pattern, line, false, true)
}

// GetLines returns the current lines (useful for debugging)
func (g *GoFile) GetLines() []string {
	return g.lines
}

// GetLineCount returns the number of lines
func (g *GoFile) GetLineCount() int {
	return len(g.lines)
}

func (g *GoFile) AddContent(content string) {
	// Add content at the end of the file
	g.lines = append(g.lines, content)
}
