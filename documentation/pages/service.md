# Service

Services implement business logic. They coordinate repositories and other services. Generate a new service with:

```bash
go run cmd/cli/main.go generate service <name> --with-repository
```

The generator creates a struct and adds it to `internal/service/registry.go` so it can be injected into handlers. Services are ideal places to keep domain rules and validation that should not live in your HTTP layer.
