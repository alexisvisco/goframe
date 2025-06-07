package i18ncmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/spf13/cobra"
)

func NewCmdI18n() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "i18n <subcommand> [flags]",
		Aliases: []string{"i"},
		Short:   "Manage i18n translations",
	}

	cmd.AddCommand(newCmd())
	cmd.AddCommand(generateCmd())
	cmd.AddCommand(syncCmd())

	return cmd
}

func newCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new i18n translation file",
		Long: `Generate i18n files from a yaml file.
Example:
	$ goframe i18n new translations
	
Will generate N * translations.{lang}.yaml files, based on the config of available languages in your config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: name")
			}

			g := generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
			}

			cfg := cmd.Context().Value("config.i18n").(configuration.I18n)

			files, err := g.I18n().NewFile(args[0], "config/i18n", cfg)
			if err != nil {
				return fmt.Errorf("error creating new i18n file: %w", err)
			}

			for _, file := range files {
				err := g.GenerateFile(file)
				if err != nil {
					return fmt.Errorf("error generating i18n file %s: %w", file.Path, err)
				}
			}

			file, err := g.I18n().CreateOrUpdateGoFile(args[0], "config/i18n", cfg)
			if err != nil {
				return fmt.Errorf("error creating or updating Go file for i18n: %w", err)
			}

			err = g.GenerateFile(file)
			if err != nil {
				return fmt.Errorf("error generating Go file for i18n: %w", err)
			}

			return nil
		},
	}
}

func generateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "generate <name>",
		Short: "Regenerate the go file for i18n translations",
		Long: `Generate go code based on yaml i18n file.
Example:
	$ goframe i18n gen translations

Will generate a Translations struct with all the translations from the translations.{default_lang}.yaml file

So if you have translations.en.yam like this one:

--- file: translations.en.yaml
	errors:
		size: "The size is invalid required {required} got {got}"
		invalid_input: "Invalid input"
		available_choices: "The choice {choice} is not available, must be one of {choices:[]string}"


You will be able to use it like this:

--- example locales usage in go
	yourvar.Errors.Size(ctx, 10, 5)
	yourvar.Errors.InvalidInput(ctx)
	yourvar.Errors.AvailableChoices(ctx, "a", []string{"a", "b", "c"})


Context is used to get the language from the request, if you haven't set the language in the context it will use the
default language in configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: name")
			}

			g := generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
			}

			cfg := cmd.Context().Value("config.i18n").(configuration.I18n)

			file, err := g.I18n().CreateOrUpdateGoFile(args[0], "config/i18n", cfg)
			if err != nil {
				return fmt.Errorf("error creating or updating Go file for i18n: %w", err)
			}

			err = g.GenerateFile(file)
			if err != nil {
				return fmt.Errorf("error generating Go file for i18n: %w", err)
			}

			return nil
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync <name>",
		Short: "Synchronize i18n translations with the current configuration",
		Long: `Synchronize file using default language, adding missing keys, removing unused keys.
Example:
	$ goframe i18n sync translations
	
Will synchronize the translations.{default_lang}.yaml file with the translations.{lang}.yaml files
If a new key is added to the default language file, it will be added to all the other languages files. 
If a new language file is added, it will be created with the default language file content.
Be careful because if a key is removed from the default language file, it will be removed from all the other languages 
files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: name")
			}

			g := generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
			}

			cfg := cmd.Context().Value("config.i18n").(configuration.I18n)

			files, err := g.I18n().SyncTranslationFiles(args[0], "", cfg)
			if err != nil {
				return fmt.Errorf("error synchronizing i18n translations: %w", err)
			}

			for _, file := range files {
				err := g.GenerateFile(file)
				if err != nil {
					return fmt.Errorf("error generating i18n file %s: %w", file.Path, err)
				}
			}

			return nil
		},
	}
}
