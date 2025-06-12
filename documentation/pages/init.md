# `init` Command

`goframe` comes with a generator that creates a full project skeleton. Running the command below scaffolds your application with everything wired:

```bash
go run github.com/alexisvisco/goframe/cli init --gomod <module-name>
```

The generator builds the following directories:

- **cmd/app** – entry point of the HTTP server
- **cmd/cli** – command line interface to manage your app
- **config** – YAML configuration loaded at startup
- **internal** – business code (repositories, services, handlers, workflows)
- **docker** – ready to use Docker Compose files for dependencies

You can pass flags to pick the database engine (`--db`), enable example handlers (`--with-web`), or choose the worker implementation. The generated project is immediately runnable with `go run ./cmd/app`.
