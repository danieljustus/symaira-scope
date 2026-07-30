package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/cache"
	"github.com/danieljustus/symaira-scope/internal/output"
	"github.com/danieljustus/symaira-scope/internal/scan"
)

func newScanCmd() *cobra.Command {
	var noCache bool
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Full inventory snapshot (ports + MCP servers + containers)",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			if !noCache {
				if snap, err := cache.Load(); err != nil {
					return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "cache load")
				} else if snap != nil {
					return out.Print(snap)
				}
			}

			snap, err := scan.Build()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "scan")
			}

			if !noCache {
				if err := cache.Save(&snap); err != nil {
					slog.Warn("cache save failed", "err", err)
				}
			}

			return out.Print(snap)
		},
	}
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Skip cache; always run a fresh scan")
	return cmd
}
