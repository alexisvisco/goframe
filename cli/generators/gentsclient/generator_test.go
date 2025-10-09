package gentsclient

import (
	"strings"
	"testing"

	"regexp"

	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/http/apidoc"
)

func TestRouteWithNoRequestFields(t *testing.T) {
	// Create a route with no serializable request fields
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create a request object with only ctx fields
	requestObj := introspect.ObjectType{
		TypeName: "test.MeRequest",
		Fields: []introspect.Field{
			{
				Name: "ctx",
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindCtx, Value: ""},
				},
			},
		},
	}

	// Create a simple response object
	responseObj := introspect.ObjectType{
		TypeName: "test.MeResponse",
		Fields: []introspect.Field{
			{
				Name: "ID",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "id"},
				},
			},
		},
	}

	route := apidoc.Route{
		Name:    "me",
		Request: &requestObj,
		Paths: map[string][]string{
			"/v1/users/@me": {"GET"},
		},
		StatusToResponse: []apidoc.StatusToResponse{
			{
				StatusPattern: regexp.MustCompile(`^2[0-9]{2}$`),
				Response:      &responseObj,
			},
		},
	}

	// Add schemas and route
	generator.AddSchema("", true, requestObj)
	generator.AddSchema("", false, responseObj)
	generator.AddRoute(route)

	// Generate the TypeScript code
	result := generator.File()

	// Verify that the function signature doesn't include a request parameter
	if strings.Contains(result, "function me(fetcher: Fetcher, request:") {
		t.Error("Expected function without request parameter, but found request parameter")
	}

	if !strings.Contains(result, "function me(fetcher: Fetcher): Promise<") {
		t.Error("Expected function with only fetcher parameter")
	}

	// Verify that there's no parsing logic
	if strings.Contains(result, "safeParse(request)") {
		t.Error("Expected no request parsing logic, but found safeParse")
	}

	// Check that the error handling in the function doesn't reference RequestParseError
	// Extract the me function code
	meStart := strings.Index(result, "export async function me(")
	if meStart == -1 {
		t.Error("Could not find me function in generated code")
	} else {
		// Find the end of the me function (next function or end of namespace)
		meEnd := strings.Index(result[meStart:], "\n  export async function")
		if meEnd == -1 {
			meEnd = strings.Index(result[meStart:], "\n}")
		}
		if meEnd == -1 {
			meEnd = len(result)
		} else {
			meEnd += meStart
		}

		meFunctionCode := result[meStart:meEnd]
		if strings.Contains(meFunctionCode, "RequestParseError") {
			t.Error("Expected no RequestParseError in me function error handling, but found it")
		}
	}
}

