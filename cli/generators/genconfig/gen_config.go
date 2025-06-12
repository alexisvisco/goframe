package genconfig

import (
	"embed"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

// ConfigGenerator generates configuration files.
type ConfigGenerator struct{ Gen *generators.Generator }

//go:embed templates
var fs embed.FS

func (c *ConfigGenerator) Generate() error {
	files := []generators.FileConfig{
		c.createConfigYaml("config/config.yml"),
		c.createConfigGo("config/config.go"),
	}
	return c.Gen.GenerateFiles(files)
}

func (c *ConfigGenerator) createConfigYaml(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/config.yml.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("db", c.Gen.DatabaseType).
				WithVar("db__filepath", "db/storage.db").
				WithVar("worker", c.Gen.WorkerType)
		},
	}
}

func (c *ConfigGenerator) createConfigGo(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/config.go.tmpl")),
	}
}
