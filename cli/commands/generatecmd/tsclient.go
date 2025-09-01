package generatecmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/gentsclient"
	"github.com/alexisvisco/goframe/core/helpers/introspect"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/http/apidoc"
	"github.com/spf13/cobra"
)

func tsclientCmd() *cobra.Command {
	var flagFile string
	var flagPkg string
	cmd := &cobra.Command{
		Use: "client [packages...]",
		RunE: func(cmd *cobra.Command, args []string) error {
			workdir, _ := cmd.Context().Value("workdir").(string)
			packages, err := genhelper.CollectRootHandlerPackages(workdir)
			if err != nil {
				return fmt.Errorf("failed to collect root handler packages: %w", err)
			}

			var paths []string
			for _, pkg := range packages {
				if pkg.Path == flagPkg {
					paths = append(paths, pkg.Path)
					paths = append(paths, pkg.Subfolders...)
					break
				}
			}

			if len(paths) == 0 {
				return fmt.Errorf("no package found with name %s", flagPkg)
			}

			routes, err := genhelper.CollectRoutesDocumentation(workdir, paths)
			if err != nil {
				return err
			}

			var rootImportPath string
			for _, r := range routes {
				if strings.HasSuffix(r.PackagePath, flagPkg) {
					rootImportPath = r.PackagePath
					break
				}
			}
			if rootImportPath == "" {
				return fmt.Errorf("failed to resolve root package import path")
			}

			prefixMap := collectTypePrefixes(routes, rootImportPath)

			generator := gentsclient.NewTypescriptClientGenerator(rootImportPath, prefixMap)

			for _, r := range routes {
				if r.Request != nil {
					generator.AddSchema("", true, *r.Request)
				}
				for _, response := range r.StatusToResponse {
					if response.Response != nil {
						generator.AddSchema("", false, *response.Response)
					}
				}
				generator.AddRoute(*r)
			}

			if flagFile != "" {
				file, err := os.OpenFile(flagFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return fmt.Errorf("failed to open output file %s: %w", flagFile, err)
				}

				defer file.Close()
				content := generator.File()
				if _, err := file.WriteString(content); err != nil {
					return fmt.Errorf("failed to write to output file %s: %w", flagFile, err)
				}
			} else {
				fmt.Println(generator.File())
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagFile, "file", "f", "", "Output file for the generated TypeScript client code")
	cmd.Flags().StringVarP(&flagPkg, "pkg", "p", "internal/v1handler", "Package name where routes are defined")

	return cmd
}

func collectTypePrefixes(routes []*apidoc.Route, rootImportPath string) map[string]string {
	type info struct {
		typeName string
		pkgPath  string
		baseName string
	}

	baseMap := make(map[string][]info)
	visited := make(map[string]bool)

	var visitField func(ft introspect.FieldType)
	var visitObject func(obj *introspect.ObjectType)

	visitObject = func(obj *introspect.ObjectType) {
		if obj == nil {
			return
		}

		// Prevent infinite recursion by tracking visited types
		if !obj.IsAnonymous {
			if visited[obj.TypeName] {
				return
			}
			visited[obj.TypeName] = true

			pkgPath := obj.TypeName[:strings.LastIndex(obj.TypeName, ".")]
			base := obj.TypeName[strings.LastIndex(obj.TypeName, ".")+1:]
			baseMap[base] = append(baseMap[base], info{obj.TypeName, pkgPath, base})
		}

		for _, f := range obj.Fields {
			visitField(f.Type)
		}
	}

	visitField = func(ft introspect.FieldType) {
		if ft.Array != nil {
			visitField(ft.Array.ItemType)
		}
		if ft.Map != nil {
			visitField(ft.Map.Key)
			visitField(ft.Map.Value)
		}
		if ft.Object != nil {
			visitObject(ft.Object)
		}
	}

	for _, r := range routes {
		if r.Request != nil {
			visitObject(r.Request)
		}
		for _, resp := range r.StatusToResponse {
			if resp.Response != nil {
				visitObject(resp.Response)
			}
		}
	}

	prefixMap := make(map[string]string)
	for _, infos := range baseMap {
		if len(infos) == 1 {
			prefixMap[infos[0].typeName] = ""
			continue
		}
		for _, inf := range infos {
			if strings.HasPrefix(inf.pkgPath, rootImportPath) {
				rel := strings.TrimPrefix(inf.pkgPath, rootImportPath)
				rel = strings.TrimPrefix(rel, "/")
				if rel == "" {
					prefixMap[inf.typeName] = ""
				} else {
					segs := strings.Split(rel, "/")
					var prefix strings.Builder
					for _, s := range segs {
						prefix.WriteString(str.ToPascalCase(s))
					}
					prefixMap[inf.typeName] = prefix.String()
				}
			} else {
				pkgName := path.Base(inf.pkgPath)
				prefixMap[inf.typeName] = str.ToPascalCase(pkgName)
			}
		}
	}

	return prefixMap
}
