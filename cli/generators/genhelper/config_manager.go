package genhelper

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigManager manages YAML configuration files while preserving structure
type ConfigManager struct {
	originalContent string
	lines           []string
	rootNode        *yaml.Node
}

// InsertOptions defines options for inserting configuration
type InsertOptions struct {
	InsertAfter         string // Insert after this key
	InsertBefore        string // Insert before this key
	MaintainIndentation bool   // Maintain proper indentation (default: true)
	AddSpacing          bool   // Add blank lines for readability (default: true)
	PreserveComments    bool   // Preserve existing comments (default: true)
}

// NewConfigManager creates a new YAML configuration manager
func NewConfigManager(yamlContent string) (*ConfigManager, error) {
	cm := &ConfigManager{
		originalContent: yamlContent,
		lines:           strings.Split(yamlContent, "\n"),
	}

	// Parse YAML while preserving structure
	if err := yaml.Unmarshal([]byte(yamlContent), &cm.rootNode); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return cm, nil
}

// AddConfig adds configuration to a specific path in the YAML structure
func (cm *ConfigManager) AddConfig(path string, config interface{}, options *InsertOptions) error {
	if options == nil {
		options = &InsertOptions{
			MaintainIndentation: true,
			AddSpacing:          true,
			PreserveComments:    true,
		}
	}

	pathParts := strings.Split(path, ".")
	if len(pathParts) == 0 {
		return fmt.Errorf("invalid path: %s", path)
	}

	// Convert config to YAML string
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return cm.insertAtPath(pathParts, string(configYaml), options)
}

// AddConfigs adds multiple configurations at once
func (cm *ConfigManager) AddConfigs(configs map[string]interface{}, options *InsertOptions) error {
	for path, config := range configs {
		if err := cm.AddConfig(path, config, options); err != nil {
			return fmt.Errorf("failed to add config at path %s: %w", path, err)
		}
	}
	return nil
}

// ToString returns the modified YAML content
func (cm *ConfigManager) ToString() string {
	return strings.Join(cm.lines, "\n")
}

// insertAtPath inserts configuration at the specified path
func (cm *ConfigManager) insertAtPath(pathParts []string, configYaml string, options *InsertOptions) error {
	if len(pathParts) == 1 {
		// Insert at root level
		return cm.insertInSection(pathParts[0], "", configYaml, options)
	}

	// Find the target section and subsection
	sectionName := pathParts[0]
	subPath := strings.Join(pathParts[1:], ".")

	return cm.insertInSection(sectionName, subPath, configYaml, options)
}

// insertInSection inserts configuration in a specific section
func (cm *ConfigManager) insertInSection(sectionName, subPath, configYaml string, options *InsertOptions) error {
	sectionStart, sectionEnd, baseIndent := cm.findSection(sectionName)
	if sectionStart == -1 {
		return fmt.Errorf("section '%s' not found", sectionName)
	}

	// If we have a subPath, find the subsection
	if subPath != "" {
		subSectionStart, subSectionEnd, subIndent := cm.findSubSection(sectionStart, sectionEnd, strings.Split(subPath, ".")[0])
		if subSectionStart == -1 {
			// Create the subsection if it doesn't exist
			return cm.createSubSection(sectionName, subPath, configYaml, options)
		}
		return cm.insertConfigInSubSection(subSectionStart, subSectionEnd, subIndent, configYaml, options)
	}

	// Insert directly in the main section
	return cm.insertConfigInSection(sectionStart, sectionEnd, baseIndent, configYaml, options)
}

// findSection finds the start and end lines of a section
func (cm *ConfigManager) findSection(sectionName string) (start, end, indent int) {
	sectionRegex := regexp.MustCompile(`^(\s*)` + regexp.QuoteMeta(sectionName) + `:\s*(.*)$`)

	for i, line := range cm.lines {
		if matches := sectionRegex.FindStringSubmatch(line); matches != nil {
			start = i
			indent = len(matches[1])

			// Find the end of this section
			end = cm.findSectionEnd(start, indent)
			return start, end, indent
		}
	}
	return -1, -1, 0
}

// findSectionEnd finds where a section ends
func (cm *ConfigManager) findSectionEnd(start, baseIndent int) int {
	for i := start + 1; i < len(cm.lines); i++ {
		line := cm.lines[i]

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.TrimSpace(line)[0] == '#' {
			continue
		}

		// Check if this line starts a new section at the same or higher level
		currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))
		if currentIndent <= baseIndent && strings.Contains(line, ":") {
			return i - 1
		}
	}
	return len(cm.lines) - 1
}

