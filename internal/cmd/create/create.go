package create

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	mainAppPath = "cmd/app/main.go"
)

func NewInitCmd() *cobra.Command {
	i := &initializer{}

	defaultFiles := map[string]func(path string) error{
		"cmd":                             i.createEmptyFolders,
		mainAppPath:                       i.generateCmdApp,
		"cmd/cli/main.go":                 i.generateCmdCli,
		"config":                          i.createEmptyFolders,
		"config/config.yaml":              i.generateConfigYaml,
		"config/config.go":                i.generateConfigGo,
		"db":                              i.createEmptyFolders,
		"db/migrations.go":                i.generateDBMigrations,
		"internal/providers":              i.createEmptyFolders,
		"internal/providers/providers.go": i.generateProviders,
		"internal/types":                  i.createEmptyFolders,
		"internal/services":               i.createEmptyFolders,
		"internal/repositories":           i.createEmptyFolders,
		"internal/jobs":                   i.createEmptyFolders,
		"internal/mails":                  i.createEmptyFolders,
	}

	webFiles := map[string]func(path string) error{

		"internal/web/server.go":                       i.generateWebServer,
		"internal/web/routes.go":                       i.generateWebRoutes,
		"internal/web/example/find_example_handler.go": i.generateWebFindExampleHandler,
		"internal/types/example.go":                    i.generateWebExample,
		"internal/services/example_service.go":         i.generateWebExampleService,
		"internal/repositories/example_repository.go":  i.generateWebExampleRepository,
	}

	dockerFiles := map[string]func(path string) error{
		"Dockerfile":          i.generateDockerfile,
		"docker-compose.yaml": i.generateDockerCompose,
		".dockerignore":       i.generateDockerIgnore,
	}

	goModFile := map[string]func(path string) error{
		"go.mod": i.generateGoMod,
	}

	binariesThatMustBeInstalled := []string{
		"docker",
		"go",
	}

	// Dynamically generate the help text sections
	defaultFilesHelp := formatFilesList(defaultFiles)
	webFilesHelp := formatFilesList(webFiles)
	dockerFilesHelp := formatFilesList(dockerFiles)
	gomodFilesHelp := formatFilesList(goModFile)
	binariesHelp := formatBinariesList(binariesThatMustBeInstalled)

	longDescription := fmt.Sprintf(`Initialize a new project with the necessary configuration files and directories.

The 'goframe init' command creates a new Goframe application with a default
directory structure and configuration at the current path or path that you specify.

The following files will be created:
%s

Files if --gomod=true:
%s

Files if --web=true:
%s

Files if --docker=true:
%s

Required binaries:
%s`, defaultFilesHelp, gomodFilesHelp, webFilesHelp, dockerFilesHelp, binariesHelp)

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

			for path, fn := range dockerFiles {
				err := fn(path)
				if err != nil {
					return fmt.Errorf("failed to create %s: %v", path, err)
				}
			}

			for path, fn := range goModFile {
				err := fn(path)
				if err != nil {
					return fmt.Errorf("failed to create %s: %v", path, err)
				}
			}

			for _, info := range i.filesCreated {
				fmt.Printf("Created: %s\n", filepath.Clean(filepath.Join(i.folder, info.path)))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&i.databaseName, "databaseName", "d", "postgres", "Database type: mysql, postgres, sqlite")
	cmd.Flags().BoolVarP(&i.docker, "docker", "D", true, "Initialize with Docker support (docker-compose & Dockerfile)")
	cmd.Flags().StringVarP(&i.folder, "folder", "f", ".", "Project folder name")
	cmd.Flags().BoolVarP(&i.web, "web", "w", false, "Initialize a web application")
	cmd.Flags().StringVarP(&i.goModName, "gomod", "g", "", "Create a go.mod file with go module name if set")
	cmd.Flags().BoolVarP(&i.maintainer, "maintainer", "m", false, "Add specific maintainer thing to test the framework")

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
	maintainer   bool
	web          bool
	docker       bool
	filesCreated []fileinfo
}

// Functions for generating default files
func (i *initializer) generateCmdApp(path string) error {
	// Generate main application entry point
	return nil
}

func (i *initializer) generateCmdCli(path string) error {
	// Generate CLI application entry point
	return nil
}

func (i *initializer) generateConfigYaml(path string) error {
	// Generate default config.yaml file
	return nil
}

func (i *initializer) generateConfigGo(path string) error {
	// Generate config.go file for loading configuration
	return nil
}

func (i *initializer) createEmptyFolders(path string) error {
	// Create empty folders for migrations, types, services, repositories, jobs, and mails
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return fmt.Errorf("failed to create folder: %v", err)
	}

	i.filesCreated = append(i.filesCreated, fileinfo{dir: true, path: path})

	return nil
}

func (i *initializer) generateDBMigrations(path string) error {
	// Generate databaseName migrations file
	return nil
}

func (i *initializer) generateProviders(path string) error {
	// Generate providers.go file
	return nil
}

// Functions for web-related files
func (i *initializer) generateWebServer(path string) error {
	// Generate web server implementation
	return nil
}

func (i *initializer) generateWebRoutes(path string) error {
	// Generate routes configuration for web server
	return nil
}

func (i *initializer) generateWebFindExampleHandler(path string) error {
	// Generate example handler for web server
	return nil
}

func (i *initializer) generateWebExample(path string) error {
	// Generate example type definition
	return nil
}

func (i *initializer) generateWebExampleService(path string) error {
	// Generate example service implementation
	return nil
}

func (i *initializer) generateWebExampleRepository(path string) error {
	// Generate example repository implementation
	return nil
}

