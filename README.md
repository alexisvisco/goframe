# GoFrame

GoFrame is an opinionated framework to build web applications in Go. It provides a command line interface that scaffolds your project and manages common tasks so you can focus on business logic.

A quick overview of what you get out of the box:

- **Command line interface** to generate boilerplate and run utilities.
- **HTTP server** powered by `net/http` with routing helpers and middleware support.
- **Database** integration through `GORM` with migration helpers.
- **Internationalization** using YAML translations with typed accessors.
- **Workers** based on [Temporal](https://temporal.io) for background jobs.
- **Mailer** with MJML and text templates for asynchronous email sending.
- **Task runner** for reusable CLI commands.
- **Structured logging** via the canonical log helper built on `slog`.
- **Storage** abstraction to handle attachments on disk or S3.

## Getting Started

Install the CLI:

```bash
go install github.com/alexisvisco/goframe/cli/goframe@latest
```

Create a new project:

```bash
goframe init <module name> [flags]
```

Start the development services:

```bash
docker compose up -d
```

Run the application:

```bash
go run cmd/app/main.go
```

For more commands check the built‑in help:

```bash
bin/goframe --help
```

`bin/goframe` always tries to compile the CLI into `bin/goframe.bin`. When compilation fails it reuses the previous binary if available. If no binary exists, it attempts `go mod tidy` once before asking you to fix the build errors.

## Documentation

Full documentation is available at [goframe.alexisvis.co](https://goframe.alexisvis.co).

