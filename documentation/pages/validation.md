# Validation

The framework uses `coretypes.ValidationError` to represent invalid input. When returned from a handler, `httpx` automatically converts it into a `400 Bad Request` with a JSON body listing each field error.

You can validate structs using your library of choice. Many projects rely on [zog](https://github.com/Oudwins/zog). Convert issues to a `ValidationError` via `ValidationErrorFromZog` so they integrate with the HTTP response helpers.
