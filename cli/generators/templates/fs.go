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

	//go:embed providers__provide_http.go.tmpl
	ProvidersProvideHTTPServerGo []byte

	//go:embed internal__repository__repository_example.go.tmpl
	InternalRepositoryExampleGo []byte

	//go:embed internal__service__service_example.go.tmpl
	InternalServiceExampleGo []byte

	//go:embed internal__v1handler__example.go.tmpl
	InternalV1HandlerExampleGo []byte

	//go:embed internal__types__example.go.tmpl
	InternalTypesExampleGo []byte

	//go:embed internal__v1handler__router.go.tmpl
	InternalV1HandlerRouterGo []byte

	//go:embed bin__goframe.tmpl
	InternalBinGoframe []byte

	//go:embed internal__app__module.go.tmpl
	InternalAppModuleGo []byte

	//go:embed internal__task__newtask.go.tmpl
	InternalTaskNewTaskGo []byte

	//go:embed config__i18n__translation.go.tmpl
	ConfigI18nTranslationGo []byte
)
