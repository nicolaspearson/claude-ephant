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
	// Abs fails only without a working directory; the raw value still names it.
	if directory, err := filepath.Abs(configDirectory); err == nil {
		configDirectory = directory
	}
	return nameOf(configDirectory)
}

func Discover(home string, configDirectory string) ([]Root, []string, error) {
	candidates, err := filepath.Glob(filepath.Join(home, ".claude-*"))
	if err != nil {
		return nil, nil, fmt.Errorf("discover profiles: %w", err)
	}
	candidates = append(candidates, filepath.Join(home, ".claude"))
	var roots []Root
	for _, directory := range candidates {
		holds, err := holdsProjects(directory)
		if err != nil {
			return nil, nil, fmt.Errorf("discover profiles: %w", err)
		}
		if holds {
			roots = append(roots, Root{Directory: directory, Name: nameOf(directory)})
		}
	}
	var skipped []string
	if configDirectory != "" {
		root, reason, err := configRoot(roots, configDirectory)
		if err != nil {
			return nil, nil, fmt.Errorf("discover profiles: %w", err)
		}
		if root != nil {
			roots = append(roots, *root)
		}
		if reason != "" {
			skipped = append(skipped, reason)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Name < roots[j].Name })
	return roots, skipped, nil
}

// The archive keys a file by profile name, so a second directory under a taken name would share another's tree.
func configRoot(roots []Root, configDirectory string) (*Root, string, error) {
	directory, err := filepath.Abs(configDirectory)
	if err != nil {
		return nil, "", err
	}
	holds, err := holdsProjects(directory)
	if err != nil || !holds {
		return nil, "", err
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return nil, "", err
	}
	for _, root := range roots {
		rootReal, err := filepath.EvalSymlinks(root.Directory)
		if err != nil {
			return nil, "", err
		}
		if rootReal == resolved {
			return nil, "", nil
		}
	}
	name := nameOf(directory)
	for _, root := range roots {
		if root.Name == name {
			return nil, fmt.Sprintf("profile %s: its name %s is taken by %s", directory, name, root.Directory), nil
		}
	}
	return &Root{Directory: directory, Name: name}, "", nil
}

func holdsProjects(directory string) (bool, error) {
	if !isDirectory(directory) {
		return false, nil
	}
	info, err := os.Stat(filepath.Join(directory, "projects"))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
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
