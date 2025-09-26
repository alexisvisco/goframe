package genstorage

import (
	"embed"
	"path/filepath"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

type StorageGenerator struct {
	Gen   *generators.Generator
	DBGen *gendb.DatabaseGenerator
}

//go:embed templates
var fs embed.FS

func (p *StorageGenerator) Generate() error {
	files := []generators.FileConfig{
		p.CreateStorageProvider("internal/provide/provide_storage.go"),
	}

	files = append(files, p.CreateMigrations()...)

	return p.Gen.GenerateFiles(files)
}

// CreateStorageProvider creates the FileConfig for the storage provider
func (p *StorageGenerator) CreateStorageProvider(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/provide_storage.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.Gen.GoModuleName, "config"), "config")
		},
	}
}

func (p *StorageGenerator) CreateMigrations() []generators.FileConfig {
	up := p.getSQLMigration()

	return p.DBGen.CreateMigration(gendb.CreateMigrationParams{
		Sql:  true,
		Name: "storage",
		At:   time.Now(),
		Up:   up,
		Down: "drop table if exists attachments;",
	})
}

// getSQLMigration returns the SQL for creating the attachments table based on database type
func (p *StorageGenerator) getSQLMigration() string {
	switch p.Gen.DatabaseType {
	case configuration.DatabaseTypePostgres:
		return `
CREATE TABLE IF NOT EXISTS attachments (
	id TEXT PRIMARY KEY,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL,
	byte_size BIGINT NOT NULL,
	key TEXT NOT NULL UNIQUE,
	checksum TEXT NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT timezone('utc', now()),
	deleted_at TIMESTAMP WITH TIME ZONE NULL
);
CREATE INDEX IF NOT EXISTS idx_attachments_created_at ON attachments(created_at);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
		`
	default:
		return ""
	}
}
