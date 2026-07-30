package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/cache"
	"github.com/danieljustus/symaira-scope/internal/output"
)

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "cache", Short: "Inspect or manage the snapshot cache"}

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show cache status and metadata",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			return out.Print(cache.Stats())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Delete the snapshot cache file",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := cache.Clear(); err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "cache clear")
			}
			fmt.Println("Cache cleared.")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:    "stats",
		Short:  "Print cache statistics as JSON",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			fmt.Fprintln(os.Stderr, "warning: 'cache stats' is deprecated, use 'cache show' instead")
			return out.Print(cache.Stats())
		},
	})

	return cmd
}
