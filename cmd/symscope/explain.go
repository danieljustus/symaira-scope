package main

import (
	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"

	"github.com/danieljustus/symaira-scope/internal/explain"
	"github.com/danieljustus/symaira-scope/internal/output"
)

func newExplainCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "explain", Short: "Explain what uses a port or server"}

	var port int
	portCmd := &cobra.Command{
		Use:   "port",
		Short: "Explain what's using a specific port",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			exp, err := explain.ExplainPort(port)
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "explain port")
			}
			return out.Print(exp)
		},
	}
	portCmd.Flags().IntVar(&port, "number", 0, "Port number to explain")
	portCmd.MarkFlagRequired("number")
	cmd.AddCommand(portCmd)

	var serverName string
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Explain a specific MCP server",
		RunE: func(_ *cobra.Command, _ []string) error {
			out := output.New("json")
			exp, err := explain.ExplainServer(serverName)
			if err != nil {
				return exitcodes.Wrap(err, exitcodes.ExitSoftware, exitcodes.KindInternal, "explain server")
			}
			return out.Print(exp)
		},
	}
	serverCmd.Flags().StringVar(&serverName, "name", "", "Server name to explain")
	serverCmd.MarkFlagRequired("name")
	cmd.AddCommand(serverCmd)

	return cmd
}
