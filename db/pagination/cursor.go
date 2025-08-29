package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// CursorParams represents cursor-based pagination parameters.
// This method is ideal for large datasets and provides consistent
// performance regardless of the pagination position.
type CursorParams struct {
	Cursor    string `json:"cursor" query:"cursor"`       // Base64 encoded cursor for pagination position
	PageSize  int    `json:"page_size" query:"page_size"` // Number of items per page
	Direction string `json:"direction" query:"direction"` // "next" or "prev" for navigation direction
}

// CursorPagination represents pagination metadata for cursor-based pagination.
// It provides cursors for navigation without exposing internal record positions.
type CursorPagination struct {
	NextCursor string `json:"next_cursor,omitempty"` // Cursor for the next page (if available)
	PrevCursor string `json:"prev_cursor,omitempty"` // Cursor for the previous page (if available)
	HasNext    bool   `json:"has_next"`              // Whether there is a next page
	HasPrev    bool   `json:"has_prev"`              // Whether there is a previous page
	PageSize   int    `json:"page_size"`             // Items per page
}

// CursorData represents the internal structure of a cursor.
// This is encoded as base64 JSON for transmission.
type CursorData struct {
	ID    interface{} `json:"id"`    // The value of the ordering field
	Field string      `json:"field"` // The field name used for ordering
}

// NewCursorParams creates new cursor pagination parameters with validation and defaults.
// Invalid directions default to "next", and invalid page sizes default to 20.
// Page sizes are limited to a maximum of 100 to prevent excessive memory usage.
func NewCursorParams(cursor string, pageSize int, direction string) CursorParams {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	if direction != "next" && direction != "prev" {
		direction = "next"
	}
	return CursorParams{
		Cursor:    cursor,
		PageSize:  pageSize,
		Direction: direction,
	}
}

// PaginateCursor performs cursor-based pagination on a GORM query.
// The orderField parameter specifies which field to use for ordering (e.g., "id", "created_at").
//
// This method provides consistent O(log n) performance regardless of the pagination position,
// making it ideal for large datasets where offset-based pagination would be slow.
//
// Example:
//
//	// First page
//	params := pagination.NewCursorParams("", 20, "next")
//	var users []User
//	pagination, users, err := pagination.PaginateCursor(db, params, &users, "id")
//
//	// Navigate to next page
//	if pagination.HasNext {
//		nextParams := pagination.NewCursorParams(pagination.NextCursor, 20, "next")
//		nextPagination, nextUsers, err := pagination.PaginateCursor(db, nextParams, &users, "id")
//	}
//
//	// Navigate backwards
//	if pagination.HasPrev {
//		prevParams := pagination.NewCursorParams(pagination.PrevCursor, 20, "prev")
//		prevPagination, prevUsers, err := pagination.PaginateCursor(db, prevParams, &users, "id")
//	}
//
// The function supports ordering by any comparable field. Ensure the orderField
// is properly indexed in your database for optimal performance.
func PaginateCursor[T any](db *gorm.DB, params CursorParams, dest *[]T, orderField string) (*CursorPagination, []T, error) {
	query := db

	// Parse cursor if provided
	var cursorData *CursorData
	if params.Cursor != "" {
		var err error
		cursorData, err = decodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode cursor: %w", err)
		}
	}

	// Apply cursor filtering
	if cursorData != nil {
		operator := ">"
		order := "ASC"
		if params.Direction == "prev" {
			operator = "<"
			order = "DESC"
		}
		query = query.Where(fmt.Sprintf("%s %s ?", orderField, operator), cursorData.ID)
		query = query.Order(fmt.Sprintf("%s %s", orderField, order))
	} else {
		// Default ordering for first page
		order := "ASC"
		if params.Direction == "prev" {
			order = "DESC"
		}
		query = query.Order(fmt.Sprintf("%s %s", orderField, order))
	}

	// Fetch one extra record to check if there are more pages
	query = query.Limit(params.PageSize + 1)

	if err := query.Find(dest).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to fetch cursor paginated data: %w", err)
	}

	hasNext := len(*dest) > params.PageSize
	hasPrev := cursorData != nil

	// Remove the extra record if present
	var data []T
	if hasNext {
		data = (*dest)[:params.PageSize]
	} else {
		data = *dest
	}

	// If we're going backwards, reverse the results
	if params.Direction == "prev" {
		reverseSlice(&data)
	}

	pagination := &CursorPagination{
		HasNext:  hasNext,
		HasPrev:  hasPrev,
		PageSize: params.PageSize,
	}

	// Generate cursors
	if len(data) > 0 {
		if hasNext || params.Direction == "prev" {
			nextCursor, err := encodeCursor(getFieldValue(data[len(data)-1], orderField), orderField)
			if err == nil {
				pagination.NextCursor = nextCursor
			}
		}

		if hasPrev || params.Direction == "next" {
			prevCursor, err := encodeCursor(getFieldValue(data[0], orderField), orderField)
			if err == nil {
				pagination.PrevCursor = prevCursor
			}
		}
	}

	return pagination, data, nil
}

