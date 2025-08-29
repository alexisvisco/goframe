package pagination

import (
	"context"
	"testing"

	"github.com/alexisvisco/goframe/db/dbutil"
	"github.com/nrednav/cuid2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test models for integration
type OrderModel struct {
	ID         string `gorm:"primaryKey" json:"id"`
	CustomerID string `json:"customer_id"`
	Amount     int    `json:"amount"`
	Status     string `json:"status"`
	CreatedAt  int64  `json:"created_at"`
}

func setupIntegrationDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	err = db.AutoMigrate(&OrderModel{})
	require.NoError(t, err)

	return db
}

func seedOrders(db *gorm.DB, count int) []OrderModel {
	orders := make([]OrderModel, count)
	statuses := []string{"pending", "processing", "completed", "cancelled"}

	for i := 0; i < count; i++ {
		orders[i] = OrderModel{
			ID:         cuid2.Generate(),
			CustomerID: cuid2.Generate(),
			Amount:     (i + 1) * 100,
			Status:     statuses[i%len(statuses)],
			CreatedAt:  int64(1000 + i*100),
		}
	}

	for _, order := range orders {
		db.Create(&order)
	}

	return orders
}

// TestPaginationWithDBUtil demonstrates integration with GoFrame's dbutil package
func TestPaginationWithDBUtil(t *testing.T) {
	db := setupIntegrationDB(t)
	seedOrders(db, 12)

	t.Run("offset pagination with context", func(t *testing.T) {
		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		params := NewParams(2, 5) // page 2, 5 items per page
		var orders []OrderModel
		pagination, results, err := Paginate(contextDB, params, &orders)
		require.NoError(t, err)

		assert.Len(t, results, 5)
		assert.Equal(t, 2, pagination.Page)
		assert.Equal(t, int64(12), pagination.Total)
		assert.Equal(t, 3, pagination.TotalPages)
		assert.True(t, pagination.HasNext)
		assert.True(t, pagination.HasPrev)
	})

	t.Run("cursor pagination with context", func(t *testing.T) {
		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		params := NewCursorParams("", 4, "next")
		var orders []OrderModel
		pagination, results, err := PaginateCursor(contextDB, params, &orders, "created_at")
		require.NoError(t, err)

		assert.Len(t, results, 4)
		assert.True(t, pagination.HasNext)
		assert.False(t, pagination.HasPrev)
		assert.NotEmpty(t, pagination.NextCursor)
	})
}

// TestPaginationInTransaction demonstrates pagination within database transactions
func TestPaginationInTransaction(t *testing.T) {
	db := setupIntegrationDB(t)
	seedOrders(db, 8)

	t.Run("offset pagination in transaction", func(t *testing.T) {
		ctx := context.Background()

		err := dbutil.Transaction(ctx, db, func(ctx context.Context) error {
			txDB := dbutil.DB(ctx, db)

			// Update some records in transaction
			// Update the record with specific columns
			err := txDB.Model(&OrderModel{}).
				Where("status = ?", "pending").
				Update("status", "processing").Error
			require.NoError(t, err)

			// Paginate the updated results
			filteredQuery := txDB.Where("status = ?", "processing")
			params := NewParams(1, 10)
			var orders []OrderModel
			_, results, err := Paginate(filteredQuery, params, &orders)
			require.NoError(t, err)

			// Verify all results have the updated status
			for _, order := range results {
				assert.Equal(t, "processing", order.Status)
			}

			return nil
		})

		require.NoError(t, err)
	})

	t.Run("cursor pagination in transaction", func(t *testing.T) {
		ctx := context.Background()

		err := dbutil.Transaction(ctx, db, func(ctx context.Context) error {
			txDB := dbutil.DB(ctx, db)

			// Create a new order in transaction
			newOrder := OrderModel{
				ID:         cuid2.Generate(),
				CustomerID: cuid2.Generate(),
				Amount:     999,
				Status:     "urgent",
				CreatedAt:  9999,
			}
			err := txDB.Create(&newOrder).Error
			require.NoError(t, err)

			// Paginate including the new record
			params := NewCursorParams("", 5, "next")
			var orders []OrderModel
			pagination, results, err := PaginateCursor(txDB, params, &orders, "amount")
			require.NoError(t, err)

			assert.True(t, len(results) > 0)
			assert.True(t, pagination.HasNext)

			return nil
		})

		require.NoError(t, err)
	})
}

