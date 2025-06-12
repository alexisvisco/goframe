# Project Architecture

A generated project follows a clean separation of concerns. Each package has a dedicated role and can be tested independently.

- **internal/app** – defines the `fx.App` module wiring all dependencies. New repositories, services and handlers register themselves here.
- **internal/repository** – data access layer. Each repository contains CRUD methods for a single model.
- **internal/service** – business logic orchestrating repositories and other services.
- **internal/v1handler** – HTTP handlers for your REST API. Routing is set up in `router.go`.
- **internal/workflow** – background workflows and activities executed by Temporal.

This layout keeps domain logic separate from infrastructure so files remain small and easy to understand.
