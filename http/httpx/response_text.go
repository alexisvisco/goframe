package httpx

import "net/http"

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

func NewTextResponse(statusCode int, data string) TextResponse {
	return TextResponse{
		StatusCode: statusCode,
		Data:       data,
	}
}
