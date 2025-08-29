package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test models
type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
}

type Product struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Name       string `json:"name"`
	Price      int    `json:"price"`
	CategoryID uint   `json:"category_id"`
	CreatedAt  int64  `json:"created_at"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&User{}, &Product{})
	require.NoError(t, err)

	return db
}

func seedUsers(db *gorm.DB, count int) []User {
	users := make([]User, count)
	for i := 0; i < count; i++ {
		users[i] = User{
			Name:      "User" + string(rune('A'+i%26)),
			Email:     "user" + string(rune('0'+i%10)) + "@example.com",
			CreatedAt: int64(1000 + i*100),
		}
	}

	for _, user := range users {
		db.Create(&user)
	}

	return users
}

func TestNewParams(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		expected Params
	}{
		{
			name:     "valid parameters",
			page:     2,
			pageSize: 10,
			expected: Params{Page: 2, PageSize: 10},
		},
		{
			name:     "invalid page defaults to 1",
			page:     0,
			pageSize: 10,
			expected: Params{Page: 1, PageSize: 10},
		},
		{
			name:     "negative page defaults to 1",
			page:     -5,
			pageSize: 10,
			expected: Params{Page: 1, PageSize: 10},
		},
		{
			name:     "invalid pageSize defaults to 20",
			page:     1,
			pageSize: 0,
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "negative pageSize defaults to 20",
			page:     1,
			pageSize: -10,
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "pageSize too large defaults to 20",
			page:     1,
			pageSize: 101,
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "pageSize at limit is allowed",
			page:     1,
			pageSize: 100,
			expected: Params{Page: 1, PageSize: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewParams(tt.page, tt.pageSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParamsOffset(t *testing.T) {
	tests := []struct {
		name   string
		params Params
		want   int
	}{
		{
			name:   "first page",
			params: Params{Page: 1, PageSize: 10},
			want:   0,
		},
		{
			name:   "second page",
			params: Params{Page: 2, PageSize: 10},
			want:   10,
		},
		{
			name:   "third page with different page size",
			params: Params{Page: 3, PageSize: 5},
			want:   10,
		},
		{
			name:   "large page number",
			params: Params{Page: 100, PageSize: 25},
			want:   2475,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.params.Offset())
		})
	}
}

func TestParamsLimit(t *testing.T) {
	tests := []struct {
		name   string
		params Params
		want   int
	}{
		{
			name:   "small page size",
			params: Params{Page: 1, PageSize: 5},
			want:   5,
		},
		{
			name:   "large page size",
			params: Params{Page: 1, PageSize: 100},
			want:   100,
		},
		{
			name:   "default page size",
			params: Params{Page: 1, PageSize: 20},
			want:   20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.params.Limit())
		})
	}
}

func TestPaginate(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db, 10)

	tests := []struct {
		name            string
		params          Params
		expectedCount   int
		expectedPage    int
		expectedTotal   int64
		expectedHasNext bool
		expectedHasPrev bool
	}{
		{
			name:            "first page",
			params:          NewParams(1, 3),
			expectedCount:   3,
			expectedPage:    1,
			expectedTotal:   10,
			expectedHasNext: true,
			expectedHasPrev: false,
		},
		{
			name:            "middle page",
			params:          NewParams(2, 3),
			expectedCount:   3,
			expectedPage:    2,
			expectedTotal:   10,
			expectedHasNext: true,
			expectedHasPrev: true,
		},
		{
			name:            "last page",
			params:          NewParams(4, 3),
			expectedCount:   1,
			expectedPage:    4,
			expectedTotal:   10,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
		{
			name:            "page beyond data",
			params:          NewParams(10, 3),
			expectedCount:   0,
			expectedPage:    10,
			expectedTotal:   10,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
		{
			name:            "single large page",
			params:          NewParams(1, 20),
			expectedCount:   10,
			expectedPage:    1,
			expectedTotal:   10,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var users []User
			pagination, results, err := Paginate(db, tt.params, &users)
			require.NoError(t, err)

			assert.Len(t, results, tt.expectedCount)
			assert.Equal(t, tt.expectedPage, pagination.Page)
			assert.Equal(t, tt.expectedTotal, pagination.Total)
			assert.Equal(t, tt.expectedHasNext, pagination.HasNext)
			assert.Equal(t, tt.expectedHasPrev, pagination.HasPrev)
			assert.Equal(t, tt.params.PageSize, pagination.PageSize)

			// Calculate expected total pages
			expectedTotalPages := int((tt.expectedTotal + int64(tt.params.PageSize) - 1) / int64(tt.params.PageSize))
			assert.Equal(t, expectedTotalPages, pagination.TotalPages)
		})
	}
}

func TestPaginateWithFilters(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db, 10)

	t.Run("filtered query", func(t *testing.T) {
		// Filter users whose names contain 'A' (should be UserA)
		filteredQuery := db.Where("name LIKE ?", "%A%")
		params := NewParams(1, 10)

		var users []User
		pagination, results, err := Paginate(filteredQuery, params, &users)
		require.NoError(t, err)

		assert.Len(t, results, 1) // Only UserA
		assert.Equal(t, int64(1), pagination.Total)
		assert.False(t, pagination.HasNext)
		assert.False(t, pagination.HasPrev)
	})

	t.Run("filtered query with multiple results", func(t *testing.T) {
		// Filter users with emails containing '0' or '1'
		filteredQuery := db.Where("email LIKE ? OR email LIKE ?", "%0%", "%1%")
		params := NewParams(1, 5)

		var users []User
		pagination, results, err := Paginate(filteredQuery, params, &users)
		require.NoError(t, err)

		assert.Equal(t, int64(2), pagination.Total) // user0 and user1
		assert.Len(t, results, 2)
		assert.False(t, pagination.HasNext)
		assert.False(t, pagination.HasPrev)
	})
}

func TestPaginateWithOrdering(t *testing.T) {
	db := setupTestDB(t)
	seedUsers(db, 5)

	t.Run("ordered by name descending", func(t *testing.T) {
		query := db.Order("name DESC")
		params := NewParams(1, 3)

		var users []User
		pagination, results, err := Paginate(query, params, &users)
		require.NoError(t, err)

		assert.Len(t, results, 3)
		assert.Equal(t, int64(5), pagination.Total)

		// Verify ordering (UserE, UserD, UserC should be first three)
		assert.True(t, results[0].Name >= results[1].Name)
		assert.True(t, results[1].Name >= results[2].Name)
	})

	t.Run("ordered by created_at ascending", func(t *testing.T) {
		query := db.Order("created_at ASC")
		params := NewParams(1, 2)

		var users []User
		_, results, err := Paginate(query, params, &users)
		require.NoError(t, err)

		assert.Len(t, results, 2)
		// Verify ordering
		assert.True(t, results[0].CreatedAt <= results[1].CreatedAt)
	})
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name     string
		page     string
		pageSize string
		expected Params
	}{
		{
			name:     "valid parameters",
			page:     "2",
			pageSize: "10",
			expected: Params{Page: 2, PageSize: 10},
		},
		{
			name:     "empty parameters use defaults",
			page:     "",
			pageSize: "",
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "invalid parameters use defaults",
			page:     "invalid",
			pageSize: "invalid",
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "zero page uses default",
			page:     "0",
			pageSize: "10",
			expected: Params{Page: 1, PageSize: 10},
		},
		{
			name:     "negative page uses default",
			page:     "-5",
			pageSize: "10",
			expected: Params{Page: 1, PageSize: 10},
		},
		{
			name:     "zero pageSize uses default",
			page:     "1",
			pageSize: "0",
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "negative pageSize uses default",
			page:     "1",
			pageSize: "-10",
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "too large pageSize uses default",
			page:     "1",
			pageSize: "101",
			expected: Params{Page: 1, PageSize: 20},
		},
		{
			name:     "pageSize at limit is allowed",
			page:     "1",
			pageSize: "100",
			expected: Params{Page: 1, PageSize: 100},
		},
		{
			name:     "floating point numbers are truncated",
			page:     "2.7",
			pageSize: "15.9",
			expected: Params{Page: 1, PageSize: 20}, // Invalid, so defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseParams(tt.page, tt.pageSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaginateEmptyTable(t *testing.T) {
	db := setupTestDB(t)
	// Don't seed any data

	params := NewParams(1, 10)
	var users []User
	pagination, results, err := Paginate(db, params, &users)
	require.NoError(t, err)

	assert.Empty(t, results)
	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, int64(0), pagination.Total)
	assert.Equal(t, 0, pagination.TotalPages)
	assert.False(t, pagination.HasNext)
	assert.False(t, pagination.HasPrev)
}

func TestPaginateWithDifferentModels(t *testing.T) {
	db := setupTestDB(t)

	// Seed products
	products := []Product{
		{Name: "Laptop", Price: 1000, CategoryID: 1, CreatedAt: 1000},
		{Name: "Mouse", Price: 25, CategoryID: 1, CreatedAt: 2000},
		{Name: "Keyboard", Price: 75, CategoryID: 1, CreatedAt: 3000},
	}
	for _, product := range products {
		db.Create(&product)
	}

	t.Run("paginate products", func(t *testing.T) {
		params := NewParams(1, 2)
		var productResults []Product
		pagination, results, err := Paginate(db, params, &productResults)
		require.NoError(t, err)

		assert.Len(t, results, 2)
		assert.Equal(t, int64(3), pagination.Total)
		assert.True(t, pagination.HasNext)
		assert.False(t, pagination.HasPrev)
	})

	t.Run("paginate with price filter", func(t *testing.T) {
		query := db.Where("price > ?", 50)
		params := NewParams(1, 10)
		var productResults []Product
		pagination, results, err := Paginate(query, params, &productResults)
		require.NoError(t, err)

		assert.Len(t, results, 2) // Laptop and Keyboard
		assert.Equal(t, int64(2), pagination.Total)

		// Verify all results have price > 50
		for _, product := range results {
			assert.Greater(t, product.Price, 50)
		}
	})
}

func TestPaginateDatabaseError(t *testing.T) {
	// Create a database and then close it to simulate connection errors
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	params := NewParams(1, 10)
	var users []User
	pagination, results, err := Paginate(db, params, &users)

	assert.Error(t, err)
	assert.Nil(t, pagination)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "failed to count records")
}
