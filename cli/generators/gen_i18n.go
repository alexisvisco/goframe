package generators

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"gopkg.in/yaml.v3"
)

type (
	I18nGenerator struct {
		g *Generator
	}

	i18nSyncOperation string

	i18nSyncOperationInfo struct {
		Operation i18nSyncOperation
		Key       string
	}
)

const (
	operationAdd    i18nSyncOperation = "add"
	operationDelete i18nSyncOperation = "delete"
)

func (d *I18nGenerator) NewFile(name, path string, cfg configuration.I18n) ([]FileConfig, error) {
	// Create example translations
	example := map[string]interface{}{
		"welcome": "Welcome {user} to our application",
		"errors": map[string]interface{}{
			"not_found":     "Resource {code:int} not found",
			"system_error":  "A system error occurred",
			"invalid_input": "Invalid input ({input}) provided by {user}, available options: {options:[]string}",
		},
		"messages": map[string]interface{}{
			"success": "Operation completed successfully",
			"status": map[string]interface{}{
				"pending":  "Your request is pending",
				"approved": "Your request has been approved",
				"rejected": "Your request has been rejected",
			},
		},
	}

	var files []FileConfig
	for _, locale := range cfg.SupportedLocales {
		filePath := filepath.Join(path, fmt.Sprintf("%s.%s.yml", str.ToSnakeCase(name), locale))
		// Create YAML file
		content, err := yaml.Marshal(example)
		if err != nil {
			return nil, fmt.Errorf("error marshaling YAML: %w", err)
		}

		files = append(files, FileConfig{
			Path:      filePath,
			Template:  content,
			Category:  CategoryI18n,
			Condition: true,
		})
	}

	return files, nil
}

func (g *I18nGenerator) CreateOrUpdateGoFile(name, path string, cfg configuration.I18n) (FileConfig, error) {
	content, err := os.ReadFile(filepath.Join(path, fmt.Sprintf("%s.%s.yml", str.ToSnakeCase(name), cfg.DefaultLocale)))
	if err != nil {
		return FileConfig{}, fmt.Errorf("error reading input file: %w", err)
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return FileConfig{}, fmt.Errorf("error parsing YAML: %w", err)
	}

	tree := buildTranslationTree(data, "")
	needsStrings := g.needsStringsPackage(tree)
	structsCode := g.generateStructCode(tree, str.ToPascalCase(name), str.ToPascalCase(name), "")
	embedsFilesVariablesCode, files := g.generateEmbedsFilesVariablesCode(name, cfg.SupportedLocales)

	return FileConfig{
		Path:     filepath.Join(path, fmt.Sprintf("%s.gen.go", str.ToSnakeCase(name))),
		Template: templates.ConfigI18nTranslationGo,
		Gen: func(gen *genhelper.GenHelper) {
			if needsStrings {
				gen.WithImport("strings", "strings")
			}

			gen.WithVar("package_name", cfg.Package).
				WithVar("struct_name", str.ToPascalCase(name)).
				WithVar("structs_code", structsCode).
				WithVar("embeds_files_variables_code", embedsFilesVariablesCode).
				WithVar("files", files)
		},
		Condition: true,
		Category:  CategoryI18n,
	}, nil
}

func (g *I18nGenerator) SyncTranslationFiles(name string, locale string, cfg configuration.I18n) ([]FileConfig, error) {
	if locale == "" {
		locale = cfg.DefaultLocale
	}
	// Read base translations
	baseContent, err := os.ReadFile(filepath.Join(cfg.Folder, fmt.Sprintf("%s.%s.yml", str.ToSnakeCase(name), locale)))
	if err != nil {
		return nil, fmt.Errorf("error reading base translations: %w", err)
	}

	var baseTranslations map[string]interface{}
	if err := yaml.Unmarshal(baseContent, &baseTranslations); err != nil {
		return nil, fmt.Errorf("error parsing base translations: %w", err)
	}

	// Sync each locale
	files := make([]FileConfig, 0)

	for _, locale := range cfg.SupportedLocales {
		if locale == cfg.DefaultLocale {
			continue
		}

		var localeTranslations map[string]interface{}
		localeFile := filepath.Join(cfg.Folder, fmt.Sprintf("%s.%s.yml", str.ToSnakeCase(name), locale))
		// Read existing translations or create new
		content, err := os.ReadFile(localeFile)
		if err == nil {
			if err := yaml.Unmarshal(content, &localeTranslations); err != nil {
				return nil, fmt.Errorf("error parsing %s translations: %w", locale, err)
			}

			// Sync translations
			translations, infos := g.syncTranslations(baseTranslations, localeTranslations)
			if len(infos) == 0 {
				slog.Info("no changes", "file", localeFile)
				continue
			} else {
				slog.Info("changes detected", "file", localeFile, "changes", len(infos))
			}
			localeTranslations = translations
		} else {
			localeTranslations = baseTranslations
		}

		// Write updated translations
		content, err = yaml.Marshal(localeTranslations)
		if err != nil {
			return nil, fmt.Errorf("error marshaling %s translations: %w", locale, err)
		}

		files = append(files, FileConfig{
			Path:      localeFile,
			Template:  content,
			Condition: true,
			Category:  CategoryI18n,
		})
	}

	return files, nil
}

