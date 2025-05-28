package genhelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImportRegistry(t *testing.T) {
	tests := []struct {
		name          string
		importPath    string
		alias         string
		expectedAlias string
		description   string
	}{
		{
			name:          "basic_import_no_alias",
			importPath:    "github.com/gorilla/mux",
			alias:         "",
			expectedAlias: "mux",
			description:   "Basic import without alias should use package name",
		},
		{
			name:          "duplicate_import",
			importPath:    "github.com/gorilla/mux",
			alias:         "",
			expectedAlias: "mux",
			description:   "Same import added again should return same alias",
		},
		{
			name:          "custom_alias",
			importPath:    "database/sql",
			alias:         "db",
			expectedAlias: "db",
			description:   "Import with custom alias should use provided alias",
		},
		{
			name:          "alias_conflict_default",
			importPath:    "github.com/gin-gonic/gin/mux",
			alias:         "",
			expectedAlias: "mux1",
			description:   "Different import with same default alias should get incremental alias",
		},
		{
			name:          "alias_conflict_second",
			importPath:    "net/http/mux",
			alias:         "",
			expectedAlias: "mux2",
			description:   "Third import with same base alias should get mux2",
		},
		{
			name:          "custom_alias_conflict",
			importPath:    "github.com/lib/pq",
			alias:         "db",
			expectedAlias: "db1",
			description:   "Custom alias conflict should get incremental alias",
		},
		{
			name:          "empty_alias_uses_default",
			importPath:    "encoding/json",
			alias:         "",
			expectedAlias: "json",
			description:   "Empty alias should use package name",
		},
		{
			name:          "single_package_name",
			importPath:    "fmt",
			alias:         "",
			expectedAlias: "fmt",
			description:   "Single package name should work correctly",
		},
	}

	registry := newImportRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.alias == "" {
				result = registry.addImport(tt.importPath)
			} else {
				result = registry.addImport(tt.importPath, tt.alias)
			}

			assert.Equal(t, tt.expectedAlias, result, tt.description)
		})
	}

	// Verify registry state
	assert.Len(t, registry.imports, 7, "Should have 7 unique imports")
	assert.Contains(t, registry.alias, "github.com/gorilla/mux")
	assert.Equal(t, "mux", registry.alias["github.com/gorilla/mux"])
}
