package cmd

import (
	"fmt"

	"github.com/diffpal/diffpal/internal/initcmd"
	ip "github.com/diffpal/diffpal/internal/provider"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Generate starter workspace configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			workingDir, err := currentWorkingDir()
			if err != nil {
				return err
			}
			configPath, _ := cmd.Flags().GetString("config")
			statePath, _ := cmd.Flags().GetString("state")
			force, _ := cmd.Flags().GetBool("force")
			detected := ip.AutodetectProviders()
			detectedKeys := make([]string, 0, len(detected))
			for _, d := range detected {
				detectedKeys = append(detectedKeys, d.Key)
			}
			result, err := initcmd.InitWorkspace(initcmd.InitOptions{
				WorkingDir: workingDir,
				ConfigPath: configPath,
				StatePath:  statePath,
				Force:      force,
			}, detectedKeys)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "initialized diffpal workspace at %s\n", result.ConfigPath)
			return err
		},
	}
	initCmd.Flags().String("config", "", "Path to write repo config (defaults to .config/diffpal/config.yaml)")
	initCmd.Flags().String("state", "", "State directory for local cache (defaults to .config/diffpal/state)")
	initCmd.Flags().Bool("force", false, "Overwrite existing files")
	return initCmd
}
