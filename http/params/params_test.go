package params

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Define test structs
type TestUser struct {
	Name      string   `json:"name" xml:"name" form:"name" query:"name" headers:"X-User-Name" cookie:"user_name" default:"DefaultUser"`
	Age       int      `json:"age" xml:"age" form:"age" query:"age" headers:"X-User-Age" cookie:"user_age" default:"30"`
	IsActive  bool     `json:"is_active" xml:"is_active" form:"is_active" query:"is_active" headers:"X-User-Active" default:"true"`
	CreatedAt string   `json:"created_at" xml:"created_at" form:"created_at" query:"created_at" default:"2023-01-01"`
	Tags      []string `json:"tags" xml:"tags" form:"tags" query:"tags" headers:"X-User-Tags" exploder:","`
	UserID    string   `ctx:"userID" default:"guest"`
}

type UserWithFile struct {
	Name     string                  `form:"name" default:"DefaultUser"`
	Avatar   *multipart.FileHeader   `file:"avatar"`
	Pictures []*multipart.FileHeader `files:"pictures"`
}

type OrderedField struct {
	Field string
	Type  string
}

func (o *OrderedField) UnmarshalText(text []byte) error {
	parts := bytes.Split(text, []byte(":"))
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, expected field:order, got %s", text)
	}
	o.Field = string(parts[0])
	o.Type = string(parts[1])
	return nil
}

type Pagination struct {
	Page    uint            `query:"page" default:"1"`
	Limit   uint            `query:"limit" default:"50"`
	OrderBy []*OrderedField `query:"orderBy" exploder:"," default:"name:ASC"`
}