func (g *I18nGenerator) needsStringsPackage(node *I18nTranslationNode) bool {
	for _, child := range node.Children {
		// Check if any parameter is a slice type
		for _, param := range child.Parameters {
			if strings.HasPrefix(param.Type, "[]") {
				return true
			}
		}
		// Recursively check children
		if len(child.Children) > 0 && g.needsStringsPackage(child) {
			return true
		}
	}

	return false
}

func (g *I18nGenerator) generateStructCode(node *I18nTranslationNode, baseStruct, currentStruct, prefix string) string {
	var sb strings.Builder

	// WriteTo struct
	sb.WriteString(fmt.Sprintf("type %s struct {\n", currentStruct))
	sb.WriteString("\ttranslations *i18n.Translations\n")

	// Add nested struct fields
	hasChildren := false
	for key, child := range node.Children {
		if len(child.Children) > 0 {
			hasChildren = true
			fieldName := formatStructName(key)
			nestedStructName := fmt.Sprintf("%s%s", baseStruct, fieldName)
			sb.WriteString(fmt.Sprintf("\t%s *%s\n", fieldName, nestedStructName))
		}
	}
	sb.WriteString("}\n\n")

	// WriteTo initialization method if struct has children
	if hasChildren {
		sb.WriteString(fmt.Sprintf("func (t *%s) initializeStructs() {\n", currentStruct))
		for key, child := range node.Children {
			if len(child.Children) > 0 {
				fieldName := formatStructName(key)
				nestedStructName := fmt.Sprintf("%s%s", baseStruct, fieldName)
				sb.WriteString(fmt.Sprintf("\tt.%s = &%s{translations: t.translations}\n", fieldName, nestedStructName))
				if g.containsNestedChildren(child) {
					sb.WriteString(fmt.Sprintf("\tt.%s.initializeStructs()\n", fieldName))
				}
			}
		}
		sb.WriteString("}\n\n")
	}

	// WriteTo nested structs
	for key, child := range node.Children {
		if len(child.Children) > 0 {
			fieldName := formatStructName(key)
			nestedStructName := fmt.Sprintf("%s%s", baseStruct, fieldName)
			sb.WriteString(g.generateStructCode(child, baseStruct, nestedStructName, joinPrefix(prefix, key)))
		}
	}

	// WriteTo methods
	for key, child := range node.Children {
		if child.Value != "" {
			methodName := formatStructName(key)
			fullKey := joinPrefix(prefix, key)

			// WriteTo method signature
			sb.WriteString(fmt.Sprintf("func (t *%s) %s(ctx context.Context, ", currentStruct, methodName))
			for i, param := range child.Parameters {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s %s", param.Name, param.Type))
			}
			sb.WriteString(") string {\n")

			// WriteTo return statement
			if len(child.Parameters) > 0 {
				// Create format string with proper format specifiers
				args := make([]string, len(child.Parameters))
				for i, param := range child.Parameters {
					args[i] = g.generateArgFormatting(param)
				}
				sb.WriteString(fmt.Sprintf("\treturn fmt.Sprintf(t.translations.Get(ctx, \"%s\"), %s)\n",
					fullKey, strings.Join(args, ", ")))
			} else {
				sb.WriteString(fmt.Sprintf("\treturn t.translations.Get(ctx, \"%s\")\n", fullKey))
			}
			sb.WriteString("}\n\n")
		}
	}

	return sb.String()
}

// Helper function to check if a node contains any nested children
func (g *I18nGenerator) containsNestedChildren(node *I18nTranslationNode) bool {
	for _, child := range node.Children {
		if len(child.Children) > 0 {
			return true
		}
	}
	return false
}
func (g *I18nGenerator) generateArgFormatting(param i18nParameter) string {
	if strings.HasPrefix(param.Type, "[]") {
		return fmt.Sprintf("strings.Join(%s, \", \")", param.Name)
	}
	return param.Name
}

func (g *I18nGenerator) generateEmbedsFilesVariablesCode(file string, locales []string) (string, map[string]string) {
	var sb strings.Builder
	files := make(map[string]string)

	// WriteTo //go:embed directive for all locale files
	sb.WriteString("//go:embed")
	for _, locale := range locales {
		sb.WriteString(fmt.Sprintf(" %s.%s.yml", file, locale))
	}
	sb.WriteString("\n")

	// Create embed.FS variable
	embedVarName := fmt.Sprintf("%sTranslations", file)
	sb.WriteString(fmt.Sprintf("var %s embed.FS\n\n", embedVarName))

	// WriteTo variables for each locale's content
	for _, locale := range locales {
		varName := str.ToCamelCase(fmt.Sprintf("%s_%s_Translations", file, formatStructName(locale)))
		filePath := fmt.Sprintf("%s.%s.yml", file, locale)

		// Add entry to files map
		files[locale] = varName

		// WriteTo variable declaration
		sb.WriteString(fmt.Sprintf("var %s = func() []byte {\n", varName))
		sb.WriteString(fmt.Sprintf("\tcontent, err := %s.ReadFile(\"%s\")\n", embedVarName, filePath))
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString(fmt.Sprintf("\t\tpanic(fmt.Sprintf(\"failed to read embedded %s file: %%v\", err))\n", locale))
		sb.WriteString("\t}\n")
		sb.WriteString("\treturn content\n")
		sb.WriteString("}()\n\n")
	}

	return sb.String(), files
}