func TestRecursiveStructGeneration(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create the ExerciseEventType enum
	eventTypeEnum := &introspect.FieldTypeEnum{
		TypeName: "test.ExerciseEventType",
		KeyValuesString: map[string]string{
			"ExerciseEventTypeStart": "start",
			"ExerciseEventTypeEnd":   "end",
		},
	}

	// Create the ExerciseEventKind enum
	eventKindEnum := &introspect.FieldTypeEnum{
		TypeName: "test.ExerciseEventKind",
		KeyValuesString: map[string]string{
			"ExerciseEventKindRep":  "rep",
			"ExerciseEventKindSet":  "set",
			"ExerciseEventKindRest": "rest",
		},
	}

	// Create a self-referencing ExerciseEvent struct
	exerciseEventObj := introspect.ObjectType{
		TypeName: "test.ExerciseEvent",
		Fields: []introspect.Field{
			{
				Name: "ExerciseEventType",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveEnum,
					Enum:      eventTypeEnum,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "event_type"},
				},
			},
			{
				Name: "Kind",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveEnum,
					Enum:      eventKindEnum,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "kind"},
				},
			},
			{
				Name: "DurationSec",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveInt,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "duration_sec", Options: []string{"omitempty"}},
				},
				Optional: true,
			},
			{
				Name: "SetNumber",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveInt,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "set_number", Options: []string{"omitempty"}},
				},
				Optional: true,
			},
			{
				Name: "RepetitionNumber",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveInt,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "repetition_number", Options: []string{"omitempty"}},
				},
				Optional: true,
			},
		},
	}

	// Add the self-referencing Loop field - this creates the recursion
	loopField := introspect.Field{
		Name: "Loop",
		Type: introspect.FieldType{
			Array: &introspect.FieldTypeArray{
				ItemType: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveObject,
					Object:    &exerciseEventObj,
				},
			},
		},
		Tags: []introspect.FieldTag{
			{Key: introspect.FieldKindJSON, Value: "loop", Options: []string{"omitempty"}},
		},
		Optional: true,
	}
	exerciseEventObj.Fields = append(exerciseEventObj.Fields, loopField)

	// Add the schema
	generator.AddSchema("", false, exerciseEventObj)

	// Generate the TypeScript code
	result := generator.File()

	// Verify that z.lazy() is used for the recursive schema
	if !strings.Contains(result, "export const exerciseEventSchema: z.ZodType<ExerciseEvent> = z.lazy(() =>") {
		t.Error("Expected exerciseEventSchema to use z.lazy() for recursive type")
	}

	// Verify the schema structure includes the recursive array field
	if !strings.Contains(result, "loop: z.array(exerciseEventSchema).optional()") {
		t.Error("Expected loop field to reference exerciseEventSchema recursively")
	}

	// Verify the closing parenthesis for z.lazy()
	if !strings.Contains(result, ").passthrough(),\n);") {
		t.Error("Expected proper closing for z.lazy() schema")
	}

	// Verify TypeScript interface is still generated correctly
	if !strings.Contains(result, "export interface ExerciseEvent {") {
		t.Error("Expected ExerciseEvent interface to be generated")
	}

	if !strings.Contains(result, "loop?: Array<ExerciseEvent>;") {
		t.Error("Expected loop field to be optional array of ExerciseEvent in interface")
	}

	// Verify enums are still generated correctly
	if !strings.Contains(result, "export const ExerciseEventTypeEnum") {
		t.Error("Expected ExerciseEventType enum to be generated")
	}

	if !strings.Contains(result, "export const ExerciseEventKindEnum") {
		t.Error("Expected ExerciseEventKind enum to be generated")
	}
}

func TestNonRecursiveStructRemainsSame(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create a simple non-recursive struct
	simpleObj := introspect.ObjectType{
		TypeName: "test.SimpleObject",
		Fields: []introspect.Field{
			{
				Name: "ID",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "id"},
				},
			},
			{
				Name: "Name",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "name"},
				},
			},
		},
	}

	// Add the schema
	generator.AddSchema("", false, simpleObj)

	// Generate the TypeScript code
	result := generator.File()

	// Verify that z.lazy() is NOT used for non-recursive types
	if strings.Contains(result, "z.lazy()") {
		t.Error("Expected non-recursive schema to NOT use z.lazy()")
	}

	// Verify normal schema generation
	if !strings.Contains(result, "export const simpleObjectSchema = z.object({") {
		t.Error("Expected normal schema declaration for non-recursive type")
	}

	// Verify normal closing
	if !strings.Contains(result, "}).passthrough();") {
		t.Error("Expected normal closing for non-recursive schema")
	}
}

func TestRouteWithRequestFields(t *testing.T) {
	// Create a route with serializable request fields
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create a request object with serializable fields
	requestObj := introspect.ObjectType{
		TypeName: "test.GetUserRequest",
		Fields: []introspect.Field{
			{
				Name: "ID",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindPath, Value: "id"},
				},
			},
		},
	}

	// Create a simple response object
	responseObj := introspect.ObjectType{
		TypeName: "test.GetUserResponse",
		Fields: []introspect.Field{
			{
				Name: "ID",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "id"},
				},
			},
		},
	}

	route := apidoc.Route{
		Name:    "getUser",
		Request: &requestObj,
		Paths: map[string][]string{
			"/v1/users/{id}": {"GET"},
		},
		StatusToResponse: []apidoc.StatusToResponse{
			{
				StatusPattern: regexp.MustCompile(`^2[0-9]{2}$`),
				Response:      &responseObj,
			},
		},
	}

	// Add schemas and route
	generator.AddSchema("", true, requestObj)
	generator.AddSchema("", false, responseObj)
	generator.AddRoute(route)

	// Generate the TypeScript code
	result := generator.File()

	// Verify that the function signature includes a request parameter
	if !strings.Contains(result, "function getUser(fetcher: Fetcher, request:") {
		t.Error("Expected function with request parameter, but found no request parameter")
	}

	// Verify that there's parsing logic
	if !strings.Contains(result, "safeParse(request)") {
		t.Error("Expected request parsing logic, but found no safeParse")
	}
}

