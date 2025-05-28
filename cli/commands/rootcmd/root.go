package rootcmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/commands/dbcmd"
	"github.com/alexisvisco/goframe/cli/commands/generatecmd"
	"github.com/alexisvisco/goframe/db/migrate"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
)

type OptionFunc func(*options)

type options struct {
	Migrations []migrate.Migration
	DB         func() (*sql.DB, error)
}

func WithMigrations(migrations []migrate.Migration) OptionFunc {
	return func(o *options) {
		o.Migrations = migrations
	}
}

func WithDB(dbConnector func() (*sql.DB, error)) OptionFunc {
	return func(o *options) {
		o.DB = dbConnector
	}
}

func NewCmdRoot(opts ...OptionFunc) *cobra.Command {
	// Default options
	defaultOpts := &options{
		Migrations: []migrate.Migration{},
		DB:         nil,
	}

	// Apply options
	for _, opt := range opts {
		opt(defaultOpts)
	}

	cmd := &cobra.Command{
		Use:           "<command> <subcommand> [flags]",
		Short:         "Goframe CLI",
		Long:          `Manage your goframe application with the CLI.`,
		SilenceErrors: true,
		Annotations: map[string]string{
			"versionInfo": "0.0.1",
		},
		SilenceUsage: true,
		Version:      "0.0.1",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// find the go mod file and find the module name to set it in the ctx
			// if not found go .. until it finds it
			// if not found, return an error
			goModPath, err := findGoMod(".")
			if err != nil {
				return fmt.Errorf("failed to find go.mod: %w", err)
			}

			moduleName, err := parseModuleName(goModPath)
			if err != nil {
				return fmt.Errorf("failed to parse module name from go.mod: %w", err)
			}

			// Set the module name in the command context
			cmd.SetContext(context.WithValue(cmd.Context(), "module", moduleName))
			cmd.SetContext(context.WithValue(cmd.Context(), "migrations", defaultOpts.Migrations))
			cmd.SetContext(context.WithValue(cmd.Context(), "db", defaultOpts.DB))

			return nil
		},
	}

	cmd.AddCommand(dbcmd.NewCmdRootMigrate())
	cmd.AddCommand(generatecmd.NewCmdRootGenerate())

	return cmd
}

// findGoMod searches for go.mod file starting from current directory and going up
func findGoMod(startDir string) (string, error) {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return goModPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod file not found")
}

// parseModuleName extracts module name from go.mod file using the official parser
func parseModuleName(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse go.mod: %w", err)
	}

	if modFile.Module == nil || modFile.Module.Mod.Path == "" {
		return "", fmt.Errorf("module declaration not found in go.mod")
	}

	return modFile.Module.Mod.Path, nil
}
