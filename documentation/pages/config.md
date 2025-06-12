# Configuration

Application settings live in `config/config.yml`. The file is parsed at startup and mapped onto a struct. Values can reference environment variables using `${VAR}` or `${VAR:default}` placeholders.

Example configuration snippet:

```yaml
database:
  dsn: ${DATABASE_URL:postgres://user:pass@localhost/dbname}
worker:
  queue: default
```

The generated `Config` type provides helper methods like `GetDatabase()` or `GetWorker()` so your code stays decoupled from the YAML layout. You can extend the configuration by adding new sections in the file and fields in the struct.
