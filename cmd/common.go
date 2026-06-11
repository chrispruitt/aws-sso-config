package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveConfigFile() (string, error) {
	if env := os.Getenv("AWS_CONFIG_FILE"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".aws", "config"), nil
}
