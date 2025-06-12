package generatecmd

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
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

func withFileDiff(run func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		before, _ := snapshotDir(".")
		if err := run(cmd, args); err != nil {
			return err
		}
		after, _ := snapshotDir(".")
		added, changed, deleted := diffSnapshots(before, after)
		if len(added)+len(changed)+len(deleted) == 0 {
			fmt.Println("No file changes detected.")
			return nil
		}
		fmt.Println("Files summary:")
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
