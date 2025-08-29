// Package pagination provides comprehensive pagination support for GORM queries with both
// offset-based (traditional) and cursor-based pagination methods.
//
// The package offers two main pagination approaches:
//
//  1. Offset-based pagination: Traditional page/limit pagination with page numbers,
//     suitable for small to medium datasets where users need to navigate to specific pages.
//
//  2. Cursor-based pagination: High-performance pagination using cursors, ideal for
//     large datasets and real-time data where consistent performance is required.
//
// Both methods work seamlessly with existing GORM query chains and GoFrame's dbutil package.
//
// # Offset-based Pagination Example
//
//	params := pagination.NewParams(1, 20) // page 1, 20 items per page
//	var users []User
//	result, users, err := pagination.Paginate(db, params, &users)
//	if err != nil {
//		return err
//	}
//	fmt.Printf("Page %d of %d (Total: %d)\n", result.Page, result.TotalPages, result.Total)
//
// # Cursor-based Pagination Example
//
//	params := pagination.NewCursorParams("", 20, "next") // first page, 20 items
//	var users []User
//	result, users, err := pagination.PaginateCursor(db, params, &users, "id")
//	if err != nil {
//		return err
//	}
//	// Navigate to next page using the cursor
//	if result.HasNext {
//		nextParams := pagination.NewCursorParams(result.NextCursor, 20, "next")
//		// Use nextParams for next request
//	}
//
// # HTTP API Integration
//
// The package provides parsing functions for easy HTTP parameter handling:
//
//	// Offset pagination: GET /users?page=2&page_size=10
//	params := pagination.ParseParams(
//		r.URL.Query().Get("page"),
//		r.URL.Query().Get("page_size"),
//	)
//
//	// Cursor pagination: GET /users?cursor=abc123&page_size=10&direction=next
//	params := pagination.ParseCursorParams(
//		r.URL.Query().Get("cursor"),
//		r.URL.Query().Get("page_size"),
//		r.URL.Query().Get("direction"),
//	)
//
// # Performance Considerations
//
// Offset pagination performance degrades with large offsets due to database OFFSET behavior,
// while cursor pagination maintains consistent O(log n) performance regardless of position.
// For optimal cursor pagination performance, ensure your order field is properly indexed.
//
// # Field Ordering Support
//
// Cursor pagination supports ordering by any comparable field:
//
//	// Order by ID (default, most common)
//	PaginateCursor(db, params, &users, "id")
//
//	// Order by timestamp
//	PaginateCursor(db, params, &users, "created_at")
//
//	// Order by numeric fields
//	PaginateCursor(db, params, &products, "price")
//
// Field name matching is flexible and supports exact field names, lowercase variants,
// JSON tag names, and GORM column names.
package pagination

import (
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

// Params represents standard offset-based pagination parameters.
// This is suitable for traditional pagination where users need page numbers
// and the ability to jump to specific pages.
type Params struct {
	Page     int `json:"page" query:"page"`           // Current page number (1-based)
	PageSize int `json:"page_size" query:"page_size"` // Number of items per page
}

// Pagination represents pagination metadata for offset-based pagination.
// It includes all necessary metadata for implementing pagination controls
// in user interfaces.
type Pagination struct {
	Page       int   `json:"page"`        // Current page number
	PageSize   int   `json:"page_size"`   // Items per page
	Total      int64 `json:"total"`       // Total number of items across all pages
	TotalPages int   `json:"total_pages"` // Total number of pages
	HasNext    bool  `json:"has_next"`    // Whether there is a next page
	HasPrev    bool  `json:"has_prev"`    // Whether there is a previous page
}

// NewParams creates new pagination parameters with validation and defaults.
// Invalid page numbers default to 1, and invalid page sizes default to 20.
// Page sizes are limited to a maximum of 100 to prevent excessive memory usage.
func NewParams(page, pageSize int) Params {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return Params{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset calculates the database offset for the current page.
// This is used internally by GORM's Offset() method.
func (p Params) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit returns the page size as the query limit.
// This is used internally by GORM's Limit() method.
func (p Params) Limit() int {
	return p.PageSize
}

// Paginate performs offset-based pagination on a GORM query.
// It counts the total records, calculates pagination metadata,
// and fetches the requested page of data.
//
// The function works with any GORM query chain, allowing for
// complex filtering and joining before pagination is applied.
//
// Example:
//
//	// Basic pagination
//	var users []User
//	pagination, users, err := pagination.Paginate(db, params, &users)
//
//	// With filtering
//	query := db.Where("active = ?", true).Order("created_at DESC")
//	pagination, users, err := pagination.Paginate(query, params, &users)
//
// Note: This method performs two queries - one for counting and one for data.
// For large datasets, consider using cursor-based pagination instead.
func Paginate[T any](db *gorm.DB, params Params, dest *[]T) (*Pagination, []T, error) {
	var total int64

	// Count total records
	if err := db.Model(new(T)).Count(&total).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to count records: %w", err)
	}

	// Calculate pagination metadata
	totalPages := int((total + int64(params.PageSize) - 1) / int64(params.PageSize))
	hasNext := params.Page < totalPages
	hasPrev := params.Page > 1

	// Fetch paginated data
	if err := db.Offset(params.Offset()).Limit(params.Limit()).Find(dest).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to fetch paginated data: %w", err)
	}

	pagination := &Pagination{
		Page:       params.Page,
		PageSize:   params.PageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	return pagination, *dest, nil
}

// ParseParams parses offset pagination parameters from query strings.
// This is a convenience function for HTTP handlers that need to convert
// string parameters to validated Params.
//
// Example usage in HTTP handler:
//
//	params := pagination.ParseParams(
//		r.URL.Query().Get("page"),
//		r.URL.Query().Get("page_size"),
//	)
//
// Invalid or missing parameters will use sensible defaults.
func ParseParams(page, pageSize string) Params {
	p := 1
	if page != "" {
		if parsed, err := strconv.Atoi(page); err == nil && parsed > 0 {
			p = parsed
		}
	}

	size := 20
	if pageSize != "" {
		if parsed, err := strconv.Atoi(pageSize); err == nil && parsed > 0 && parsed <= 100 {
			size = parsed
		}
	}

	return NewParams(p, size)
}
