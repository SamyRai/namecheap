package config

import (
	"os"
	"path/filepath"
)

// FindProjectConfigPath looks for config file in the project directory
func FindProjectConfigPath() string {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Look for config in current directory and parent directories
	for {
		configPath := filepath.Join(cwd, "configs", ".zonekit.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		// Move up one directory
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break // Reached root
		}
		cwd = parent
	}

	return ""
}