func TestHasRequestFieldsWithOnlyCtxTags(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create a route with only ctx fields
	requestObj := introspect.ObjectType{
		TypeName: "test.TestRequest",
		Fields: []introspect.Field{
			{
				Name: "ctx",
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindCtx, Value: ""},
				},
			},
		},
	}

	route := apidoc.Route{
		Name:    "test",
		Request: &requestObj,
	}

	// Should return false since only ctx tags are present
	hasFields := generator.hasRequestFields(route)
	if hasFields {
		t.Error("Expected hasRequestFields to return false for ctx-only fields")
	}
}

func TestHasRequestFieldsWithSerializableFields(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create a route with serializable fields
	requestObj := introspect.ObjectType{
		TypeName: "test.TestRequest",
		Fields: []introspect.Field{
			{
				Name: "query",
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindQuery, Value: "q"},
				},
			},
		},
	}

	route := apidoc.Route{
		Name:    "test",
		Request: &requestObj,
	}

	// Should return true since serializable fields are present
	hasFields := generator.hasRequestFields(route)
	if !hasFields {
		t.Error("Expected hasRequestFields to return true for serializable fields")
	}
}

func TestPointerTypesAreOptional(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create an object with pointer and non-pointer fields
	requestObj := introspect.ObjectType{
		TypeName: "test.TestRequest",
		Fields: []introspect.Field{
			{
				Name: "RequiredField",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "required_field"},
				},
				Optional: false, // Non-pointer field
			},
			{
				Name: "OptionalField",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "optional_field"},
				},
				Optional: true, // Pointer field (should be optional)
			},
		},
	}

	// Create a response object with pointer and non-pointer fields
	responseObj := introspect.ObjectType{
		TypeName: "test.TestResponse",
		Fields: []introspect.Field{
			{
				Name: "ID",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "id"},
				},
				Optional: false, // Non-pointer field
			},
			{
				Name: "Name",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "name"},
				},
				Optional: true, // Pointer field (should be optional)
			},
		},
	}

	// Add schemas
	generator.AddSchema("", true, requestObj)
	generator.AddSchema("", false, responseObj)

	// Generate the TypeScript code
	result := generator.File()

	// Verify Zod schema has .optional() for pointer fields
	if !strings.Contains(result, "required_field: z.string(),") {
		t.Error("Expected required field to not have .optional() in Zod schema")
	}

	if !strings.Contains(result, "optional_field: z.string().optional(),") {
		t.Error("Expected optional field to have .optional() in Zod schema")
	}

	// Verify TypeScript interface has ? for pointer fields
	if !strings.Contains(result, "required_field: string;") {
		t.Error("Expected required field to not have ? in TypeScript interface")
	}

	if !strings.Contains(result, "optional_field?: string;") {
		t.Error("Expected optional field to have ? in TypeScript interface")
	}

	// Check response interface as well
	if !strings.Contains(result, "id: string;") {
		t.Error("Expected required response field to not have ? in TypeScript interface")
	}

	if !strings.Contains(result, "name?: string;") {
		t.Error("Expected optional response field to have ? in TypeScript interface")
	}
}

