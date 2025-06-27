package refactorcmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/spf13/cobra"
)

func NewCmdRefactor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refactor",
		Short: "Set of utilies to refactor existing code",
	}

	cmd.AddCommand(splitHandlerCmd())

	return cmd
}

func splitHandlerCmd() *cobra.Command {
	yesForAllFlag := false
	cmd := &cobra.Command{
		Use: "split-handler <handler name> <file name>",
		Short: `Refactor handlers, it search all methods in the file for the given handler struct name and ask you if you
want to extract them into a separate file (with request response if any)`,
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("handler name is required")
			}

			if len(args) < 2 {
				return fmt.Errorf("file name is required")
			}

			handlerName := args[0]
			fileName := args[1]

			pkg, err := genhelper.LoadGoPkg("internal/v1handler")
			if err != nil {
				return fmt.Errorf("failed to load v1handler package: %w", err)
			}

			methods := pkg.FindAllMethodsForStruct(handlerName, fileName)
			for _, method := range methods {
				newFile := fmt.Sprintf("%s_%s.go", strings.TrimSuffix(fileName, ".go"), str.ToSnakeCase(method))
				relatedStructs := pkg.FindAllStructRegexp(regexp.MustCompile(fmt.Sprintf(`^%s`, method)))
				filteredStructs := make([]string, 0, len(relatedStructs))
				for _, s := range relatedStructs {
					if strings.Contains(s.FileName, fileName) {
						filteredStructs = append(filteredStructs, s.Name)
					}
				}

				scanner := bufio.NewScanner(os.Stdin)
				if !yesForAllFlag {
					prompt := fmt.Sprintf("Do you want to extract the following content into the new file %s?\n", newFile)

					for _, filteredStruct := range filteredStructs {
						prompt += fmt.Sprintf("- struct %s\n", filteredStruct)
					}
					prompt += fmt.Sprintf("- method %s\n", method)
					prompt += "Type 'yes' to confirm or 'no' to skip: "

					fmt.Print(prompt)

					if !scanner.Scan() {
						if err := scanner.Err(); err != nil {
							return fmt.Errorf("failed to read input: %w", err)
						}
						fmt.Println("Skipping due to EOF")
						continue
					}

					input := strings.TrimSpace(strings.ToLower(scanner.Text()))

					switch input {
					case "yes", "y":
						fmt.Printf("Extracting to %s...\n", newFile)
						err := extractMethodAndStruct(handlerName, method, filteredStructs, fileName, newFile, pkg)
						if err != nil {
							return fmt.Errorf("failed to extract method %s: %w", method, err)
						}
					case "no", "n":
						fmt.Printf("Skipping %s\n", method)
						continue
					default:
						fmt.Printf("Invalid input '%s'. Skipping %s\n", input, method)
						continue
					}
				} else {
					question := fmt.Sprintf("Automatically extracting method %s to file %s with structs: %s\n", method, newFile, strings.Join(filteredStructs, ", "))
					fmt.Print(question)
					err := extractMethodAndStruct(handlerName, method, filteredStructs, fileName, newFile, pkg)
					if err != nil {
						return fmt.Errorf("failed to extract method %s: %w", method, err)
					}
				}
			}

			return nil
		}),
	}

	cmd.Flags().BoolVarP(&yesForAllFlag, "yes", "y", false, "Automatically answer yes to all methods")

	return cmd
}

func extractMethodAndStruct(handlerName string, method string, structs []string, oldFile, newFile string, pkg *genhelper.GoPkg) error {
	for _, s := range structs {
		err := pkg.ExtractStruct(s, oldFile, newFile)
		if err != nil {
			return fmt.Errorf("failed to extract struct %s: %w", s, err)
		}
	}

	err := pkg.ExtractMethod(handlerName, method, oldFile, newFile)
	if err != nil {
		return fmt.Errorf("failed to extract method %s from %s: %w", method, handlerName, err)
	}

	return nil
}
