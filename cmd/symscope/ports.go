package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/config"
	"github.com/danieljustus/symaira-scope/internal/output"
	"github.com/danieljustus/symaira-scope/internal/ports"
)

func newPortsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "ports", Short: "List or suggest local ports"}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List local listening ports",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			p, err := ports.ListListening()
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "list ports")
			}
			return out.Print(p)
		},
	})

	var count, from, to int
	suggest := &cobra.Command{
		Use:   "suggest",
		Short: "Suggest free TCP ports in a range",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := output.New("json")
			cfg, err := config.Load()
			if err != nil {
				slog.Warn("config load failed, using defaults", "err", err)
				cfg = config.Defaults()
			}
			if !cmd.Flags().Changed("from") {
				from = cfg.Ports.SuggestFrom
			}
			if !cmd.Flags().Changed("to") {
				to = cfg.Ports.SuggestTo
			}
			return out.Print(map[string]any{"free": ports.SuggestFree(count, from, to)})
		},
	}
	suggest.Flags().IntVar(&count, "count", 3, "How many free ports to suggest")
	suggest.Flags().IntVar(&from, "from", 3000, "Range start (default from config)")
	suggest.Flags().IntVar(&to, "to", 9999, "Range end (default from config)")
	cmd.AddCommand(suggest)

	return cmd
}
