package generatecmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/spf13/cobra"
)

func handlerCmd() *cobra.Command {
	var defaultPackage = "internal/v1handler"
	var flagPackage string
	var flagForced bool
	cmd := &cobra.Command{
		Use:   "handler <name> [services...]",
		Short: "Create a handler for handling http requests",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("handler name is required")
			}
			name := args[0]
			services := args[1:]

			g := cmd.Context().Value("generator").(*generators.Generator)
			genHandler := &genhttp.HTTPGenerator{Gen: g, BasePath: flagPackage}

			if flagPackage != defaultPackage {
				if err := preparePackage(flagPackage, flagForced, genHandler); err != nil {
					return fmt.Errorf("failed to prepare package: %w", err)
				}
			}

			if err := genHandler.GenerateHandler(name, services); err != nil {
				return fmt.Errorf("failed to create handler: %w", err)
			}
			return nil
		}),
	}

	cmd.Flags().StringVar(&flagPackage, "pkg", defaultPackage, "Package name for the generated handler")
	cmd.Flags().BoolVar(&flagForced, "force", false, "If true and the pkg is inside a directory that contains router.go or registry.go, it will generate them inside the pkg instead of the directory of the existing registry and router")

	return cmd
}

func preparePackage(pkg string, forced bool, genHandler *genhttp.HTTPGenerator) error {
	if _, err := os.Stat(pkg); os.IsNotExist(err) {
		if err := os.MkdirAll(pkg, 0755); err != nil {
			os.Exit(1)
		}
	}

	// Check if router.go and registry.go exist in any parent directory
	rootPath, ok := hasInitRootPath(pkg, forced)
	genHandler.RootPath = rootPath
	if !ok {
		if err := genHandler.Generate(); err != nil {
			return fmt.Errorf("failed to generate router and registry: %w", err)
		}
	}

	return nil
}

func hasInitRootPath(startPath string, forced bool) (string, bool) {

	currentPath := startPath
	for {
		if forced {
			break
		}
		routerPath := filepath.Join(currentPath, "router.go")
		registryPath := filepath.Join(currentPath, "registry.go")

		// Check if both files exist
		if _, err := os.Stat(routerPath); err == nil {
			if _, err := os.Stat(registryPath); err == nil {
				return currentPath, true
			}
		}

		parent := filepath.Dir(currentPath)
		// Stop if we've reached the root directory
		if parent == currentPath {
			break
		}
		currentPath = parent
	}

	return startPath, false
}