func formatStructName(name string) string {
	name = str.ToPascalCase(name)
	if len(name) > 0 {
		r := []rune(name)
		r[0] = unicode.ToUpper(r[0])
		name = string(r)
	}
	if name == "" {
		return "Translations"
	}
	return name
}

// i18nParameter represents a translation parameter
type i18nParameter struct {
	Name string
	Type string
}

// I18nTranslationNode represents a node in the translation tree
type I18nTranslationNode struct {
	Value      string
	Parameters []i18nParameter
	Children   map[string]*I18nTranslationNode
}

func newTranslationNode() *I18nTranslationNode {
	return &I18nTranslationNode{
		Children: make(map[string]*I18nTranslationNode),
	}
}

func buildTranslationTree(data map[string]interface{}, prefix string) *I18nTranslationNode {
	root := newTranslationNode()

	var process func(map[string]interface{}, *I18nTranslationNode, string)
	process = func(data map[string]interface{}, node *I18nTranslationNode, currentPrefix string) {
		for key, value := range data {
			switch v := value.(type) {
			case map[string]interface{}:
				childNode := newTranslationNode()
				node.Children[key] = childNode
				process(v, childNode, joinPrefix(currentPrefix, key))
			default:
				childNode := newTranslationNode()
				str := fmt.Sprint(v)
				childNode.Value, childNode.Parameters = parseParameters(str)
				node.Children[key] = childNode
			}
		}
	}

	process(data, root, prefix)
	return root
}

// Parse parameters from a translation string - modified to clean format strings
func parseParameters(value string) (string, []i18nParameter) {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(value, -1)

	var params []i18nParameter
	resultStr := value

	for _, match := range matches {
		full := match[0]
		param := match[1]

		// Parse parameter type
		parts := strings.Split(param, ":")
		paramName := parts[0]
		paramType := "string"
		if len(parts) > 1 {
			paramType = parts[1]
		}

		params = append(params, i18nParameter{
			Name: paramName,
			Type: paramType,
		})

		// Replace with %s for slice types, %v for others
		var formatSpecifier string
		if strings.HasPrefix(paramType, "[]") {
			formatSpecifier = "%s"
		} else {
			formatSpecifier = "%v"
		}
		resultStr = strings.Replace(resultStr, full, formatSpecifier, 1)
	}

	return resultStr, params
}

func joinPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func (g *I18nGenerator) syncTranslations(base, target map[string]interface{}) (map[string]any, []i18nSyncOperationInfo) {
	ops := make([]i18nSyncOperationInfo, 0)

	flattenBase := make(map[string]string)
	flattenForSync(base, "", flattenBase) // Use new function that preserves interpolation

	flattenTarget := make(map[string]string)
	flattenForSync(target, "", flattenTarget) // Use new function that preserves interpolation

	for key, baseValue := range flattenBase {
		if _, ok := flattenTarget[key]; !ok {
			ops = append(ops, i18nSyncOperationInfo{Operation: operationAdd, Key: key})
			flattenTarget[key] = baseValue
		}
	}

	for key, _ := range flattenTarget {
		if _, ok := flattenBase[key]; !ok {
			ops = append(ops, i18nSyncOperationInfo{Operation: operationDelete, Key: key})
			delete(flattenTarget, key)
		}
	}

	return unflatten(flattenTarget), ops
}

// New function for syncing that preserves interpolation syntax
func flattenForSync(m map[string]interface{}, prefix string, translations map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch v := v.(type) {
		case map[string]interface{}:
			flattenForSync(v, key, translations)
		default:
			// Keep original string without parsing parameters for sync
			translations[key] = fmt.Sprint(v)
		}
	}
}

// Keep the original flatten function for other uses
func flatten(m map[string]interface{}, prefix string, translations map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch v := v.(type) {
		case map[string]interface{}:
			flatten(v, key, translations)
		default:
			fmtStr, _ := parseParameters(fmt.Sprint(v))
			translations[key] = fmtStr
		}
	}
}

func unflatten(m map[string]string) map[string]interface{} {
	root := make(map[string]interface{})
	for k, v := range m {
		keys := strings.Split(k, ".")
		current := root
		for i, key := range keys {
			if i == len(keys)-1 {
				current[key] = v
			} else {
				if _, ok := current[key]; !ok {
					current[key] = make(map[string]interface{})
				}
				current = current[key].(map[string]interface{})
			}
		}
	}
	return root
}