func TestSliceOfPointersNotAutomaticallyOptional(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create an object with a slice of pointers
	requestObj := introspect.ObjectType{
		TypeName: "test.TestRequest",
		Fields: []introspect.Field{
			{
				Name: "Items",
				Type: introspect.FieldType{
					Array: &introspect.FieldTypeArray{
						ItemType: introspect.FieldType{
							Primitive: introspect.FieldTypePrimitiveString,
						},
					},
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "items"},
				},
				Optional: false, // Slice itself should not be optional just because items are pointers
			},
			{
				Name: "OptionalItems",
				Type: introspect.FieldType{
					Array: &introspect.FieldTypeArray{
						ItemType: introspect.FieldType{
							Primitive: introspect.FieldTypePrimitiveString,
						},
					},
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "optional_items", Options: []string{"omitempty"}},
				},
				Optional: true, // This should be optional because of omitempty tag
			},
		},
	}

	// Add schema
	generator.AddSchema("", true, requestObj)

	// Generate the TypeScript code
	result := generator.File()

	// Verify that slice field without omitempty is required
	if !strings.Contains(result, "items: z.array(z.string()),") {
		t.Error("Expected items field to be required (no .optional()) in Zod schema")
	}

	// Verify that slice field with omitempty is optional
	if !strings.Contains(result, "optional_items: z.array(z.string()).optional(),") {
		t.Error("Expected optional_items field to have .optional() in Zod schema")
	}

	// Verify TypeScript interface
	if !strings.Contains(result, "items: Array<string>;") {
		t.Error("Expected items field to be required (no ?) in TypeScript interface")
	}

	if !strings.Contains(result, "optional_items?: Array<string>;") {
		t.Error("Expected optional_items field to have ? in TypeScript interface")
	}
}

func TestComprehensivePointerBehavior(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create an object demonstrating different pointer scenarios
	requestObj := introspect.ObjectType{
		TypeName: "test.PointerTestRequest",
		Fields: []introspect.Field{
			{
				Name: "PointerString",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "pointer_string"},
				},
				Optional: true, // *string - should be optional
			},
			{
				Name: "RegularString",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveString,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "regular_string"},
				},
				Optional: false, // string - should be required
			},
			{
				Name: "SliceOfPointers",
				Type: introspect.FieldType{
					Array: &introspect.FieldTypeArray{
						ItemType: introspect.FieldType{
							Primitive: introspect.FieldTypePrimitiveString,
						},
					},
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "slice_of_pointers"},
				},
				Optional: false, // []*string without omitempty - should be required
			},
			{
				Name: "OptionalSlice",
				Type: introspect.FieldType{
					Array: &introspect.FieldTypeArray{
						ItemType: introspect.FieldType{
							Primitive: introspect.FieldTypePrimitiveString,
						},
					},
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "optional_slice", Options: []string{"omitempty"}},
				},
				Optional: true, // []*string with omitempty - should be optional
			},
		},
	}

	// Add schema
	generator.AddSchema("", false, requestObj)

	// Generate the TypeScript code
	result := generator.File()

	// Test Zod schema generation
	// Pointer string should be optional
	if !strings.Contains(result, "pointer_string: z.string().optional(),") {
		t.Error("Expected pointer_string to have .optional() in Zod schema")
	}

	// Regular string should be required
	if !strings.Contains(result, "regular_string: z.string(),") {
		t.Error("Expected regular_string to be required (no .optional()) in Zod schema")
	}

	// Slice of pointers should be required (array itself)
	if !strings.Contains(result, "slice_of_pointers: z.array(z.string()),") {
		t.Error("Expected slice_of_pointers to be required (no .optional()) in Zod schema")
	}

	// Optional slice should be optional
	if !strings.Contains(result, "optional_slice: z.array(z.string()).optional(),") {
		t.Error("Expected optional_slice to have .optional() in Zod schema")
	}

	// Test TypeScript interface generation
	// Pointer string should be optional
	if !strings.Contains(result, "pointer_string?: string;") {
		t.Error("Expected pointer_string to have ? in TypeScript interface")
	}

	// Regular string should be required
	if !strings.Contains(result, "regular_string: string;") {
		t.Error("Expected regular_string to be required (no ?) in TypeScript interface")
	}

	// Slice of pointers should be required
	if !strings.Contains(result, "slice_of_pointers: Array<string>;") {
		t.Error("Expected slice_of_pointers to be required (no ?) in TypeScript interface")
	}

	// Optional slice should be optional
	if !strings.Contains(result, "optional_slice?: Array<string>;") {
		t.Error("Expected optional_slice to have ? in TypeScript interface")
	}
}

