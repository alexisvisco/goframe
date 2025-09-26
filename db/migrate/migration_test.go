package migrate

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test migration implementations
type testMigration struct {
	name       string
	timestamp  time.Time
	upCalled   bool
	downCalled bool
	onDown     func(string) // callback for tracking rollback order
}

func (t *testMigration) Up(ctx context.Context) error {
	t.upCalled = true
	return nil
}

func (t *testMigration) Down(ctx context.Context) error {
	t.downCalled = true
	if t.onDown != nil {
		t.onDown(t.name)
	}
	return nil
}

func (t *testMigration) Version() (string, time.Time) {
	return t.name, t.timestamp
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

func createTestMigrations() []Migration {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return []Migration{
		&testMigration{name: "first_migration", timestamp: baseTime},
		&testMigration{name: "second_migration", timestamp: baseTime.Add(time.Hour)},
		&testMigration{name: "third_migration", timestamp: baseTime.Add(2 * time.Hour)},
		&testMigration{name: "fourth_migration", timestamp: baseTime.Add(3 * time.Hour)},
		&testMigration{name: "fifth_migration", timestamp: baseTime.Add(4 * time.Hour)},
	}
}

func TestMigratorDownWithSteps(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()
	migrations := createTestMigrations()

	// First, apply all migrations
	err := migrator.Up(ctx, migrations)
	require.NoError(t, err)

	// Verify all migrations were applied
	applied, err := migrator.getAppliedMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 5)

	t.Run("rollback 2 steps", func(t *testing.T) {
		// Reset the test migrations down called flags
		for _, m := range migrations {
			tm := m.(*testMigration)
			tm.downCalled = false
		}

		// Rollback 2 steps (should rollback the last 2 migrations)
		err := migrator.Down(ctx, migrations, 2)
		require.NoError(t, err)

		// Check which migrations were rolled back (last 2 in reverse chronological order)
		// fifth_migration and fourth_migration should have been rolled back
		assert.False(t, migrations[0].(*testMigration).downCalled) // first_migration
		assert.False(t, migrations[1].(*testMigration).downCalled) // second_migration
		assert.False(t, migrations[2].(*testMigration).downCalled) // third_migration
		assert.True(t, migrations[3].(*testMigration).downCalled)  // fourth_migration
		assert.True(t, migrations[4].(*testMigration).downCalled)  // fifth_migration

		// Verify only 3 migrations remain applied
		applied, err := migrator.getAppliedMigrations(ctx)
		require.NoError(t, err)
		assert.Len(t, applied, 3)
	})
}

func TestMigratorDownAll(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()
	migrations := createTestMigrations()

	// Apply all migrations
	err := migrator.Up(ctx, migrations)
	require.NoError(t, err)

	// Rollback all migrations using DownAll
	err = migrator.DownAll(ctx, migrations)
	require.NoError(t, err)

	// Verify all migrations were rolled back
	applied, err := migrator.getAppliedMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 0)
}

func TestMigratorDownSteps(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()
	migrations := createTestMigrations()

	// Apply all migrations
	err := migrator.Up(ctx, migrations)
	require.NoError(t, err)

	// Rollback 3 steps using DownSteps
	err = migrator.DownSteps(ctx, migrations, 3)
	require.NoError(t, err)

	// Verify only 2 migrations remain applied
	applied, err := migrator.getAppliedMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 2)
}

func TestMigratorDownWithStepsExceedsAvailable(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()
	migrations := createTestMigrations()

	// Apply only first 3 migrations
	err := migrator.Up(ctx, migrations[:3])
	require.NoError(t, err)

	// Try to rollback 5 steps when only 3 are applied
	// Should rollback all available migrations
	err = migrator.Down(ctx, migrations, 5)
	require.NoError(t, err)

	// Verify all migrations were rolled back
	applied, err := migrator.getAppliedMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 0)
}

func TestMigratorDownWithZeroSteps(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()
	migrations := createTestMigrations()

	// Apply all migrations
	err := migrator.Up(ctx, migrations)
	require.NoError(t, err)

	// Rollback with 0 steps (should rollback all)
	err = migrator.Down(ctx, migrations, 0)
	require.NoError(t, err)

	// Verify all migrations were rolled back
	applied, err := migrator.getAppliedMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 0)
}

func TestMigratorDownOrder(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()

	// Create migrations with specific timestamps to test ordering
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var rollbackOrder []string
	migrations := []Migration{
		&testMigration{
			name:      "migration_1",
			timestamp: baseTime,
			onDown:    func(name string) { rollbackOrder = append(rollbackOrder, name) },
		},
		&testMigration{
			name:      "migration_2",
			timestamp: baseTime.Add(time.Hour),
			onDown:    func(name string) { rollbackOrder = append(rollbackOrder, name) },
		},
		&testMigration{
			name:      "migration_3",
			timestamp: baseTime.Add(2 * time.Hour),
			onDown:    func(name string) { rollbackOrder = append(rollbackOrder, name) },
		},
	}

	// Apply all migrations
	err := migrator.Up(ctx, migrations)
	require.NoError(t, err)

	// Rollback 2 steps
	err = migrator.Down(ctx, migrations, 2)
	require.NoError(t, err)

	// Verify rollback order is reverse chronological (newest first)
	expectedOrder := []string{"migration_3", "migration_2"}
	assert.Equal(t, expectedOrder, rollbackOrder)
}

func TestMigratorDownNoAppliedMigrations(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()
	migrations := createTestMigrations()

	// Try to rollback when no migrations are applied
	err := migrator.Down(ctx, migrations, 2)
	require.NoError(t, err)

	// Should complete without error
	applied, err := migrator.getAppliedMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 0)
}

func TestMigratorDownWithGlobalTransaction(t *testing.T) {
	db := setupTestDB(t)
	migrator := New(db)
	ctx := context.Background()
	migrations := createTestMigrations()

	// Apply all migrations
	err := migrator.Up(ctx, migrations)
	require.NoError(t, err)

	// Rollback 2 steps with global transaction
	err = migrator.Down(ctx, migrations, 2, GlobalTransactionOption(true))
	require.NoError(t, err)

	// Verify correct number of migrations remain
	applied, err := migrator.getAppliedMigrations(ctx)
	require.NoError(t, err)
	assert.Len(t, applied, 3)
}
