package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DB represents a database connection that can execute SQL statements with context.
// Both *sql.DB and *sql.Tx implement this interface when using context methods.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// Migration represents a database migration with up and down operations.
type Migration interface {
	Up(context.Context, DB) error
	Down(context.Context, DB) error
	Version() (name string, at time.Time)
}

// MigrationWithTx is an optional interface for migrations that want to control transaction usage.
type MigrationWithTx interface {
	// UseTx returns whether this migration should run in a transaction for the given operation.
	// kind is either "up" or "down". Only applicable when globalTransaction is false.
	UseTx(kind string) bool
}

// Option configures migration execution.
type Option func(*options)

// options holds migration execution configuration.
type options struct {
	globalTransaction bool
	timeout           time.Duration
	logger            *slog.Logger
}

// GlobalTransactionOption configures whether all migrations run in a single transaction.
func GlobalTransactionOption(enabled bool) Option {
	return func(c *options) {
		c.globalTransaction = enabled
	}
}

// TimeoutOption configures the timeout for database operations.
func TimeoutOption(timeout time.Duration) Option {
	return func(c *options) {
		c.timeout = timeout
	}
}

// LoggerOption configures the logger for migration operations.
// Pass nil to disable logging.
func LoggerOption(logger *slog.Logger) Option {
	return func(c *options) {
		c.logger = logger
	}
}

// Migrator handles database migrations.
type Migrator struct {
	db *sql.DB
}

// New creates a new Migrator instance.
func New(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// Up executes pending migrations in chronological order.
func (m *Migrator) Up(ctx context.Context, migrations []Migration, opts ...Option) error {
	cfg := &options{
		timeout: 15 * time.Second,                            // default timeout
		logger:  slog.Default().With("component", "migrate"), // default logger
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "starting migration up process")
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	if err := m.ensureTable(ctx); err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to ensure migrations table", "error", err)
		}
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to get applied migrations", "error", err)
		}
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	pending := m.filterPending(migrations, applied)

	if len(pending) == 0 {
		if cfg.logger != nil {
			cfg.logger.InfoContext(ctx, "no pending migrations found")
		}
		return nil
	}

	if cfg.globalTransaction {
		return m.runInGlobalTransaction(ctx, pending, true, cfg)
	}

	for _, migration := range pending {
		if err := m.runUp(ctx, migration, cfg); err != nil {
			name, at := migration.Version()
			if cfg.logger != nil {
				cfg.logger.ErrorContext(ctx, "failed to run migration",
					"migration_version", formatVersion(name, at),
					"migration_name", name,
					"error", err)
			}
			return fmt.Errorf("failed to run migration %s_%s: %w",
				at.UTC().Format("20060102150405"), name, err)
		}
	}

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "migration up process completed successfully",
			"applied_count", len(pending))
	}

	return nil
}

// Down executes applied migrations in reverse chronological order.
func (m *Migrator) Down(ctx context.Context, migrations []Migration, opts ...Option) error {
	cfg := &options{
		timeout: 15 * time.Second, // default timeout
		logger:  slog.Default(),   // default logger
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "starting migration down process",
			"total_migrations", len(migrations))
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	if err := m.ensureTable(ctx); err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to ensure migrations table",
				"error", err)
		}
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to get applied migrations",
				"error", err)
		}
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Sort migrations by timestamp descending for rollback
	sorted := make([]Migration, 0, len(migrations))
	for _, migration := range migrations {
		name, at := migration.Version()
		version := formatVersion(name, at)
		if applied[version] {
			sorted = append(sorted, migration)
		}
	}

	sort.Slice(sorted, func(i, j int) bool {
		_, atI := sorted[i].Version()
		_, atJ := sorted[j].Version()
		return atI.After(atJ)
	})

	if len(sorted) == 0 {
		if cfg.logger != nil {
			cfg.logger.InfoContext(ctx, "no migrations to rollback")
		}
		return nil
	}

	if cfg.globalTransaction {
		return m.runInGlobalTransaction(ctx, sorted, false, cfg)
	}

	for _, migration := range sorted {
		if err := m.runDown(ctx, migration, cfg); err != nil {
			name, at := migration.Version()
			if cfg.logger != nil {
				cfg.logger.ErrorContext(ctx, "failed to rollback migration",
					"migration_version", formatVersion(name, at),
					"migration_name", name,
					"error", err)
			}
			return fmt.Errorf("failed to rollback migration %s_%s: %w",
				at.UTC().Format("20060102150405"), name, err)
		}
	}

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "migration down process completed successfully",
			"rollback_count", len(sorted))
	}

	return nil
}

