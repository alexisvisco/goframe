package genhelper

import (
	"bytes"
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
