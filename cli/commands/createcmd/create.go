package createcmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genconfig"
	"github.com/alexisvisco/goframe/cli/generators/gencore"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/gendocker"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/cli/generators/genmailer"
	"github.com/alexisvisco/goframe/cli/generators/genrepository"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
	"github.com/alexisvisco/goframe/cli/generators/genstorage"
	"github.com/alexisvisco/goframe/cli/generators/genworker"
	"github.com/alexisvisco/goframe/cli/termcolor"
	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/spf13/cobra"
)

var (
	mainAppPath = "cmd/app/main.go"
)

func NewInitCmd() *cobra.Command {
	i := &initializer{}

	binariesThatMustBeInstalled := []string{
		"docker",
		"go",
	}

	longDescription := fmt.Sprintf(`Generates a new project`)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Long:  longDescription,
		RunE: func(cmd *cobra.Command, args []string) error {

			err := i.mayCreateAndChdirFolder()
			if err != nil {
				return err
			}

			err = i.mustHaveValidDatabase()
			if err != nil {
				return err
			}

			err = i.mustProjectNotBeInitialized()
			if err != nil {
				return fmt.Errorf("project already initialized: %v", err)
			}

			err = i.mustHaveBinaries(binariesThatMustBeInstalled)
			if err != nil {
				return fmt.Errorf("missing required binaries: %v", err)
			}

			err = i.ensureGoModCanBeCreated()
			if err != nil {
				return err
			}

			g := &generators.Generator{
				GoModuleName: i.goModName,
				DatabaseType: configuration.DatabaseType(i.databaseName),
				ORMType:      i.orm,
				Maintainer:   i.maintainer,
				HTTPServer:   i.http,
				WorkerType:   i.worker,
			}

			cfgGen := &genconfig.ConfigGenerator{Gen: g}
			coreGen := &gencore.CoreGenerator{Gen: g}
			repoGen := &genrepository.RepositoryGenerator{Gen: g}
			svcGen := &genservice.ServiceGenerator{Gen: g}
			dbGen := &gendb.DatabaseGenerator{Gen: g}
			storageGen := &genstorage.StorageGenerator{Gen: g, DBGen: dbGen}
			dockerGen := &gendocker.DockerGenerator{Gen: g}
			httpGen := &genhttp.HTTPGenerator{Gen: g}
			workerGen := &genworker.WorkerGenerator{Gen: g}
			mailerGen := &genmailer.MailerGenerator{Gen: g, Wf: workerGen}
			//exampleHttpGen := &genhttpexample.NoteExampleGenerator{
			//	Gen:     g,
			//	GenHTTP: httpGen,
			//	GenSvc:  svcGen,
			//	GenRepo: repoGen,
			//}

			filesGenerators := []generators.FilesGenerator{
				cfgGen,
				coreGen,
				repoGen,
				svcGen,
				dbGen,
				storageGen,
				dockerGen,
				workerGen,
				httpGen,
				mailerGen,
			}

			for _, gen := range filesGenerators {
				err := gen.Generate()
				if err != nil {
					return fmt.Errorf("failed to generate files: %v", err)
				}
			}

			fmt.Println(termcolor.WrapCyan(`   _______     ______    _______   _______        __       ___      ___   _______  
  /" _   "|   /    " \  /"     "| /"      \      /""\     |"  \    /"  | /"     "| 
 (: ( \___)  // ____  \(: ______)|:        |    /    \     \   \  //   |(: ______) 
  \/ \      /  /    ) :)\/    |  |_____/   )   /' /\  \    /\\  \/.    | \/    |   
  //  \ ___(: (____/ // // ___)   //      /   //  __'  \  |: \.        | // ___)_  
 (:   _(  _|\        / (:  (     |:  __   \  /   /  \\  \ |.  \    /:  |(:      "| 
  \_______)  \"_____/   \__/     |__|  \___)(___/    \___)|___|\__/|___| \_______) 
	                                                                      v.alpha.beta.omega `))

			type kv struct {
				Key   string
				Value string
			}

			var keyvalues []kv

			if i.databaseName == string(configuration.DatabaseTypePostgres) {
				keyvalues = append(keyvalues, kv{"Database", fmt.Sprintf("Postgres (%s)", termcolor.WrapBlue("localhost:7894"))})
			} else if i.databaseName == string(configuration.DatabaseTypeSQLite) {
				keyvalues = append(keyvalues, kv{"Database", fmt.Sprintf("SQLite (%s)", termcolor.WrapBlue(i.folder))})
			}

			if i.worker == string(configuration.WorkerTypeTemporal) {
				keyvalues = append(keyvalues, kv{"Worker", fmt.Sprintf("Temporal (%s, UI at %s)", termcolor.WrapBlue("localhost:7233"), termcolor.WrapBlue("http://localhost:8233"))})
			}

			if i.http {
				keyvalues = append(keyvalues, kv{"HTTP Server", termcolor.WrapBlue("http://localhost:8080")})
			}

			keyvalues = append(keyvalues, kv{"Mailpit", fmt.Sprintf("%s (for email testing)", termcolor.WrapBlue("http://localhost:8025"))})

			tw := new(tabwriter.Writer)

			tw.Init(os.Stdout, 0, 8, 1, '\t', 0)
			for _, kv := range keyvalues {
				fmt.Fprintf(tw, "  %s\t%s\n", kv.Key, kv.Value)
			}

			tw.Flush()
			fmt.Println("\nRun 'docker compose up -d' then 'go run cmd/app/main.go' to start the app")

			return nil
		},
	}

	cmd.Flags().StringVarP(&i.databaseName, "db-name", "d", "postgres", "Database type: postgres, sqlite")
	cmd.Flags().StringVarP(&i.folder, "folder", "f", ".", "Project folder name")
	cmd.Flags().BoolVarP(&i.http, "http-server", "w", true, "Initialize a http application")
	cmd.Flags().StringVarP(&i.goModName, "gomod", "g", "", "GenerateHandler a go.mod file with go module name if set")
	cmd.Flags().BoolVarP(&i.maintainer, "maintainer", "m", false, "Add specific maintainer thing to test the framework")
	cmd.Flags().StringVarP(&i.orm, "orm", "o", "gorm", "ORM to use (only gorm is supported for now)")
	cmd.Flags().StringVar(&i.worker, "worker", "temporal", "Worker type to use (only temporal is supported for now)")

	return cmd
}

