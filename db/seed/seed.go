package seed

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/alexisvisco/goframe/db/dbutil"
)

// Seed represents a database seed.
type Seed interface {
	Run(context.Context) error
	Version() (name string, at time.Time)
}

// SeedWithTx optionally controls transaction usage for a seed.
type SeedWithTx interface {
	UseTx() bool
}

// Option configures seed execution.
type Option func(*options)

type options struct {
	globalTransaction bool
	timeout           time.Duration
	logger            *slog.Logger
}

// GlobalTransactionOption configures whether all seeds run in a single transaction.
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

// LoggerOption sets the logger used during seeding.
func LoggerOption(logger *slog.Logger) Option {
	return func(c *options) {
		c.logger = logger
	}
}

// Seeder handles database seeds.
type Seeder struct {
	db *gorm.DB
}

// New creates a new Seeder.
func New(db *gorm.DB) *Seeder {
	return &Seeder{db: db}
}

// Up executes all pending seeds.
func (s *Seeder) Up(ctx context.Context, seeds []Seed, opts ...Option) error {
	cfg := &options{
		timeout: 15 * time.Second,
		logger:  slog.Default().With("component", "seed"),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	if err := s.ensureTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure seeds table: %w", err)
	}

	applied, err := s.getAppliedSeeds(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied seeds: %w", err)
	}

	pending := s.filterPending(seeds, applied)
	if len(pending) == 0 {
		if cfg.logger != nil {
			cfg.logger.InfoContext(ctx, "no pending seeds found")
		}
		return nil
	}

	if cfg.globalTransaction {
		tx := s.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			return tx.Error
		}
		txCtx := dbutil.WithDB(ctx, tx)
		for _, seed := range pending {
			name, at := seed.Version()
			version := formatVersion(name, at)
			if cfg.logger != nil {
				cfg.logger.InfoContext(ctx, "running seed", "seed_version", version, "seed_name", name)
			}
			if err := seed.Run(txCtx); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to run seed %s: %w", version, err)
			}
			if err := tx.Exec("INSERT INTO schema_seeds (version) VALUES ($1)", version).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to record seed %s: %w", version, err)
			}
		}
		return tx.Commit().Error
	}

	for _, seed := range pending {
		name, at := seed.Version()
		version := formatVersion(name, at)
		if cfg.logger != nil {
			cfg.logger.InfoContext(ctx, "running seed", "seed_version", version, "seed_name", name)
		}
		var err error
		if useTx(seed) {
			err = dbutil.Transaction(ctx, s.db, func(txCtx context.Context) error {
				if err := seed.Run(txCtx); err != nil {
					return err
				}
				return dbutil.DB(txCtx, nil).Exec("INSERT INTO schema_seeds (version) VALUES ($1)", version).Error
			})
		} else {
			runCtx := dbutil.WithDB(ctx, s.db)
			if err = seed.Run(runCtx); err == nil {
				err = s.db.WithContext(ctx).Exec("INSERT INTO schema_seeds (version) VALUES ($1)", version).Error
			}
		}
		if err != nil {
			return fmt.Errorf("failed to run seed %s: %w", version, err)
		}
	}

	return nil
}

// Applied returns all applied seed versions sorted chronologically.
func (s *Seeder) Applied(ctx context.Context) ([]string, error) {
	rows, err := s.db.WithContext(ctx).Raw("SELECT version FROM schema_seeds ORDER BY version ASC").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		list = append(list, version)
	}
	return list, rows.Err()
}

func (s *Seeder) ensureTable(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS schema_seeds (
        version VARCHAR NOT NULL PRIMARY KEY
    )`
	return s.db.WithContext(ctx).Exec(query).Error
}

func (s *Seeder) getAppliedSeeds(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.WithContext(ctx).Raw("SELECT version FROM schema_seeds").Rows()
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

func (s *Seeder) filterPending(seeds []Seed, applied map[string]bool) []Seed {
	sort.Slice(seeds, func(i, j int) bool {
		_, atI := seeds[i].Version()
		_, atJ := seeds[j].Version()
		return atI.Before(atJ)
	})
	var pending []Seed
	for _, seed := range seeds {
		name, at := seed.Version()
		version := formatVersion(name, at)
		if !applied[version] {
			pending = append(pending, seed)
		}
	}
	return pending
}

func formatVersion(name string, at time.Time) string {
	timestamp := at.UTC().Format("20060102150405")
	return fmt.Sprintf("%s_%s", timestamp, name)
}

func useTx(s Seed) bool {
	if txSeed, ok := s.(SeedWithTx); ok {
		return txSeed.UseTx()
	}
	return true
}

// SeedFromSQL creates a Seed from a SQL file.
func SeedFromSQL(fsys fs.FS, filename string) Seed {
	content, err := fs.ReadFile(fsys, filename)
	if err != nil {
		panic(fmt.Errorf("failed to read file %s: %w", filename, err))
	}

	name, timestamp, err := parseSQLFileName(filename)
	if err != nil {
		panic(fmt.Errorf("invalid file name %s: %w", filename, err))
	}

	sql, useTx, err := parseSQLContent(string(content))
	if err != nil {
		panic(fmt.Errorf("failed to parse SQL content in %s: %w", filename, err))
	}

	return &sqlSeed{
		name:  name,
		at:    timestamp,
		sql:   sql,
		useTx: useTx,
	}
}

// parseSQLFileName expects format {timestamp}_{name}.sql
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

// parseSQLContent reads SQL file content and optional transaction directive.
func parseSQLContent(content string) (sql string, useTx bool, err error) {
	useTx = true
	lines := strings.Split(content, "\n")
	pattern := regexp.MustCompile(`(?i)^--\s*seed:transaction=(\w+)\s*$`)
	if len(lines) > 0 && pattern.MatchString(strings.TrimSpace(lines[0])) {
		matches := pattern.FindStringSubmatch(strings.TrimSpace(lines[0]))
		if len(matches) > 1 {
			useTx = parseBool(matches[1], true)
		}
		lines = lines[1:]
	}
	sql = strings.TrimSpace(strings.Join(lines, "\n"))
	if sql == "" {
		return "", useTx, fmt.Errorf("empty SQL content")
	}
	return sql, useTx, nil
}

func parseBool(s string, defaultValue bool) bool {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return defaultValue
}

type sqlSeed struct {
	name  string
	at    time.Time
	sql   string
	useTx bool
}

func (s *sqlSeed) Run(ctx context.Context) error {
	return dbutil.DB(ctx, nil).Exec(s.sql).Error
}

func (s *sqlSeed) Version() (string, time.Time) {
	return s.name, s.at
}

func (s *sqlSeed) UseTx() bool {
	return s.useTx
}
