# CLI

The command line interface under `cmd/cli` is your entry point for generators, database migrations and custom tasks. The root command sets up a context containing configuration, database connection and other services.

Explore available commands with:

```bash
go run cmd/cli/main.go --help
```

Generators like `generate service` or `generate repository` update the project structure automatically. You can add your own subcommands and share services using the provided options.
