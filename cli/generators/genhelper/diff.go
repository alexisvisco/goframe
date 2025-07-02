package genhelper

import (
	"crypto/sha256"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/imports"
)

type fileSnapshot map[string][32]byte

func snapshotDir(root string) (fileSnapshot, error) {
	snap := make(fileSnapshot)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		snap[rel] = sha256.Sum256(data)
		return nil
	})
	return snap, err
}

func diffSnapshots(before, after fileSnapshot) (added, changed, deleted []string) {
	for path, sum := range after {
		b, ok := before[path]
		if !ok {
			added = append(added, path)
			continue
		}
		if b != sum {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			deleted = append(deleted, path)
		}
	}

	sort.Strings(added)
	sort.Strings(changed)
	sort.Strings(deleted)
	return
}

func formatGoFile(path string) error {
	// Read the file
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Process the source with imports (removes unused imports and formats imports)
	formatted, err := imports.Process(path, src, &imports.Options{
		Comments:  true,
		TabIndent: true,
		TabWidth:  8,
	})
	if err != nil {
		// If imports processing fails, try basic formatting
		formatted, err = format.Source(src)
		if err != nil {
			return err
		}
	}

	// Write the formatted source back to the file
	return os.WriteFile(path, formatted, 0644)
}

func WithFileDiff(run func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		before, _ := snapshotDir(".")
		if err := run(cmd, args); err != nil {
			return err
		}
		after, _ := snapshotDir(".")
		added, changed, deleted := diffSnapshots(before, after)

		// Format Go files that were added or modified
		for _, f := range append(added, changed...) {
			if strings.HasSuffix(f, ".go") {
				if err := formatGoFile(f); err != nil {
					fmt.Printf("Warning: Could not format %s: %v\n", f, err)
				}
			}
		}

		if len(added)+len(changed)+len(deleted) == 0 {
			fmt.Println("No file changes detected.")
			return nil
		}
		for _, f := range added {
			fmt.Println("A", f)
		}
		for _, f := range changed {
			fmt.Println("M", f)
		}
		for _, f := range deleted {
			fmt.Println("D", f)
		}
		return nil
	}
}
