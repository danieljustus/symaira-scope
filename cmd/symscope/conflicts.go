package main

import (
	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/mcpcfg"
	"github.com/danieljustus/symaira-scope/internal/output"
	"github.com/danieljustus/symaira-scope/internal/ports"
)

func newConflictsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "conflicts",
		Short: "Report ports bound by more than one process or occupied by configured services",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			p, err := ports.ListListening()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "list ports")
			}
			all := ports.Conflicts(p)
			servers, _ := mcpcfg.Discover(mcpcfg.DefaultSources())
			all = append(all, ports.MCPServerConflicts(servers, p)...)
			return out.Print(all)
		},
	}
}
