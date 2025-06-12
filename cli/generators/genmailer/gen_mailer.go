package genmailer

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genworker"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

// MailerGenerator manages mailer files.
type MailerGenerator struct {
	Gen *generators.Generator
	Wf  *genworker.WorkerGenerator
}

//go:embed templates
var fs embed.FS

// Generate creates the initial mailer infrastructure files.
func (m *MailerGenerator) Generate() error {
	files := []generators.FileConfig{
		m.createRegistry(),
		m.createSendEmailWorkflow(),
	}

	if err := m.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if err := m.Wf.Update(); err != nil {
		return err
	}

	return m.updateAppModule()
}

// Update refreshes the mailer registry with current mailers.
func (m *MailerGenerator) Update() error {
	err := m.Gen.GenerateFile(m.createRegistry())
	if err != nil {
		return fmt.Errorf("failed to generate registry: %w", err)
	}

	return m.updateAppModule()
}

func (m *MailerGenerator) createSendEmailWorkflow() generators.FileConfig {
	return generators.FileConfig{
		Path:     "internal/workflow/workflow_send_email.go",
		Template: typeutil.Must(fs.ReadFile("templates/workflow_send_email.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport("go.temporal.io/sdk/workflow", "workflow")
		},
	}
}

// GenerateMailer creates a new mailer with the specified action.
func (m *MailerGenerator) GenerateMailer(name, action string) error {
	files := []generators.FileConfig{
		m.createMailerFile(name),
		m.createRegistry(),
		m.createMailView(name, action, "txt"),
		m.createMailView(name, action, "mjml"),
	}

	if err := m.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if err := m.ensureAction(name, action); err != nil {
		return fmt.Errorf("failed to create mailer action: %w", err)
	}

	return m.updateAppModule()
}

func (m *MailerGenerator) createMailerFile(name string) generators.FileConfig {
	path := filepath.Join("internal/mailer", fmt.Sprintf("mailer_%s.go", str.ToSnakeCase(name)))

	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/new_mailer.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("mailer_pascal", str.ToPascalCase(name)).
				WithImport("context", "context").
				WithImport("go.temporal.io/sdk/client", "client").
				WithImport("go.uber.org/fx", "fx").
				WithImport("github.com/nrednav/cuid2", "cuid2").
				WithImport(filepath.Join(m.Gen.GoModuleName, "internal/workflow"), "workflow")
		},
	}
}

func (m *MailerGenerator) createMailView(name, action, format string) generators.FileConfig {
	var templateFile string
	var filePath string

	if format == "txt" {
		templateFile = "templates/views_new_mail.txt.tmpl"
		filePath = filepath.Join("views/mails", fmt.Sprintf("%s_%s.txt.tmpl",
			str.ToSnakeCase(name), str.ToSnakeCase(action)))
	} else {
		templateFile = "templates/views_new_mail.mjml.tmpl"
		filePath = filepath.Join("views/mails", fmt.Sprintf("%s_%s.mjml.tmpl",
			str.ToSnakeCase(name), str.ToSnakeCase(action)))
	}

	return generators.FileConfig{
		Path:     filePath,
		Template: typeutil.Must(fs.ReadFile(templateFile)),
	}
}

func (m *MailerGenerator) createRegistry() generators.FileConfig {
	mailers, _ := m.listMailers() // Ignoring error as empty slice is acceptable if dir doesn't exist

	return generators.FileConfig{
		Path:     "internal/mailer/registry.go",
		Template: typeutil.Must(fs.ReadFile("templates/registry.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("mailers", mailers)
		},
	}
}

func (m *MailerGenerator) ensureAction(name, action string) error {
	path := filepath.Join("internal/mailer", fmt.Sprintf("mailer_%s.go", str.ToSnakeCase(name)))
	gofile, err := genhelper.LoadGoFile(path)
	if err != nil {
		return fmt.Errorf("failed to load mailer file %s: %w", path, err)
	}

	pascalMailer := str.ToPascalCase(name)
	pascalAction := str.ToPascalCase(action)

	if gofile.HasMethod(pascalMailer+"Mailer", pascalAction) {
		return fmt.Errorf("action %s already exists in mailer %s", action, name)
	}

	gh := genhelper.New("mailer", typeutil.Must(fs.ReadFile("templates/new_mailer_action.go.tmpl")))
	gh.WithVar("mailer_pascal", pascalMailer).
		WithVar("mailer_snake", str.ToSnakeCase(name)).
		WithVar("action_pascal", pascalAction).
		WithVar("action_snake", str.ToSnakeCase(action))

	actionContent, err := gh.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate action %s for mailer %s: %w", action, name, err)
	}

	gofile.AddNamedImport("", "github.com/alexisvisco/goframe/mail")
	gofile.AddNamedImport("", "go.temporal.io/sdk/temporal")
	gofile.AddContent(actionContent)

	return gofile.Save()
}

func (m *MailerGenerator) updateAppModule() error {
	path := "internal/app/module.go"
	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return nil
	}

	gf.AddNamedImport("", filepath.Join(m.Gen.GoModuleName, "internal/mailer"))
	gf.AddLineAfterString("return []fx.Option{", "\t\tfx.Provide(mailer.Dependencies...),")

	return gf.Save()
}

func (m *MailerGenerator) listMailers() ([]string, error) {
	entries, err := os.ReadDir("internal/mailer")
	if err != nil {
		return nil, err
	}

	var mailers []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		if e.Name() == "registry.go" {
			continue
		}

		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		name = strings.TrimPrefix(name, "mailer_")
		pascal := str.ToPascalCase(name)

		if !strings.HasSuffix(pascal, "Mailer") {
			pascal += "Mailer"
		}

		mailers = append(mailers, pascal)
	}

	return mailers, nil
}