func TestEnumDetection(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create an enum type
	statusEnum := &introspect.FieldTypeEnum{
		TypeName: "test.StatusType",
		KeyValuesString: map[string]string{
			"StatusTypeActive":   "active",
			"StatusTypePending":  "pending",
			"StatusTypeInactive": "inactive",
		},
	}

	// Create an object that uses the enum
	requestObj := introspect.ObjectType{
		TypeName: "test.StatusRequest",
		Fields: []introspect.Field{
			{
				Name: "Status",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveEnum,
					Enum:      statusEnum,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "status"},
				},
				Optional: false,
			},
			{
				Name: "OptionalStatus",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveEnum,
					Enum:      statusEnum,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "optional_status"},
				},
				Optional: true,
			},
		},
	}

	// Add schema
	generator.AddSchema("", false, requestObj)

	// Generate the TypeScript code
	result := generator.File()

	// Verify enum constants are generated
	if !strings.Contains(result, "export const StatusTypeEnum = {") {
		t.Error("Expected StatusTypeEnum constant object to be generated")
	}

	if !strings.Contains(result, "ACTIVE: 'active',") {
		t.Error("Expected ACTIVE enum value in enum constant")
	}

	if !strings.Contains(result, "PENDING: 'pending',") {
		t.Error("Expected PENDING enum value in enum constant")
	}

	if !strings.Contains(result, "INACTIVE: 'inactive',") {
		t.Error("Expected INACTIVE enum value in enum constant")
	}

	// Verify enum type is generated
	if !strings.Contains(result, "export type StatusTypeEnum = ValueOf<typeof StatusTypeEnum>;") {
		t.Error("Expected StatusTypeEnum type to be generated")
	}

	// Verify Zod schema for enum is generated
	if !strings.Contains(result, "export const statusTypeEnumSchema = z.union([") {
		t.Error("Expected statusTypeEnumSchema Zod union to be generated")
	}

	if !strings.Contains(result, "z.literal('active')") {
		t.Error("Expected z.literal('active') in Zod schema")
	}

	if !strings.Contains(result, "z.literal('pending')") {
		t.Error("Expected z.literal('pending') in Zod schema")
	}

	if !strings.Contains(result, "z.literal('inactive')") {
		t.Error("Expected z.literal('inactive') in Zod schema")
	}

	// Verify object schema uses enum schema
	if !strings.Contains(result, "status: statusTypeEnumSchema,") {
		t.Error("Expected status field to use statusTypeEnumSchema")
	}

	if !strings.Contains(result, "optional_status: statusTypeEnumSchema.optional(),") {
		t.Error("Expected optional_status field to use statusTypeEnumSchema.optional()")
	}

	// Verify TypeScript interface uses enum type
	if !strings.Contains(result, "status: StatusTypeEnum;") {
		t.Error("Expected status field to use StatusTypeEnum type")
	}

	if !strings.Contains(result, "optional_status?: StatusTypeEnum;") {
		t.Error("Expected optional_status field to use StatusTypeEnum type with optional marker")
	}
}

