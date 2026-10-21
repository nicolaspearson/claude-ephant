package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Root struct {
	Directory string
	Name      string
}

func Current(configDirectory string) string {
	if configDirectory == "" {
		return "claude"
	}
	return nameOf(configDirectory)
}

func Discover(home string) ([]Root, error) {
	candidates, err := filepath.Glob(filepath.Join(home, ".claude-*"))
	if err != nil {
		return nil, fmt.Errorf("discover profiles: %w", err)
	}
	candidates = append(candidates, filepath.Join(home, ".claude"))
	var roots []Root
	for _, directory := range candidates {
		if !isDirectory(directory) {
			continue
		}
		info, err := os.Stat(filepath.Join(directory, "projects"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("discover profiles: %w", err)
		}
		if !info.IsDir() {
			continue
		}
		roots = append(roots, Root{Directory: directory, Name: nameOf(directory)})
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Name < roots[j].Name })
	return roots, nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func nameOf(directory string) string {
	name := strings.TrimPrefix(strings.TrimPrefix(filepath.Base(directory), ".claude"), "-")
	if name == "" {
		return "claude"
	}
	return name
}