func TestBindJSON(t *testing.T) {
	// GenerateHandler test data
	jsonData := `{"name":"John Doe","age":25,"is_active":true,"created_at":"2023-01-15","tags":["tag1","tag2","tag3"]}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind JSON: %v", err)
	}

	// Verify bound data
	if user.Name != "John Doe" {
		t.Errorf("Expected Name to be 'John Doe', got '%s'", user.Name)
	}
	if user.Age != 25 {
		t.Errorf("Expected Age to be 25, got %d", user.Age)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if user.CreatedAt != "2023-01-15" {
		t.Errorf("Expected CreatedAt to be '2023-01-15', got '%s'", user.CreatedAt)
	}
	if len(user.Tags) != 3 || user.Tags[0] != "tag1" || user.Tags[1] != "tag2" || user.Tags[2] != "tag3" {
		t.Errorf("Tags not bound correctly, got %v", user.Tags)
	}
}

func TestBindXML(t *testing.T) {
	// GenerateHandler test data
	xmlData := `
<TestUser>
  <name>Jane Doe</name>
  <age>28</age>
  <is_active>true</is_active>
  <created_at>2023-02-15</created_at>
  <tags>tag1</tags>
  <tags>tag2</tags>
</TestUser>`
	req := httptest.NewRequest("POST", "/", strings.NewReader(xmlData))
	req.Header.Set("Content-Type", "application/xml")

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind XML: %v", err)
	}

	// Verify bound data
	if user.Name != "Jane Doe" {
		t.Errorf("Expected Name to be 'Jane Doe', got '%s'", user.Name)
	}
	if user.Age != 28 {
		t.Errorf("Expected Age to be 28, got %d", user.Age)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if user.CreatedAt != "2023-02-15" {
		t.Errorf("Expected CreatedAt to be '2023-02-15', got '%s'", user.CreatedAt)
	}
}

func TestBindForm(t *testing.T) {
	// GenerateHandler form data
	form := url.Values{}
	form.Add("name", "Alice Smith")
	form.Add("age", "35")
	form.Add("is_active", "true")
	form.Add("created_at", "2023-03-15")
	form.Add("tags[]", "tag1")
	form.Add("tags[]", "tag2")

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Form = form

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind form: %v", err)
	}

	// Verify bound data
	if user.Name != "Alice Smith" {
		t.Errorf("Expected Name to be 'Alice Smith', got '%s'", user.Name)
	}
	if user.Age != 35 {
		t.Errorf("Expected Age to be 35, got %d", user.Age)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if user.CreatedAt != "2023-03-15" {
		t.Errorf("Expected CreatedAt to be '2023-03-15', got '%s'", user.CreatedAt)
	}
	if len(user.Tags) != 2 || user.Tags[0] != "tag1" || user.Tags[1] != "tag2" {
		t.Errorf("Tags not bound correctly, got %v", user.Tags)
	}
}

func TestBindQuery(t *testing.T) {
	// GenerateHandler query parameters
	req := httptest.NewRequest("GET", "/user?name=Bob+Johnson&age=40&is_active=true&created_at=2023-04-15&tags[]=tag1&tags[]=tag2&tags[]=tag3", nil)

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind query: %v", err)
	}

	// Verify bound data
	if user.Name != "Bob Johnson" {
		t.Errorf("Expected Name to be 'Bob Johnson', got '%s'", user.Name)
	}
	if user.Age != 40 {
		t.Errorf("Expected Age to be 40, got %d", user.Age)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if user.CreatedAt != "2023-04-15" {
		t.Errorf("Expected CreatedAt to be '2023-04-15', got '%s'", user.CreatedAt)
	}
	if len(user.Tags) != 3 || user.Tags[0] != "tag1" || user.Tags[1] != "tag2" || user.Tags[2] != "tag3" {
		t.Errorf("Tags not bound correctly, got %v", user.Tags)
	}
}

func TestBindHeaders(t *testing.T) {
	// GenerateHandler headers
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-User-Name", "Charlie Brown")
	req.Header.Set("X-User-Age", "45")
	req.Header.Set("X-User-Active", "true")
	req.Header.Add("X-User-Tags", "tag1")
	req.Header.Add("X-User-Tags", "tag2")
	req.Header.Add("X-User-Tags", "tag3")

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind headers: %v", err)
	}

	// Verify bound data
	if user.Name != "Charlie Brown" {
		t.Errorf("Expected Name to be 'Charlie Brown', got '%s'", user.Name)
	}
	if user.Age != 45 {
		t.Errorf("Expected Age to be 45, got %d", user.Age)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if len(user.Tags) != 3 || user.Tags[0] != "tag1" || user.Tags[1] != "tag2" || user.Tags[2] != "tag3" {
		t.Errorf("Tags not bound correctly, got %v", user.Tags)
	}
}

func TestBindCookies(t *testing.T) {
	// GenerateHandler cookies
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "user_name", Value: "David Miller"})
	req.AddCookie(&http.Cookie{Name: "user_age", Value: "50"})

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind cookies: %v", err)
	}

	// Verify bound data
	if user.Name != "David Miller" {
		t.Errorf("Expected Name to be 'David Miller', got '%s'", user.Name)
	}
	if user.Age != 50 {
		t.Errorf("Expected Age to be 50, got %d", user.Age)
	}
}

func TestBindContext(t *testing.T) {
	// GenerateHandler context with values
	ctx := context.WithValue(context.Background(), "userID", "12345")
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind context: %v", err)
	}

	// Verify bound data
	if user.UserID != "12345" {
		t.Errorf("Expected UserID to be '12345', got '%s'", user.UserID)
	}
}

func TestBindDefault(t *testing.T) {
	// Empty request
	req := httptest.NewRequest("GET", "/", nil)

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind defaults: %v", err)
	}

	// Verify bound data
	if user.Name != "DefaultUser" {
		t.Errorf("Expected Name to be 'DefaultUser', got '%s'", user.Name)
	}
	if user.Age != 30 {
		t.Errorf("Expected Age to be 30, got %d", user.Age)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if user.CreatedAt != "2023-01-01" {
		t.Errorf("Expected CreatedAt to be '2023-01-01', got '%s'", user.CreatedAt)
	}
	if user.UserID != "guest" {
		t.Errorf("Expected UserID to be 'guest', got '%s'", user.UserID)
	}
}

func TestBindFile(t *testing.T) {
	// GenerateHandler a multipart form with a file
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add form field
	if err := w.WriteField("name", "FileUser"); err != nil {
		t.Fatal(err)
	}

	// Add a file
	fileWriter, err := w.CreateFormFile("avatar", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.WriteString(fileWriter, "test file content")
	if err != nil {
		t.Fatal(err)
	}

	// Add multiple files
	for i := 1; i <= 3; i++ {
		fw, err := w.CreateFormFile("pictures", fmt.Sprintf("pic%d.jpg", i))
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.WriteString(fw, fmt.Sprintf("picture %d content", i))
		if err != nil {
			t.Fatal(err)
		}
	}

	// Close the writer
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}

	// GenerateHandler the request
	req := httptest.NewRequest("POST", "/", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	// GenerateHandler target struct
	user := &UserWithFile{}

	// Bind data
	err = Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind file: %v", err)
	}

	// Verify bound data
	if user.Name != "FileUser" {
		t.Errorf("Expected Name to be 'FileUser', got '%s'", user.Name)
	}
	if user.Avatar == nil {
		t.Errorf("Expected Avatar to be set")
	} else if user.Avatar.Filename != "test.txt" {
		t.Errorf("Expected Avatar filename to be 'test.txt', got '%s'", user.Avatar.Filename)
	}

	if len(user.Pictures) != 3 {
		t.Errorf("Expected 3 pictures, got %d", len(user.Pictures))
	}
}

func TestBindCustomUnmarshaler(t *testing.T) {
	// GenerateHandler query string with custom formatted value
	req := httptest.NewRequest("GET", "/pagination?page=2&limit=20&orderBy=name:ASC&orderBy=date:DESC", nil)

	// GenerateHandler target struct
	pagination := &Pagination{}

	// Bind data
	err := Bind(pagination, req)
	if err != nil {
		t.Fatalf("Failed to bind with custom unmarshaler: %v", err)
	}

	// Verify bound data
	if pagination.Page != 2 {
		t.Errorf("Expected Page to be 2, got %d", pagination.Page)
	}
	if pagination.Limit != 20 {
		t.Errorf("Expected Limit to be 20, got %d", pagination.Limit)
	}

	// Check OrderBy results
	if len(pagination.OrderBy) != 2 {
		t.Fatalf("Expected 2 OrderBy values, got %d", len(pagination.OrderBy))
	}

	if pagination.OrderBy[0].Field != "name" || pagination.OrderBy[0].Type != "ASC" {
		t.Errorf("First OrderBy value incorrect: %+v", pagination.OrderBy[0])
	}

	if pagination.OrderBy[1].Field != "date" || pagination.OrderBy[1].Type != "DESC" {
		t.Errorf("Second OrderBy value incorrect: %+v", pagination.OrderBy[1])
	}
}

func TestBindWithOptions(t *testing.T) {
	// Test strict mode with invalid input
	jsonData := `{"name":"John Doe","age":"invalid"}`
	req := httptest.NewRequest("POST", "/", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data with strict mode
	err := Bind(user, req, WithStrictMode(true))
	if err == nil {
		t.Fatalf("Expected binding to fail in strict mode")
	}
}

func TestBindingErrors(t *testing.T) {
	// Invalid target (not a pointer)
	var invalidUser TestUser
	req := httptest.NewRequest("GET", "/", nil)
	err := Bind(invalidUser, req)
	if err == nil || err != ErrInvalidTarget {
		t.Errorf("Expected ErrInvalidTarget, got %v", err)
	}

	// Invalid target (nil)
	err = Bind(nil, req)
	if err == nil || err != ErrInvalidTarget {
		t.Errorf("Expected ErrInvalidTarget, got %v", err)
	}

	// Invalid target (not a struct)
	var invalidTarget string
	err = Bind(&invalidTarget, req)
	if err == nil || err != ErrInvalidTarget {
		t.Errorf("Expected ErrInvalidTarget, got %v", err)
	}

	// Invalid JSON
	jsonData := `{"name":"Invalid JSON`
	req = httptest.NewRequest("POST", "/", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	user := &TestUser{}
	err = Bind(user, req)
	if err == nil {
		t.Errorf("Expected JSON parsing error, got nil")
	}

	// Invalid XML
	xmlData := `<TestUser><name>Invalid XML`
	req = httptest.NewRequest("POST", "/", strings.NewReader(xmlData))
	req.Header.Set("Content-Type", "application/xml")
	user = &TestUser{}
	err = Bind(user, req)
	if err == nil {
		t.Errorf("Expected XML parsing error, got nil")
	}
}

func TestCombinedDataSources(t *testing.T) {
	// GenerateHandler data from multiple sources
	jsonData := `{"name":"JSON User"}`
	req := httptest.NewRequest("POST", "/?age=55&is_active=true", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Tags", "json,query,header")

	// Context
	ctx := context.WithValue(context.Background(), "userID", "json_user_id")
	req = req.WithContext(ctx)

	// Cookies
	req.AddCookie(&http.Cookie{Name: "user_age", Value: "99"}) // This should not override query param

	// GenerateHandler target struct
	user := &TestUser{}

	// Bind data
	err := Bind(user, req)
	if err != nil {
		t.Fatalf("Failed to bind combined data: %v", err)
	}

	// Verify bound data
	if user.Name != "JSON User" {
		t.Errorf("Expected Name to be 'JSON User', got '%s'", user.Name)
	}
	if user.Age != 55 {
		t.Errorf("Expected Age to be 55, got %d", user.Age)
	}
	if !user.IsActive {
		t.Errorf("Expected IsActive to be true")
	}
	if user.UserID != "json_user_id" {
		t.Errorf("Expected UserID to be 'json_user_id', got '%s'", user.UserID)
	}
	if len(user.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d %s", len(user.Tags), user.Tags)
	}
}

func BenchmarkBindJSON(b *testing.B) {
	jsonData, _ := json.Marshal(TestUser{
		Name:      "Benchmark User",
		Age:       35,
		IsActive:  true,
		CreatedAt: "2023-01-01",
		Tags:      []string{"benchmark", "test", "performance"},
		UserID:    "12345",
	})

	req := httptest.NewRequest("POST", "/", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	for i := 0; i < b.N; i++ {
		user := &TestUser{}
		_ = Bind(user, req)
	}
}

func BenchmarkBindCombined(b *testing.B) {
	jsonData := `{"name":"Benchmark User"}`
	req := httptest.NewRequest("POST", "/?age=35&is_active=true", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Tags", "benchmark,test,performance")

	ctx := context.WithValue(context.Background(), "userID", "12345")
	req = req.WithContext(ctx)

	for i := 0; i < b.N; i++ {
		user := &TestUser{}
		_ = Bind(user, req.WithContext(ctx))
	}
}
