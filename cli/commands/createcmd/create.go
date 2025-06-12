package createcmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/alexisvisco/goframe/cli/generators"
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

	longDescription := fmt.Sprintf(`Generates a new project with the following modules:
	- cmd/app/main.go: Main application entry point
	- cmd/cli/main.go: CLI application entry point
	
	- configuration files
	- databases related files
	- Docker files (if --docker=true)
	- web files (if --web=true)
	- storage provider
	`)

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
				GoModuleName:    i.goModName,
				DatabaseType:    configuration.DatabaseType(i.databaseName),
				ORMType:         i.orm,
				Maintainer:      i.maintainer,
				WebFiles:        i.web,
				ExampleWebFiles: i.webExamples,
				DockerFiles:     i.docker,
				WorkerType:      i.worker,
			}

			generators := []generators.FileCreator{
				g.Core(),
				g.Config(),
				g.Databases(),
				g.Docker(),
				g.Storage(),
				g.Web(),
				g.Worker(),
			}

			for _, gen := range generators {
				err := gen.Generate()
				if err != nil {
					return fmt.Errorf("failed to generate files: %v", err)
				}
			}

			fmt.Println("Project initialized!")
			fmt.Println("Postgres port: 7894 (if enabled)")
			fmt.Println("Temporal UI port: 8233 (if enabled)")
			fmt.Println("Mailpit UI port: 8888")
			fmt.Println("Run 'docker compose up -d' then 'go run cmd/app/main.go' to start the app")

			return nil
		},
	}

	cmd.Flags().StringVarP(&i.databaseName, "databaseName", "d", "postgres", "Database type: postgres, sqlite")
	cmd.Flags().BoolVarP(&i.docker, "docker", "D", true, "Initialize with DockerFiles support (docker-compose & Dockerfile)")
	cmd.Flags().StringVarP(&i.folder, "folder", "f", ".", "Project folder name")
	cmd.Flags().BoolVarP(&i.web, "web", "w", true, "Initialize a web application")
	cmd.Flags().BoolVarP(&i.webExamples, "web-examples", "W", true, "Initialize a web application with examples")
	cmd.Flags().StringVarP(&i.goModName, "gomod", "g", "", "Create a go.mod file with go module name if set")
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
	web          bool
	webExamples  bool
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
