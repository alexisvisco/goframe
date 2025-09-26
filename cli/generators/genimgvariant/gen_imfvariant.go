package genimgvariant

import (
	"embed"
	"path/filepath"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
	"github.com/alexisvisco/goframe/cli/generators/genworker"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

type ImageVariantGenerator struct {
	Gen               *generators.Generator
	ServiceGenerator  *genservice.ServiceGenerator
	WorkflowGenerator *genworker.WorkerGenerator
	DBGenerator       *gendb.DatabaseGenerator
}

//go:embed templates
var fs embed.FS

func (g *ImageVariantGenerator) Generate() error {
	files := []generators.FileConfig{
		g.createImageVariantType("internal/types/imgvariant.go"),
		g.createImageVariantTypeScope("internal/types/scope_imgvariant.go"),
		g.createImageVariantService("internal/service/service_imgvariant.go"),
		g.createImageVariantProvide("internal/provide/provide_imgvariant.go"),
		g.createGenerateImageVariantsWorkflow("internal/workflow/workflow_generate_image_variants.go"),
		g.createGenerateImageVariantActivity("internal/workflow/activity/activity_generate_image_variant.go"),
	}

	if err := g.createMigrations(); err != nil {
		return err
	}

	if err := g.Gen.GenerateFiles(files); err != nil {
		return err
	}

	if err := g.addProviderToAppModule(); err != nil {
		return err
	}

	if err := g.ServiceGenerator.Update(); err != nil {
		return err
	}
	if err := g.WorkflowGenerator.Update(); err != nil {
		return err
	}

	return nil
}

func (g *ImageVariantGenerator) createImageVariantType(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/types_imgvariant.go.tmpl")),
	}
}

func (g *ImageVariantGenerator) createImageVariantTypeScope(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/types_imgvariant_scope.go.tmpl")),
	}
}

func (g *ImageVariantGenerator) createImageVariantService(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/imgvariant_service.go.tmpl")),
		Gen: func(x *genhelper.GenHelper) {
			x.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
		},
	}
}

func (g *ImageVariantGenerator) createImageVariantProvide(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/provide_image_variant.go.tmpl")),
		Gen: func(x *genhelper.GenHelper) {
			x.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
			x.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/service"), "service")
		},
	}
}

func (g *ImageVariantGenerator) addProviderToAppModule() error {
	gf, err := genhelper.LoadGoFile("internal/app/module.go")
	if err != nil {
		return err
	}

	gf.AddLineAfterString("return []fx.Option{", "\tfx.Invoke(provide.ImageVariant),")

	return gf.Save()
}

func (g *ImageVariantGenerator) createGenerateImageVariantsWorkflow(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/wf_generate_image_variants.go.tmpl")),
		Gen: func(x *genhelper.GenHelper) {
			x.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
			x.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/workflow/activity"), "activity")
		},
	}
}

func (g *ImageVariantGenerator) createGenerateImageVariantActivity(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/activity_generate_image_variant.go.tmpl")),
		Gen: func(x *genhelper.GenHelper) {
			x.WithImport(filepath.Join(g.Gen.GoModuleName, "internal/types"), "types")
		},
	}
}

func (g *ImageVariantGenerator) createMigrations() error {
	return g.DBGenerator.GenerateMigration(gendb.CreateMigrationParams{
		Sql:  true,
		Name: "image_variant_attachments",
		At:   time.Date(2025, 6, 30, 14, 15, 5, 0, time.UTC),
		Up: `CREATE TABLE image_variant_sets (
				id TEXT PRIMARY KEY,
				original_attachment_id TEXT NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
				kind TEXT,
    		kind_id TEXT,
				created_at timestamp with time zone NOT NULL DEFAULT timezone('utc', now()),
				updated_at timestamp with time zone NOT NULL DEFAULT timezone('utc', now())
);

CREATE TABLE image_variants (
				id TEXT PRIMARY KEY,
				attachment_id TEXT NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
				image_variant_set_id TEXT NOT NULL REFERENCES image_variant_sets(id) ON DELETE CASCADE,
				name TEXT NOT NULL,
				metadata JSONB,
				created_at timestamp with time zone NOT NULL DEFAULT timezone('utc', now())
);

CREATE UNIQUE INDEX idx_image_variant_set_name_unique ON image_variants(image_variant_set_id, name);`,
		Down: `DROP TABLE IF EXISTS image_variants;
DROP TABLE IF EXISTS image_variant_sets;`,
	})
}
