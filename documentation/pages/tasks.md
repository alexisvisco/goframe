# Tasks

Tasks are long-running or administrative commands executed from the CLI. They live in `internal/task` and integrate with `cobradi` so they automatically register themselves with the root command.

Create a task using:

```bash
go run cmd/cli/main.go generate task <name>
```

Tasks can leverage the same services and providers as your HTTP handlers, letting you reuse business logic for maintenance jobs or cron-style automation.
