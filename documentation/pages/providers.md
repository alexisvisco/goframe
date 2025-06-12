# Providers

Providers are small constructors located in `internal/providers`. They build infrastructure pieces such as the database connection, mailer or storage client. Each provider exposes an `fx` option and registers itself in `internal/app/module.go`.

Providers can declare `fx.Lifecycle` hooks to start and stop resources. When you generate a new feature, the corresponding provider is automatically added to the application's module so everything is ready to use.
