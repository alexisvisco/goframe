package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

type MailerGenerator struct {
	g *Generator
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

func (m *MailerGenerator) updateRegistry() error {
	mailers, err := m.listMailers()
	if err != nil {
		return err
	}
	path := "internal/mailer/registry.go"
	var file *os.File
	if _, err := os.Stat(path); os.IsNotExist(err) {
		file, err = os.Create(path)
		if err != nil {
			return err
		}
	} else {
		file, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
	}
	defer file.Close()

	gh := genhelper.New("mailer", templates.InternalMailerRegistryGo)
	m.g.TrackFile(path, false, CategoryMailer)
	return gh.WithVar("mailers", mailers).Generate(file)
}

func (m *MailerGenerator) ensureMailerFile(name string) error {
	path := filepath.Join("internal/mailer", fmt.Sprintf("mailer_%s.go", str.ToSnakeCase(name)))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return m.g.GenerateFile(FileConfig{
			Path:      path,
			Template:  templates.InternalMailerNewGo,
			Category:  CategoryMailer,
			Condition: true,
			Gen: func(g *genhelper.GenHelper) {
				g.WithVar("mailer_pascal", str.ToPascalCase(name)).
					WithImport("context", "context").
					WithImport("go.temporal.io/sdk/client", "client").
					WithImport("go.uber.org/fx", "fx").
					WithImport("github.com/nrednav/cuid2", "cuid2").
					WithImport(filepath.Join(m.g.GoModuleName, "internal/workflow"), "workflow")
			},
		})
	}
	return nil
}

func (m *MailerGenerator) ensureAction(name, action string) error {
	path := filepath.Join("internal/mailer", fmt.Sprintf("mailer_%s.go", str.ToSnakeCase(name)))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// todo imports if not exists :
	// "go.temporal.io/sdk/temporal"
	// "github.com/alexisvisco/goframe/mail"

	pascalMailer := str.ToPascalCase(name)
	pascalAction := str.ToPascalCase(action)
	signature := fmt.Sprintf("func (m *%sMailer) %s", pascalMailer, pascalAction)
	if strings.Contains(string(data), signature) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString("\n")
	gh := genhelper.New("mailer", templates.InternalMailerActionGo)
	gh.WithVar("mailer_pascal", pascalMailer).
		WithVar("mailer_snake", str.ToSnakeCase(name)).
		WithVar("action_pascal", pascalAction).
		WithVar("action_snake", str.ToSnakeCase(action))
	m.g.TrackFile(path, false, CategoryMailer)
	return gh.Generate(f)
}

func (m *MailerGenerator) createViews(name, action string) error {
	txtPath := filepath.Join("views/mails", fmt.Sprintf("%s_%s.txt.tmpl", str.ToSnakeCase(name), str.ToSnakeCase(action)))
	htmlPath := filepath.Join("views/mails", fmt.Sprintf("%s_%s.mjml.tmpl", str.ToSnakeCase(name), str.ToSnakeCase(action)))

	if err := m.g.GenerateFile(FileConfig{Path: txtPath, Template: templates.ViewMailTxt, Category: CategoryMailer, Condition: true}); err != nil {
		return err
	}
	return m.g.GenerateFile(FileConfig{Path: htmlPath, Template: templates.ViewMailMJML, Category: CategoryMailer, Condition: true})
}

func (m *MailerGenerator) Create(name, action string) error {
	if err := m.ensureMailerFile(name); err != nil {
		return err
	}
	if err := m.ensureAction(name, action); err != nil {
		return err
	}
	if err := m.createViews(name, action); err != nil {
		return err
	}
	if err := m.updateRegistry(); err != nil {
		return err
	}
	if err := m.updateAppModule(); err != nil {
		return err
	}
	return nil
}

func (m *MailerGenerator) updateAppModule() error {
	path := "internal/app/module.go"
	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return nil
	}

	gf.AddNamedImport("", filepath.Join(m.g.GoModuleName, "internal/mailer"))
	if err := gf.AddLineBeforeRegex("^\\s*\\}$", "\tfx.Provide(mailer.Dependencies...),"); err != nil {
		return err
	}
	return gf.Save()
}
