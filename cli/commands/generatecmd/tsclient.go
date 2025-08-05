package generatecmd

import (
	"fmt"
	"os"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/gentsclient"
	"github.com/spf13/cobra"
)

func tsclientCmd() *cobra.Command {
	var flagFile string
	var flagPkg string
	cmd := &cobra.Command{
		Use: "client [packages...]",
		RunE: func(cmd *cobra.Command, args []string) error {
			workdir, _ := cmd.Context().Value("workdir").(string)
			packages, err := genhelper.CollectRootHandlerPackages(workdir)
			if err != nil {
				return fmt.Errorf("failed to collect root handler packages: %w", err)
			}

			var paths []string
			for _, pkg := range packages {
				if pkg.Path == flagPkg {
					paths = append(paths, pkg.Path)
					paths = append(paths, pkg.Subfolders...)
					break
				}
			}

			if len(paths) == 0 {
				return fmt.Errorf("no package found with name %s", flagPkg)
			}

			routes, err := genhelper.CollectRoutesDocumentation(workdir, paths)
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

			if flagFile != "" {
				file, err := os.OpenFile(flagFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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

	cmd.Flags().StringVarP(&flagFile, "file", "f", "", "Output file for the generated TypeScript client code")
	cmd.Flags().StringVarP(&flagPkg, "pkg", "p", "internal/v1handler", "Package name where routes are defined")

	return cmd
}
