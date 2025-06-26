package genhelper

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GoPkg represents a parsed Go package directory.
type GoPkg struct {
	Path           string
	Files          map[string]*PackageFile // relative path -> PackageFile
	fset           *token.FileSet
	recursive      bool
	rootPath       string
	rootImportPath string // Import path of the root package
}

// PackageFile contains metadata and AST of a single Go file.
type PackageFile struct {
	File        *ast.File
	PackageName string
	ImportPath  string
	RelPath     string // relative path from root directory
}

// StructInfo represents a found struct type.
type StructInfo struct {
	Name       string
	FileName   string
	Package    string
	ImportPath string
	RelPath    string
	Self       bool
}

// LoadGoPkg parses Go source files in the specified directory.
func LoadGoPkg(pkgPath string, recursive ...bool) (*GoPkg, error) {
	isRecursive := len(recursive) > 0 && recursive[0]

	fset := token.NewFileSet()
	files := make(map[string]*PackageFile)

	absPath, err := filepath.Abs(pkgPath)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories unless recursive is enabled
		if info.IsDir() && !isRecursive && path != absPath {
			return filepath.SkipDir
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(absPath, filepath.Dir(path))
		if err != nil {
			return err
		}
		if relPath == "." {
			relPath = ""
		}

		importPath := buildImportPath(absPath, filepath.Dir(path))

		packageFile := &PackageFile{
			File:        file,
			PackageName: file.Name.Name,
			ImportPath:  importPath,
			RelPath:     relPath,
		}

		relFilePath := filepath.Join(relPath, info.Name())
		if relPath == "" {
			relFilePath = info.Name()
		}
		files[relFilePath] = packageFile

		return nil
	})

	if err != nil {
		return nil, err
	}

	rootImportPath := buildImportPath(absPath, absPath)

	return &GoPkg{
		Path:           pkgPath,
		Files:          files,
		fset:           fset,
		recursive:      isRecursive,
		rootPath:       absPath,
		rootImportPath: rootImportPath,
	}, nil
}

func buildImportPath(rootPath, currentPath string) string {
	modPath := findGoMod(currentPath)
	if modPath == "" {
		relPath, _ := filepath.Rel(rootPath, currentPath)
		if relPath == "." || relPath == "" {
			return filepath.Base(rootPath)
		}
		return filepath.ToSlash(filepath.Join(filepath.Base(rootPath), relPath))
	}

	modName := getModuleName(modPath)
	if modName == "" {
		relPath, _ := filepath.Rel(rootPath, currentPath)
		if relPath == "." || relPath == "" {
			return filepath.Base(rootPath)
		}
		return filepath.ToSlash(filepath.Join(filepath.Base(rootPath), relPath))
	}

	relPath, err := filepath.Rel(filepath.Dir(modPath), currentPath)
	if err != nil || relPath == "." {
		return modName
	}
	return filepath.ToSlash(filepath.Join(modName, relPath))
}

func findGoMod(startPath string) string {
	for path := startPath; ; path = filepath.Dir(path) {
		mod := filepath.Join(path, "go.mod")
		if _, err := os.Stat(mod); err == nil {
			return mod
		}
		if parent := filepath.Dir(path); parent == path {
			break
		}
	}
	return ""
}

func getModuleName(goModPath string) string {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(strings.TrimPrefix(line, "module "))
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}
	return ""
}

func (g *GoPkg) FindAllStructRegexp(pattern *regexp.Regexp) []StructInfo {
	var results []StructInfo

	for filePath, pkgFile := range g.Files {
		ast.Inspect(pkgFile.File, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if _, ok := ts.Type.(*ast.StructType); ok {
					name := ts.Name.Name
					if pattern.MatchString(name) {
						results = append(results, StructInfo{
							Name:       name,
							FileName:   filepath.Base(filePath),
							Package:    pkgFile.PackageName,
							ImportPath: pkgFile.ImportPath,
							RelPath:    pkgFile.RelPath,
							Self:       pkgFile.ImportPath == g.rootImportPath,
						})
					}
				}
			}
			return true
		})
	}

	return results
}

func (g *GoPkg) GetImportPathFor(packageDir string) string {
	absPath, err := filepath.Abs(packageDir)
	if err != nil {
		return ""
	}
	return buildImportPath(g.rootPath, absPath)
}

func (g *GoPkg) GetRootImportPath() string {
	return g.rootImportPath
}

func (g *GoPkg) GetFileNames() []string {
	var list []string
	for name := range g.Files {
		list = append(list, name)
	}
	return list
}

func (g *GoPkg) GetPackageNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, file := range g.Files {
		if !seen[file.PackageName] {
			seen[file.PackageName] = true
			names = append(names, file.PackageName)
		}
	}
	return names
}

func (g *GoPkg) IsRecursive() bool {
	return g.recursive
}
