package gendb

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

type CreateSeedParams struct {
	Sql  bool
	Name string
	At   time.Time

	// only for sql seeds
	Run string
}

// GenerateSeed génère une nouvelle seed
func (g *DatabaseGenerator) GenerateSeed(params CreateSeedParams) error {
	if len(params.Name) == 0 {
		return fmt.Errorf("seed name is required")
	}

	if params.At.IsZero() {
		params.At = time.Now()
	}

	// Créer le dossier seeds s'il n'existe pas
	if err := g.Gen.CreateDirectory("db/seeds"); err != nil {
		return fmt.Errorf("failed to create seeds directory: %w", err)
	}

	files := g.CreateSeed(params)

	if err := g.Gen.GenerateFiles(files); err != nil {
		return fmt.Errorf("failed to generate seed files: %w", err)
	}

	return nil
}

// CreateSeed crée les fichiers de configuration pour une seed
func (g *DatabaseGenerator) CreateSeed(params CreateSeedParams) []generators.FileConfig {
	var seedFile generators.FileConfig

	if params.Sql {
		seedFile = g.createSQLSeedFile(params)
	} else {
		seedFile = g.createGoSeedFile(params)
	}

	return []generators.FileConfig{seedFile, g.updateOrCreateSeeds("db/seeds.go")}
}

// createGoSeedFile crée le FileConfig pour un fichier de seed Go
func (g *DatabaseGenerator) createGoSeedFile(c CreateSeedParams) generators.FileConfig {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	namePascalCase := str.ToPascalCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)
	structName := namePascalCase + timestamp

	nowUTC := time.Now().UTC()
	date := fmt.Sprintf("%d, %d, %d, %d, %d, %d, %d",
		nowUTC.Year(), nowUTC.Month(), nowUTC.Day(),
		nowUTC.Hour(), nowUTC.Minute(), nowUTC.Second(), time.Now().Nanosecond())

	return generators.FileConfig{
		Path:     fmt.Sprintf("db/seeds/%s.go", version),
		Template: typeutil.Must(fs.ReadFile("templates/new_seed.go.tmpl")),
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithVar("struct", structName).
				WithVar("date", date).
				WithVar("version", version).
				WithVar("name", nameSnakeCase)
		},
	}
}

// createSQLSeedFile crée le FileConfig pour un fichier de seed SQL
func (g *DatabaseGenerator) createSQLSeedFile(c CreateSeedParams) generators.FileConfig {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)

	return generators.FileConfig{
		Path:     fmt.Sprintf("db/seeds/%s.sql", version),
		Template: typeutil.Must(fs.ReadFile("templates/new_seed.sql.tmpl")),
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithVar("name", version).
				WithVar("run", c.Run)
		},
	}
}

// updateOrCreateSeeds met à jour ou crée le fichier seeds.go
func (g *DatabaseGenerator) updateOrCreateSeeds(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/seeds.go.tmpl")),
		Gen: func(gh *genhelper.GenHelper) {
			hasSQLSeeds, hasGoSeeds, list := g.buildSeedList()

			if hasSQLSeeds {
				gh.WithImport("embed", "embed").
					WithVar("has_sql_seeds", "true")
			}

			if hasGoSeeds {
				gh.WithImport(filepath.Join(g.Gen.GoModuleName, "db/seeds"), "seeds")
			}

			gh.WithVar("seeds", list)
		},
	}
}

// buildSeedList construit la liste des seeds disponibles
func (g *DatabaseGenerator) buildSeedList() (bool, bool, []string) {
	seedsDir := "db/seeds"

	entries, err := os.ReadDir(seedsDir)
	if err != nil {
		return false, false, []string{}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var seeds []string
	hasSqlFiles := false
	hasGoFiles := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		if strings.HasSuffix(filename, ".go") {
			hasGoFiles = true
			nameWithoutExt := strings.TrimSuffix(filename, ".go")
			parts := strings.SplitN(nameWithoutExt, "_", 2)

			if len(parts) == 2 {
				timestamp := parts[0]
				nameSnakeCase := parts[1]
				namePascalCase := str.ToPascalCase(nameSnakeCase)
				structName := namePascalCase + timestamp
				seeds = append(seeds, "seeds."+structName+"{},")
			}
		} else if strings.HasSuffix(filename, ".sql") {
			hasSqlFiles = true
			seeds = append(seeds, fmt.Sprintf(`seed.SeedFromSQL(fs, "seeds/%s"),`, filename))
		}
	}

	return hasSqlFiles, hasGoFiles, seeds
}