// runInGlobalTransaction executes all migrations in a single transaction.
func (m *Migrator) runInGlobalTransaction(ctx context.Context, migrations []Migration, isUp bool, cfg *options) error {
	if cfg.logger != nil {
		action := "up"
		if !isUp {
			action = "down"
		}
		cfg.logger.InfoContext(ctx, "starting global transaction for migrations",
			"action", action,
			"migration_count", len(migrations))
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to begin global transaction", "error", err)
		}
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, migration := range migrations {
		name, at := migration.Version()
		version := formatVersion(name, at)

		if cfg.logger != nil {
			action := "applying"
			if !isUp {
				action = "rolling back"
			}
			cfg.logger.InfoContext(ctx, fmt.Sprintf("migration %s", action),
				"migration_version", version,
				"migration_name", name)
		}

		var execErr error
		if isUp {
			execErr = migration.Up(ctx, tx)
		} else {
			execErr = migration.Down(ctx, tx)
		}

		if execErr != nil {
			action := "run"
			if !isUp {
				action = "rollback"
			}
			if cfg.logger != nil {
				cfg.logger.ErrorContext(ctx, fmt.Sprintf("failed to %s migration in global transaction", action),
					"migration_version", version,
					"migration_name", name,
					"error", execErr)
			}
			return fmt.Errorf("failed to %s migration %s_%s: %w",
				action, at.UTC().Format("20060102150405"), name, execErr)
		}

		// Update migration table
		if isUp {
			_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
		} else {
			_, err = tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = $1", version)
		}

		if err != nil {
			if cfg.logger != nil {
				cfg.logger.ErrorContext(ctx, "failed to update migration table in global transaction",
					"migration_version", version, "error", err)
			}
			return fmt.Errorf("failed to update migration table for %s: %w", version, err)
		}

		if cfg.logger != nil {
			action := "applied"
			if !isUp {
				action = "rolled back"
			}
			cfg.logger.InfoContext(ctx, fmt.Sprintf("migration %s successfully", action),
				"migration_version", version,
				"migration_name", name)
		}
	}

	if err := tx.Commit(); err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to commit global transaction", "error", err)
		}
		return fmt.Errorf("failed to commit global transaction: %w", err)
	}

	if cfg.logger != nil {
		action := "up"
		if !isUp {
			action = "down"
		}
		cfg.logger.InfoContext(ctx, "global transaction completed successfully",
			"action", action,
			"migration_count", len(migrations))
	}

	return nil
}

