package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alexisvisco/goframe/core/coretypes"
	"github.com/alexisvisco/goframe/core/helpers/clog"
	"github.com/alexisvisco/goframe/http/params"
)

// Response represents an HTTP response
type Response interface {
	WriteTo(w http.ResponseWriter, r *http.Request) error
}

// Error represents a structured error response
type Error struct {
	Message    string         `json:"message,omitempty"`
	Code       string         `json:"code"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	StatusCode int            `json:"-"`
}

func (e Error) Error() string {
	return e.Message
}

// HandlerFunc represents our custom handler that returns a Response and error
type HandlerFunc func(r *http.Request) (Response, error)

// Middleware represents a function that wraps a HandlerFunc
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain combines multiple middleware into a single middleware
// Example: Chain(middleware1, middleware2)(handler)
// -> Middleware1(Middleware2(HandlerFunc))
func Chain(middlewares ...Middleware) func(HandlerFunc) http.HandlerFunc {
	return func(handler HandlerFunc) http.HandlerFunc {
		// Convert our custom HandlerFunc to standard http.HandlerFunc
		httpHandler := Wrap(handler)

		// Apply middleware in reverse order (last to first)
		for i := len(middlewares) - 1; i >= 0; i-- {
			httpHandler = middlewares[i](httpHandler)
		}

		return httpHandler
	}
}

func Wrap(handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := handler(r)
		if err != nil {
			ctx, line := clog.FromContext(r.Context())
			line.Error(err)
			r.WithContext(ctx)
		}
		if resp != nil {
			if err := resp.WriteTo(w, r); err != nil {
				ctx, line := clog.FromContext(r.Context())
				line.Error(err)
				r.WithContext(ctx)

				onError(w, r, err)
			}
		} else if err != nil {
			onError(w, r, err)
		}
	}
}

// JSONResponse represents a JSON HTTP response
type JSONResponse struct {
	StatusCode int
	Data       interface{}
	Headers    map[string]string
}

func (j JSONResponse) WriteTo(w http.ResponseWriter, r *http.Request) error {
	// Set headers
	for key, value := range j.Headers {
		w.Header().Set(key, value)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(j.StatusCode)

	if j.Data != nil {
		return json.NewEncoder(w).Encode(j.Data)
	}
	return nil
}

// WithHeader adds a single header to the JSON response
func (j JSONResponse) WithHeader(key, value string) JSONResponse {
	if j.Headers == nil {
		j.Headers = make(map[string]string)
	}
	j.Headers[key] = value
	return j
}

// WithHeaders adds multiple headers to the JSON response
func (j JSONResponse) WithHeaders(headers map[string]string) JSONResponse {
	if j.Headers == nil {
		j.Headers = make(map[string]string)
	}
	for k, v := range headers {
		j.Headers[k] = v
	}
	return j
}

// TextResponse represents a plain text HTTP response
type TextResponse struct {
	StatusCode int
	Data       string
	Headers    map[string]string
}

func (t TextResponse) WriteTo(w http.ResponseWriter, r *http.Request) error {
	// Set headers
	for key, value := range t.Headers {
		w.Header().Set(key, value)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(t.StatusCode)

	_, err := w.Write([]byte(t.Data))
	return err
}

// WithHeader adds a single header to the text response
func (t TextResponse) WithHeader(key, value string) TextResponse {
	if t.Headers == nil {
		t.Headers = make(map[string]string)
	}
	t.Headers[key] = value
	return t
}

// WithHeaders adds multiple headers to the text response
func (t TextResponse) WithHeaders(headers map[string]string) TextResponse {
	if t.Headers == nil {
		t.Headers = make(map[string]string)
	}
	for k, v := range headers {
		t.Headers[k] = v
	}
	return t
}

// WithContentType sets the Content-Type header (overrides default text/plain)
func (t TextResponse) WithContentType(contentType string) TextResponse {
	return t.WithHeader("Content-Type", contentType)
}

type RedirectResponse struct {
	url        string
	statusCode int
}

func (r RedirectResponse) WriteTo(w http.ResponseWriter, req *http.Request) error {
	http.Redirect(w, req, r.url, r.statusCode)
	return nil
}

func NewRedirectResponse(statusCode int, url string) RedirectResponse {
	allowedStatusCodes := map[int]bool{
		http.StatusMovedPermanently:  true,
		http.StatusFound:             true,
		http.StatusSeeOther:          true,
		http.StatusTemporaryRedirect: true,
		http.StatusPermanentRedirect: true,
	}
	if _, ok := allowedStatusCodes[statusCode]; !ok {
		// If the status code is not one of the allowed redirect codes, default to 302 Found
		statusCode = http.StatusFound
	}
	return RedirectResponse{
		url:        url,
		statusCode: statusCode,
	}
}

// EmptyResponse represents an empty HTTP response
type EmptyResponse struct {
	StatusCode int
	Headers    map[string]string
}

func (e EmptyResponse) WriteTo(w http.ResponseWriter, r *http.Request) error {
	// Set headers
	for key, value := range e.Headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(e.StatusCode)
	return nil
}

// WithHeader adds a single header to the empty response
func (e EmptyResponse) WithHeader(key, value string) EmptyResponse {
	if e.Headers == nil {
		e.Headers = make(map[string]string)
	}
	e.Headers[key] = value
	return e
}

// WithHeaders adds multiple headers to the empty response
func (e EmptyResponse) WithHeaders(headers map[string]string) EmptyResponse {
	if e.Headers == nil {
		e.Headers = make(map[string]string)
	}
	for k, v := range headers {
		e.Headers[k] = v
	}
	return e
}

// WithLocation sets the Location header (useful for redirects)
func (e EmptyResponse) WithLocation(url string) EmptyResponse {
	return e.WithHeader("Location", url)
}

// JSON provides helper methods for JSON responses
var JSON = jsonHelper{}

type jsonHelper struct{}

func (jsonHelper) Ok(data interface{}) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusOK,
		Data:       data,
	}
}

func (jsonHelper) Created(data interface{}) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusCreated,
		Data:       data,
	}
}

func (jsonHelper) Accepted(data interface{}) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusAccepted,
		Data:       data,
	}
}

func (jsonHelper) NoContent() JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusNoContent,
		Data:       nil,
	}
}

func (jsonHelper) BadRequest(message string) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusBadRequest,
		Data: Error{
			Message: message,
			Code:    "BAD_REQUEST",
		},
	}
}

func (jsonHelper) Unauthorized(message string) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusUnauthorized,
		Data: Error{
			Message: message,
			Code:    "UNAUTHORIZED",
		},
	}
}

func (jsonHelper) Forbidden(message string) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusForbidden,
		Data: Error{
			Message: message,
			Code:    "FORBIDDEN",
		},
	}
}

func (jsonHelper) NotFound(message string) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusNotFound,
		Data: Error{
			Message: message,
			Code:    "NOT_FOUND",
		},
	}
}

func (jsonHelper) Conflict(message string) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusConflict,
		Data: Error{
			Message: message,
			Code:    "CONFLICT",
		},
	}
}

func (jsonHelper) InternalServerError(message string) JSONResponse {
	return JSONResponse{
		StatusCode: http.StatusInternalServerError,
		Data: Error{
			Message: message,
			Code:    "INTERNAL_SERVER_ERROR",
		},
	}
}

func (jsonHelper) WithHeaders(resp Response, headers map[string]string) Response {
	switch r := resp.(type) {
	case JSONResponse:
		return r.WithHeaders(headers)
	case TextResponse:
		return r.WithHeaders(headers)
	case EmptyResponse:
		return r.WithHeaders(headers)
	default:
		return resp
	}
}

// Text provides helper methods for text responses
var Text = textHelper{}

type textHelper struct{}

func (textHelper) Ok(data string) TextResponse {
	return TextResponse{
		StatusCode: http.StatusOK,
		Data:       data,
	}
}

func (textHelper) Created(data string) TextResponse {
	return TextResponse{
		StatusCode: http.StatusCreated,
		Data:       data,
	}
}

// Empty provides helper methods for empty responses
var Empty = emptyHelper{}

type emptyHelper struct{}

func (emptyHelper) Ok() EmptyResponse {
	return EmptyResponse{
		StatusCode: http.StatusOK,
	}
}

func (emptyHelper) NoContent() EmptyResponse {
	return EmptyResponse{
		StatusCode: http.StatusNoContent,
	}
}

func (emptyHelper) Created() EmptyResponse {
	return EmptyResponse{
		StatusCode: http.StatusCreated,
	}
}

func (emptyHelper) Accepted() EmptyResponse {
	return EmptyResponse{
		StatusCode: http.StatusAccepted,
	}
}

var ErrorMapper = map[error]Error{}

// DefaultHTTPError is the default error handler for unexpected errors
var DefaultHTTPError = func(w http.ResponseWriter, r *http.Request, err error) {
	var validationError *coretypes.ValidationError
	if errors.As(err, &validationError) {
		m := make(map[string]any, len(validationError.Errors))
		for k, v := range validationError.Errors {
			m[k] = v
		}

		err := Error{
			Message:  "validation error",
			Code:     "VALIDATION_ERROR",
			Metadata: m,
		}

		_ = NewJSONResponse(http.StatusBadRequest, err).WriteTo(w, r)
	} else {
		_ = JSON.InternalServerError("").WriteTo(w, r)
	}
}

// onError handles errors that occur during request processing
func onError(w http.ResponseWriter, r *http.Request, err error) {
	if customError, ok := ErrorMapper[err]; ok {
		resp := NewJSONResponse(customError.StatusCode, customError)
		_ = resp.WriteTo(w, r)
		return
	}

	var bindingError *params.BindingError
	if errors.As(err, &bindingError) {
		fmt.Println("Using binding error handler")
		resp := NewJSONResponse(http.StatusBadRequest, Error{
			Message: "binding error",
			Code:    "BINDING_ERROR",
		})
		_ = resp.WriteTo(w, r)
		return
	}

	fmt.Println("Using default error handler")

	DefaultHTTPError(w, r, err)
}

func NewJSONResponse(statusCode int, data interface{}) JSONResponse {
	return JSONResponse{
		StatusCode: statusCode,
		Data:       data,
	}
}

func NewTextResponse(statusCode int, data string) TextResponse {
	return TextResponse{
		StatusCode: statusCode,
		Data:       data,
	}
}

func NewEmptyResponse(statusCode int) EmptyResponse {
	return EmptyResponse{
		StatusCode: statusCode,
	}
}
