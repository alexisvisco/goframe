package httpx

import "net/http"

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

func NewEmptyResponse(statusCode int) EmptyResponse {
	return EmptyResponse{
		StatusCode: statusCode,
	}
}
