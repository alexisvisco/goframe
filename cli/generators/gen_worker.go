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

type WorkerGenerator struct {
	g *Generator
}

func (w *WorkerGenerator) Generate() error {
	dirs := []string{
		"internal/workflow",
		"internal/workflow/activity",
		"internal/providers",
	}

	for _, dir := range dirs {
		if err := w.g.CreateDirectory(dir, CategoryWorker); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	files := []FileConfig{
		w.createWorkerProvider("internal/providers/provider_worker.go"),
	}

	for _, file := range files {
		if err := w.g.GenerateFile(file); err != nil {
			return fmt.Errorf("failed to create worker file %s: %w", file.Path, err)
		}
	}

	return w.updateOrCreateRegistrations()
}

func (w *WorkerGenerator) createWorkerProvider(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.ProvidersProvideWorkerGo,
		Gen: func(g *genhelper.GenHelper) {
			g.WithImport("context", "context").
				WithImport("log/slog", "slog").
				WithImport("go.temporal.io/sdk/client", "client").
				WithImport("go.temporal.io/sdk/worker", "worker").
				WithImport("go.uber.org/fx", "fx").
				WithImport(filepath.Join(w.g.GoModuleName, "config"), "config").
				WithImport(filepath.Join(w.g.GoModuleName, "internal/workflow"), "workflow")
		},
		Category:  CategoryWorker,
		Condition: true,
	}
}

func (w *WorkerGenerator) updateOrCreateRegistrations() error {
	var file *os.File
	regPath := "internal/workflow/register.go"
	if _, err := os.Stat(regPath); os.IsNotExist(err) {
		var err2 error
		file, err2 = os.Create(regPath)
		if err2 != nil {
			return fmt.Errorf("failed to create register.go file: %w", err2)
		}
	} else {
		var err2 error
		file, err2 = os.OpenFile(regPath, os.O_WRONLY|os.O_TRUNC, 0644)
		if err2 != nil {
			return fmt.Errorf("failed to open register.go file: %w", err2)
		}
	}
	defer file.Close()

	hasActivities, hasWorkflows, activities, workflows := w.buildRegistrationList()

	gh := genhelper.New("workflow", templates.InternalWorkflowRegistryGo)
	gh.WithImport("go.temporal.io/sdk/worker", "worker")
	if hasActivities {
		gh.WithImport(filepath.Join(w.g.GoModuleName, "internal/workflow/activity"), "activity")
	}

	w.g.TrackFile(regPath, false, CategoryWorker)

	return gh.WithVar("activities", activities).
		WithVar("workflows", workflows).
		Generate(file)
}

func (w *WorkerGenerator) buildRegistrationList() (bool, bool, []string, []string) {
	actDir := "internal/workflow/activity"
	wfDir := "internal/workflow"

	actEntries, _ := os.ReadDir(actDir)
	wfEntries, _ := os.ReadDir(wfDir)

	var acts []string
	var wfs []string
	hasActivities := false
	hasWorkflows := false

	for _, e := range actEntries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		if e.Name() == "register.go" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		structName := str.ToPascalCase(name)
		acts = append(acts, fmt.Sprintf("activity.%s", structName))
		hasActivities = true
	}

	for _, e := range wfEntries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		if e.Name() == "register.go" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		structName := str.ToPascalCase(name)
		wfs = append(wfs, structName)
		hasWorkflows = true
	}

	return hasActivities, hasWorkflows, acts, wfs
}
