package gentask

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

// TaskGenerator manages task files.
type TaskGenerator struct{ Gen *generators.Generator }

//go:embed templates
var fs embed.FS

func (t *TaskGenerator) createTaskFile(name, description string) error {
	path := filepath.Join("internal/task", fmt.Sprintf("task_%s.go", str.ToSnakeCase(name)))
	return t.Gen.GenerateFile(generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/new_task.go.tmpl")),
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
	gf.AddNamedImport("", filepath.Join(t.Gen.GoModuleName, "internal/task"))
	gf.AddLineAfterString("cmdRoot := rootcmd.NewCmdRoot(", line)
	return gf.Save()
}

func (t *TaskGenerator) GenerateTask(name, description string) error {
	if err := t.createTaskFile(name, description); err != nil {
		return err
	}
	return t.updateCliMain(name)
}
