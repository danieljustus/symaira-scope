package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/output"
	"github.com/danieljustus/symaira-scope/internal/scan"
	"github.com/danieljustus/symaira-scope/internal/watch"
)

func newWatchCmd() *cobra.Command {
	var interval time.Duration
	var format string
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch for changes in ports, conflicts, and MCP configs",
		RunE: func(_ *cobra.Command, _ []string) error {
			if format != "ndjson" {
				return exitcodes.Wrap(fmt.Errorf("unsupported format %q (only ndjson is supported)", format), exitcodes.ExitConfig, exitcodes.KindValidation, "watch")
			}
			if interval <= 0 {
				return exitcodes.Wrap(fmt.Errorf("interval must be greater than 0"), exitcodes.ExitConfig, exitcodes.KindValidation, "watch")
			}

			out := output.New("ndjson")

			oldSnap, err := scan.Build()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "watch initial scan")
			}

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for range ticker.C {
				newSnap, err := scan.Build()
				if err != nil {
					slog.Warn("scan failed", "err", err)
					continue
				}

				events := watch.Diff(oldSnap, newSnap)
				for _, e := range events {
					if err := out.Print(e); err != nil {
						slog.Warn("failed to encode event", "err", err)
					}
				}
				oldSnap = newSnap
			}
			return nil
		},
	}
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Polling interval (e.g. 1s, 500ms)")
	cmd.Flags().StringVar(&format, "format", "ndjson", "Output format (ndjson)")
	return cmd
}
