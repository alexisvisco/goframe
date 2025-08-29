package pagination

import (
	"testing"

	"github.com/nrednav/cuid2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test models for cursor pagination
type UserWithCUID struct {
	ID        string `gorm:"primaryKey" json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
}

type ProductWithScore struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	Name       string  `json:"name"`
	Price      int     `json:"price"`
	Score      float64 `json:"score"`
	CategoryID uint    `json:"category_id"`
	CreatedAt  int64   `json:"created_at"`
}

func setupCursorTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&UserWithCUID{}, &ProductWithScore{})
	require.NoError(t, err)

	return db
}

func seedUsersWithCUID(db *gorm.DB, count int) []UserWithCUID {
	users := make([]UserWithCUID, count)
	for i := 0; i < count; i++ {
		users[i] = UserWithCUID{
			ID:        cuid2.Generate(),
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

func seedProductsWithScore(db *gorm.DB, count int) []ProductWithScore {
	products := make([]ProductWithScore, count)
	for i := 0; i < count; i++ {
		products[i] = ProductWithScore{
			Name:       "Product" + string(rune('A'+i%26)),
			Price:      (i + 1) * 10,
			Score:      float64(i+1) * 1.5,
			CategoryID: uint(i%3 + 1),
			CreatedAt:  int64(1000 + i*100),
		}
	}

	for _, product := range products {
		db.Create(&product)
	}

	return products
}

func TestNewCursorParams(t *testing.T) {
	tests := []struct {
		name      string
		cursor    string
		pageSize  int
		direction string
		expected  CursorParams
	}{
		{
			name:      "valid parameters",
			cursor:    "abc123",
			pageSize:  10,
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 10, Direction: "next"},
		},
		{
			name:      "invalid direction defaults to next",
			cursor:    "abc123",
			pageSize:  10,
			direction: "invalid",
			expected:  CursorParams{Cursor: "abc123", PageSize: 10, Direction: "next"},
		},
		{
			name:      "empty direction defaults to next",
			cursor:    "abc123",
			pageSize:  10,
			direction: "",
			expected:  CursorParams{Cursor: "abc123", PageSize: 10, Direction: "next"},
		},
		{
			name:      "prev direction",
			cursor:    "abc123",
			pageSize:  10,
			direction: "prev",
			expected:  CursorParams{Cursor: "abc123", PageSize: 10, Direction: "prev"},
		},
		{
			name:      "invalid pageSize defaults to 20",
			cursor:    "abc123",
			pageSize:  0,
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 20, Direction: "next"},
		},
		{
			name:      "negative pageSize defaults to 20",
			cursor:    "abc123",
			pageSize:  -5,
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 20, Direction: "next"},
		},
		{
			name:      "pageSize too large defaults to 20",
			cursor:    "abc123",
			pageSize:  101,
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 20, Direction: "next"},
		},
		{
			name:      "pageSize at limit is allowed",
			cursor:    "abc123",
			pageSize:  100,
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 100, Direction: "next"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewCursorParams(tt.cursor, tt.pageSize, tt.direction)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPaginateCursorFirstPage(t *testing.T) {
	db := setupCursorTestDB(t)
	seedUsersWithCUID(db, 10)

	params := NewCursorParams("", 3, "next")
	var users []UserWithCUID
	pagination, results, err := PaginateCursor(db, params, &users, "created_at")
	require.NoError(t, err)

	assert.Len(t, results, 3)
	assert.True(t, pagination.HasNext)
	assert.False(t, pagination.HasPrev)
	assert.NotEmpty(t, pagination.NextCursor)
	assert.NotEmpty(t, pagination.PrevCursor) // PrevCursor is generated when direction is "next"
	assert.Equal(t, 3, pagination.PageSize)

	// Verify ordering by created_at
	for i := 1; i < len(results); i++ {
		assert.LessOrEqual(t, results[i-1].CreatedAt, results[i].CreatedAt)
	}
}

func TestPaginateCursorWithCUID(t *testing.T) {
	db := setupCursorTestDB(t)
	seedUsersWithCUID(db, 10)

	t.Run("first page ordered by CUID", func(t *testing.T) {
		params := NewCursorParams("", 4, "next")
		var users []UserWithCUID
		pagination, results, err := PaginateCursor(db, params, &users, "id")
		require.NoError(t, err)

		assert.Len(t, results, 4)
		assert.True(t, pagination.HasNext)
		assert.False(t, pagination.HasPrev)
		assert.NotEmpty(t, pagination.NextCursor)

		// Verify all IDs are valid CUIDs
		for _, user := range results {
			assert.NotEmpty(t, user.ID)
			assert.True(t, len(user.ID) > 20) // CUIDs are typically longer than 20 chars
		}
	})

	t.Run("navigate with CUID cursor", func(t *testing.T) {
		// Get first page
		params := NewCursorParams("", 3, "next")
		var users []UserWithCUID
		firstPagination, firstResults, err := PaginateCursor(db, params, &users, "id")
		require.NoError(t, err)
		require.True(t, firstPagination.HasNext)

		// Use cursor for next page
		var nextUsers []UserWithCUID
		nextParams := NewCursorParams(firstPagination.NextCursor, 3, "next")
		nextPagination, nextResults, err := PaginateCursor(db, nextParams, &nextUsers, "id")
		require.NoError(t, err)

		assert.Len(t, nextResults, 3)
		assert.True(t, nextPagination.HasPrev)

		// Ensure no overlap between pages by checking IDs
		firstPageIDs := make(map[string]bool)
		for _, user := range firstResults {
			firstPageIDs[user.ID] = true
		}

		for _, user := range nextResults {
			assert.False(t, firstPageIDs[user.ID], "Found duplicate ID between pages: %s", user.ID)
		}
	})
}

func TestPaginateCursorWithNumericOrdering(t *testing.T) {
	db := setupCursorTestDB(t)
	seedProductsWithScore(db, 8)

	t.Run("order by integer price", func(t *testing.T) {
		params := NewCursorParams("", 3, "next")
		var products []ProductWithScore
		pagination, results, err := PaginateCursor(db, params, &products, "price")
		require.NoError(t, err)

		assert.Len(t, results, 3)
		assert.True(t, pagination.HasNext)

		// Verify ordering by price
		for i := 1; i < len(results); i++ {
			assert.LessOrEqual(t, results[i-1].Price, results[i].Price)
		}
	})

	t.Run("order by float score", func(t *testing.T) {
		params := NewCursorParams("", 3, "next")
		var products []ProductWithScore
		pagination, results, err := PaginateCursor(db, params, &products, "score")
		require.NoError(t, err)

		assert.Len(t, results, 3)
		assert.True(t, pagination.HasNext)

		// Verify ordering by score
		for i := 1; i < len(results); i++ {
			assert.LessOrEqual(t, results[i-1].Score, results[i].Score)
		}
	})

	t.Run("navigate with numeric cursor", func(t *testing.T) {
		// Get first page ordered by price
		params := NewCursorParams("", 2, "next")
		var products []ProductWithScore
		firstPagination, firstResults, err := PaginateCursor(db, params, &products, "price")
		require.NoError(t, err)

		// Get second page
		var nextProducts []ProductWithScore
		nextParams := NewCursorParams(firstPagination.NextCursor, 2, "next")
		_, nextResults, err := PaginateCursor(db, nextParams, &nextProducts, "price")
		require.NoError(t, err)

		assert.Len(t, nextResults, 2)

		// Verify continuation of ordering
		lastPriceFirstPage := firstResults[len(firstResults)-1].Price
		firstPriceSecondPage := nextResults[0].Price
		assert.Less(t, lastPriceFirstPage, firstPriceSecondPage)
	})
}

func TestPaginateCursorPrevDirection(t *testing.T) {
	db := setupCursorTestDB(t)
	seedUsersWithCUID(db, 6)

	// Navigate to middle of dataset first
	var users []UserWithCUID
	params := NewCursorParams("", 2, "next")
	firstPagination, firstResults, err := PaginateCursor(db, params, &users, "created_at")
	require.NoError(t, err)

	var secondPageUsers []UserWithCUID
	secondParams := NewCursorParams(firstPagination.NextCursor, 2, "next")
	secondPagination, _, err := PaginateCursor(db, secondParams, &secondPageUsers, "created_at")
	require.NoError(t, err)

	// Now go backwards from second page cursor
	var prevUsers []UserWithCUID
	prevParams := NewCursorParams(secondPagination.NextCursor, 2, "prev")
	prevPagination, prevResults, err := PaginateCursor(db, prevParams, &prevUsers, "created_at")
	require.NoError(t, err)

	assert.Len(t, prevResults, 2)
	assert.True(t, prevPagination.HasPrev)
	assert.True(t, prevPagination.HasNext)

	// Results should be equivalent to first page (but order might be different due to reverse)
	assert.Equal(t, len(firstResults), len(prevResults))
}

func TestPaginateCursorWithFilters(t *testing.T) {
	db := setupCursorTestDB(t)
	seedProductsWithScore(db, 10)

	t.Run("filtered cursor pagination", func(t *testing.T) {
		// Filter products with price > 50
		filteredQuery := db.Where("price > ?", 50)
		params := NewCursorParams("", 3, "next")
		var products []ProductWithScore
		_, results, err := PaginateCursor(filteredQuery, params, &products, "price")
		require.NoError(t, err)

		assert.True(t, len(results) > 0)

		// Verify all results match filter
		for _, product := range results {
			assert.Greater(t, product.Price, 50)
		}

		// Verify ordering is maintained
		for i := 1; i < len(results); i++ {
			assert.LessOrEqual(t, results[i-1].Price, results[i].Price)
		}
	})

	t.Run("navigate filtered results", func(t *testing.T) {
		filteredQuery := db.Where("category_id = ?", 1)
		params := NewCursorParams("", 2, "next")
		var products []ProductWithScore
		firstPagination, _, err := PaginateCursor(filteredQuery, params, &products, "id")
		require.NoError(t, err)

		if firstPagination.HasNext {
			var nextProducts []ProductWithScore
			nextParams := NewCursorParams(firstPagination.NextCursor, 2, "next")
			_, nextResults, err := PaginateCursor(filteredQuery, nextParams, &nextProducts, "id")
			require.NoError(t, err)

			// Verify all results still match filter
			for _, product := range nextResults {
				assert.Equal(t, uint(1), product.CategoryID)
			}
		}
	})
}

func TestPaginateCursorEmptyResults(t *testing.T) {
	db := setupCursorTestDB(t)
	// Don't seed any data

	params := NewCursorParams("", 10, "next")
	var users []UserWithCUID
	pagination, results, err := PaginateCursor(db, params, &users, "id")
	require.NoError(t, err)

	assert.Empty(t, results)
	assert.False(t, pagination.HasNext)
	assert.False(t, pagination.HasPrev)
	assert.Empty(t, pagination.NextCursor)
	assert.Empty(t, pagination.PrevCursor)
}

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		id    interface{}
		field string
	}{
		{
			name:  "integer id",
			id:    123,
			field: "id",
		},
		{
			name:  "string id",
			id:    "abc123",
			field: "uuid",
		},
		{
			name:  "float id",
			id:    123.45,
			field: "score",
		},
		{
			name:  "cuid2 id",
			id:    cuid2.Generate(),
			field: "id",
		},
		{
			name:  "negative number",
			id:    -42,
			field: "balance",
		},
		{
			name:  "zero value",
			id:    0,
			field: "count",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor, err := encodeCursor(tt.id, tt.field)
			require.NoError(t, err)
			assert.NotEmpty(t, cursor)

			decoded, err := decodeCursor(cursor)
			require.NoError(t, err)
			assert.Equal(t, tt.field, decoded.Field)

			// For interface{} comparison, we need to handle type conversion
			switch v := tt.id.(type) {
			case int:
				assert.Equal(t, float64(v), decoded.ID.(float64))
			case float64:
				assert.Equal(t, v, decoded.ID.(float64))
			case string:
				assert.Equal(t, v, decoded.ID.(string))
			}
		})
	}
}

func TestDecodeInvalidCursor(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{
			name:   "invalid base64",
			cursor: "invalid-base64!@#",
		},
		{
			name:   "valid base64 but invalid json",
			cursor: "aW52YWxpZCBqc29u", // "invalid json" in base64
		},
		{
			name:   "empty cursor",
			cursor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cursor == "" {
				// Empty cursor should be handled gracefully by PaginateCursor
				return
			}

			decoded, err := decodeCursor(tt.cursor)
			assert.Error(t, err)
			assert.Nil(t, decoded)
		})
	}
}

func TestParseCursorParams(t *testing.T) {
	tests := []struct {
		name      string
		cursor    string
		pageSize  string
		direction string
		expected  CursorParams
	}{
		{
			name:      "valid parameters",
			cursor:    "abc123",
			pageSize:  "10",
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 10, Direction: "next"},
		},
		{
			name:      "empty parameters use defaults",
			cursor:    "",
			pageSize:  "",
			direction: "",
			expected:  CursorParams{Cursor: "", PageSize: 20, Direction: "next"},
		},
		{
			name:      "invalid pageSize uses default",
			cursor:    "abc123",
			pageSize:  "invalid",
			direction: "prev",
			expected:  CursorParams{Cursor: "abc123", PageSize: 20, Direction: "prev"},
		},
		{
			name:      "zero pageSize uses default",
			cursor:    "abc123",
			pageSize:  "0",
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 20, Direction: "next"},
		},
		{
			name:      "negative pageSize uses default",
			cursor:    "abc123",
			pageSize:  "-5",
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 20, Direction: "next"},
		},
		{
			name:      "pageSize too large uses default",
			cursor:    "abc123",
			pageSize:  "101",
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 20, Direction: "next"},
		},
		{
			name:      "pageSize at limit is allowed",
			cursor:    "abc123",
			pageSize:  "100",
			direction: "next",
			expected:  CursorParams{Cursor: "abc123", PageSize: 100, Direction: "next"},
		},
		{
			name:      "invalid direction uses default",
			cursor:    "abc123",
			pageSize:  "15",
			direction: "invalid",
			expected:  CursorParams{Cursor: "abc123", PageSize: 15, Direction: "next"},
		},
		{
			name:      "cuid2 cursor",
			cursor:    cuid2.Generate(),
			pageSize:  "25",
			direction: "prev",
			expected:  CursorParams{Cursor: "", PageSize: 25, Direction: "prev"}, // cursor will be set to the generated value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseCursorParams(tt.cursor, tt.pageSize, tt.direction)
			if tt.name == "cuid2 cursor" {
				// Special case: just check the other fields since cursor is generated
				assert.Equal(t, tt.expected.PageSize, result.PageSize)
				assert.Equal(t, tt.expected.Direction, result.Direction)
				assert.NotEmpty(t, result.Cursor)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetFieldValue(t *testing.T) {
	user := UserWithCUID{
		ID:        cuid2.Generate(),
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: 1234567890,
	}

	product := ProductWithScore{
		ID:         123,
		Name:       "Test Product",
		Price:      100,
		Score:      85.5,
		CategoryID: 1,
		CreatedAt:  9876543210,
	}

	tests := []struct {
		name      string
		item      interface{}
		fieldName string
		expected  interface{}
		checkType bool
	}{
		{
			name:      "get user ID field",
			item:      user,
			fieldName: "ID",
			expected:  user.ID,
			checkType: true,
		},
		{
			name:      "get user id field (lowercase)",
			item:      user,
			fieldName: "id",
			expected:  user.ID,
			checkType: true,
		},
		{
			name:      "get user Name field",
			item:      user,
			fieldName: "Name",
			expected:  "Test User",
			checkType: true,
		},
		{
			name:      "get user created_at field",
			item:      user,
			fieldName: "created_at",
			expected:  int64(1234567890),
			checkType: true,
		},
		{
			name:      "get product Price field",
			item:      product,
			fieldName: "Price",
			expected:  100,
			checkType: true,
		},
		{
			name:      "get product score field",
			item:      product,
			fieldName: "score",
			expected:  85.5,
			checkType: true,
		},
		{
			name:      "get product price (lowercase)",
			item:      product,
			fieldName: "price",
			expected:  100,
			checkType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFieldValue(tt.item, tt.fieldName)
			if tt.checkType {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NotNil(t, result)
			}
		})
	}

	t.Run("nonexistent field returns nil", func(t *testing.T) {
		result := getFieldValue(user, "nonexistent")
		assert.Nil(t, result)
	})

	t.Run("field from pointer", func(t *testing.T) {
		result := getFieldValue(&user, "Name")
		assert.Equal(t, "Test User", result)
	})
}

func TestReverseSlice(t *testing.T) {
	t.Run("reverse integers", func(t *testing.T) {
		slice := []int{1, 2, 3, 4, 5}
		expected := []int{5, 4, 3, 2, 1}
		reverseSlice(&slice)
		assert.Equal(t, expected, slice)
	})

	t.Run("reverse strings", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		expected := []string{"c", "b", "a"}
		reverseSlice(&slice)
		assert.Equal(t, expected, slice)
	})

	t.Run("reverse CUIDs", func(t *testing.T) {
		cuid1 := cuid2.Generate()
		cuid2_val := cuid2.Generate()
		cuid3 := cuid2.Generate()

		slice := []string{cuid1, cuid2_val, cuid3}
		expected := []string{cuid3, cuid2_val, cuid1}
		reverseSlice(&slice)
		assert.Equal(t, expected, slice)
	})

	t.Run("empty slice", func(t *testing.T) {
		var slice []int
		reverseSlice(&slice)
		assert.Empty(t, slice)
	})

	t.Run("single element slice", func(t *testing.T) {
		slice := []int{42}
		expected := []int{42}
		reverseSlice(&slice)
		assert.Equal(t, expected, slice)
	})

	t.Run("two element slice", func(t *testing.T) {
		slice := []string{"first", "second"}
		expected := []string{"second", "first"}
		reverseSlice(&slice)
		assert.Equal(t, expected, slice)
	})
}

func TestPaginateCursorDatabaseError(t *testing.T) {
	// Create a database and then close it to simulate connection errors
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	params := NewCursorParams("", 10, "next")
	var users []UserWithCUID
	pagination, results, err := PaginateCursor(db, params, &users, "id")

	assert.Error(t, err)
	assert.Nil(t, pagination)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "failed to fetch cursor paginated data")
}

func TestPaginateCursorInvalidCursor(t *testing.T) {
	db := setupCursorTestDB(t)
	seedUsersWithCUID(db, 5)

	params := NewCursorParams("invalid-cursor-data", 10, "next")
	var users []UserWithCUID
	pagination, results, err := PaginateCursor(db, params, &users, "id")

	assert.Error(t, err)
	assert.Nil(t, pagination)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "failed to decode cursor")
}
