package cmd

import (
	"fmt"

	"github.com/diffpal/diffpal/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the DiffPal version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "diffpal %s\n", version.String())
			if err != nil {
				return err
			}
			return nil
		},
	}
}
