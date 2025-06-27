package genauth

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/cli/generators/genmailer"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

type AuthGenerator struct {
	Gen              *generators.Generator
	MailerGenerator  *genmailer.MailerGenerator
	ServiceGenerator *genservice.ServiceGenerator
	HTTPGenerator    *genhttp.HTTPGenerator
	DBGenerator      *gendb.DatabaseGenerator
}

//go:embed templates
var fs embed.FS

func (g *AuthGenerator) Generate() error {
	var files []generators.FileConfig

	files = append(files, g.createTypes()...)
	files = append(files, g.createHandler()...)
	files = append(files, g.createMailer()...)
	files = append(files, g.createService()...)

	if err := g.Gen.GenerateFiles(files); err != nil {
		return err
	}

	err := g.generateMigration()
	if err != nil {
		return fmt.Errorf("failed to create migration: %w", err)
	}

	if err := g.updateConfig(); err != nil {
		return err
	}

	if err := g.updateRouter(); err != nil {
		return fmt.Errorf("failed to update router: %w", err)
	}

	if err := g.MailerGenerator.Update(); err != nil {
		return err
	}

	if err := g.ServiceGenerator.Update(); err != nil {
		return err
	}

	if err := g.HTTPGenerator.UpdateRouter("UserHandler"); err != nil {
		return err
	}

	if err := g.HTTPGenerator.Update(); err != nil {
		return err
	}

	if err := g.Gen.RunGoModTidy(); err != nil {
		return err
	}

	if err := g.Gen.SyncMails(); err != nil {
		return fmt.Errorf("failed to sync mail templates: %w", err)
	}

	return nil
}

func (g *AuthGenerator) createTypes() []generators.FileConfig {
	return []generators.FileConfig{
		{
			Path:     "internal/types/user.go",
			Template: typeutil.Must(fs.ReadFile("templates/types_user.go.tmpl")),
		},
		{
			Path:     "internal/types/user_code.go",
			Template: typeutil.Must(fs.ReadFile("templates/types_user_code.go.tmpl")),
		},
		{
			Path:     "internal/types/user_oauth_provider.go",
			Template: typeutil.Must(fs.ReadFile("templates/types_user_oauth_provider.go.tmpl")),
		},
		{
			Path:     "internal/types/auth.go",
			Template: typeutil.Must(fs.ReadFile("templates/types_auth.go.tmpl")),
		},
		{
			Path:     "internal/types/errors.go",
			Template: typeutil.Must(fs.ReadFile("templates/types_errors.go.tmpl")),
		},
		{
			Path:     "internal/types/mailer.go",
			Template: typeutil.Must(fs.ReadFile("templates/types_mailer.go.tmpl")),
		},
	}
}

func (g *AuthGenerator) createHandler() []generators.FileConfig {
	return []generators.FileConfig{
		{
			Path:     "internal/v1handler/handler_user.go",
			Template: typeutil.Must(fs.ReadFile("templates/v1handler_handler_user.go.tmpl")),
			Gen: func(gen *genhelper.GenHelper) {
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
			},
		},
		{
			Path:     "internal/v1handler/middleware_user.go",
			Template: typeutil.Must(fs.ReadFile("templates/v1handler_middleware_user.go.tmpl")),
			Gen: func(gen *genhelper.GenHelper) {
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
			},
		},
	}
}

func (g *AuthGenerator) createMailer() []generators.FileConfig {
	return []generators.FileConfig{
		{
			Path:     "internal/mailer/mailer_user.go",
			Template: typeutil.Must(fs.ReadFile("templates/mailer_mailer_user.go.tmpl")),
			Gen: func(gen *genhelper.GenHelper) {
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
			},
		},
		{
			Path:     "views/mails/user_email_verification.mjml.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_email_verification.mjml.tmpl")),
		},
		{
			Path:     "views/mails/user_email_verification.txt.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_email_verification.txt.tmpl")),
		},
		{
			Path:     "views/mails/user_magic_link.mjml.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_magic_link.mjml.tmpl")),
		},
		{
			Path:     "views/mails/user_magic_link.txt.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_magic_link.txt.tmpl")),
		},
		{
			Path:     "views/mails/user_oauth_provider_verification.mjml.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_oauth_provider_verification.mjml.tmpl")),
		},
		{
			Path:     "views/mails/user_oauth_provider_verification.txt.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_oauth_provider_verification.txt.tmpl")),
		},
		{
			Path:     "views/mails/user_password_reset.mjml.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_password_reset.mjml.tmpl")),
		},
		{
			Path:     "views/mails/user_password_reset.txt.tmpl",
			RawFile:  true,
			Template: typeutil.Must(fs.ReadFile("templates/views_user_password_reset.txt.tmpl")),
		},
	}
}

