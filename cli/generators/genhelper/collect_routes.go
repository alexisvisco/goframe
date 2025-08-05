package genhelper

import (
	"go/ast"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alexisvisco/goframe/http/apidoc"
)

func CollectRoutesDocumentation(workdir string, packagePaths []string) ([]*apidoc.Route, error) {
	var routes []*apidoc.Route
	for _, pkg := range packagePaths {
		gopkg, err := LoadGoPkg(pkg, false)
		if err != nil {
			return nil, err
		}

		for _, file := range gopkg.Files {
			ast.Inspect(file.File, func(n ast.Node) bool {
				fd, ok := n.(*ast.FuncDecl)
				if !ok || fd.Doc == nil || fd.Recv == nil {
					return true
				}
				hasRoute := false
				for _, c := range fd.Doc.List {
					if strings.Contains(c.Text, "goframe:http_route") {
						hasRoute = true
						break
					}
				}
				if !hasRoute {
					return true
				}

				var structName string
				switch t := fd.Recv.List[0].Type.(type) {
				case *ast.StarExpr:
					if ident, ok := t.X.(*ast.Ident); ok {
						structName = ident.Name
					}
				case *ast.Ident:
					structName = t.Name
				}
				if structName == "" {
					return true
				}

				r, err := apidoc.ParseRoute(workdir, file.ImportPath, structName, fd.Name.Name)
				if err == nil {
					routes = append(routes, r)
				}
				return true
			})
		}
	}

	slices.SortFunc(routes, func(a, b *apidoc.Route) int {
		parentStructNameA := ""
		if a.ParentStructName != nil {
			parentStructNameA = *a.ParentStructName
		}

		parentStructNameB := ""
		if b.ParentStructName != nil {
			parentStructNameB = *b.ParentStructName
		}

		return strings.Compare(
			strings.Join([]string{a.PackagePath, parentStructNameA, a.Name}, "/"),
			strings.Join([]string{b.PackagePath, parentStructNameB, b.Name}, "/"),
		)
	})

	return routes, nil
}

type RootHandlerPackage struct {
	Path       string   // Path where the handler is located (root path)
	Subfolders []string // Subfolders that contain others handlers
}

// CollectRootHandlerPackages collects all subfolders that contain HTTP handlers.
// A root handler is where the router.go or registry.go files are located.
// It returns a slice of RootHandlerPackage, each representing a root handler package
// and its subfolders that contain HTTP handlers but no router or registry files.
// The workdir can be the go.mod root directory.
// example: command run in the module root directory and have:
// - internal/v1handler/dashboard root
// - internal/v1handler/dashboard/taxes
// - internal/v1handler root
// - internal/v1handler/user
// - internal/v1handler/user/company
// - internal/v1handler/profile
// - internal/v2handler root
// Will return:
// - internal/v1handler/dashboard root with subfolders: ["internal/v1handler/dashboard/taxes"]
// - internal/v1handler root with subfolders: ["internal/v1handler/user", "internal/v1handler/profile", "internal/v1handler/user/company"]
// - internal/v2handler root with subfolders: []
func CollectRootHandlerPackages(workdir string) ([]RootHandlerPackage, error) {
	var handlerDirs []string
	var rootDirs []string

	// Single pass: collect all Go directories and identify roots
	err := filepath.Walk(workdir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || !info.IsDir() {
			return walkErr
		}

		hasGoFiles, err := containsGoFiles(path)
		if err != nil || !hasGoFiles {
			return err
		}

		relPath, err := filepath.Rel(workdir, path)
		if err != nil || relPath == "." {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		handlerDirs = append(handlerDirs, relPath)

		// Check if this is a root handler directory
		if isRoot, err := isRootHandlerDir(path); err != nil {
			return err
		} else if isRoot {
			rootDirs = append(rootDirs, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Build result: for each root, find its subfolders
	var result []RootHandlerPackage
	for _, root := range rootDirs {
		var subfolders []string

		for _, dir := range handlerDirs {
			if isSubfolder(root, dir) {
				subfolders = append(subfolders, dir)
			}
		}

		result = append(result, RootHandlerPackage{
			Path:       root,
			Subfolders: subfolders,
		})
	}

	// Sort the result by root path
	slices.SortFunc(result, func(a, b RootHandlerPackage) int {
		return strings.Compare(a.Path, b.Path)
	})

	return result, nil
}

// containsGoFiles checks if a directory contains any .go files
func containsGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			return true, nil
		}
	}

	return false, nil
}

// isRootHandlerDir checks if a directory contains both router.go and registry.go files
func isRootHandlerDir(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	var hasRouter, hasRegistry bool
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch entry.Name() {
		case "router.go":
			hasRouter = true
		case "registry.go":
			hasRegistry = true
		}
	}

	return hasRouter && hasRegistry, nil
}

// isSubfolder checks if childPath is a direct or indirect subfolder of parentPath
func isSubfolder(parentPath, childPath string) bool {
	// Normalize paths
	parentPath = filepath.ToSlash(parentPath)
	childPath = filepath.ToSlash(childPath)

	// Child must start with parent followed by separator and be longer
	return strings.HasPrefix(childPath, parentPath+"/")
}
