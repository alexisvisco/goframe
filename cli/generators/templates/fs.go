package templates

import _ "embed"

var (
	//go:embed ../core/templates/app_main.go.tmpl
	CmdAppMainGo []byte

	//go:embed ../config/templates/config.yml.tmpl
	ConfigConfigYml []byte

	//go:embed ../config/templates/config.go.tmpl
	ConfigConfigGo []byte

	//go:embed ../db/templates/migrations.go.tmpl
	DBMigrationsGo []byte

	//go:embed ../db/templates/provide_db.go.tmpl
	ProvidersProvideDBGo []byte

	//go:embed ../storage/templates/provide_storage.go.tmpl
	ProvidersProvideStorageGo []byte

	//go:embed ../db/templates/migrations_file.go.tmpl
	DBMigrationsFileGo []byte

	//go:embed ../db/templates/migrations_file.sql.tmpl
	DBMigrationsFileSQL []byte

	//go:embed ../core/templates/cli_main.go.tmpl
	CmdCLIMainGo []byte

	//go:embed ../docker/templates/Dockerfile.tmpl
	Dockerfile []byte

	//go:embed ../docker/templates/docker-compose.yml.tmpl
	DockerComposeYml []byte

	//go:embed ../web/templates/provide_http.go.tmpl
	ProvidersProvideHTTPServerGo []byte

	//go:embed ../repository/templates/repository_example.go.tmpl
	InternalRepositoryExampleGo []byte

	//go:embed ../service/templates/service_example.go.tmpl
	InternalServiceExampleGo []byte

	//go:embed ../handler/templates/example.go.tmpl
	InternalV1HandlerExampleGo []byte

	//go:embed ../types/templates/example.go.tmpl
	InternalTypesExampleGo []byte

	//go:embed ../handler/templates/router.go.tmpl
	InternalV1HandlerRouterGo []byte

	//go:embed ../handler/templates/registry.go.tmpl
	InternalV1HandlerRegistryGo []byte

	//go:embed ../handler/templates/new.go.tmpl
	InternalV1HandlerNewGo []byte

	//go:embed ../core/templates/goframe.tmpl
	InternalBinGoframe []byte

	//go:embed ../core/templates/mjml.tmpl
	BinMJML []byte

	//go:embed ../core/templates/module.go.tmpl
	InternalAppModuleGo []byte

	//go:embed ../task/templates/newtask.go.tmpl
	InternalTaskNewTaskGo []byte

	//go:embed ../worker/templates/registry.go.tmpl
	InternalWorkflowRegistryGo []byte

	//go:embed ../worker/templates/newworkflow.go.tmpl
	InternalWorkflowNewWorkflowGo []byte

	//go:embed ../worker/templates/activity/newactivity.go.tmpl
	InternalWorkflowActivityNewActivityGo []byte

	//go:embed ../worker/templates/provide_worker.go.tmpl
	ProvidersProvideWorkerGo []byte

	//go:embed ../i18n/templates/translation.go.tmpl
	ConfigI18nTranslationGo []byte

	//go:embed ../repository/templates/new.go.tmpl
	InternalRepositoryNewGo []byte

	//go:embed ../service/templates/new.go.tmpl
	InternalServiceNewGo []byte

	//go:embed ../repository/templates/registry.go.tmpl
	InternalRepositoryRegistryGo []byte

	//go:embed ../service/templates/registry.go.tmpl
	InternalServiceRegistryGo []byte

	//go:embed ../types/templates/new.go.tmpl
	InternalTypesNewGo []byte

	//go:embed ../mailer/templates/newmailer.go.tmpl
	InternalMailerNewGo []byte

	//go:embed ../mailer/templates/newmailer_action.go.tmpl
	InternalMailerActionGo []byte

	//go:embed ../mailer/templates/registry.go.tmpl
	InternalMailerRegistryGo []byte

	//go:embed ../mailer/templates/mail.txt.tmpl
	ViewMailTxt []byte

	//go:embed ../mailer/templates/mail.mjml.tmpl
	ViewMailMJML []byte

	//go:embed ../worker/templates/send_email.go.tmpl
	InternalWorkflowSendEmailGo []byte
)
