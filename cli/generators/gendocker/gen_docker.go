package gendocker

import (
	"embed"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

// DockerGenerator generates Docker related files.
type DockerGenerator struct {
	Gen *generators.Generator
}

//go:embed templates
var fs embed.FS

// Generate installs all docker files.
func (d *DockerGenerator) Generate() error {
	files := []generators.FileConfig{
		d.createDockerfile("Dockerfile"),
		d.createDockerCompose("docker-compose.yaml"),
		d.createDockerIgnore(".dockerignore"),
	}

	return d.Gen.GenerateFiles(files)
}

// createDockerfile returns the Dockerfile definition.
func (d *DockerGenerator) createDockerfile(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/Dockerfile.tmpl")),
	}
}

// createDockerCompose returns the docker-compose file definition.
func (d *DockerGenerator) createDockerCompose(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/docker-compose.yml.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("db", d.Gen.DatabaseType).
				WithVar("worker", d.Gen.WorkerType)
		},
	}
}

// createDockerIgnore returns the .dockerignore file definition.
func (d *DockerGenerator) createDockerIgnore(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: d.getDockerIgnoreTemplate(),
	}
}

// getDockerIgnoreTemplate build the .dockerignore template.
func (d *DockerGenerator) getDockerIgnoreTemplate() []byte {
	lines := []string{
		"*.log",
		"*.tmp",
		"*.db",
		"db/storage.db",
		".env",
		".git",
	}

	return []byte(strings.Join(lines, "\n"))
}
