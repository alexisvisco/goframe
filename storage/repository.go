package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/coretypes"
)

type (
	DefaultRepository struct {
		db     *sql.DB
		driver string
	}
)

func NewRepository(cfg configuration.Database) func(db *sql.DB) (*DefaultRepository, error) {
	return func(db *sql.DB) (*DefaultRepository, error) {
		repo := &DefaultRepository{
			db:     db,
			driver: cfg.Type,
		}

		// Create tables if they don't exist
		if err := repo.createTables(); err != nil {
			return nil, fmt.Errorf("failed to create tables: %w", err)
		}

		return repo, nil
	}
}

// createTables creates the attachments table with appropriate SQL for each database
func (r *DefaultRepository) createTables() error {
	var createTableSQL string

	switch r.driver {
	case "sqlite3":
		createTableSQL = `
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
	case "postgres":
		createTableSQL = `
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
		return fmt.Errorf("unsupported database driver: %s", r.driver)
	}

	// Execute each statement separately for better error handling
	statements := strings.Split(createTableSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := r.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement '%s': %w", stmt, err)
		}
	}

	return nil
}

// SaveAttachment saves an attachment record to the database
func (r *DefaultRepository) SaveAttachment(ctx context.Context, attachment *coretypes.Attachment) error {
	query := `
		INSERT INTO attachments (id, filename, content_type, byte_size, key, checksum, created_at, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Adjust query syntax for different databases
	switch r.driver {
	case "postgres":
		query = `
			INSERT INTO attachments (id, filename, content_type, byte_size, key, checksum, created_at, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		attachment.ID,
		attachment.Filename,
		attachment.ContentType,
		attachment.ByteSize,
		attachment.Key,
		attachment.Checksum,
		attachment.CreatedAt,
		attachment.DeletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save attachment: %w", err)
	}

	return nil
}

// GetAttachment retrieves an attachment by ID
func (r *DefaultRepository) GetAttachment(ctx context.Context, id string) (*coretypes.Attachment, error) {
	query := `
		SELECT id, filename, content_type, byte_size, key, checksum, created_at, deleted_at
		FROM attachments
		WHERE id = ? AND deleted_at IS NULL
	`

	if r.driver == "postgres" {
		query = strings.ReplaceAll(query, "?", "$1")
	}

	row := r.db.QueryRowContext(ctx, query, id)

	var attachment coretypes.Attachment
	var deletedAt sql.NullTime

	err := row.Scan(
		&attachment.ID,
		&attachment.Filename,
		&attachment.ContentType,
		&attachment.ByteSize,
		&attachment.Key,
		&attachment.Checksum,
		&attachment.CreatedAt,
		&deletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Attachment not found
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	if deletedAt.Valid {
		attachment.DeletedAt = &deletedAt.Time
	}

	return &attachment, nil
}

// DeleteAttachment marks an attachment as deleted and executes cleanup function
func (r *DefaultRepository) DeleteAttachment(ctx context.Context, id string, cleanup func() error) error {
	// Start a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Mark as deleted in database
	query := `
		UPDATE attachments 
		SET deleted_at = ? 
		WHERE id = ? AND deleted_at IS NULL
	`

	if r.driver == "postgres" {
		query = `
			UPDATE attachments 
			SET deleted_at = $1 
			WHERE id = $2 AND deleted_at IS NULL
		`
	}

	now := time.Now()
	result, err := tx.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark attachment as deleted: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("attachment not found or already deleted")
	}

	// Execute cleanup function (delete from S3)
	if cleanup != nil {
		if err := cleanup(); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
