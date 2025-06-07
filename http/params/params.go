// Package params provides functionality for binding HTTP request data to Go structs.
// It supports binding from JSON, XML, headers, query parameters, form values,
// context values, cookies, files, and defaults through struct tags.
package params

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Common errors that may be returned during binding.
var (
	ErrInvalidTarget   = errors.New("binding target must be a non-nil pointer to a struct")
	ErrUnsupportedType = errors.New("unsupported type for binding")
	ErrFileNotFound    = errors.New("file not found in request")
)

// BindingError represents a specific error that occurred during binding.
type BindingError struct {
	Field   string
	Type    string
	Message string
	Err     error
}

func (e *BindingError) Error() string {
	return fmt.Sprintf("binding error for field '%s' (type %s): %s", e.Field, e.Type, e.Message)
}

func (e *BindingError) Unwrap() error {
	return e.Err
}

// Option represents a configuration option for the Bind function.
type Option func(*bindOptions)

type bindOptions struct {
	// Add options here like strict mode, custom tag names, etc.
	strictMode bool
}

// WithStrictMode enables strict mode where all errors are returned immediately.
func WithStrictMode(strict bool) Option {
	return func(o *bindOptions) {
		o.strictMode = strict
	}

}

// Bind attempts to bind data from an HTTP request to the destination struct.
// The destination must be a non-nil pointer to a struct.
// Binding is performed based on struct tags defined in the destination type.
func Bind(dest interface{}, req *http.Request, opts ...Option) error {
	// Apply options
	options := &bindOptions{
		strictMode: false,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Validate destination
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ErrInvalidTarget
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return ErrInvalidTarget
	}

	// Perform binding
	return bindStruct(v, req, options)
}

