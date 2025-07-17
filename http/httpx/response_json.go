package httpx

import (
	"encoding/json"
	"net/http"
)

// JSONResponse represents a JSON HTTP response
type JSONResponse struct {
	StatusCode int
	Data       interface{}
	Headers    map[string]string
}

// JSON provides helper methods for JSON responses
var JSON = jsonHelper{}

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

func NewJSONResponse(statusCode int, data interface{}) JSONResponse {
	return JSONResponse{
		StatusCode: statusCode,
		Data:       data,
	}
}
