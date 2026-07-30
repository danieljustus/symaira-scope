package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/updatecheck"
	"github.com/danieljustus/symaira-corekit/versionkit"
)

func newVersionCmd() *cobra.Command {
	var check bool
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(_ *cobra.Command, _ []string) error {
			info := versionkit.New("symscope", version, 1)
			if jsonOut {
				return info.Write(os.Stdout)
			}
			fmt.Println(info.String())
			if !check {
				return nil
			}
			release, err := updatecheck.NewChecker("danieljustus", "symaira-scope").Check(context.Background(), version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "update check failed: %v\n", err)
				return nil
			}
			if release != nil {
				fmt.Printf("Update available: %s\n%s\n", release.TagName, release.HTMLURL)
			} else {
				fmt.Println("Already up to date.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Check for updates on GitHub")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	return cmd
}
