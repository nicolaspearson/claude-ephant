package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type Root struct {
	Directory string
	Name      string
}

func Current(home string, configDirectory string) string {
	if configDirectory == "" {
		return "claude"
	}
	// Abs fails only without a working directory; the raw value still names it.
	if directory, err := filepath.Abs(configDirectory); err == nil {
		configDirectory = directory
	}
	name := nameOf(configDirectory)
	if home == "" {
		return name
	}
	roots, _, err := Discover(home, configDirectory)
	if err != nil {
		return name
	}
	resolved, resolveErr := filepath.EvalSymlinks(configDirectory)
	for _, root := range roots {
		if rootReal, err := filepath.EvalSymlinks(root.Directory); resolveErr == nil && err == nil && rootReal == resolved {
			return root.Name
		}
	}
	// Name held by another profile, skipped or not; the path tells them apart.
	for _, root := range roots {
		if root.Name == name {
			return configDirectory
		}
	}
	return name
}

func Discover(home string, configDirectory string) ([]Root, []string, error) {
	entries, err := os.ReadDir(home)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("discover profiles: %w", err)
	}
	// ~/.claude first: it keeps its name and its real path against a link or a name clash.
	candidates := []string{filepath.Join(home, ".claude")}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".claude-") {
			candidates = append(candidates, filepath.Join(home, entry.Name()))
		}
	}
	var roots []Root
	realPaths := map[string]int{}
	for _, directory := range candidates {
		holds, err := holdsProjects(directory)
		if err != nil {
			return nil, nil, fmt.Errorf("discover profiles: %w", err)
		}
		if !holds {
			continue
		}
		resolved, err := filepath.EvalSymlinks(directory)
		if err != nil {
			return nil, nil, fmt.Errorf("discover profiles: %w", err)
		}
		kept, seen := realPaths[resolved]
		switch {
		case !seen:
			realPaths[resolved] = len(roots)
			roots = append(roots, Root{Directory: directory})
		case roots[kept].Directory != candidates[0] && !isSymlink(directory):
			roots[kept].Directory = directory
		}
	}
	var skipped []string
	named := roots[:0]
	for _, root := range roots {
		root.Name = nameOf(root.Directory)
		if taken := slices.IndexFunc(named, func(other Root) bool { return other.Name == root.Name }); taken >= 0 {
			skipped = append(skipped, fmt.Sprintf("profile %s: its name %s is taken by %s", root.Directory, root.Name, named[taken].Directory))
			continue
		}
		named = append(named, root)
	}
	roots = named
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

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&fs.ModeSymlink != 0
}

func nameOf(directory string) string {
	name := strings.TrimPrefix(strings.TrimPrefix(filepath.Base(directory), ".claude"), "-")
	if name == "" {
		return "claude"
	}
	return name
}
