package datadir

import (
	"fmt"
	"os"
	"path/filepath"
)

func Path() (string, error) {
	if override := os.Getenv("ELE_HOME"); override != "" {
		absolute, err := filepath.Abs(override)
		if err != nil {
			return "", fmt.Errorf("resolve ELE_HOME %s: %w", override, err)
		}
		return absolute, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve data dir: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "claude-ephant"), nil
}
