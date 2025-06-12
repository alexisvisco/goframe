package generators

import (
	"fmt"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

type TaskGenerator struct {
	g *Generator
}

func (t *TaskGenerator) createTaskFile(name, description string) error {
	path := filepath.Join("internal/task", fmt.Sprintf("task_%s.go", str.ToSnakeCase(name)))
	return t.g.GenerateFile(FileConfig{
		Path:      path,
		Template:  templates.InternalTaskNewTaskGo,
		Category:  CategoryTasks,
		Condition: true,
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("name_kebab_case", str.ToKebabCase(name)).
				WithVar("name_pascal_case", str.ToPascalCase(name)).
				WithVar("name_camel_case", str.ToCamelCase(name)).
				WithVar("description", str.ToSnakeCase(description))
		},
	})
}

func (t *TaskGenerator) updateCliMain(name string) error {
	path := "cmd/cli/main.go"
	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return err
	}

	line := fmt.Sprintf("\t\trootcmd.WithCommand(\"task\", task.New%sTask(app.Module(cfg))),", str.ToPascalCase(name))
	gf.AddNamedImport("", filepath.Join(t.g.GoModuleName, "internal/task"))
	gf.AddLineAfterString("cmdRoot := rootcmd.NewCmdRoot(", line)
	return gf.Save()
}

func (t *TaskGenerator) Create(name, description string) error {
	if err := t.createTaskFile(name, description); err != nil {
		return err
	}
	return t.updateCliMain(name)
}