func (g *AuthGenerator) createService() []generators.FileConfig {
	return []generators.FileConfig{
		{
			Path:     "internal/service/service_auth.go",
			Template: typeutil.Must(fs.ReadFile("templates/service_service_auth.go.tmpl")),
			Gen: func(gen *genhelper.GenHelper) {
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "config"), "config")
			},
		},
		{
			Path:     "internal/service/service_oauth_state.go",
			Template: typeutil.Must(fs.ReadFile("templates/service_service_oauth_state.go.tmpl")),
			Gen: func(gen *genhelper.GenHelper) {
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
				gen.WithImport("github.com/alexisvisco/goframe/db/dbutil", "dbutil")
			},
		},
		{
			Path:     "internal/service/service_user.go",
			Template: typeutil.Must(fs.ReadFile("templates/service_service_user.go.tmpl")),
			Gen: func(gen *genhelper.GenHelper) {
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
				gen.WithImport("github.com/alexisvisco/goframe/db/dbutil", "dbutil")
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "config"), "config")
			},
		},
		{
			Path:     "internal/service/service_user_code.go",
			Template: typeutil.Must(fs.ReadFile("templates/service_service_user_code.go.tmpl")),
			Gen: func(gen *genhelper.GenHelper) {
				gen.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
				gen.WithImport("github.com/alexisvisco/goframe/db/dbutil", "dbutil")
			},
		},
	}
}

func (g *AuthGenerator) updateConfig() error {
	file, err := genhelper.LoadGoFile("config/config.go")
	if err != nil {
		return err
	}

	file.AddLineAfterRegex("FrontendURL\\s+string", "\tAuth Auth `yaml:\"auth\"`")

	authStruct, err := fs.ReadFile("templates/config_auth_struct.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read auth struct template: %w", err)
	}

	file.AddLineBeforeRegex("type\\s+Environment\\s+struct", string(authStruct))

	file.AddContent(`func (c *Config) GetAuth() Auth {
	return c.getEnvironment().Auth
}
`)

	err = file.Save("config/config.go")
	if err != nil {
		return fmt.Errorf("failed to save updated config.go: %w", err)
	}

	yamlFile, err := os.ReadFile("config/config.yml")
	if err != nil {
		return fmt.Errorf("failed to read config.yml: %w", err)
	}

	configManager, err := genhelper.NewConfigManager(string(yamlFile))
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	authConfig := map[string]any{
		"verify_email_url":       "${FRONTEND_URL}/auth/verify-email",
		"magic_link_url":         "${FRONTEND_URL}/auth/magic-link",
		"reset_password_url":     "${FRONTEND_URL}/auth/reset-password",
		"oauth_verify_email_url": "${FRONTEND_URL}/auth/oauth/verify-email",
		"oauth_redirect_url":     "${FRONTEND_URL}/auth/oauth",
		"github_client_id":       "${GITHUB_CLIENT_ID}",
		"github_client_secret":   "${GITHUB_CLIENT_SECRET}",
		"apple_client_id":        "${APPLE_CLIENT_ID}",
		"apple_team_id":          "${APPLE_TEAM_ID}",
		"apple_key_id":           "${APPLE_KEY_ID}",
		"apple_private_key_b64":  "${APPLE_PRIVATE_KEY_BASE64}",
		"discord_client_id":      "${DISCORD_CLIENT_ID}",
		"discord_client_secret":  "${DISCORD_CLIENT_SECRET}",
	}
	err = configManager.AddConfigs(map[string]any{
		"production.auth":  authConfig,
		"development.auth": authConfig,
	}, &genhelper.InsertOptions{
		AddSpacing: false,
	})
	if err != nil {
		return fmt.Errorf("failed to add auth config: %w", err)
	}

	err = os.WriteFile("config/config.yml", []byte(configManager.ToString()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated config.yml: %w", err)
	}

	return nil
}