type fileinfo struct {
	dir  bool
	path string
}
type initializer struct {
	folder       string
	goModName    string
	databaseName string
	orm          string
	maintainer   bool
	http         bool
	httpExample  bool
	docker       bool
	worker       string
}

func (i *initializer) mustProjectNotBeInitialized() error {
	if _, err := os.Stat(mainAppPath); err == nil {
		return fmt.Errorf(mainAppPath + " already exists, please remove it or choose a different folder")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking project initialization: %v", err)
	}

	return nil
}

func (i *initializer) mayCreateAndChdirFolder() error {
	// check if . or valid create folder
	// if the is not created we create it
	if i.folder == "." {
		i.folder = "."
	} else {
		if _, err := os.Stat(i.folder); os.IsNotExist(err) {
			err := os.MkdirAll(i.folder, 0755)
			if err != nil {
				return fmt.Errorf("failed to create folder: %v", err)
			}
		} else if err != nil {
			return fmt.Errorf("error checking folder: %v", err)
		}

		err := os.Chdir(i.folder)
		if err != nil {
			return fmt.Errorf("failed to change directory: %v", err)
		}
	}

	return nil
}

func (i *initializer) mustHaveValidDatabase() error {
	var validDatabases = []string{"postgres", "sqlite"}
	for _, db := range validDatabases {
		if db == i.databaseName {
			return nil
		}
	}
	return fmt.Errorf("invalid databaseName type: %s, allowed values are: %v", i.databaseName, validDatabases)
}

func (i *initializer) mustHaveBinaries(installed []string) error {
	for _, binary := range installed {
		if _, err := exec.LookPath(binary); err != nil {
			return fmt.Errorf("missing required binary: %s", binary)
		}
	}
	return nil
}

func (i *initializer) goModTidy() error {
	cmd := exec.Command("go", "mod", "tidy")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run go mod tidy: %v", err)
	}

	return nil
}

func (i *initializer) ensureGoModCanBeCreated() error {
	// check if the go.mod file exists
	if _, err := os.Stat("go.mod"); err == nil {
		return fmt.Errorf("go.mod file already exists, please remove it or choose a different folder")
	}

	if i.goModName == "" {
		return fmt.Errorf("go module name must be specified with --gomod flag")
	}

	return nil
}
