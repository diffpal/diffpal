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
			wizard, _ := cmd.Flags().GetBool("wizard")
			if wizard {
				setup, _ := cmd.Flags().GetString("setup")
				platform, _ := cmd.Flags().GetString("platform")
				profile, _ := cmd.Flags().GetString("profile")
				blockOn, _ := cmd.Flags().GetString("block-on")
				result, err := initcmd.InitWizardWorkspace(initcmd.WizardOptions{
					InitOptions: initcmd.InitOptions{
						WorkingDir: workingDir,
						ConfigPath: configPath,
						StatePath:  statePath,
						Force:      force,
					},
					Setup:    setup,
					Platform: platform,
					Profile:  profile,
					BlockOn:  blockOn,
				})
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "initialized diffpal workspace at %s\nsetup: %s\nplatform: %s\nprofile: %s\nblock_on: %s\n", result.ConfigPath, result.Setup, result.Platform, result.Profile, result.BlockOn)
				return err
			}
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
	initCmd.Flags().Bool("wizard", false, "Generate first-run onboarding config")
	initCmd.Flags().String("setup", "codex-api-key", "Wizard setup: codex-api-key, codex-subscription, copilot-github-token, opencode-acp, or generic-acp")
	initCmd.Flags().String("platform", "auto", "Wizard platform: auto, github, gitlab, azure, or none")
	initCmd.Flags().String("profile", "ci", "Wizard review profile to generate")
	initCmd.Flags().String("block-on", "high", "Wizard gate threshold: low, medium, high, or critical")
	return initCmd
}
