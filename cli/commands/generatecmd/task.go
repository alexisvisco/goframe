package generatecmd

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/spf13/cobra"
)

func taskCmd() *cobra.Command {
	var flagDescription string
	cmd := &cobra.Command{
		Use:   "task <name>",
		Short: "Create a new task file",
		Long:  "Create a new task file with the specified name.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("migration name is required")
			}

			name := args[0]

			g := generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
				ORMType:      "gorm",
			}

			path := fmt.Sprintf("internal/task/task_%s.go", str.ToSnakeCase(name))
			err := g.GenerateFile(generators.FileConfig{
				Template:  templates.InternalTaskNewTaskGo,
				Path:      path,
				Condition: true,
				Category:  generators.CategoryTasks,
				Gen: func(g *genhelper.GenHelper) {
					g.WithVar("name_kebab_case", str.ToKebabCase(name)).
						WithVar("name_pascal_case", str.ToPascalCase(name)).
						WithVar("name_camel_case", str.ToCamelCase(name)).
						WithVar("description", str.ToSnakeCase(flagDescription))
				},
			})

			slog.Info("created: task file", slog.String("file", path))

			if err != nil {
				return fmt.Errorf("failed to create task file: %w", err)
			}

			err = tryAddTaskToCliMain(name, cmd.Context().Value("module").(string))
			if err != nil {
				return fmt.Errorf("failed to add task to cli main: %w", err)
			}

			slog.Info("modified: cli main", slog.String("file", "cmd/cli/main.go"))

			return nil
		},
	}

	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Description of the task")

	return cmd
}

func tryAddTaskToCliMain(name, modname string) error {
	lineToFind := `cmdRoot := rootcmd.NewCmdRoot(`

	file, err := os.ReadFile("cmd/cli/main.go")
	if err != nil {
		return fmt.Errorf("failed to read main.go: %w", err)
	}

	if !strings.Contains(string(file), lineToFind) {
		return fmt.Errorf("line not found in main.go: %s", lineToFind)
	}

	// add import for the new task
	importLine := `import (`
	if !strings.Contains(string(file), importLine) {
		return fmt.Errorf("import line not found in main.go: %s", importLine)
	}
	fileByLine := strings.Split(string(file), "\n")
	newImport := fmt.Sprintf("\t\"%s/internal/task\"\n", modname)
	if !strings.Contains(string(file), newImport) {
		fileByLine = genhelper.InsertLineAfter(fileByLine, importLine, newImport)
	}

	newCommand := fmt.Sprintf("\t\t"+`rootcmd.WithCommand("task", task.New%sTask(app.Module(cfg))),`, str.ToPascalCase(name))
	if !strings.Contains(string(file), newCommand) {
		fileByLine = genhelper.InsertLineAfter(fileByLine, lineToFind, newCommand)
	}

	if err := os.WriteFile("cmd/cli/main.go", []byte(strings.Join(fileByLine, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	return nil
}
