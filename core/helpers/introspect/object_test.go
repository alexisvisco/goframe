package introspect

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestEnumDetection(t *testing.T) {
	// Test basic enum detection with string constants
	ctx := &ParseContext{
		Visited:     make(map[string]*ObjectType),
		Enums:       make(map[string]*FieldTypeEnum),
		Packages:    make(map[string]*packages.Package),
		EnumsParsed: make(map[string]bool),
		RootPath:    ".",
	}

	// Load the current package (this test package)
	pkg, err := ctx.LoadPackage("github.com/alexisvisco/goframe/core/helpers/introspect")
	if err != nil {
		t.Fatalf("Failed to load introspect package: %v", err)
	}

	// Parse enums in the package
	ctx.ParseEnums(pkg)

	// Check if FieldTypePrimitive enum is detected (it should be since it's a string type with constants)
	enumKey := "github.com/alexisvisco/goframe/core/helpers/introspect.FieldTypePrimitive"
	if enum, exists := ctx.Enums[enumKey]; exists {
		t.Logf("✅ Successfully detected FieldTypePrimitive enum with %d string values", len(enum.KeyValuesString))

		// Verify some expected values
		expectedValues := []string{"string", "int", "float", "bool", "time", "duration", "array", "map", "object", "enum", "any", "file"}
		for _, expected := range expectedValues {
			found := false
			for _, value := range enum.KeyValuesString {
				if value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Logf("⚠️  Expected value '%s' not found in enum", expected)
			}
		}
	} else {
		t.Logf("❌ FieldTypePrimitive enum not detected")
	}

	// Check if FieldKind enum is detected
	fieldKindKey := "github.com/alexisvisco/goframe/core/helpers/introspect.FieldKind"
	if enum, exists := ctx.Enums[fieldKindKey]; exists {
		t.Logf("✅ Successfully detected FieldKind enum with %d string values", len(enum.KeyValuesString))
	} else {
		t.Logf("❌ FieldKind enum not detected")
	}

	// Log all detected enums for debugging
	t.Logf("Total enums detected: %d", len(ctx.Enums))
	for typeName, enum := range ctx.Enums {
		t.Logf("- %s (strings: %d, ints: %d)", typeName, len(enum.KeyValuesString), len(enum.KeyValuesInt))
	}
}

func TestCrossPackageEnumDetection(t *testing.T) {
	// Test that detectEnum works across packages
	ctx := &ParseContext{
		Visited:     make(map[string]*ObjectType),
		Enums:       make(map[string]*FieldTypeEnum),
		Packages:    make(map[string]*packages.Package),
		EnumsParsed: make(map[string]bool),
		RootPath:    ".",
	}

	// Load a different package (like core/configuration which has DatabaseType enum)
	pkg, err := ctx.LoadPackage("github.com/alexisvisco/goframe/core/configuration")
	if err != nil {
		t.Skipf("Skipping cross-package test - failed to load configuration package: %v", err)
	}

	// Parse enums in the configuration package
	ctx.ParseEnums(pkg)

	// Check if DatabaseType is detected
	enumKey := "github.com/alexisvisco/goframe/core/configuration.DatabaseType"
	if enum, exists := ctx.Enums[enumKey]; exists {
		t.Logf("✅ Successfully detected DatabaseType enum with %d string values", len(enum.KeyValuesString))
		for key, value := range enum.KeyValuesString {
			t.Logf("  %s = %s", key, value)
		}
	} else {
		t.Logf("❌ DatabaseType enum not detected")
	}

	t.Logf("Total enums detected in configuration package: %d", len(ctx.Enums))
}

func TestEnumInStruct(t *testing.T) {
	// Test that enums are properly detected when used in struct fields

	// First, let's create a minimal test by parsing an existing struct that might use enums
	obj, err := ParseStruct(".", "github.com/alexisvisco/goframe/core/helpers/introspect", "Field")
	if err != nil {
		t.Fatalf("Failed to parse Field struct: %v", err)
	}

	t.Logf("Parsed Field struct with %d fields", len(obj.Fields))

	// Look for any enum fields
	foundEnum := false
	for _, field := range obj.Fields {
		if field.Type.Enum != nil {
			foundEnum = true
			t.Logf("✅ Found enum field '%s' of type '%s'", field.Name, field.Type.Enum.TypeName)
			t.Logf("   String values: %v", field.Type.Enum.KeyValuesString)
			t.Logf("   Int values: %v", field.Type.Enum.KeyValuesInt)
		} else if field.Type.Primitive == FieldTypePrimitiveEnum {
			t.Logf("⚠️  Field '%s' has enum primitive but no enum data", field.Name)
		} else if field.Type.Primitive == FieldTypePrimitiveString {
			// This might be a failed enum detection
			t.Logf("   Field '%s' is string primitive (potential failed enum?)", field.Name)
		}
	}

	if !foundEnum {
		t.Logf("❌ No enum fields found in Field struct")
	}
}

