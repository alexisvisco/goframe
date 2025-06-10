package mailcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewCmdRootMail(subCommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mails <subcommand>",
		Short: "Mail utilities",
	}
	cmd.AddCommand(syncCmd())
	for _, sc := range subCommands {
		cmd.AddCommand(sc)
	}
	return cmd
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Compile MJML templates to HTML",
		RunE: func(cmd *cobra.Command, args []string) error {
			htmlDir := filepath.Join("views", "mails", "html")
			if err := os.MkdirAll(htmlDir, 0o755); err != nil {
				return fmt.Errorf("create html dir: %w", err)
			}
			entries, err := os.ReadDir(filepath.Join("views", "mails"))
			if err != nil {
				return fmt.Errorf("read mails dir: %w", err)
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".mjml.tmpl") {
					continue
				}
				base := strings.TrimSuffix(e.Name(), ".mjml.tmpl")
				src := filepath.Join("views", "mails", e.Name())
				dst := filepath.Join(htmlDir, base+".html")
				out, err := exec.Command("bin/mjml", src, "-o", dst).CombinedOutput()
				if err != nil {
					return fmt.Errorf("mjml failed: %w: %s", err, string(out))
				}
			}
			return nil
		},
	}
}
