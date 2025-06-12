# HTTP Struct Binding

The `params` package binds query strings, form values, path parameters and headers into Go structs. Declare tags on your struct fields to specify the source:

```go
type UserInput struct {
    ID    int    `path:"id"`
    Name  string `query:"name"`
    Token string `headers:"X-Auth"`
}
```

Call `params.Bind(&input, r)` in your handler to populate the struct. Binding works with JSON bodies and multipart forms as well, providing a simple way to validate user input.