func TestEnumInNestedStructure(t *testing.T) {
	generator := NewTypescriptClientGenerator("test/pkg", map[string]string{})

	// Create an enum type
	eventTypeEnum := &introspect.FieldTypeEnum{
		TypeName: "test.EventType",
		KeyValuesString: map[string]string{
			"EventTypeStart": "start",
			"EventTypeEnd":   "end",
		},
	}

	// Create a nested struct that uses the enum
	eventObj := introspect.ObjectType{
		TypeName: "test.Event",
		Fields: []introspect.Field{
			{
				Name: "Type",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveEnum,
					Enum:      eventTypeEnum,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "type"},
				},
			},
			{
				Name: "Duration",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveInt,
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "duration"},
				},
				Optional: true,
			},
		},
	}

	// Create a request that uses the event in a map
	requestObj := introspect.ObjectType{
		TypeName: "test.EventMapRequest",
		Fields: []introspect.Field{
			{
				Name: "Events",
				Type: introspect.FieldType{
					Primitive: introspect.FieldTypePrimitiveMap,
					Map: &introspect.FieldTypeMap{
						Key: introspect.FieldType{
							Primitive: introspect.FieldTypePrimitiveString,
						},
						Value: introspect.FieldType{
							Primitive: introspect.FieldTypePrimitiveObject,
							Object:    &eventObj,
						},
					},
				},
				Tags: []introspect.FieldTag{
					{Key: introspect.FieldKindJSON, Value: "events"},
				},
			},
		},
	}

	// Add schemas - this simulates the nested processing that happens in real usage
	generator.AddSchema("", false, eventObj)
	generator.AddSchema("", false, requestObj)

	// Generate the TypeScript code
	result := generator.File()

	// Verify enum is still detected and generated correctly
	if !strings.Contains(result, "export const EventTypeEnum = {") {
		t.Error("Expected EventTypeEnum to be generated even when used in nested structure")
	}

	if !strings.Contains(result, "START: 'start',") {
		t.Error("Expected START enum value")
	}

	if !strings.Contains(result, "END: 'end',") {
		t.Error("Expected END enum value")
	}

	// Verify the enum is used in the nested object schema
	if !strings.Contains(result, "type: eventTypeEnumSchema,") {
		t.Error("Expected nested object to use eventTypeEnumSchema")
	}

	// Verify the map structure is correct
	if !strings.Contains(result, "events: z.record(z.string(), eventSchema),") {
		t.Error("Expected events field to be a record/map with Event objects")
	}

	// Verify TypeScript interface for the nested structure
	if !strings.Contains(result, "type: EventTypeEnum;") {
		t.Error("Expected Event interface to use EventTypeEnum")
	}

	if !strings.Contains(result, "events: Record<string, Event>;") {
		t.Error("Expected events field to be Record<string, Event>")
	}
}

func TestTimeTypeGenerationBug(t *testing.T) {
	generator := NewTypescriptClientGenerator("internal/testhandler", make(map[string]string))

	// Create a struct that contains a time.Time field
	dateObject := introspect.ObjectType{
		TypeName: "DateObject",
		Fields: []introspect.Field{
			{
				Name:     "Time",
				Type:     introspect.FieldType{Primitive: introspect.FieldTypePrimitiveTime},
				Optional: false,
				Tags:     []introspect.FieldTag{{Key: introspect.FieldKindJSON, Value: "time"}},
			},
		},
	}

	generator.AddSchema("DateObject", false, dateObject)

	result := generator.File()

	// Check that dateSchema is correctly generated as a primitive
	expectedDateSchema := "const dateSchema = z.string().datetime({ offset: true }).transform((value) => new Date(value));"
	if !strings.Contains(result, expectedDateSchema) {
		t.Errorf("Expected dateSchema to be generated as primitive, but got:\n%s", result)
	}

	// Check that DateObject schema uses dateSchema correctly, not recursively
	expectedObjectSchema := "export const dateObjectDateObjectSchema = z.object({\n  time: dateSchema,\n}).passthrough();"
	if !strings.Contains(result, expectedObjectSchema) {
		t.Errorf("Expected DateObject schema to reference dateSchema correctly, but got:\n%s", result)
	}

	// Most importantly, ensure we don't have a recursive reference
	unexpectedRecursive := "export const dateSchema = z.object({\n  Time: dateSchema,"
	if strings.Contains(result, unexpectedRecursive) {
		t.Errorf("Found recursive dateSchema definition, which is incorrect:\n%s", result)
	}
}

