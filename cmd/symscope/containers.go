package main

import (
	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-scope/internal/containers"
	"github.com/danieljustus/symaira-scope/internal/output"
)

func newContainersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "containers",
		Short: "List running containers and published ports",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			c, notes := containers.List()
			return out.Print(map[string]any{"containers": c, "notes": notes})
		},
	}
}
