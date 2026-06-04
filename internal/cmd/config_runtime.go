package cmd

import (
	"os"

	"github.com/diffpal/diffpal/internal/config"
)

func currentWorkingDir() (string, error) {
	return os.Getwd()
}

func loadRequiredConfig() (config.Config, error) {
	workingDir, err := currentWorkingDir()
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := config.LoadConfig(workingDir, rootConfigDir, rootProfile)
	if err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}
