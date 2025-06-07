package i18n

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Translations struct {
	translations map[string]map[string]string
}

// Parameter represents a translation parameter
type Parameter struct {
	Name string
	Type string
}

func (t *Translations) Get(ctx context.Context, key string) string {
	language, ok := ctx.Value("language").(string)
	if !ok {
		language = "en"
	}
	if translations, ok := t.translations[language]; ok {
		if translation, ok := translations[key]; ok {
			return translation
		}
	}
	return key
}

func TranslationsFromFiles(files map[string][]byte) (*Translations, error) {
	t := &Translations{translations: make(map[string]map[string]string)}
	for language, content := range files {
		rawMap := make(map[string]any)

		err := yaml.Unmarshal(content, &rawMap)
		if err != nil {
			return nil, fmt.Errorf("error parsing translations %s file: %w", language, err)
		}
		flatMap := make(map[string]string)
		flatten(rawMap, "", flatMap)

		t.translations[language] = flatMap
	}

	return t, nil
}

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

// Parse parameters from a translation string - modified to clean format strings
func parseParameters(value string) (string, []Parameter) {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(value, -1)

	var params []Parameter
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

		params = append(params, Parameter{
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
