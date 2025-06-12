# Worker

Goframe relies on [Temporal](https://temporal.io/) for background processing. A worker listens on a task queue and executes workflows defined in `internal/workflow`.

Temporal offers strong guarantees around retries and state management, making it a great fit for distributed systems. The generator creates a provider that starts a worker when your application launches. You can add activities and workflows just like you would add services or repositories.
