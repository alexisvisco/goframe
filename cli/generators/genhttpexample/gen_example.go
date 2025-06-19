package genhttpexample

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/cli/generators/genrepository"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

//go:embed templates
var fs embed.FS

type NoteExampleGenerator struct {
	Gen     *generators.Generator
	GenHTTP *genhttp.HTTPGenerator
	GenSvc  *genservice.ServiceGenerator
	GenRepo *genrepository.RepositoryGenerator
}

func (n *NoteExampleGenerator) Generate() error {
	files := []generators.FileConfig{
		n.CreateHandler("internal/v1handler/handler_note.go"),
		n.CreateTypes("internal/types/note.go"),
		n.CreateRepository("internal/repository/repository_note.go"),
		n.CreateService("internal/service/service_note.go"),
	}

	if err := n.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if err := n.GenHTTP.Update(); err != nil {
		return fmt.Errorf("failed to update HTTP files: %w", err)
	}

	if err := n.GenSvc.Update(); err != nil {
		return fmt.Errorf("failed to update service files: %w", err)
	}

	if err := n.GenRepo.Update(); err != nil {
		return fmt.Errorf("failed to update repository files: %w", err)
	}

	if err := n.addRoutesIfNotExists(); err != nil {
		return fmt.Errorf("failed to add routes: %w", err)
	}

	return nil
}

func (n *NoteExampleGenerator) CreateHandler(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/handler_note.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(n.Gen.GoModuleName, "internal/types"), "types")
		},
	}
}

func (n *NoteExampleGenerator) CreateRepository(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/repository_note.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(n.Gen.GoModuleName, "internal/types"), "types")
		},
	}
}

func (n *NoteExampleGenerator) CreateService(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/service_note.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(n.Gen.GoModuleName, "internal/types"), "types")
		},
	}
}

func (n *NoteExampleGenerator) CreateTypes(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/type_note.go.tmpl")),
	}
}

func (n *NoteExampleGenerator) addRoutesIfNotExists() error {
	routes := []string{
		`p.Mux.HandleFunc("POST /v1/notes", p.NoteHandler.CreateNote())`,
		`p.Mux.HandleFunc("GET /v1/notes", p.NoteHandler.ListNotes())`,
		`p.Mux.HandleFunc("PUT /v1/notes/{id}", p.NoteHandler.UpdateNote())`,
		`p.Mux.HandleFunc("DELETE /v1/notes/{id}", p.NoteHandler.DeleteNote())`,
	}

	gofile, err := genhelper.LoadGoFile("internal/v1handler/router.go")
	if err != nil {
		return fmt.Errorf("failed to load router.go: %w", err)
	}
	for _, route := range routes {
		gofile.AddLineAfterString(`func Router(p RouterParams) {`, "\t"+route)
	}

	if err := gofile.Save(); err != nil {
		return fmt.Errorf("failed to save router.go: %w", err)
	}

	return nil
}
