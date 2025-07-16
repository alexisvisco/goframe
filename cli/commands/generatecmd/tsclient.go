package generatecmd

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/gentsclient"
	"github.com/alexisvisco/goframe/http/apidoc"
	"github.com/spf13/cobra"
)

func tsclientCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use: "client [packages...]",
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

			if file != "" {
				file, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return fmt.Errorf("failed to open output file %s: %w", file, err)
				}

				defer file.Close()
				content := generator.File()
				if _, err := file.WriteString(content); err != nil {
					return fmt.Errorf("failed to write to output file %s: %w", file, err)
				}
			} else {
				fmt.Println(generator.File())
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Output file for the generated TypeScript client code")

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