// ensureTable creates the schema_migrations table if it doesn't exist.
func (m *Migrator) ensureTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR NOT NULL PRIMARY KEY
	)`

	_, err := m.db.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations returns a set of applied migration versions.
func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	rows, err := m.db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// filterPending returns migrations that haven't been applied yet, sorted by timestamp.
func (m *Migrator) filterPending(migrations []Migration, applied map[string]bool) []Migration {
	var pending []Migration

	for _, migration := range migrations {
		name, at := migration.Version()
		version := formatVersion(name, at)

		if !applied[version] {
			pending = append(pending, migration)
		}
	}

	// Sort by timestamp ascending
	sort.Slice(pending, func(i, j int) bool {
		_, atI := pending[i].Version()
		_, atJ := pending[j].Version()
		return atI.Before(atJ)
	})

	return pending
}

// runUp executes a migration and records it as applied.
func (m *Migrator) runUp(ctx context.Context, migration Migration, cfg *options) error {
	name, at := migration.Version()
	version := formatVersion(name, at)

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "applying migration",
			"migration_version", version,
			"migration_name", name)
	}

	if useTx(migration, "up") {
		return m.runInTransaction(ctx, migration, true, cfg)
	}

	if err := migration.Up(ctx, m.db); err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to apply migration",
				"migration_version", version,
				"migration_name", name,
				"error", err)
		}
		return err
	}

	_, err := m.db.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
	if err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to record migration as applied",
				"migration_version", version,
				"error", err)
		}
		return err
	}

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "migration applied successfully",
			"migration_version", version,
			"migration_name", name)
	}

	return nil
}

// runDown executes a migration rollback and removes it from applied migrations.
func (m *Migrator) runDown(ctx context.Context, migration Migration, cfg *options) error {
	name, at := migration.Version()
	version := formatVersion(name, at)

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "rolling back migration",
			"migration_version", version,
			"migration_name", name)
	}

	if useTx(migration, "down") {
		return m.runInTransaction(ctx, migration, false, cfg)
	}

	if err := migration.Down(ctx, m.db); err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to rollback migration",
				"migration_version", version,
				"migration_name", name,
				"error", err)
		}
		return err
	}

	_, err := m.db.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = $1", version)
	if err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to remove migration from applied list",
				"migration_version", version,
				"error", err)
		}
		return err
	}

	if cfg.logger != nil {
		cfg.logger.InfoContext(ctx, "migration rolled back successfully",
			"migration_version", version,
			"migration_name", name)
	}

	return nil
}

// runInTransaction executes a single migration in a transaction.
func (m *Migrator) runInTransaction(ctx context.Context, migration Migration, isUp bool, cfg *options) error {
	name, at := migration.Version()
	version := formatVersion(name, at)

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		if cfg.logger != nil {
			cfg.logger.ErrorContext(ctx, "failed to begin transaction for migration",
				"migration_version", version,
				"error", err)
		}
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if isUp {
		if err := migration.Up(ctx, tx); err != nil {
			return err
		}
	} else {
		if err := migration.Down(ctx, tx); err != nil {
			return err
		}
	}

	if isUp {
		_, err = tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
	} else {
		_, err = tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE version = $1", version)
	}

	if err != nil {
		return err
	}

	return tx.Commit()
}

// formatVersion creates a version string in the format {timestamp}_{name}.
func formatVersion(name string, at time.Time) string {
	timestamp := at.UTC().Format("20060102150405")
	return fmt.Sprintf("%s_%s", timestamp, name)
}

// useTx checks if a migration should use transactions for the given operation.
// Returns true by default, unless the migration implements MigrationWithTx and returns false.
func useTx(migration Migration, kind string) bool {
	if txMigration, ok := migration.(MigrationWithTx); ok {
		return txMigration.UseTx(kind)
	}
	return true // default to using transactions
}

// MigrationFromSQL creates a Migration from a SQL file using the provided filesystem.
// Expected file format: single file with -- migrate:up and -- migrate:down separators.
// Optional transaction control: -- migrate:up transaction=false, -- migrate:down transaction=true
func MigrationFromSQL(fsys fs.FS, filename string) Migration {
	content, err := fs.ReadFile(fsys, filename)
	if err != nil {
		panic(fmt.Errorf("failed to read file %s: %w", filename, err))
	}

	name, timestamp, err := parseSQLFileName(filename)
	if err != nil {
		panic(fmt.Errorf("invalid file name %s: %w", filename, err))
	}

	upSQL, downSQL, upUseTx, downUseTx, err := parseSQLContent(string(content))
	if err != nil {
		panic(fmt.Errorf("failed to parse SQL content in %s: %w", filename, err))
	}

	return &sqlMigration{
		name:      name,
		at:        timestamp,
		upSQL:     upSQL,
		downSQL:   downSQL,
		upUseTx:   upUseTx,
		downUseTx: downUseTx,
	}
}

// parseSQLFileName extracts migration information from a SQL file name.
// Expected format: {timestamp}_{name}.sql
func parseSQLFileName(filename string) (name string, timestamp time.Time, err error) {
	if !strings.HasSuffix(filename, ".sql") {
		return "", time.Time{}, fmt.Errorf("not a SQL file")
	}

	filename = filepath.Base(filename)

	base := strings.TrimSuffix(filename, ".sql")
	underscoreIndex := strings.Index(base, "_")
	if underscoreIndex == -1 || underscoreIndex == 0 {
		return "", time.Time{}, fmt.Errorf("invalid file format, expected {timestamp}_{name}.sql")
	}

	timestampStr := base[:underscoreIndex]
	name = base[underscoreIndex+1:]

	timestamp, err = time.Parse("20060102150405", timestampStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	return name, timestamp, nil
}

// parseSQLContent parses SQL file content with -- migrate:up and -- migrate:down separators.
func parseSQLContent(content string) (upSQL, downSQL string, upUseTx, downUseTx bool, err error) {
	upUseTx = true   // default to using transactions
	downUseTx = true // default to using transactions

	// Regular expressions for parsing separators with optional transaction parameter
	upPattern := regexp.MustCompile(`(?i)^--\s*migrate:up(?:\s+transaction=(\w+))?\s*$`)
	downPattern := regexp.MustCompile(`(?i)^--\s*migrate:down(?:\s+transaction=(\w+))?\s*$`)

	lines := strings.Split(content, "\n")
	var currentSection string
	var upLines, downLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := upPattern.FindStringSubmatch(trimmed); matches != nil {
			currentSection = "up"
			if len(matches) > 1 && matches[1] != "" {
				upUseTx = parseBool(matches[1], true)
			}
			continue
		}

		if matches := downPattern.FindStringSubmatch(trimmed); matches != nil {
			currentSection = "down"
			if len(matches) > 1 && matches[1] != "" {
				downUseTx = parseBool(matches[1], true)
			}
			continue
		}

		switch currentSection {
		case "up":
			upLines = append(upLines, line)
		case "down":
			downLines = append(downLines, line)
		}
	}

	if len(upLines) == 0 {
		return "", "", upUseTx, downUseTx, fmt.Errorf("no -- migrate:up section found")
	}

	upSQL = strings.TrimSpace(strings.Join(upLines, "\n"))
	downSQL = strings.TrimSpace(strings.Join(downLines, "\n"))

	return upSQL, downSQL, upUseTx, downUseTx, nil
}

// parseBool parses a string to boolean with a default value.
func parseBool(s string, defaultValue bool) bool {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return defaultValue
}

// sqlMigration implements both Migration and MigrationWithTx interfaces for SQL file migrations.
type sqlMigration struct {
	name      string
	at        time.Time
	upSQL     string
	downSQL   string
	upUseTx   bool
	downUseTx bool
}

// Up executes the SQL migration.
func (s *sqlMigration) Up(ctx context.Context, db DB) error {
	if s.upSQL == "" {
		return fmt.Errorf("no up SQL found for migration %s", s.name)
	}
	_, err := db.ExecContext(ctx, s.upSQL)
	return err
}

// Down executes the SQL migration rollback.
func (s *sqlMigration) Down(ctx context.Context, db DB) error {
	if s.downSQL == "" {
		return fmt.Errorf("no down SQL found for migration %s", s.name)
	}
	_, err := db.ExecContext(ctx, s.downSQL)
	return err
}

// Version returns the migration name and timestamp.
func (s *sqlMigration) Version() (string, time.Time) {
	return s.name, s.at
}

// UseTx implements MigrationWithTx interface.
func (s *sqlMigration) UseTx(kind string) bool {
	switch kind {
	case "up":
		return s.upUseTx
	case "down":
		return s.downUseTx
	default:
		return true // default to using transactions for unknown kinds
	}
}