// Functions for Docker-related files
func (i *initializer) generateDockerfile(path string) error {
	if i.docker == false {
		return nil
	}

	// Generate Dockerfile for the application
	dockerFileLines := []string{
		"FROM golang:1.20-alpine AS builder",
		"WORKDIR /app",
		"COPY go.mod .",
		"COPY go.sum .",
		"RUN go mod download",
		"COPY . .",
		"RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/app",
		"",
		"FROM alpine:latest",
		"WORKDIR /root/",
		"COPY --from=builder /app .",
		"CMD [\"./app\"]",
		"",
	}
	err := os.WriteFile(path, []byte(strings.Join(dockerFileLines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to create Dockerfile: %v", err)
	}

	i.filesCreated = append(i.filesCreated, fileinfo{dir: false, path: path})

	return nil
}

func (i *initializer) generateDockerCompose(path string) error {
	if i.docker == false {
		return nil
	}

	dcLines := []string{
		"version: '3.8'",
		"services:",
		"  mailpit:",
		"    image: axllent/mailpit:latest",
		"    ports:",
		"      - 8025:8025",
		"      - 8026:8026",
		"    volumes:",
		"      - ./data:/data",
		"    environment:",
		"      MP_MAX_MESSAGES: 5000",
		"      MP_DATABASE: /data/mailpit.db",
		"      MP_SMTP_AUTH_ACCEPT_ANY: 1",
		"      MP_SMTP_AUTH_ALLOW_INSECURE: 1",
	}

	if i.databaseName == "mysql" {
		dcLines = append(dcLines, "  mysql:")
		dcLines = append(dcLines, "    image: mysql:latest")
		dcLines = append(dcLines, "    environment:")
		dcLines = append(dcLines, "      MYSQL_ROOT_PASSWORD: root")
		dcLines = append(dcLines, "      MYSQL_DATABASE: test")
		dcLines = append(dcLines, "    ports:")
		dcLines = append(dcLines, "      - 3306:3306")
		dcLines = append(dcLines, "    volumes:")
		dcLines = append(dcLines, "      - mysql_data:/var/lib/mysql")
	}

	if i.databaseName == "postgres" {
		dcLines = append(dcLines, "  postgres:")
		dcLines = append(dcLines, "    image: postgres:latest")
		dcLines = append(dcLines, "    environment:")
		dcLines = append(dcLines, "      POSTGRES_PASSWORD: root")
		dcLines = append(dcLines, "      POSTGRES_DB: test")
		dcLines = append(dcLines, "    ports:")
		dcLines = append(dcLines, "      - 5432:5432")
		dcLines = append(dcLines, "    volumes:")
		dcLines = append(dcLines, "      - postgres_data:/var/lib/postgresql/data")
	}

	err := os.WriteFile(path, []byte(strings.Join(dcLines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to create docker-compose.yaml: %v", err)
	}

	i.filesCreated = append(i.filesCreated, fileinfo{dir: false, path: path})

	return nil
}

func (i *initializer) generateDockerIgnore(path string) error {
	if i.docker == false {
		return nil
	}

	dockerIgnoreLines := []string{
		"*.log",
		"*.tmp",
		"*.db",
		"*.sqlite",
		"db/storage.db",
		".env",
		".git",
	}

	err := os.WriteFile(path, []byte(strings.Join(dockerIgnoreLines, "\n")), 0644)
	if err != nil {
		return fmt.Errorf("failed to create .dockerignore: %v", err)
	}

	return nil
}

func (i *initializer) generateGoMod(_ string) error {
	out := bytes.NewBuffer(nil)
	cmd := exec.Command("go", "mod", "init", i.goModName)
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create go.mod file: %v", err)
	}

	// add dependencies
	file, err := os.OpenFile("go.mod", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open go.mod file: %v", err)
	}

	defer file.Close()

	dependencies := []string{
		"github.com/alexisvisco/goframe",
	}

	for _, dep := range dependencies {
		file.WriteString(fmt.Sprintf(`require "%s" latest`, dep))
		file.WriteString("\n")
	}

	if i.maintainer {
		file.WriteString(fmt.Sprintf(`replace "%s" => "%s"`, "github.com/alexisvisco/goframe", "../"))
		file.WriteString("\n")
	}

	out = bytes.NewBuffer(nil)
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to tidy go.mod file: %v", err)
	}

	i.filesCreated = append(i.filesCreated, fileinfo{dir: false, path: "go.mod"})

	return nil
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

			stat, _ := os.Stat(i.folder)
			i.filesCreated = append(i.filesCreated, fileinfo{dir: stat.IsDir(), path: "."})
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
	var validDatabases = []string{"mysql", "postgres", "sqlite"}
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

func (i *initializer) installDependency(dep string) error {
	cmd := exec.Command("go", "get", dep)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to install dependency %s: %v", dep, err)
	}
	return nil
}

// formatFilesList converts a slice of file paths to a formatted string with indentation and dashes
func formatFilesList(files map[string]func(path string) error) string {
	var result strings.Builder
	for file := range files {
		result.WriteString(fmt.Sprintf("  - %s\n", file))
	}
	return strings.TrimSuffix(result.String(), "\n")
}

// formatBinariesList formats the list of required binaries with conditionals where appropriate
func formatBinariesList(binaries []string) string {
	var result strings.Builder
	for _, binary := range binaries {
		line := fmt.Sprintf("  - %s", binary)
		if binary == "docker" {
			line += " (if --docker=true)"
		}
		result.WriteString(line + "\n")
	}
	return strings.TrimSuffix(result.String(), "\n")
}