func TestDateTypeNamingConflict(t *testing.T) {
	generator := NewTypescriptClientGenerator("internal/testhandler", make(map[string]string))

	// Create a struct named "Date" that contains a time.Time field
	// This reproduces the exact issue where a Go type "Date" creates a schema named "dateSchema"
	// which conflicts with the primitive dateSchema for time.Time
	dateStruct := introspect.ObjectType{
		TypeName: "Date",
		Fields: []introspect.Field{
			{
				Name:     "Time",
				Type:     introspect.FieldType{Primitive: introspect.FieldTypePrimitiveTime},
				Optional: false,
				Tags:     []introspect.FieldTag{{Key: introspect.FieldKindJSON, Value: "Time"}},
			},
		},
	}

	// First add the Date struct - this will create a dateSchema entry in lookup
	generator.AddSchema("", false, dateStruct)

	// Then try to generate the file, which should trigger the createDateSchema() call
	// when processing the Time field
	result := generator.File()

	// The bug should be fixed: we should NOT have a recursive definition
	problematicRecursive := "export const dateSchema = z.object({\n  Time: dateSchema,"
	if strings.Contains(result, problematicRecursive) {
		t.Errorf("BUG STILL EXISTS! Found recursive dateSchema definition:\n%s", result)
	}

	// What we should have after the fix:
	// 1. A primitive dateSchema for time.Time
	expectedPrimitiveDateSchema := "const dateSchema = z.string().datetime({ offset: true }).transform((value) => new Date(value));"
	if !strings.Contains(result, expectedPrimitiveDateSchema) {
		t.Errorf("Expected primitive dateSchema to be present, but got:\n%s", result)
	}

	// 2. A differently named schema for the Date struct (should be dateTypeSchema now)
	expectedStructSchema := "export const dateTypeSchema = z.object({\n  Time: dateSchema,"
	if !strings.Contains(result, expectedStructSchema) {
		t.Errorf("Expected Date struct to use dateTypeSchema name to avoid conflict, but got:\n%s", result)
	}

	// 3. Verify the Date struct schema correctly references the primitive dateSchema
	if !strings.Contains(result, "Time: dateSchema,") {
		t.Errorf("Expected Date struct to reference primitive dateSchema, but got:\n%s", result)
	}
}

func TestDurationTypeNamingConflict(t *testing.T) {
	generator := NewTypescriptClientGenerator("internal/testhandler", make(map[string]string))

	// Create a struct named "Duration" that contains a time.Duration field
	// This should also avoid naming conflicts like the Date type
	durationStruct := introspect.ObjectType{
		TypeName: "Duration",
		Fields: []introspect.Field{
			{
				Name:     "Value",
				Type:     introspect.FieldType{Primitive: introspect.FieldTypePrimitiveDuration},
				Optional: false,
				Tags:     []introspect.FieldTag{{Key: introspect.FieldKindJSON, Value: "Value"}},
			},
		},
	}

	generator.AddSchema("", false, durationStruct)
	result := generator.File()

	// Should not have recursive durationSchema definition
	problematicRecursive := "export const durationSchema = z.object({\n  Value: durationSchema,"
	if strings.Contains(result, problematicRecursive) {
		t.Errorf("Found recursive durationSchema definition:\n%s", result)
	}

	// Should have primitive durationSchema
	expectedPrimitiveDurationSchema := "const durationSchema = z.number().int().nonnegative().transform((value) => new Duration(value))"
	if !strings.Contains(result, expectedPrimitiveDurationSchema) {
		t.Errorf("Expected primitive durationSchema to be present, but got:\n%s", result)
	}

	// Should have durationTypeSchema for the Duration struct
	expectedStructSchema := "export const durationTypeSchema = z.object({\n  Value: durationSchema,"
	if !strings.Contains(result, expectedStructSchema) {
		t.Errorf("Expected Duration struct to use durationTypeSchema name, but got:\n%s", result)
	}
}

