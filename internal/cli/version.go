package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/XKHoshizora/specmetry/internal/buildinfo"
	"github.com/XKHoshizora/specmetry/internal/store/sqlite"
	"github.com/XKHoshizora/specmetry/web"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			info := buildinfo.Get()
			commit := info.Commit
			if info.Modified {
				commit += " (modified)"
			}
			dashboard := "not embedded (built with -tags nowebui)"
			if web.Embedded() {
				dashboard = "embedded"
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "specmetry %s\n", info.Version)
			_, _ = fmt.Fprintf(out, "commit:          %s\n", commit)
			_, _ = fmt.Fprintf(out, "go:              %s\n", info.GoVersion)
			_, _ = fmt.Fprintf(out, "platform:        %s\n", info.Platform)
			_, _ = fmt.Fprintf(out, "database schema: v%d\n", sqlite.SchemaVersion())
			_, _ = fmt.Fprintf(out, "dashboard:       %s\n", dashboard)
			return nil
		}),
	}
}
