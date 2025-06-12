# Handler

Handlers expose HTTP endpoints and translate requests into service calls. A handler struct may depend on one or more services. Routes are registered in `internal/v1handler/router.go` using the standard `http.ServeMux`.

Handlers return responses through the `httpx` package which provides helpers for JSON, text or file responses. Input data can be bound into structs using the `params` package so your handler functions remain clean.
