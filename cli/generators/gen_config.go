package generators

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
)

// ConfigGenerator generates configuration files
type ConfigGenerator struct {
	g *Generator
}

func (c *ConfigGenerator) Generate() error {
	files := []FileConfig{
		c.CreateConfigYaml("config/config.yml"),
		c.CreateConfigGo("config/config.go"),
	}

	for _, file := range files {
		if err := c.g.GenerateFile(file); err != nil {
			return fmt.Errorf("failed to create config file %s: %w", file.Path, err)
		}
	}

	return nil
}

// CreateConfigYaml generates the config.yml file
func (c *ConfigGenerator) CreateConfigYaml(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.ConfigConfigYml,
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("db", c.g.DatabaseType).
				WithVar("db__filepath", "db/storage.db")
		},
		Category:  CategoryConfig,
		Condition: true,
	}
}

// CreateConfigGo generates the config.go file
func (c *ConfigGenerator) CreateConfigGo(path string) FileConfig {
	return FileConfig{
		Path:      path,
		Template:  templates.ConfigConfigGo,
		Condition: true,
		Category:  CategoryConfig,
	}
}