// TestPaginationWithComplexQueries demonstrates pagination with joins and complex filtering
func TestPaginationWithComplexQueries(t *testing.T) {
	db := setupIntegrationDB(t)
	seedOrders(db, 15)

	t.Run("paginate with multiple filters", func(t *testing.T) {
		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		// Complex query with multiple conditions
		query := contextDB.
			Where("amount > ?", 500).
			Where("status IN (?)", []string{"completed", "processing"}).
			Order("amount DESC")

		params := NewParams(1, 3)
		var orders []OrderModel
		_, results, err := Paginate(query, params, &orders)
		require.NoError(t, err)

		// Verify all results match the filters
		for _, order := range results {
			assert.Greater(t, order.Amount, 500)
			assert.Contains(t, []string{"completed", "processing"}, order.Status)
		}

		// Verify ordering
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].Amount, results[i].Amount)
		}
	})

	t.Run("cursor pagination with complex ordering", func(t *testing.T) {
		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		// Query ordered by multiple fields (simulated with amount DESC)
		query := contextDB.
			Where("status != ?", "cancelled").
			Order("amount DESC")

		params := NewCursorParams("", 4, "next")
		var orders []OrderModel
		_, results, err := PaginateCursor(query, params, &orders, "amount")
		require.NoError(t, err)

		// Verify all results exclude cancelled orders
		for _, order := range results {
			assert.NotEqual(t, "cancelled", order.Status)
		}

		// Verify descending order by amount
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].Amount, results[i].Amount)
		}
	})
}

// TestPaginationWithCUID demonstrates pagination using CUID2 as ordering field
func TestPaginationWithCUID(t *testing.T) {
	db := setupIntegrationDB(t)
	seedOrders(db, 6)

	t.Run("cursor pagination ordered by CUID", func(t *testing.T) {
		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		// First page
		params := NewCursorParams("", 3, "next")
		var firstPageOrders []OrderModel
		firstPagination, firstPageResults, err := PaginateCursor(contextDB, params, &firstPageOrders, "id")
		require.NoError(t, err)

		assert.Len(t, firstPageResults, 3)
		assert.True(t, firstPagination.HasNext)
		assert.NotEmpty(t, firstPagination.NextCursor)

		// Verify all IDs are valid CUIDs
		for _, order := range firstPageResults {
			assert.NotEmpty(t, order.ID)
			assert.True(t, len(order.ID) > 20) // CUIDs are typically longer
		}

		// Second page using cursor
		if firstPagination.HasNext {
			var secondPageOrders []OrderModel
			nextParams := NewCursorParams(firstPagination.NextCursor, 3, "next")
			secondPagination, secondPageResults, err := PaginateCursor(contextDB, nextParams, &secondPageOrders, "id")
			require.NoError(t, err)

			assert.True(t, len(secondPageResults) > 0)
			assert.True(t, secondPagination.HasPrev)

			// Ensure no ID overlap between pages
			firstPageIDs := make(map[string]bool)
			for _, order := range firstPageResults {
				firstPageIDs[order.ID] = true
			}

			for _, order := range secondPageResults {
				assert.False(t, firstPageIDs[order.ID], "Found duplicate ID between pages")
			}
		}
	})

	t.Run("navigate backwards with CUID cursor", func(t *testing.T) {
		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		// Navigate to middle, then go backwards
		params := NewCursorParams("", 2, "next")
		var orders []OrderModel
		firstPagination, _, err := PaginateCursor(contextDB, params, &orders, "id")
		require.NoError(t, err)

		if firstPagination.HasNext {
			// Go to second page
			var secondPageOrders []OrderModel
			secondParams := NewCursorParams(firstPagination.NextCursor, 2, "next")
			secondPagination, _, err := PaginateCursor(contextDB, secondParams, &secondPageOrders, "id")
			require.NoError(t, err)

			// Now go backwards
			var backwardOrders []OrderModel
			backParams := NewCursorParams(secondPagination.NextCursor, 2, "prev")
			backPagination, backPageResults, err := PaginateCursor(contextDB, backParams, &backwardOrders, "id")
			require.NoError(t, err)

			assert.Len(t, backPageResults, 2)
			assert.True(t, backPagination.HasPrev)
		}
	})
}