// ParseCursorParams parses cursor pagination parameters from query strings.
// This is a convenience function for HTTP handlers that need to convert
// string parameters to validated CursorParams.
//
// Example usage in HTTP handler:
//
//	params := pagination.ParseCursorParams(
//		r.URL.Query().Get("cursor"),
//		r.URL.Query().Get("page_size"),
//		r.URL.Query().Get("direction"),
//	)
//
// Invalid or missing parameters will use sensible defaults.
func ParseCursorParams(cursor, pageSize, direction string) CursorParams {
	size := 20
	if pageSize != "" {
		if parsed, err := strconv.Atoi(pageSize); err == nil && parsed > 0 && parsed <= 100 {
			size = parsed
		}
	}

	return NewCursorParams(cursor, size, direction)
}

// encodeCursor creates a base64 encoded cursor from an ID and field name.
// The cursor contains both the field value and the field name to ensure
// consistency across different queries.
func encodeCursor(id interface{}, field string) (string, error) {
	cursorData := CursorData{
		ID:    id,
		Field: field,
	}

	jsonData, err := json.Marshal(cursorData)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(jsonData), nil
}

// decodeCursor decodes a base64 cursor back to CursorData.
// Returns an error if the cursor is malformed or cannot be decoded.
func decodeCursor(cursor string) (*CursorData, error) {
	jsonData, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, err
	}

	var cursorData CursorData
	if err := json.Unmarshal(jsonData, &cursorData); err != nil {
		return nil, err
	}

	return &cursorData, nil
}

// getFieldValue extracts the value of a specific field from a struct using reflection.
// It supports various field name formats including exact matches, case-insensitive matches,
// JSON tag names, and GORM column names.
func getFieldValue(item interface{}, fieldName string) interface{} {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Try direct field access first
	field := v.FieldByName(strings.Title(fieldName))
	if field.IsValid() {
		return field.Interface()
	}

	// Try case-insensitive field access
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		if strings.EqualFold(fieldType.Name, fieldName) {
			return v.Field(i).Interface()
		}

		// Check json tags
		if jsonTag := fieldType.Tag.Get("json"); jsonTag != "" {
			tagName := strings.Split(jsonTag, ",")[0]
			if strings.EqualFold(tagName, fieldName) {
				return v.Field(i).Interface()
			}
		}

		// Check gorm column tags
		if gormTag := fieldType.Tag.Get("gorm"); gormTag != "" {
			if strings.Contains(strings.ToLower(gormTag), "column:"+strings.ToLower(fieldName)) {
				return v.Field(i).Interface()
			}
		}
	}

	return nil
}

// reverseSlice reverses a slice in place.
// This is used when paginating backwards to maintain correct order.
func reverseSlice[T any](slice *[]T) {
	s := *slice
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
