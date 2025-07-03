package dbutil

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// DB returns the gorm.DB associated with the provided context if any or
// the provided defaultDB otherwise. The returned DB carries the context
// so all executed queries are tied to it.
func DB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if ctxDB, ok := ctx.Value("db").(*gorm.DB); ok {
		return ctxDB.Session(&gorm.Session{Context: ctx}).Debug()
	}
	return defaultDB.Session(&gorm.Session{Context: ctx}).Debug()
}

// WithDB stores a *gorm.DB in the context so it can be retrieved later with DB.
func WithDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, "db", db)
}

// Transaction wraps the given handler in a gorm transaction bound to the context.
func Transaction(ctx context.Context, defaultDB *gorm.DB, h func(ctx context.Context) error) error {
	return DB(ctx, defaultDB).Transaction(func(tx *gorm.DB) error {
		newCtx := WithDB(ctx, tx)
		return h(newCtx)
	})
}

// IsUniqueConstraintError checks whether the error represents a violation of the
// given unique index. Only PostgreSQL errors are detected for now.
func IsUniqueConstraintError(err error, indexName string) bool {
	if err == nil {
		return false
	}

	matchErr := fmt.Sprintf(`ERROR: duplicate key value violates unique constraint "%s" (SQLSTATE 23505)`, indexName)
	return strings.Contains(err.Error(), matchErr)
}
