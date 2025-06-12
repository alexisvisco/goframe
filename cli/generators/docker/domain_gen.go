package generators

import (
	"fmt"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
)

// DockerGenerator generates DockerFiles-related files
type DockerGenerator struct {
	g *Generator
}

func (d *DockerGenerator) Generate() error {
	files := []FileConfig{
		d.createDockerfile("Dockerfile"),
		d.createDockerCompose("docker-compose.yaml"),
		d.createDockerIgnore(".dockerignore"),
	}

	for _, file := range files {
		if err := d.g.GenerateFile(file); err != nil {
			return fmt.Errorf("failed to create docker file %s: %w", file.Path, err)
		}
	}

	return nil
}

// createDockerfile creates the FileConfig for the Dockerfile
func (d *DockerGenerator) createDockerfile(path string) FileConfig {
	return FileConfig{
		Path:      path,
		Template:  templates.Dockerfile,
		Category:  CategoryDocker,
		Condition: d.g.DockerFiles,
	}
}

// createDockerCompose creates the FileConfig for the docker-compose.yaml file
func (d *DockerGenerator) createDockerCompose(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.DockerComposeYml,
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("db", d.g.DatabaseType).
				WithVar("worker", d.g.WorkerType)
		},
		Category:  CategoryDocker,
		Condition: d.g.DockerFiles,
	}
}

// createDockerIgnore creates the FileConfig for the .dockerignore file
func (d *DockerGenerator) createDockerIgnore(path string) FileConfig {
	return FileConfig{
		Path:      path,
		Template:  d.getDockerIgnoreTemplate(),
		Category:  CategoryDocker,
		Condition: d.g.DockerFiles,
	}
}

// getDockerIgnoreTemplate returns the .dockerignore content as a template
func (d *DockerGenerator) getDockerIgnoreTemplate() []byte {
	dockerIgnoreLines := []string{
		"*.log",
		"*.tmp",
		"*.db",
		"db/storage.db",
		".env",
		".git",
	}

	return []byte(strings.Join(dockerIgnoreLines, "\n"))
}
