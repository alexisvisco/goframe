package genworker

import (
	"embed"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

// WorkerGenerator handles workflow files.
type WorkerGenerator struct {
	Gen *generators.Generator
}

//go:embed templates
var fs embed.FS

// Generate creates the initial workflow infrastructure files.
func (w *WorkerGenerator) Generate() error {
	// Create required directories
	dirs := []string{"internal/workflow", "internal/workflow/activity"}
	for _, dir := range dirs {
		if err := w.Gen.CreateDirectory(dir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	files := []generators.FileConfig{
		w.createProviderFile(),
		w.createOrUpdateRegistry(),
	}

	if err := w.Gen.GenerateFiles(files); err != nil {
		return err
	}

	return w.updateAppModule()
}

// Update refreshes workflow registrations.
func (w *WorkerGenerator) Update() error {
	err := w.Gen.GenerateFile(w.createOrUpdateRegistry())
	if err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	return w.updateAppModule()
}

// GenerateWorkflow creates a new workflow with specified activities.
func (w *WorkerGenerator) GenerateWorkflow(name string, activities []string) error {
	// Create each activity file first
	var activityFiles []generators.FileConfig
	for _, act := range activities {
		activityFiles = append(activityFiles, w.createActivityFile(act))
	}

	files := append(activityFiles, []generators.FileConfig{
		w.createWorkflowFile(name, activities),
		w.createOrUpdateRegistry(),
	}...)

	if err := w.Gen.GenerateFiles(files); err != nil {
		return err
	}

	return w.updateAppModule()
}

// GenerateActivity creates a new activity.
func (w *WorkerGenerator) GenerateActivity(name string) error {
	files := []generators.FileConfig{
		w.createActivityFile(name),
		w.createOrUpdateRegistry(),
	}

	if err := w.Gen.GenerateFiles(files); err != nil {
		return err
	}

	return w.updateAppModule()
}

func (w *WorkerGenerator) createProviderFile() generators.FileConfig {
	return generators.FileConfig{
		Path:     "internal/provide/provide_worker.go",
		Template: typeutil.Must(fs.ReadFile("templates/provide_worker.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport("context", "context").
				WithImport("log/slog", "slog").
				WithImport("go.temporal.io/sdk/client", "client").
				WithImport("go.temporal.io/sdk/worker", "worker").
				WithImport("go.uber.org/fx", "fx").
				WithImport("github.com/alexisvisco/goframe/core/helpers/temporalutil", "temporalutil").
				WithImport(filepath.Join(w.Gen.GoModuleName, "config"), "config").
				WithImport(filepath.Join(w.Gen.GoModuleName, "internal/workflow"), "workflow")
		},
	}
}

func (w *WorkerGenerator) createWorkflowFile(name string, activities []string) generators.FileConfig {
	path := fmt.Sprintf("internal/workflow/workflow_%s.go", str.ToSnakeCase(name))

	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/new_workflow.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			pascalName := str.ToPascalCase(name)
			g.WithVar("name_pascal_case", pascalName)

			var acts []string
			for _, a := range activities {
				p := str.ToPascalCase(a)
				if !strings.HasSuffix(p, "Activity") {
					p += "Activity"
				}
				acts = append(acts, p)
			}

			if len(acts) > 0 {
				g.WithImport(filepath.Join(w.Gen.GoModuleName, "internal/workflow/activity"), "activity")
			}

			g.WithImport("go.temporal.io/sdk/workflow", "workflow").
				WithImport("go.uber.org/fx", "fx").
				WithVar("activities", acts)
		},
	}
}

func (w *WorkerGenerator) createActivityFile(name string) generators.FileConfig {
	path := fmt.Sprintf("internal/workflow/activity/activity_%s.go", str.ToSnakeCase(name))

	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/new_activity.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("name_pascal_case", str.ToPascalCase(name))
		},
	}
}

func (w *WorkerGenerator) createOrUpdateRegistry() generators.FileConfig {
	return generators.FileConfig{
		Path:     "internal/workflow/registry.go",
		Template: typeutil.Must(fs.ReadFile("templates/registry.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			activities, workflows := w.buildRegistrationList()

			g.WithImport("go.temporal.io/sdk/worker", "worker")

			for _, r := range append(workflows, activities...) {
				if !r.SelfPackage {
					g.WithImport(r.ImportPath, r.PackageName)
				}
			}

			g.WithVar("activities", activities).
				WithVar("workflows", workflows)
		},
	}
}

func (w *WorkerGenerator) updateAppModule() error {
	path := "internal/app/module.go"
	gf, err := genhelper.LoadGoFile(path)
	if err != nil {
		return nil // File might not exist yet, which is acceptable
	}

	gf.AddNamedImport("", filepath.Join(w.Gen.GoModuleName, "internal/workflow"))
	gf.AddLineAfterString("return []fx.Option{", "\tfx.Provide(workflow.Dependencies...),")

	return gf.Save()
}

type temporalRegistration struct {
	StructName  string
	MethodName  string
	ImportPath  string
	SelfPackage bool
	PackageName string
}

func (tr temporalRegistration) Constructor() string {
	if tr.SelfPackage {
		return fmt.Sprintf("New%s", tr.StructName)
	}
	return fmt.Sprintf("%s.New%s", tr.PackageName, tr.StructName)
}

func (tr temporalRegistration) Type() string {
	if tr.SelfPackage {
		return tr.StructName
	}
	return fmt.Sprintf("%s.%s", tr.PackageName, tr.StructName)
}

func (w *WorkerGenerator) buildRegistrationList() ([]temporalRegistration, []temporalRegistration) {
	pkg, err := genhelper.LoadGoPkg("internal/workflow", true)
	if err != nil {
		return nil, nil
	}

	structActivities := pkg.FindAllStructRegexp(regexp.MustCompile(`(\w+)Activity$`))
	structWorkflows := pkg.FindAllStructRegexp(regexp.MustCompile(`(\w+)Workflow$`))

	var acts []temporalRegistration
	var wfs []temporalRegistration
	for _, sa := range structActivities {
		name := sa.Name
		if !strings.HasSuffix(name, "Activity") {
			name += "Activity"
		}
		acts = append(acts, temporalRegistration{
			StructName:  name,
			MethodName:  str.ToPascalCase(strings.TrimSuffix(name, "Activity")),
			ImportPath:  sa.ImportPath,
			SelfPackage: sa.Self,
			PackageName: sa.Package,
		})
	}
	for _, sw := range structWorkflows {
		name := sw.Name
		if !strings.HasSuffix(name, "Workflow") {
			name += "Workflow"
		}
		wfs = append(wfs, temporalRegistration{
			StructName:  name,
			MethodName:  str.ToPascalCase(strings.TrimSuffix(name, "Workflow")),
			ImportPath:  sw.ImportPath,
			SelfPackage: sw.Self,
			PackageName: sw.Package,
		})
	}

	return acts, wfs
}
