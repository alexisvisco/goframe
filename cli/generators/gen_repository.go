package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

type RepositoryGenerator struct {
	g *Generator
}

func (r *RepositoryGenerator) ensureTypes(name string) error {
	path := filepath.Join("internal/types", fmt.Sprintf("%s.go", str.ToSnakeCase(name)))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return r.g.GenerateFile(FileConfig{
			Path:      path,
			Template:  templates.InternalTypesNewGo,
			Category:  CategoryWeb,
			Condition: true,
			Gen: func(g *genhelper.GenHelper) {
				g.WithVar("name_pascal", str.ToPascalCase(name)).WithVar("name_snake", str.ToSnakeCase(name))
			},
		})
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	pascal := str.ToPascalCase(name)
	snake := str.ToSnakeCase(name)
	var add []string
	if !strings.Contains(content, "type "+pascal+"Repository") {
		add = append(add, fmt.Sprintf("type %sRepository interface{}\n", pascal))
	}
	if !strings.Contains(content, "type "+pascal+"Service") {
		add = append(add, fmt.Sprintf("type %sService interface{}\n", pascal))
	}
	if !strings.Contains(content, "Err"+pascal+"NotFound") {
		add = append(add, fmt.Sprintf("var Err%sNotFound = fmt.Errorf(\"%s not found\")\n", pascal, snake))
	}
	if len(add) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, l := range add {
		if _, err := f.WriteString("\n" + l); err != nil {
			return err
		}
	}
	return nil
}

func (r *RepositoryGenerator) createRepositoryFile(name string) error {
	path := filepath.Join("internal/repository", fmt.Sprintf("repository_%s.go", str.ToSnakeCase(name)))
	return r.g.GenerateFile(FileConfig{
		Path:      path,
		Template:  templates.InternalRepositoryNewGo,
		Category:  CategoryWeb,
		Condition: true,
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("name_pascal", str.ToPascalCase(name))
		},
	})
}

func (r *RepositoryGenerator) listRepositories() ([]string, error) {
	entries, err := os.ReadDir("internal/repository")
	if err != nil {
		return nil, err
	}
	var repos []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		if e.Name() == "registry.go" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		name = strings.TrimPrefix(name, "repository_")
		pascal := str.ToPascalCase(name)
		if !strings.HasSuffix(pascal, "Repository") {
			pascal += "Repository"
		}
		repos = append(repos, pascal)
	}
	return repos, nil
}

func (r *RepositoryGenerator) updateRegistry() error {
	repos, err := r.listRepositories()
	if err != nil {
		return err
	}
	path := "internal/repository/registry.go"
	var file *os.File
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err = os.Create(path)
		if err != nil {
			return err
		}
	} else {
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	gh := genhelper.New("repository", templates.InternalRepositoryRegistryGo)
	gh.WithImport(filepath.Join(r.g.GoModuleName, "internal/types"), "types").
		WithImport("github.com/alexisvisco/goframe/core/helpers/fxutil", "fxutil")
	r.g.TrackFile(path, false, CategoryWeb)
	return gh.WithVar("repositories", repos).Generate(file)
}

func (r *RepositoryGenerator) updateAppModule() error {
	path := "internal/app/module.go"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	hasImport := false
	for _, l := range lines {
		if strings.Contains(l, "/internal/repository") {
			hasImport = true
			break
		}
	}
	if !hasImport {
		for i, l := range lines {
			if strings.TrimSpace(l) == "import (" {
				importLine := fmt.Sprintf("\t\"%s\"", filepath.Join(r.g.GoModuleName, "internal/repository"))
				lines = append(lines[:i+1], append([]string{importLine}, lines[i+1:]...)...)
				break
			}
		}
	}
	hasProvide := false
	for _, l := range lines {
		if strings.Contains(l, "repository.Dependencies") {
			hasProvide = true
			break
		}
	}
	if !hasProvide {
		for i, l := range lines {
			if strings.Contains(l, "fx.Provide(") {
				lines = append(lines[:i], append([]string{"    fx.Provide(repository.Dependencies...),"}, lines[i:]...)...)
				break
			}
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

func (r *RepositoryGenerator) Create(name string) error {
	if err := r.createRepositoryFile(name); err != nil {
		return err
	}
	if err := r.ensureTypes(name); err != nil {
		return err
	}
	if err := r.updateRegistry(); err != nil {
		return err
	}
	return r.updateAppModule()
}
