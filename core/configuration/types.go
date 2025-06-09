package configuration

type (
	Database struct {
		Type DatabaseType `yaml:"type"`

		Host     string `yaml:"host,omitempty"`
		Port     int    `yaml:"port,omitempty"`
		Username string `yaml:"username,omitempty"`
		Password string `yaml:"password,omitempty"`
		Database string `yaml:"database,omitempty"`
		SSLMode  string `yaml:"ssl_mode,omitempty"`

		File string `yaml:"file,omitempty"`
	}

	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
		URL  string `yaml:"url"`
	}

	Logging struct {
		Level string `yaml:"level"`
	}

	Storage struct {
		Type StorageType `yaml:"type"`

		Endpoint        string `yaml:"endpoint"`
		Bucket          string `yaml:"bucket"`
		Region          string `yaml:"region"`
		AccessKeyID     string `yaml:"access_key_id"`
		SecretAccessKey string `yaml:"secret_access_key"`
		Secure          bool   `yaml:"secure"`

		Directory string `yaml:"directory"`
	}
	Worker struct {
		Type      WorkerType `yaml:"type"`
		HostPort  string     `yaml:"host_port"`
		Namespace string     `yaml:"namespace"`
		TaskQueue string     `yaml:"task_queue"`
	}

	I18n struct {
		DefaultLocale    string   `yaml:"default_locale"`
		SupportedLocales []string `yaml:"supported_locales"`
		FallbackLocale   []string `yaml:"fallback_locale"`
		Folder           string   `yaml:"folder"`
		Package          string   `yaml:"package"`
	}

	StorageType string

	DatabaseType string
	WorkerType   string
)

const (
	StorageTypeDisk StorageType = "disk"
	StorageTypeS3   StorageType = "s3"
)

const (
	DatabaseTypeSQLite   DatabaseType = "sqlite"
	DatabaseTypePostgres DatabaseType = "postgres"
)

const (
	WorkerTypeTemporal WorkerType = "temporal"
)
