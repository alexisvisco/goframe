package templates

import _ "embed"

var (
	//go:embed cmd__app__main.go.tmpl
	CmdAppMainGo []byte

	//go:embed config__config.yml.tmpl
	ConfigConfigYml []byte

	//go:embed config__config.go.tmpl
	ConfigConfigGo []byte

	//go:embed db__migrations.go.tmpl
	DBMigrationsGo []byte

	//go:embed providers__provide_db.go.tmpl
	ProvidersProvideDBGo []byte

	//go:embed providers__provide_storage.go.tmpl
	ProvidersProvideStorageGo []byte

	//go:embed db__migrations__file.go.tmpl
	DBMigrationsFileGo []byte

	//go:embed db__migrations__file.sql.tmpl
	DBMigrationsFileSQL []byte

	//go:embed cmd__cli__main.go.tmpl
	CmdCLIMainGo []byte

	//go:embed Dockerfile.tmpl
	Dockerfile []byte

	//go:embed docker-compose.yml.tmpl
	DockerComposeYml []byte
)
