package gencache

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

type CacheGenerator struct {
	Gen   *generators.Generator
	DBGen *gendb.DatabaseGenerator
}

//go:embed templates
var fs embed.FS

func (c *CacheGenerator) Generate() error {
	files := []generators.FileConfig{
		c.createProvider("internal/provide/provide_cache.go"),
	}
	files = append(files, c.CreateMigrations()...)
	return c.Gen.GenerateFiles(files)
}

func (c *CacheGenerator) createProvider(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/provide_cache.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport(filepath.Join(c.Gen.GoModuleName, "config"), "config").
				WithImport("github.com/alexisvisco/goframe/cache", "cache").
				WithImport("github.com/alexisvisco/goframe/core/contracts", "contracts").
				WithImport("gorm.io/gorm", "gorm")
		},
	}
}

func (p *CacheGenerator) CreateMigrations() []generators.FileConfig {
	up := p.getSQLMigration()
	down := p.getSQLMigrationDown()

	return p.DBGen.CreateMigration(gendb.CreateMigrationParams{
		Sql:  true,
		Name: "cache",
		At:   time.Now(),
		Up:   up,
		Down: down,
	})
}

func (p *CacheGenerator) getSQLMigration() string {
	switch p.Gen.DatabaseType {
	case configuration.DatabaseTypePostgres:
		return `
CREATE TABLE IF NOT EXISTS cache_entries (
    key TEXT PRIMARY KEY,
    value BYTEA NOT NULL,
    expires_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_cache_entries_expires_at ON cache_entries(expires_at);

-- Create function to notify cache events
CREATE OR REPLACE FUNCTION notify_cache_event() RETURNS TRIGGER AS $$
DECLARE
    payload json;
    operation_type text;
BEGIN
    -- Determine operation type
    IF TG_OP = 'DELETE' THEN
        operation_type := 'delete';
        payload := json_build_object(
            'type', operation_type,
            'key', OLD.key,
            'value', null
        );
    ELSIF TG_OP = 'UPDATE' THEN
        operation_type := 'update';
        payload := json_build_object(
            'type', operation_type,
            'key', NEW.key,
            'value', NEW.value::text
        );
    ELSIF TG_OP = 'INSERT' THEN
        operation_type := 'put';
        payload := json_build_object(
            'type', operation_type,
            'key', NEW.key,
            'value', NEW.value::text
        );
    END IF;
    
    -- Send notification
    PERFORM pg_notify('cache_events', payload::text);
    
    -- Return appropriate record
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    ELSE
        RETURN NEW;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create triggers on cache_entries table
CREATE TRIGGER cache_notify_insert
    AFTER INSERT ON cache_entries
    FOR EACH ROW EXECUTE FUNCTION notify_cache_event();

CREATE TRIGGER cache_notify_update
    AFTER UPDATE ON cache_entries
    FOR EACH ROW EXECUTE FUNCTION notify_cache_event();

CREATE TRIGGER cache_notify_delete
    AFTER DELETE ON cache_entries
    FOR EACH ROW EXECUTE FUNCTION notify_cache_event();
`
	default:
		return ""
	}
}

func (p *CacheGenerator) getSQLMigrationDown() string {
	switch p.Gen.DatabaseType {
	case configuration.DatabaseTypePostgres:
		return `
-- Drop cache_entries table
DROP TABLE IF EXISTS cache_entries;
		
-- Drop triggers
DROP TRIGGER IF EXISTS cache_notify_insert ON cache_entries;
DROP TRIGGER IF EXISTS cache_notify_update ON cache_entries;
DROP TRIGGER IF EXISTS cache_notify_delete ON cache_entries;

-- Drop function
DROP FUNCTION IF EXISTS notify_cache_event();
`
	default:
		return ""
	}
}
