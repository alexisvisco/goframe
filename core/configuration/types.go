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
		Type              WorkerType `yaml:"type"`
		TemporalAddress   string     `yaml:"temporal_address"`
		TemporalNamespace string     `yaml:"temporal_namespace"`
		TemporalTaskQueue string     `yaml:"temporal_task_queue"`
	}

	Mail struct {
		Host      string        `yaml:"host"`
		Port      int           `yaml:"port"`
		Username  string        `yaml:"username"`
		Password  string        `yaml:"password"`
		AuthType  MailAuthType  `yaml:"auth_type"`  // Note: Fixed typo from AuthTYpe to AuthType
		TLSPolicy MailTLSPolicy `yaml:"tls_policy"` // TLS policy for SMTP connection

		DefaultFrom string `yaml:"default_from"`
	}

	I18n struct {
		DefaultLocale    string   `yaml:"default_locale"`
		SupportedLocales []string `yaml:"supported_locales"`
		FallbackLocale   []string `yaml:"fallback_locale"`
		Folder           string   `yaml:"folder"`
		Package          string   `yaml:"package"`
	}

	StorageType string

	DatabaseType  string
	WorkerType    string
	MailAuthType  string
	MailTLSPolicy string
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

const (
	MailAuthTypeNone    MailAuthType = "none"
	MailAuthTypePlain   MailAuthType = "plain"
	MailAuthTypeLogin   MailAuthType = "login"
	MailAuthTypeCRAMMD5 MailAuthType = "crammd5"
)

// TLS policy for SMTP connections
const (
	TLSPolicyNone          MailTLSPolicy = "none"          // No TLS (insecure, use only for testing)
	TLSPolicyOpportunistic MailTLSPolicy = "opportunistic" // Use STARTTLS if available (default)
	TLSPolicyMandatory     MailTLSPolicy = "mandatory"     // Require TLS via STARTTLS
)
