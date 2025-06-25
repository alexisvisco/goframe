package generators

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/configuration"
)

type Generator struct {
	GoModuleName     string
	DatabaseType     configuration.DatabaseType
	ORMType          string
	Maintainer       bool
	HTTPServer       bool
	ExampleHTTPFiles bool
	WorkerType       string
}

type FilesGenerator interface {
	Generate() error // Generate the file or directory
}

type FileConfig struct {
	Path       string
	Template   []byte
	Gen        func(g *genhelper.GenHelper)
	Skip       bool
	Executable bool // If true, the file will be executable
}

func (g *Generator) GenerateFiles(files []FileConfig) error {
	for _, f := range files {
		if err := g.GenerateFile(f); err != nil {
			return fmt.Errorf("failed to generate file %s: %w", f.Path, err)
		}
	}
	return nil
}

func (g *Generator) GenerateFile(f FileConfig) error {
	if f.Skip {
		return nil
	}

	if err := g.CreateDirectory(filepath.Dir(f.Path)); err != nil {
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

// CreateDirectory creates a directory if it doesn't exist
func (g *Generator) CreateDirectory(path string) error {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	return nil
}

func (g *Generator) SkipDirectoryIfExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false // Directory does not exist, so we can create it
	}
	return true // Directory exists, so we skip creating it
}

func (g *Generator) SkipFileIfExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
