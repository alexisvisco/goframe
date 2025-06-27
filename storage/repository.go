package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/alexisvisco/goframe/core/contracts"
	"github.com/alexisvisco/goframe/core/coretypes"
	"github.com/alexisvisco/goframe/db/dbutil"
	"gorm.io/gorm"
)

type (
	DefaultRepository struct {
		db     *gorm.DB
		driver configuration.DatabaseType
	}
)

var _ contracts.StorageRepository = (*DefaultRepository)(nil)

func NewRepository(cfg configuration.Database, db *gorm.DB) *DefaultRepository {
	repo := &DefaultRepository{
		db:     db,
		driver: cfg.Type,
	}
	return repo

}

// Migration returns the SQL statements for creating and dropping the attachments table
func (r *DefaultRepository) Migration(databaseType configuration.DatabaseType) (string, string, error) {
	switch databaseType {
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
		`, "DROP TABLE IF EXISTS attachments;", nil
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
		`, "DROP TABLE IF EXISTS attachments;", nil

	default:
		return "", "", fmt.Errorf("unsupported database driver: %s", r.driver)
	}
}

// SaveAttachment saves an attachment record to the database
func (r *DefaultRepository) SaveAttachment(ctx context.Context, attachment *coretypes.Attachment) error {
	err := dbutil.DB(ctx, r.db).Create(attachment).Error
	if err != nil {
		return fmt.Errorf("failed to save attachment: %w", err)
	}

	return nil
}

// GetAttachment retrieves an attachment by ID
func (r *DefaultRepository) GetAttachment(ctx context.Context, id string) (*coretypes.Attachment, error) {
	var attachment coretypes.Attachment
	err := dbutil.DB(ctx, r.db).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&attachment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get attachment: %w", err)
	}

	return &attachment, nil
}

// DeleteAttachment marks an attachment as deleted and executes cleanup function
func (r *DefaultRepository) DeleteAttachment(ctx context.Context, id string, cleanup func() error) error {
	now := time.Now()
	return dbutil.Transaction(ctx, r.db, func(txCtx context.Context) error {
		result := dbutil.DB(txCtx, r.db).
			Model(&coretypes.Attachment{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Update("deleted_at", now)
		if result.Error != nil {
			return fmt.Errorf("failed to mark attachment as deleted: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("attachment not found or already deleted")
		}

		if cleanup != nil {
			if err := cleanup(); err != nil {
				return fmt.Errorf("cleanup failed: %w", err)
			}
		}

		return nil
	})
}