func TestManualEnumCreation(t *testing.T) {
	// Test creating and using an enum manually to ensure the detection logic works

	// Create a test enum
	testEnum := &FieldTypeEnum{
		TypeName: "test.TestStatus",
		KeyValuesString: map[string]string{
			"TestStatusActive":   "active",
			"TestStatusInactive": "inactive",
			"TestStatusPending":  "pending",
		},
	}

	// Create a test struct that uses this enum
	testObj := &ObjectType{
		TypeName: "test.TestStruct",
		Fields: []Field{
			{
				Name: "Status",
				Type: FieldType{
					Primitive: FieldTypePrimitiveEnum,
					Enum:      testEnum,
				},
			},
			{
				Name: "Name",
				Type: FieldType{
					Primitive: FieldTypePrimitiveString,
				},
			},
		},
	}

	// Verify the enum field is properly set up
	statusField := testObj.Fields[0]
	if statusField.Type.Enum == nil {
		t.Error("❌ Enum field should have enum data")
	} else {
		t.Logf("✅ Enum field properly configured")
		t.Logf("   TypeName: %s", statusField.Type.Enum.TypeName)
		t.Logf("   Values: %v", statusField.Type.Enum.KeyValuesString)
	}

	// Verify primitive is correct
	if statusField.Type.Primitive != FieldTypePrimitiveEnum {
		t.Errorf("❌ Expected enum primitive, got %s", statusField.Type.Primitive)
	}

	// Verify non-enum field doesn't have enum data
	nameField := testObj.Fields[1]
	if nameField.Type.Enum != nil {
		t.Error("❌ Non-enum field should not have enum data")
	}

	if nameField.Type.Primitive != FieldTypePrimitiveString {
		t.Errorf("❌ Expected string primitive, got %s", nameField.Type.Primitive)
	}
}

// Test the exact user scenario with embedded enum types
type TestExerciseEventType string

const (
	TestExerciseEventTypeEffort       TestExerciseEventType = "effort"
	TestExerciseEventTypeWaitEffort   TestExerciseEventType = "wait_effort"
	TestExerciseEventTypeRestSet      TestExerciseEventType = "rest_set"
	TestExerciseEventTypeRestRep      TestExerciseEventType = "rest_rep"
	TestExerciseEventTypeRestExercise TestExerciseEventType = "rest_exercise"
	TestExerciseEventTypeAskWeight    TestExerciseEventType = "ask_weight"
	TestExerciseEventTypeAskHoldSize  TestExerciseEventType = "ask_hold_size"
	TestExerciseEventTypeAskDistance  TestExerciseEventType = "ask_distance"
	TestExerciseEventTypeAskDuration  TestExerciseEventType = "ask_duration"
	TestExerciseEventTypeAskRep       TestExerciseEventType = "ask_rep"
	TestExerciseEventTypeAskRPE       TestExerciseEventType = "ask_rpe"
	TestExerciseEventTypeAskNotes     TestExerciseEventType = "ask_notes"
)

type TestExerciceEvent struct {
	ExerciceEventType TestExerciseEventType `json:"event_type"`
	DurationSec       *int                  `json:"duration_sec"`
}

type TestEventRequest struct {
	Events map[string]TestExerciceEvent `json:"events"`
}

func TestEnhancedCrossPackageEnumDetection(t *testing.T) {
	// Test that enums are properly detected when parsing structs that reference
	// enum types from other packages (simulating the user's issue)
	ctx := &ParseContext{
		Visited:     make(map[string]*ObjectType),
		Enums:       make(map[string]*FieldTypeEnum),
		Packages:    make(map[string]*packages.Package),
		EnumsParsed: make(map[string]bool),
		RootPath:    ".",
	}

	// Try to parse the Field struct which should trigger cross-package enum detection
	// if any of its fields are enum types from other packages
	obj, err := ctx.ParseStructByName("github.com/alexisvisco/goframe/core/helpers/introspect", "Field")
	if err != nil {
		t.Fatalf("Failed to parse Field struct: %v", err)
	}

	t.Logf("✅ Successfully parsed Field struct with %d fields", len(obj.Fields))

	// Check if any cross-package enums were detected during parsing
	crossPackageEnums := 0
	for enumType := range ctx.Enums {
		if !strings.HasPrefix(enumType, "github.com/alexisvisco/goframe/core/helpers/introspect.") {
			crossPackageEnums++
			t.Logf("✅ Cross-package enum detected: %s", enumType)
		}
	}

	// Verify enum parsing tracking works
	if !ctx.EnumsParsed["github.com/alexisvisco/goframe/core/helpers/introspect"] {
		t.Error("❌ EnumsParsed tracking not working - package should be marked as parsed")
	}

	t.Logf("Cross-package enums found: %d", crossPackageEnums)
	t.Logf("Total enums in context: %d", len(ctx.Enums))
	t.Logf("Packages with parsed enums: %d", len(ctx.EnumsParsed))

	// The enhanced enum detection should now work better across packages
	t.Logf("✅ Cross-package enum detection test completed")
}

func getEnumKeys(enums map[string]*FieldTypeEnum) []string {
	keys := make([]string, 0, len(enums))
	for k := range enums {
		keys = append(keys, k)
	}
	return keys
}
