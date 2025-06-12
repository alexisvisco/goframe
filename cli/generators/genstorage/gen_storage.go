package genstorage

import (
	"embed"
	"fmt"
	"path/filepath"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

type StorageGenerator struct {
	g  *generators.Generator
	db *generators.DatabaseGenerator
}

//go:embed templates
var fs embed.FS

func (p *StorageGenerator) Generate() error {
	files := []generators.FileConfig{
		p.CreateStorageProvider("internal/provide/provide_storage.go"),
	}

	if err := p.g.GenerateFiles(files); err != nil {
		return err
	}

	if err := p.AddMigrations(); err != nil {
		return fmt.Errorf("failed to generate storage migration: %w", err)
	}

	return nil
}

// CreateStorageProvider creates the FileConfig for the storage provider
func (p *StorageGenerator) CreateStorageProvider(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/provide_storage.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.g.GoModuleName, "config"), "config")
		},
	}
}

func (p *StorageGenerator) AddMigrations() error {
	up := p.getSQLMigration()

	return p.db.CreateMigration(generators.CreateMigrationParams{
		Sql:  true,
		Name: "storage",
		At:   time.Now(),
		Up:   up,
		Down: "drop table if exists attachments;",
	})
}

// getSQLMigration returns the SQL for creating the attachments table based on database type
func (p *StorageGenerator) getSQLMigration() string {
	switch p.g.DatabaseType {
	case configuration.DatabaseTypeSQLite:
		return `
CREATE TABLE IF NOT EXISTS attachments (
	id TEXT PRIMARY KEY,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL,
	byte_size INTEGER NOT NULL,
	key TEXT NOT NULL UNIQUE,
	checksum TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	deleted_at DATETIME NULL
);
CREATE INDEX IF NOT EXISTS idx_attachments_created_at ON attachments(created_at);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
		`
	case configuration.DatabaseTypePostgres:
		return `
CREATE TABLE IF NOT EXISTS attachments (
	id TEXT PRIMARY KEY,
	filename TEXT NOT NULL,
	content_type TEXT NOT NULL,
	byte_size BIGINT NOT NULL,
	key TEXT NOT NULL UNIQUE,
	checksum TEXT NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL,
	deleted_at TIMESTAMP WITH TIME ZONE NULL
);
CREATE INDEX IF NOT EXISTS idx_attachments_created_at ON attachments(created_at);
CREATE INDEX IF NOT EXISTS idx_attachments_deleted_at ON attachments(deleted_at);
		`
	default:
		return ""
	}
}
