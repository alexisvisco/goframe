package generators

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/configuration"
)

type Generator struct {
	GoModuleName string
	DatabaseType configuration.DatabaseType
	ORMType      string
	Maintainer   bool

	WebFiles        bool
	ExampleWebFiles bool
	DockerFiles     bool
	WorkerType      string
}

// Databases returns a database file generator
func (g *Generator) Databases() *DatabaseGenerator {
	return &DatabaseGenerator{g: g}
}

// Core returns a core file generator
func (g *Generator) Core() *CoreGenerator {
	return &CoreGenerator{g: g}
}

// Config returns a config file generator
func (g *Generator) Config() *ConfigGenerator {
	return &ConfigGenerator{g: g}
}

// DockerFiles returns a docker file generator
func (g *Generator) Docker() *DockerGenerator {
	return &DockerGenerator{g: g}
}

func (g *Generator) Storage() *StorageGenerator {
	return &StorageGenerator{g: g, db: g.Databases()}
}

// I18n returns an i18n file generator
func (g *Generator) I18n() *I18nGenerator {
	return &I18nGenerator{g: g}
}

// Web returns a web file generator
func (g *Generator) Web() *WebGenerator {
	return &WebGenerator{g: g}
}

func (g *Generator) Worker() *WorkerGenerator {
	return &WorkerGenerator{g: g}
}

func (g *Generator) Repository() *RepositoryGenerator {
	return &RepositoryGenerator{g: g}
}

func (g *Generator) Service() *ServiceGenerator {
	return &ServiceGenerator{g: g}
}

func (g *Generator) Mailer() *MailerGenerator {
	return &MailerGenerator{g: g}
}

func (g *Generator) Task() *TaskGenerator {
	return &TaskGenerator{g: g}
}

// CreateDirectory creates a directory if it doesn't exist
func (g *Generator) CreateDirectory(path string, category FileCategory) error {

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

type FileCreator interface {
	Generate() error // Generate the file or directory
}

type FileConfig struct {
	Path       string
	Template   []byte
	Gen        func(g *genhelper.GenHelper)
	Condition  bool
	Category   FileCategory
	Executable bool // If true, the file will be executable
}

func (g *Generator) GenerateFile(f FileConfig) error {
	if !f.Condition {
		return nil
	}

	if err := g.CreateDirectory(filepath.Dir(f.Path), f.Category); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.OpenFile(f.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	if f.Executable {
		err = file.Chmod(0777)
		if err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	defer file.Close()

	gen := genhelper.New("current", f.Template)

	if f.Gen != nil {
		f.Gen(gen)
	}

	err = gen.WriteTo(file)
	if err != nil {
		return fmt.Errorf("failed to generate file: %w", err)
	}

	return nil

}
