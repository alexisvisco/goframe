package httpx

import (
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

// Middleware is an interface for middleware functions that can be used in the middleware chain.
// It accepts both function types:
// - func(http.Handler) http.Handler
// - func(http.HandlerFunc) http.HandlerFunc
type Middleware interface{}

func Chain(middlewares ...Middleware) func(HandlerFunc) http.HandlerFunc {
	return func(handler HandlerFunc) http.HandlerFunc {
		// Convert our custom HandlerFunc to standard http.HandlerFunc
		httpHandler := Wrap(handler)

		// Apply middleware in reverse order (last to first)
		for i := len(middlewares) - 1; i >= 0; i-- {
			mw := middlewares[i]
			switch mw := mw.(type) {
			case func(http.Handler) http.Handler:
				httpHandler = mw(httpHandler).ServeHTTP
			case func(http.HandlerFunc) http.HandlerFunc:
				httpHandler = mw(httpHandler)
			default:
				panic(fmt.Sprintf("unsupported middleware type: %T", mw))
			}
		}

		return httpHandler
	}
}

// Wrap converts a HandlerFunc into an http.HandlerFunc, handling errors and writing responses.
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
