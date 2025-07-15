package generatecmd

import (
	"go/ast"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/gentsclient"
	"github.com/alexisvisco/goframe/http/apidoc"
	"github.com/spf13/cobra"
)

func NewCmdRootGenerate(subCommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate <subcommand> [flags]",
		Aliases: []string{"gen", "g"},
		Short:   "WriteTo code",
	}

	cmd.AddCommand(migrationCmd())
	cmd.AddCommand(taskCmd())
	cmd.AddCommand(workerCmd())
	cmd.AddCommand(serviceCmd())
	cmd.AddCommand(handlerCmd())
	cmd.AddCommand(mailerCmd())
	cmd.AddCommand(seedCmd())
	cmd.AddCommand(moduleCmd())
	cmd.AddCommand(NewCmdGenClient())
	for _, subCmd := range subCommands {
		cmd.AddCommand(subCmd)
	}

	return cmd
}

func NewCmdGenClient() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "client [packages...]",
		Aliases: []string{"gen", "g"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{"v1handler"}
			}

			workdir, _ := cmd.Context().Value("workdir").(string)

			routes, err := collectRoutes(workdir, args)
			if err != nil {
				return err
			}

			generator := gentsclient.NewTypescriptClientGenerator()

			for _, r := range routes {
				if r.Request != nil {
					generator.AddSchema("", true, *r.Request)
				}
				for _, response := range r.StatusToResponse {
					if response.Response != nil {
						generator.AddSchema("", false, *response.Response)
					}
				}
				generator.AddRoute(*r)
			}

			cmd.Println(generator.File())
			return nil
		},
	}

	return cmd
}

func collectRoutes(workdir string, packages []string) ([]*apidoc.Route, error) {
	var routes []*apidoc.Route
	for _, pkg := range packages {
		pkgPath := filepath.Join("internal", pkg)
		gopkg, err := genhelper.LoadGoPkg(pkgPath, true)
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
	return routes, nil
}