func (g *AuthGenerator) generateMigration() error {
	return g.DBGenerator.GenerateMigration(gendb.CreateMigrationParams{
		Sql:  true,
		Name: "user_auth",
		At:   time.Now(),
		Up: `create table if not exists users
(
    id                 text                    not null primary key,
    email              varchar(255)            not null,
    encrypted_password varchar(255),
    email_verified_at  timestamp default null,
    access_token       text,
    created_at         timestamp default now() not null,
    updated_at         timestamp default now() not null
);

-- index on email
create unique index if not exists idx_users_email on users (email);

create table if not exists user_oauth_providers
(
    id               text                    not null primary key,
    user_id          text                    not null,
    provider         varchar(50)             not null,
    provider_id      varchar(255)            not null,
    access_token     text,
    refresh_token    text,
    verified_at      timestamp default null,
    created_at       timestamp default now() not null,
    updated_at       timestamp default now() not null,
    foreign key (user_id) references users (id) on delete cascade
);

-- index unique on user_id x provider
create unique index if not exists idx_user_oauth_providers_user_provider on user_oauth_providers (user_id, provider);

create table if not exists user_codes
(
    id         text                    not null primary key,
    user_id    text                    not null,
    kind       text                    not null,
    metadata   jsonb default '{}'::jsonb,
    expires_at timestamp default now() not null,
    created_at timestamp default now() not null
);

create table if not exists oauth_state_codes (
    id text not null primary key,
    was_connected boolean not null,
    expires_at timestamp not null
);`,
		Down: `drop table if exists user_codes;
drop table if exists user_oauth_providers;
drop table if exists users;
drop table if exists oauth_state_codes;`,
	})
}

func (g *AuthGenerator) updateRouter() error {
	routes := `	p.Mux.HandleFunc("GET /v1/users/@me", p.UserHandler.Me())
	p.Mux.HandleFunc("POST /v1/users/auth/register_with_password", p.UserHandler.RegisterUserWithPassword())
	p.Mux.HandleFunc("POST /v1/users/auth/login_with_magic_link", p.UserHandler.LoginWithMagicLink())
	p.Mux.HandleFunc("POST /v1/users/auth/login_with_password", p.UserHandler.LoginWithPassword())
	p.Mux.HandleFunc("POST /v1/users/auth/verify_email/{code}", p.UserHandler.VerifyUserEmail())
	p.Mux.HandleFunc("POST /v1/users/auth/verify_magic_link/{code}", p.UserHandler.VerifyMagicLink())
	p.Mux.HandleFunc("POST /v1/users/auth/request_password_reset", p.UserHandler.RequestPasswordReset())
	p.Mux.HandleFunc("POST /v1/users/auth/reset_password/{code}", p.UserHandler.ResetPassword())
	p.Mux.HandleFunc("POST /v1/users/oauth/verify_provider/{provider_id}/{code}", p.UserHandler.VerifyOAuthProvider())
	p.Mux.HandleFunc("GET /v1/users/oauth/{provider}/login", p.UserHandler.LoginWithOAuth2())
	p.Mux.HandleFunc("POST /v1/users/oauth/{provider}/callback", p.UserHandler.OAuth2Callback())
	p.Mux.HandleFunc("GET /v1/users/oauth/{provider}/callback", p.UserHandler.OAuth2Callback())`

	file, err := genhelper.LoadGoFile("internal/v1handler/router.go")
	if err != nil {
		return fmt.Errorf("failed to load router.go: %w", err)
	}

	file.AddLineAfterRegex(`func\s+Router\(p\s+RouterParams\)\s+{`, routes)
	if err := file.Save("internal/v1handler/router.go"); err != nil {
		return fmt.Errorf("failed to save updated router.go: %w", err)
	}

	return nil
}