// TestPaginationServicePattern demonstrates a service layer pattern using pagination
func TestPaginationServicePattern(t *testing.T) {
	db := setupIntegrationDB(t)
	seedOrders(db, 20)

	// OrderService simulates a service layer
	type OrderService struct {
		db *gorm.DB
	}

	// GetOrdersPaginated demonstrates offset-based pagination in service
	getOrdersPaginated := func(ctx context.Context, service *OrderService, params Params) (*Pagination, []OrderModel, error) {
		contextDB := dbutil.DB(ctx, service.db)
		var orders []OrderModel
		return Paginate(contextDB, params, &orders)
	}

	// GetOrdersByCursor demonstrates cursor-based pagination in service
	getOrdersByCursor := func(ctx context.Context, service *OrderService, params CursorParams, orderField string) (*CursorPagination, []OrderModel, error) {
		contextDB := dbutil.DB(ctx, service.db)
		var orders []OrderModel
		return PaginateCursor(contextDB, params, &orders, orderField)
	}

	// GetOrdersByStatus demonstrates filtered pagination
	getOrdersByStatus := func(ctx context.Context, service *OrderService, status string, params Params) (*Pagination, []OrderModel, error) {
		var pagination *Pagination
		var results []OrderModel

		err := dbutil.Transaction(ctx, service.db, func(ctx context.Context) error {
			txDB := dbutil.DB(ctx, service.db)
			filteredQuery := txDB.Where("status = ?", status)
			var orders []OrderModel
			var err error
			pagination, results, err = Paginate(filteredQuery, params, &orders)
			return err
		})

		return pagination, results, err
	}

	service := &OrderService{db: db}
	ctx := context.Background()

	t.Run("service offset pagination", func(t *testing.T) {
		params := NewParams(2, 5)
		pagination, results, err := getOrdersPaginated(ctx, service, params)
		require.NoError(t, err)

		assert.Len(t, results, 5)
		assert.Equal(t, 2, pagination.Page)
		assert.Equal(t, int64(20), pagination.Total)
	})

	t.Run("service cursor pagination", func(t *testing.T) {
		params := NewCursorParams("", 7, "next")
		pagination, results, err := getOrdersByCursor(ctx, service, params, "created_at")
		require.NoError(t, err)

		assert.Len(t, results, 7)
		assert.True(t, pagination.HasNext)
	})

	t.Run("service filtered pagination", func(t *testing.T) {
		params := NewParams(1, 10)
		var err error
		_, results, err := getOrdersByStatus(ctx, service, "pending", params)
		require.NoError(t, err)

		// Verify all results have the requested status
		for _, order := range results {
			assert.Equal(t, "pending", order.Status)
		}
	})
}

// TestPaginationErrorHandling demonstrates error handling patterns
func TestPaginationErrorHandling(t *testing.T) {
	t.Run("database connection error", func(t *testing.T) {
		// Create and close database to simulate connection error
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		require.NoError(t, err)

		sqlDB, err := db.DB()
		require.NoError(t, err)
		sqlDB.Close()

		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		// Test offset pagination error
		params := NewParams(1, 10)
		var orders []OrderModel
		pagination, results, err := Paginate(contextDB, params, &orders)
		assert.Error(t, err)
		assert.Nil(t, pagination)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "failed to count records")

		// Test cursor pagination error
		cursorParams := NewCursorParams("", 10, "next")
		var cursorOrders []OrderModel
		cursorPagination, cursorResults, err := PaginateCursor(contextDB, cursorParams, &cursorOrders, "id")
		assert.Error(t, err)
		assert.Nil(t, cursorPagination)
		assert.Nil(t, cursorResults)
		assert.Contains(t, err.Error(), "failed to fetch cursor paginated data")
	})

	t.Run("invalid cursor handling", func(t *testing.T) {
		db := setupIntegrationDB(t)
		seedOrders(db, 5)

		ctx := context.Background()
		contextDB := dbutil.DB(ctx, db)

		// Test with malformed cursor
		params := NewCursorParams("invalid-cursor-123", 10, "next")
		var orders []OrderModel
		pagination, results, err := PaginateCursor(contextDB, params, &orders, "id")

		assert.Error(t, err)
		assert.Nil(t, pagination)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "failed to decode cursor")
	})
}
