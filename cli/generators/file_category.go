package generators

// FileCategory represents a Category of generated files
type FileCategory string

const (
	// CategoryCore for main application files
	CategoryCore = "core"

	// CategoryWeb for web-related files
	CategoryWeb = "Web"

	// CategoryDatabase for database-related files
	CategoryDatabase = "Database"

	// CategoryConfig for configuration files
	CategoryConfig = "Config"

	// CategoryDocker for DockerFiles-related files
	CategoryDocker = "Docker"

	// CategoryGo for Go module files
	CategoryGo = "Go"

	CategoryStorage = "Storage"

	CategoryTasks = "Task"

	CategoryI18n = "I18n"
)
