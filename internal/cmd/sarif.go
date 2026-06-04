package cmd

import (
	"fmt"

	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/sarif"
	"github.com/spf13/cobra"
)

func newSARIFCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sarif",
		Short: "Convert findings bundle to SARIF",
		RunE: func(cmd *cobra.Command, args []string) error {
			inputPath, _ := cmd.Flags().GetString("input")
			outPath, _ := cmd.Flags().GetString("out")
			bundle, err := findings.ReadBundle(inputPath)
			if err != nil {
				return withExitCode(4, err)
			}
			if outPath == "" {
				outPath = ".artifacts/diffpal/diffpal.sarif"
			}
			report := sarif.ToReport(bundle)
			raw, err := sarif.ToJSON(report)
			if err != nil {
				return withExitCode(4, err)
			}
			if err := findings.WriteStringBundle(outPath, string(raw)); err != nil {
				return withExitCode(4, err)
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"sarif=%s findings=%d version=%s\n",
				outPath,
				len(bundle.Findings),
				bundle.Version,
			)
			return withExitCode(4, err)
		},
	}
	cmd.Flags().String("input", findings.DefaultBundlePath, "Input findings bundle path")
	cmd.Flags().String("out", ".artifacts/diffpal/diffpal.sarif", "Output SARIF report path")
	return cmd
}
