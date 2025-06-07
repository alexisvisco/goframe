package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/configuration"
)

// FileInfo represents information about a generated file
type FileInfo struct {
	Path     string       // File Path
	IsDir    bool         // Whether it's a directory
	Category FileCategory // Category of the file
}

// NewFileInfo creates a new FileInfo
func NewFileInfo(path string, isDir bool, category FileCategory) FileInfo {
	return FileInfo{
		Path:     path,
		IsDir:    isDir,
		Category: category,
	}
}

type Generator struct {
	GoModuleName string
	DatabaseType configuration.DatabaseType
	ORMType      string
	Maintainer   bool

	WebFiles        bool
	ExampleWebFiles bool
	DockerFiles     bool

	filesCreated []FileInfo
}

// TrackFile adds a file to the list of created files
func (g *Generator) TrackFile(path string, isDir bool, category FileCategory) {
	g.filesCreated = append(g.filesCreated, NewFileInfo(path, isDir, category))
}

func (g *Generator) PrintCreatedFiles(rootFolder string) {
	filesByCategory := make(map[FileCategory][]FileInfo)
	seenFiles := make(map[string]bool)

	for _, file := range g.filesCreated {
		uniqueKey := fmt.Sprintf("%s-%s", file.Category, file.Path)
		if !seenFiles[uniqueKey] {
			filesByCategory[file.Category] = append(filesByCategory[file.Category], file)
			seenFiles[uniqueKey] = true
		}
	}

	// Sort categories alphabetically for consistent output
	var categories []FileCategory
	for category := range filesByCategory {
		categories = append(categories, category)
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i] < categories[j]
	})

	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("%-12s %-8s %s\n", "Type", "Kind", "File Path")
	fmt.Println(strings.Repeat("=", 70))

	// Print each Category and its files
	for _, category := range categories {
		fmt.Printf("\n%s\n\n", category)
		files := filesByCategory[category]

		// Sort files within each Category by Path
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		firstLine := true
		// Print files within each Category
		for _, file := range files {
			action := "File"
			if file.IsDir {
				action = "Directory"
			}

			kind := ""
			if firstLine {
				kind = string(category)
				firstLine = false
			}

			// Clean up paths for display
			displayPath := filepath.Clean(filepath.Join(rootFolder, file.Path))
			fmt.Printf("%-12s %-8s %s\n", action, kind, displayPath)
		}
		fmt.Println(strings.Repeat("-", 70))
	}
	fmt.Println()
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

// CreateDirectory creates a directory if it doesn't exist
func (g *Generator) CreateDirectory(path string, category FileCategory) error {

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	g.TrackFile(path, true, category)
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

	err = gen.Generate(file)
	if err != nil {
		return fmt.Errorf("failed to generate file: %w", err)
	}

	g.TrackFile(f.Path, false, f.Category)
	return nil

}