func TestErrorTypeNamingConflict(t *testing.T) {
	generator := NewTypescriptClientGenerator("internal/testhandler", make(map[string]string))

	// Create a struct named "Error" that contains some fields
	// This should avoid naming conflicts with the reserved errorSchema
	errorStruct := introspect.ObjectType{
		TypeName: "Error",
		Fields: []introspect.Field{
			{
				Name:     "Code",
				Type:     introspect.FieldType{Primitive: introspect.FieldTypePrimitiveString},
				Optional: false,
				Tags:     []introspect.FieldTag{{Key: introspect.FieldKindJSON, Value: "Code"}},
			},
			{
				Name:     "Message",
				Type:     introspect.FieldType{Primitive: introspect.FieldTypePrimitiveString},
				Optional: false,
				Tags:     []introspect.FieldTag{{Key: introspect.FieldKindJSON, Value: "Message"}},
			},
		},
	}

	generator.AddSchema("", false, errorStruct)
	result := generator.File()

	// Should not have recursive errorSchema definition
	problematicRecursive := "export const errorSchema = z.object({\n  Code: z.string(),"
	if strings.Contains(result, problematicRecursive) {
		t.Errorf("Found recursive errorSchema definition:\n%s", result)
	}

	// Should have primitive errorSchema
	expectedPrimitiveErrorSchema := "const errorSchema = z.object({\n\tmessage: z.string().optional(),"
	if !strings.Contains(result, expectedPrimitiveErrorSchema) {
		t.Errorf("Expected primitive errorSchema to be present, but got:\n%s", result)
	}

	// Should have errorTypeSchema for the Error struct
	expectedStructSchema := "export const errorTypeSchema = z.object({\n  Code: z.string(),"
	if !strings.Contains(result, expectedStructSchema) {
		t.Errorf("Expected Error struct to use errorTypeSchema name, but got:\n%s", result)
	}
}

func TestUserReportedTimeTimeBug(t *testing.T) {
	// This test reproduces the exact issue reported by the user:
	// "il y a un problème avec tsclient quand on a des time.Time il génère :
	// export const dateSchema = z.object({
	//   Time: dateSchema,
	// }).passthrough();
	// au lieu de :
	// const dateSchema = z.string().datetime({ offset: true }).transform((value) => new Date(value));"

	generator := NewTypescriptClientGenerator("internal/testhandler", make(map[string]string))

	// Simulate a Go struct that would cause this issue - a struct named something that
	// would generate "dateSchema" as its schema name
	problematicStruct := introspect.ObjectType{
		TypeName: "Date", // This would generate "dateSchema"
		Fields: []introspect.Field{
			{
				Name:     "Time",
				Type:     introspect.FieldType{Primitive: introspect.FieldTypePrimitiveTime},
				Optional: false,
				Tags:     []introspect.FieldTag{{Key: introspect.FieldKindJSON, Value: "Time"}},
			},
		},
	}

	generator.AddSchema("", false, problematicStruct)
	result := generator.File()

	// Before the fix, this would generate the problematic recursive reference:
	// export const dateSchema = z.object({
	//   Time: dateSchema,
	// }).passthrough();

	// After the fix, we should have:
	// 1. The correct primitive dateSchema
	expectedPrimitive := "const dateSchema = z.string().datetime({ offset: true }).transform((value) => new Date(value));"
	if !strings.Contains(result, expectedPrimitive) {
		t.Errorf("Missing correct primitive dateSchema. Expected:\n%s\n\nBut got:\n%s", expectedPrimitive, result)
	}

	// 2. A differently named schema for the Date struct to avoid conflict
	expectedStructSchema := "export const dateTypeSchema = z.object({\n  Time: dateSchema,\n}).passthrough();"
	if !strings.Contains(result, expectedStructSchema) {
		t.Errorf("Expected Date struct to use dateTypeSchema name. Expected:\n%s\n\nBut got:\n%s", expectedStructSchema, result)
	}

	// 3. Most importantly, we should NOT have the problematic recursive definition
	problematicPattern := "export const dateSchema = z.object({"
	if strings.Contains(result, problematicPattern) {
		t.Errorf("FAILURE: Still generating the problematic recursive dateSchema!\n\nGenerated:\n%s", result)
	}

	t.Logf("SUCCESS: User's issue has been fixed. Generated code:\n%s", result)
}
