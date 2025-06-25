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
		m.createOrUpdateRegistry(),
		m.createSendEmailActivity(),
		m.createSendEmailWorkflow(),
		m.createMailerHelperFile(),
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
	err := m.Gen.GenerateFile(m.createOrUpdateRegistry())
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
			g.WithImport("time", "time").
				WithImport("go.temporal.io/sdk/workflow", "workflow").
				WithImport("go.temporal.io/sdk/temporal", "temporal").
				WithImport("go.uber.org/fx", "fx").
				WithImport(filepath.Join(m.Gen.GoModuleName, "internal/workflow/activity"), "activity").
				WithImport("github.com/alexisvisco/goframe/mail", "mail")
		},
	}
}

func (m *MailerGenerator) createSendEmailActivity() generators.FileConfig {
	return generators.FileConfig{
		Path:     "internal/workflow/activity/activity_send_email.go",
		Template: typeutil.Must(fs.ReadFile("templates/activity_send_email.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport("go.uber.org/fx", "fx").
				WithImport("github.com/alexisvisco/goframe/mail", "mail")
		},
	}
}

func (m *MailerGenerator) createMailerHelperFile() generators.FileConfig {
	return generators.FileConfig{
		Path:     "internal/mailer/mailer_helper.go",
		Template: typeutil.Must(fs.ReadFile("templates/mailer_helper.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport("context", "context").
				WithImport("github.com/alexisvisco/goframe/mail", "mail").
				WithImport("github.com/nrednav/cuid2", "cuid2").
				WithImport("go.temporal.io/sdk/client", "client").
				WithImport("go.temporal.io/sdk/temporal", "temporal").
				WithImport(filepath.Join(m.Gen.GoModuleName, "internal/workflow"), "workflow")
		},
	}
}

// GenerateMailer creates a new mailer with the specified action.
func (m *MailerGenerator) GenerateMailer(name, action string) error {
	files := []generators.FileConfig{
		m.createMailerFile(name),
		m.createOrUpdateRegistry(),
		m.createMailView(name, action, "txt"),
		m.createMailView(name, action, "mjml"),
	}

	if err := m.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if err := m.ensureMailerTypes(name, action); err != nil {
		return fmt.Errorf("failed to update mailer types: %w", err)
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
		Skip:     m.Gen.SkipFileIfExists(path),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("mailer_pascal", str.ToPascalCase(name)).
				WithImport("context", "context").
				WithImport("go.uber.org/fx", "fx").
				WithImport(filepath.Join(m.Gen.GoModuleName, "internal/workflow"), "workflow").
				WithImport(filepath.Join(m.Gen.GoModuleName, "internal/types"), "types")
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

func (m *MailerGenerator) createOrUpdateRegistry() generators.FileConfig {
	return generators.FileConfig{
		Path:     "internal/mailer/registry.go",
		Template: typeutil.Must(fs.ReadFile("templates/registry.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			mailers, _ := m.listMailers() // directory may not exist yet
			g.WithVar("mailers", mailers)
			g.WithImport(filepath.Join(m.Gen.GoModuleName, "internal/types"), "types")
			g.WithImport("github.com/alexisvisco/goframe/core/helpers/fxutil", "fxutil")
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

	gofile.AddNamedImport("types", filepath.Join(m.Gen.GoModuleName, "internal/types"))
	gofile.AddContent(actionContent)

	return gofile.Save()
}
func (m *MailerGenerator) ensureMailerTypes(name, action string) error {
	path := filepath.Join("internal/types", "mailer.go")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for types: %w", err)
		}
		if err := os.WriteFile(path, []byte("package types\n"), 0644); err != nil {
			return fmt.Errorf("failed to create types file: %w", err)
		}
	}

	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return fmt.Errorf("failed to load type file: %w", err)
	}

	gf.AddNamedImport("", "context")

	pascalMailer := str.ToPascalCase(name)
	pascalAction := str.ToPascalCase(action)
	paramName := pascalMailer + "Mailer" + pascalAction + "Params"
	ifaceName := pascalMailer + "Mailer"

	// Ensure interface exists first
	if !gf.HasInterface(ifaceName) {
		gf.AddContent("type " + ifaceName + " interface {\n\t" + pascalAction + "(ctx context.Context, vars " + paramName + ") error\n}\n")
	} else {
		// Interface exists, check if method needs to be added
		lines := gf.GetLines()
		start := -1
		end := -1
		for i, l := range lines {
			if strings.Contains(l, "type "+ifaceName+" interface") {
				start = i
				continue
			}
			if start != -1 && strings.TrimSpace(l) == "}" {
				end = i
				break
			}
		}
		if start != -1 && end != -1 {
			exists := false
			for i := start; i < end; i++ {
				if strings.Contains(lines[i], pascalAction+"(") {
					exists = true
					break
				}
			}
			if !exists {
				gf.AddLineAfterString("type "+ifaceName+" interface {", "\t"+pascalAction+"(ctx context.Context, vars "+paramName+") error")
			}
		}
	}

	// Then ensure params struct exists
	if !gf.HasStruct(paramName) {
		gf.AddContent("type " + paramName + " struct {\n\tTo []string\n}\n")
	}

	return gf.Save()
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
		if e.Name() == "mailer_helper.go" {
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