// findSubSection finds a subsection within a section
func (cm *ConfigManager) findSubSection(sectionStart, sectionEnd int, subSectionName string) (start, end, indent int) {
	subSectionRegex := regexp.MustCompile(`^(\s*)` + regexp.QuoteMeta(subSectionName) + `:\s*(.*)$`)

	for i := sectionStart + 1; i <= sectionEnd; i++ {
		if matches := subSectionRegex.FindStringSubmatch(cm.lines[i]); matches != nil {
			start = i
			indent = len(matches[1])
			end = cm.findSectionEnd(start, indent)
			if end > sectionEnd {
				end = sectionEnd
			}
			return start, end, indent
		}
	}
	return -1, -1, 0
}

// insertConfigInSection inserts configuration within a section
func (cm *ConfigManager) insertConfigInSection(sectionStart, sectionEnd, baseIndent int, configYaml string, options *InsertOptions) error {
	// Process the config YAML to match indentation
	configLines := strings.Split(strings.TrimSpace(configYaml), "\n")
	indentedConfig := make([]string, len(configLines))

	targetIndent := baseIndent + 2 // Standard YAML indent
	indentStr := strings.Repeat(" ", targetIndent)

	for i, line := range configLines {
		if strings.TrimSpace(line) != "" {
			indentedConfig[i] = indentStr + strings.TrimSpace(line)
		} else {
			indentedConfig[i] = ""
		}
	}

	// Find insertion point
	insertPoint := cm.findInsertionPoint(sectionStart, sectionEnd, options)

	// Insert the configuration
	newLines := make([]string, 0, len(cm.lines)+len(indentedConfig))
	newLines = append(newLines, cm.lines[:insertPoint]...)

	if options.AddSpacing && insertPoint > 0 && strings.TrimSpace(cm.lines[insertPoint-1]) != "" {
		newLines = append(newLines, "")
	}

	newLines = append(newLines, indentedConfig...)

	if options.AddSpacing && insertPoint < len(cm.lines) && strings.TrimSpace(cm.lines[insertPoint]) != "" {
		newLines = append(newLines, "")
	}

	newLines = append(newLines, cm.lines[insertPoint:]...)

	cm.lines = newLines
	return nil
}

// insertConfigInSubSection inserts configuration within a subsection
func (cm *ConfigManager) insertConfigInSubSection(subStart, subEnd, baseIndent int, configYaml string, options *InsertOptions) error {
	return cm.insertConfigInSection(subStart, subEnd, baseIndent, configYaml, options)
}

// findInsertionPoint finds the best place to insert new configuration
func (cm *ConfigManager) findInsertionPoint(sectionStart, sectionEnd int, options *InsertOptions) int {
	if options.InsertAfter != "" {
		for i := sectionStart; i <= sectionEnd; i++ {
			if strings.Contains(cm.lines[i], options.InsertAfter+":") {
				return i + 1
			}
		}
	}

	if options.InsertBefore != "" {
		for i := sectionStart; i <= sectionEnd; i++ {
			if strings.Contains(cm.lines[i], options.InsertBefore+":") {
				return i
			}
		}
	}

	// Default: insert at the end of the section
	return sectionEnd + 1
}

// createSubSection creates a new subsection if it doesn't exist
func (cm *ConfigManager) createSubSection(sectionName, subPath, configYaml string, options *InsertOptions) error {
	// This is a simplified implementation - you might want to enhance this
	// to handle nested subsection creation
	sectionStart, sectionEnd, baseIndent := cm.findSection(sectionName)
	if sectionStart == -1 {
		return fmt.Errorf("section '%s' not found", sectionName)
	}

	subSectionName := strings.Split(subPath, ".")[0]
	targetIndent := baseIndent + 2
	indentStr := strings.Repeat(" ", targetIndent)

	newSubSection := fmt.Sprintf("%s%s:", indentStr, subSectionName)

	// Insert the new subsection
	insertPoint := sectionEnd + 1
	newLines := make([]string, 0, len(cm.lines)+1)
	newLines = append(newLines, cm.lines[:insertPoint]...)
	newLines = append(newLines, newSubSection)
	newLines = append(newLines, cm.lines[insertPoint:]...)

	cm.lines = newLines

	// Now insert the configuration in the new subsection
	return cm.insertConfigInSection(insertPoint, insertPoint, targetIndent, configYaml, options)
}
