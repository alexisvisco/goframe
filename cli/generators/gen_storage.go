package generators

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/configuration"
)

type StorageGenerator struct {
	g  *Generator
	db *DatabaseGenerator
}

func (p *StorageGenerator) Generate() error {
	files := []FileConfig{
		p.createStorageProvider("internal/providers/storage.go"),
	}

	for _, file := range files {
		if err := p.g.GenerateFile(file); err != nil {
			return fmt.Errorf("failed to create storage file %s: %w", file.Path, err)
		}
	}

	if err := p.GenerateStorageMigration(); err != nil {
		return fmt.Errorf("failed to generate storage migration: %w", err)
	}

	return nil
}

// createStorageProvider creates the FileConfig for the storage provider
func (p *StorageGenerator) createStorageProvider(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.ProvidersProvideStorageGo,
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(p.g.GoModuleName, "config"), "config")
		},
		Category:  CategoryStorage,
		Condition: true,
	}
}

func (p *StorageGenerator) GenerateStorageMigration() error {
	up := p.getStorageMigrationSQL()

	return p.db.CreateMigration(CreateMigrationParams{
		Sql:  true,
		Name: "storage",
		At:   time.Now(),
		Up:   up,
		Down: "drop table if exists attachments;",
	})
}

// getStorageMigrationSQL returns the SQL for creating the attachments table based on database type
func (p *StorageGenerator) getStorageMigrationSQL() string {
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
