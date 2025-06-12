# Repository

Repositories encapsulate data persistence. Each repository interface defines methods for a single aggregate. You can generate a repository with:

```bash
go run cmd/cli/main.go generate repository <name>
```

This command creates the file under `internal/repository` and updates `registry.go` so the repository is automatically injected via `fxutil.As`. Repositories are usually consumed by services and never used directly from handlers.