// bindStruct binds data to a struct based on its tags.
func bindStruct(v reflect.Value, req *http.Request, opts *bindOptions) error {
	t := v.Type()
	var errs []error

	// First pass: handle JSON/XML body if appropriate content type
	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := bindJSON(v, req); err != nil {
			bindErr := &BindingError{
				Field:   "body",
				Type:    "json",
				Message: "failed to bind JSON body",
				Err:     err,
			}
			if opts.strictMode {
				return bindErr
			}
			errs = append(errs, bindErr)
		}
	} else if strings.Contains(contentType, "application/xml") {
		if err := bindXML(v, req); err != nil {
			bindErr := &BindingError{
				Field:   "body",
				Type:    "xml",
				Message: "failed to bind XML body",
				Err:     err,
			}
			if opts.strictMode {
				return bindErr
			}
			errs = append(errs, bindErr)
		}
	}

	// Second pass: handle form data (Parse it only once)
	if strings.Contains(contentType, "multipart/form-data") ||
		strings.Contains(contentType, "application/x-www-form-urlencoded") {
		if err := req.ParseMultipartForm(32 << 20); err != nil && !errors.Is(err, http.ErrNotMultipart) {
			if err := req.ParseForm(); err != nil {
				bindErr := &BindingError{
					Field:   "form",
					Type:    "form",
					Message: "failed to parse form data",
					Err:     err,
				}
				if opts.strictMode {
					return bindErr
				}
				errs = append(errs, bindErr)
			}
		}
	}

	// Third pass: handle individual field bindings
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Check for embedded structs
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			if err := bindStruct(fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
			continue
		}

		// Process field bindings in a specific order
		// We need to bind in order of priority because some sources might override others
		// Priority:
		// 1. JSON/XML (already handled above)
		// 2. Form
		// 3. Query
		// 4. Headers
		// 5. Cookies
		// 6. Context
		// 7. File/Files (special case)
		// 8. Default (applies only if value not set from other sources)

		// Try form tag
		if formTag, ok := field.Tag.Lookup("form"); ok {
			if err := bindForm(formTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		// Try query tag (only if field is still zero)
		if queryTag, ok := field.Tag.Lookup("query"); ok && fieldValue.IsZero() {
			if err := bindQuery(queryTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		if pathTag, ok := field.Tag.Lookup("path"); ok && fieldValue.IsZero() {
			if err := bindPath(pathTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		// Try headers tag (only if field is still zero)
		if headerTag, ok := field.Tag.Lookup("headers"); ok && fieldValue.IsZero() {
			if err := bindHeader(headerTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		// Try cookie tag (only if field is still zero)
		if cookieTag, ok := field.Tag.Lookup("cookie"); ok && fieldValue.IsZero() {
			if err := bindCookie(cookieTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		// Try ctx tag (only if field is still zero)
		if ctxTag, ok := field.Tag.Lookup("ctx"); ok && fieldValue.IsZero() {
			if err := bindContext(ctxTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		// Try file tag (only if field is still zero)
		if fileTag, ok := field.Tag.Lookup("file"); ok && fieldValue.IsZero() {
			if err := bindFile(fileTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		// Try files tag (only if field is still zero)
		if filesTag, ok := field.Tag.Lookup("files"); ok && fieldValue.IsZero() {
			if err := bindFiles(filesTag, field, fieldValue, req, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}

		// Try default tag (lowest priority, only if field is still zero)
		if defaultTag, ok := field.Tag.Lookup("default"); ok && fieldValue.IsZero() {
			if err := bindDefault(defaultTag, field, fieldValue, opts); err != nil {
				if opts.strictMode {
					return err
				}
				errs = append(errs, err)
			}
		}
	}

	// Return combined errors
	if len(errs) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString("multiple binding errors occurred: ")
		for i, err := range errs {
			if i > 0 {
				errMsg.WriteString("; ")
			}
			errMsg.WriteString(err.Error())
		}
		return errors.New(errMsg.String())
	}

	return nil
}

// bindJSON binds JSON request body to the struct.
func bindJSON(v reflect.Value, req *http.Request) error {
	if req.Body == nil {
		return nil
	}

	decoder := json.NewDecoder(req.Body)
	return decoder.Decode(v.Addr().Interface())
}

// bindXML binds XML request body to the struct.
func bindXML(v reflect.Value, req *http.Request) error {
	if req.Body == nil {
		return nil
	}

	decoder := xml.NewDecoder(req.Body)
	return decoder.Decode(v.Addr().Interface())
}

// getValueSlice is a common helper for handling exploded values
func getValueSlice(values []string, exploderTag string, singleValue string) []string {
	if len(values) == 0 && singleValue != "" {
		// Use the exploder to split the single value if provided
		if exploderTag != "" {
			return strings.Split(singleValue, exploderTag)
		}
		// Otherwise use the single value as is
		return []string{singleValue}
	}
	return values
}

// processSliceValues is a common helper for binding slices
func processSliceValues(value reflect.Value, values []string, field reflect.StructField) error {
	if len(values) == 0 {
		return nil
	}

	// Create a new slice to hold the values
	slice := reflect.MakeSlice(value.Type(), 0, len(values))
	elemType := value.Type().Elem()

	// Add each value to the slice
	for _, val := range values {
		newVal := reflect.New(elemType).Elem()
		if err := setValueFromString(newVal, val, field); err != nil {
			return err
		}
		slice = reflect.Append(slice, newVal)
	}

	value.Set(slice)
	return nil
}

// bindHeader binds a value from HTTP headers.
func bindHeader(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	headerName := tag

	// If this is a slice, handle it specially
	if value.Kind() == reflect.Slice {
		exploderTag, hasExploder := field.Tag.Lookup("exploder")

		// First, get all header values (multiple headers with the same name)
		headerValues := req.Header.Values(headerName)

		// If no multiple headers but we have a single header with a value that should be exploded
		if len(headerValues) <= 1 && hasExploder {
			singleValue := req.Header.Get(headerName)
			if singleValue != "" {
				// Split the single value by the exploder
				headerValues = strings.Split(singleValue, exploderTag)
			}
		}

		// Process the values
		if err := processSliceValues(value, headerValues, field); err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "header",
				Message: "failed to set value from header",
				Err:     err,
			}
		}

		return nil
	}

	// Regular (non-slice) field
	headerValue := req.Header.Get(headerName)
	if headerValue == "" {
		return nil // No value found
	}

	return setValueFromString(value, headerValue, field)
}

// bindQuery binds a value from query parameters.
func bindQuery(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	query := req.URL.Query()
	paramName := tag

	// Check if we need to use the exploder
	exploderTag, hasExploder := field.Tag.Lookup("exploder")

	// If this is a slice, handle it specially
	if value.Kind() == reflect.Slice {
		// Get all values for this parameter
		paramValues := query[paramName]

		// Try with square brackets too (e.g., param[]=value)
		bracketParamValues := query[paramName+"[]"]
		paramValues = append(paramValues, bracketParamValues...)

		// If we have an exploder, try to use it
		singleParam := query.Get(paramName)
		exploderValue := ""
		if hasExploder {
			exploderValue = exploderTag
		}
		paramValues = getValueSlice(paramValues, exploderValue, singleParam)

		// Process all values
		if err := processSliceValues(value, paramValues, field); err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "query",
				Message: "failed to set value from query parameter",
				Err:     err,
			}
		}

		return nil
	}

	// Regular (non-slice) field
	paramValue := query.Get(paramName)
	if paramValue == "" {
		return nil // No value found
	}

	return setValueFromString(value, paramValue, field)
}

// bindPath binds a value from path parameters. It uses the new req.PathValue() method
func bindPath(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	paramName := tag

	// Check if we need to use the exploder
	exploderTag, hasExploder := field.Tag.Lookup("exploder")

	// If this is a slice, handle it specially
	if value.Kind() == reflect.Slice {
		// Get all values for this parameter
		var pathValues []string
		pathValue := req.PathValue(paramName)

		// Try with square brackets too (e.g., param[]=value)
		bracketParamValues := req.PathValue(paramName + "[]")
		pathValues = append(pathValues, pathValue, bracketParamValues)

		// If we have an exploder, try to use it
		singleParam := req.PathValue(paramName)
		exploderValue := ""
		if hasExploder {
			exploderValue = exploderTag
		}
		pathValues = getValueSlice(pathValues, exploderValue, singleParam)

		// Process all values
		if err := processSliceValues(value, pathValues, field); err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "path",
				Message: "failed to set value from path parameter",
				Err:     err,
			}
		}

		return nil
	}

	// Regular (non-slice) field
	paramValue := req.PathValue(paramName)
	if paramValue == "" {
		return nil // No value found
	}

	return setValueFromString(value, paramValue, field)
}

// bindForm binds a value from form values.
func bindForm(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	// Ensure form is parsed - handle both regular forms and multipart forms
	if req.Form == nil {
		if err := req.ParseMultipartForm(32 << 20); err != nil {
			if err != http.ErrNotMultipart {
				return &BindingError{
					Field:   field.Name,
					Type:    "form",
					Message: "failed to parse multipart form",
					Err:     err,
				}
			}

			// Not a multipart form, try regular form
			if err := req.ParseForm(); err != nil {
				return &BindingError{
					Field:   field.Name,
					Type:    "form",
					Message: "failed to parse form",
					Err:     err,
				}
			}
		}
	}

	formName := tag

	// Check if we need to use the exploder
	exploderTag, hasExploder := field.Tag.Lookup("exploder")

	// If this is a slice, handle it specially
	if value.Kind() == reflect.Slice {
		// Get all values for this parameter - from both Form and PostForm
		var formValues []string

		// Try both Form and PostForm to catch both GET and POST params
		if req.Form != nil {
			formValues = append(formValues, req.Form[formName]...)
			formValues = append(formValues, req.Form[formName+"[]"]...)
		}

		if req.PostForm != nil {
			formValues = append(formValues, req.PostForm[formName]...)
			formValues = append(formValues, req.PostForm[formName+"[]"]...)
		}

		// If we have an exploder, try to use it
		singleParam := req.FormValue(formName)
		exploderValue := ""
		if hasExploder {
			exploderValue = exploderTag
		}
		formValues = getValueSlice(formValues, exploderValue, singleParam)

		// Process all values
		if err := processSliceValues(value, formValues, field); err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "form",
				Message: "failed to set value from form parameter",
				Err:     err,
			}
		}

		return nil
	}

	// Regular (non-slice) field
	formValue := req.FormValue(formName)
	if formValue == "" {
		return nil // No value found
	}

	return setValueFromString(value, formValue, field)
}

// bindContext binds a value from request context.
func bindContext(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	ctxKey := tag
	ctxValue := req.Context().Value(ctxKey)
	if ctxValue == nil {
		return nil // No value found
	}

	// Try to set the value directly
	ctxValValue := reflect.ValueOf(ctxValue)
	if ctxValValue.Type().AssignableTo(value.Type()) {
		value.Set(ctxValValue)
		return nil
	}

	// Try converting to string and then to the target type
	strVal, ok := ctxValue.(string)
	if !ok {
		return &BindingError{
			Field:   field.Name,
			Type:    "context",
			Message: "context value is not assignable to field type and cannot be converted to string",
			Err:     ErrUnsupportedType,
		}
	}

	return setValueFromString(value, strVal, field)
}

// bindCookie binds a value from cookies.
func bindCookie(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	cookieName := tag
	cookie, err := req.Cookie(cookieName)
	if err != nil {
		return nil // No cookie found
	}

	return setValueFromString(value, cookie.Value, field)
}

// bindFile binds a single file to a multipart.FileHeader field.
func bindFile(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	if !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data") {
		return nil // Not a multipart form
	}

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		return &BindingError{
			Field:   field.Name,
			Type:    "file",
			Message: "failed to parse multipart form",
			Err:     err,
		}
	}

	fileName := tag
	file, header, err := req.FormFile(fileName)
	if err != nil {
		return &BindingError{
			Field:   field.Name,
			Type:    "file",
			Message: "file not found in request",
			Err:     ErrFileNotFound,
		}
	}
	defer file.Close()

	// Ensure the field is of type *multipart.FileHeader
	if value.Type() != reflect.TypeOf(&multipart.FileHeader{}) {
		return &BindingError{
			Field:   field.Name,
			Type:    "file",
			Message: "field must be of type *multipart.FileHeader",
			Err:     ErrUnsupportedType,
		}
	}

	value.Set(reflect.ValueOf(header))
	return nil
}

// bindFiles binds multiple files to a []*multipart.FileHeader field.
func bindFiles(tag string, field reflect.StructField, value reflect.Value, req *http.Request, opts *bindOptions) error {
	if !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data") {
		return nil // Not a multipart form
	}

	if err := req.ParseMultipartForm(32 << 20); err != nil {
		return &BindingError{
			Field:   field.Name,
			Type:    "files",
			Message: "failed to parse multipart form",
			Err:     err,
		}
	}

	fileName := tag

	// Check if the field is a slice of *multipart.FileHeader
	if value.Kind() != reflect.Slice || value.Type().Elem() != reflect.TypeOf(&multipart.FileHeader{}) {
		return &BindingError{
			Field:   field.Name,
			Type:    "files",
			Message: "field must be of type []*multipart.FileHeader",
			Err:     ErrUnsupportedType,
		}
	}

	// Get all files with this name
	if req.MultipartForm == nil || req.MultipartForm.File == nil {
		return &BindingError{
			Field:   field.Name,
			Type:    "files",
			Message: "no files found in request",
			Err:     ErrFileNotFound,
		}
	}

	headers := req.MultipartForm.File[fileName]
	if len(headers) == 0 {
		// Try with brackets
		headers = req.MultipartForm.File[fileName+"[]"]
		if len(headers) == 0 {
			return &BindingError{
				Field:   field.Name,
				Type:    "files",
				Message: "no files found in request",
				Err:     ErrFileNotFound,
			}
		}
	}

	// Create a new slice to hold the file headers
	slice := reflect.MakeSlice(value.Type(), 0, len(headers))
	for _, header := range headers {
		slice = reflect.Append(slice, reflect.ValueOf(header))
	}

	value.Set(slice)
	return nil
}

// bindDefault applies a default value from a tag.
func bindDefault(tag string, field reflect.StructField, value reflect.Value, opts *bindOptions) error {
	// Don't apply default if the value is already set
	if !value.IsZero() {
		return nil
	}

	// For slices with default values
	if value.Kind() == reflect.Slice {
		defaultValues := strings.Split(tag, ",")

		// Create a slice to hold the default values
		slice := reflect.MakeSlice(value.Type(), 0, len(defaultValues))
		elemType := value.Type().Elem()

		// Add each default value to the slice
		for _, dv := range defaultValues {
			dv = strings.TrimSpace(dv)
			if dv == "" {
				continue // Skip empty values
			}

			newVal := reflect.New(elemType).Elem()
			if err := setValueFromString(newVal, dv, field); err != nil {
				return &BindingError{
					Field:   field.Name,
					Type:    "default",
					Message: fmt.Sprintf("failed to set default value '%s'", dv),
					Err:     err,
				}
			}
			slice = reflect.Append(slice, newVal)
		}

		value.Set(slice)
		return nil
	}

	// For regular fields
	return setValueFromString(value, tag, field)
}

// setValueFromString sets a value from a string based on the field's type.
func setValueFromString(value reflect.Value, input string, field reflect.StructField) error {
	// Check if the field implements encoding.TextUnmarshaler
	if value.CanAddr() {
		ptrVal := value.Addr()
		if unmarshaler, ok := ptrVal.Interface().(encoding.TextUnmarshaler); ok {
			return unmarshaler.UnmarshalText([]byte(input))
		}
	}

	// Special case for pointer types where the pointed-to type implements TextUnmarshaler
	if value.Kind() == reflect.Ptr && value.IsNil() {
		elemType := value.Type().Elem()
		newVal := reflect.New(elemType)

		// Check if the new element implements TextUnmarshaler
		if unmarshaler, ok := newVal.Interface().(encoding.TextUnmarshaler); ok {
			if err := unmarshaler.UnmarshalText([]byte(input)); err != nil {
				return err
			}
			value.Set(newVal)
			return nil
		}

		// Not a TextUnmarshaler, continue with regular handling
		elemValue := newVal.Elem()
		if err := setValueFromString(elemValue, input, field); err != nil {
			return err
		}
		value.Set(newVal)
		return nil
	}

	// Handle based on the field's kind
	switch value.Kind() {
	case reflect.String:
		value.SetString(input)

	case reflect.Bool:
		b, err := strconv.ParseBool(input)
		if err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "conversion",
				Message: "failed to convert to bool",
				Err:     err,
			}
		}
		value.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special case for time.Duration
		if value.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(input)
			if err != nil {
				return &BindingError{
					Field:   field.Name,
					Type:    "conversion",
					Message: "failed to parse duration",
					Err:     err,
				}
			}
			value.SetInt(int64(d))
			return nil
		}

		i, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "conversion",
				Message: "failed to convert to int",
				Err:     err,
			}
		}
		value.SetInt(i)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(input, 10, 64)
		if err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "conversion",
				Message: "failed to convert to uint",
				Err:     err,
			}
		}
		value.SetUint(u)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return &BindingError{
				Field:   field.Name,
				Type:    "conversion",
				Message: "failed to convert to float",
				Err:     err,
			}
		}
		value.SetFloat(f)

	case reflect.Ptr:
		// Create a new value of the element type
		elemType := value.Type().Elem()
		newVal := reflect.New(elemType)

		// Set the element value
		elemValue := newVal.Elem()
		if err := setValueFromString(elemValue, input, field); err != nil {
			return err
		}

		// Set the pointer
		value.Set(newVal)

	case reflect.Struct:
		// Special cases for common structs
		if value.Type() == reflect.TypeOf(time.Time{}) {
			t, err := time.Parse(time.RFC3339, input)
			if err != nil {
				// Try different formats
				formats := []string{
					time.RFC3339,
					"2006-01-02",
					"2006-01-02 15:04:05",
					time.RFC1123,
					time.RFC822,
				}

				for _, format := range formats {
					t, err = time.Parse(format, input)
					if err == nil {
						break
					}
				}

				if err != nil {
					return &BindingError{
						Field:   field.Name,
						Type:    "conversion",
						Message: "failed to parse time",
						Err:     err,
					}
				}
			}
			value.Set(reflect.ValueOf(t))
			return nil
		}

		return &BindingError{
			Field:   field.Name,
			Type:    "conversion",
			Message: "unsupported struct type",
			Err:     ErrUnsupportedType,
		}

	default:
		return &BindingError{
			Field:   field.Name,
			Type:    "conversion",
			Message: "unsupported field type",
			Err:     ErrUnsupportedType,
		}
	}

	return nil
}
