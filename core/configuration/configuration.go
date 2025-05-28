package configuration

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

func Parse(content []byte, into any) error {
	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}

	if into == nil {
		return fmt.Errorf("second parameter cannot be nil")
	}

	newContent := replaceVariables(content)

	return yaml.NewDecoder(bytes.NewReader(newContent)).Decode(into)
}

// replaceVariables replaces environment variable placeholders in the content with their actual values.
// It supports both ${VAR} and $VAR formats.
// For ${VAR:default} format, it uses the default value if VAR doesn't exist or is empty.
func replaceVariables(content []byte) []byte {
	if content == nil {
		return nil
	}

	str := string(content)
	var result strings.Builder
	result.Grow(len(str)) // Pre-allocate to avoid repeated allocations

	i := 0
	for i < len(str) {
		if str[i] == '$' && i+1 < len(str) {
			// Check for $$ pattern - treat as literal $$
			if str[i+1] == '$' {
				result.WriteString("$$")
				i += 2
				continue
			}
			// Check if it's ${VAR} or ${VAR:default} format
			if str[i+1] == '{' {
				// Find closing brace
				start := i + 2
				end := start
				for end < len(str) && str[end] != '}' {
					end++
				}

				if end < len(str) && str[end] == '}' {
					// Valid ${VAR} or ${VAR:default} format
					varSpec := str[start:end]

					// Check for default value syntax
					colonIndex := strings.Index(varSpec, ":")
					var varName, defaultValue string

					if colonIndex >= 0 {
						varName = strings.TrimSpace(varSpec[:colonIndex])
						defaultValue = strings.TrimSpace(varSpec[colonIndex+1:])
					} else {
						varName = varSpec
					}

					if value, exists := os.LookupEnv(varName); exists && value != "" {
						result.WriteString(value)
					} else if colonIndex >= 0 {
						// Use default value
						result.WriteString(defaultValue)
					} else {
						// Keep original placeholder if variable doesn't exist and no default
						result.WriteString(str[i : end+1])
					}
					i = end + 1
				} else {
					// No closing brace found, treat as literal
					result.WriteByte(str[i])
					i++
				}
			} else if unicode.IsLetter(rune(str[i+1])) || str[i+1] == '_' {
				// Check if it's $VAR format (no default value support for this format)
				// Only proceed if the character after $ is a valid variable start character (letter or underscore)
				start := i + 1
				end := start

				// Find the end of the variable name (alphanumeric, underscore)
				for end < len(str) && isValidVarChar(rune(str[end])) {
					end++
				}

				// Valid $VAR format found
				varName := str[start:end]
				if value, exists := os.LookupEnv(varName); exists {
					result.WriteString(value)
				} else {
					// Keep original placeholder if variable doesn't exist
					result.WriteString(str[i:end])
				}
				i = end
			} else {
				// Character after $ is not a valid variable start character, treat $ as literal
				result.WriteByte(str[i])
				i++
			}
		} else {
			result.WriteByte(str[i])
			i++
		}
	}

	return []byte(result.String())
}

// isValidVarChar checks if a character is valid in an environment variable name
func isValidVarChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
