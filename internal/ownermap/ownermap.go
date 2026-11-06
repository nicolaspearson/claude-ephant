package ownermap

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

// Fingerprint of a missing map, so creating one counts as a change.
const absent = "absent"

// Owner becomes a folder name under notes/.
var ownerShape = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

type OwnerMap struct {
	Owners   []string          `json:"owners"`
	Paths    map[string]string `json:"paths"`
	Projects map[string]string `json:"projects"`
}

func Fingerprint(path string) (string, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return absent, nil
	}
	if err != nil {
		return "", err
	}
	return fingerprint(content), nil
}

func Load(path string) (OwnerMap, error) {
	ownerMap, _, err := Read(path)
	return ownerMap, err
}

func Moves(before OwnerMap, after OwnerMap, directories []dto.ProjectDirectory) []dto.OwnerMove {
	seen := map[dto.OwnerMove]bool{}
	var moves []dto.OwnerMove
	for _, directory := range directories {
		move := dto.OwnerMove{
			From:    before.Owner(directory.Project, directory.Directory),
			Project: directory.Project,
			To:      after.Owner(directory.Project, directory.Directory),
		}
		if move.From == move.To || seen[move] {
			continue
		}
		seen[move] = true
		moves = append(moves, move)
	}
	slices.SortFunc(moves, func(a, b dto.OwnerMove) int {
		return cmp.Or(cmp.Compare(a.Project, b.Project), cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})
	return moves
}

// Read returns the fingerprint of the bytes it parsed, so a map written after the read is not taken for the one parsed.
func Read(path string) (OwnerMap, string, error) {
	ownerMap := OwnerMap{Owners: []string{}, Paths: map[string]string{}, Projects: map[string]string{}}
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return ownerMap, absent, nil
	}
	if err != nil {
		return OwnerMap{}, "", fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(content, &ownerMap); err != nil {
		return OwnerMap{}, "", fmt.Errorf("parse %s: %w", path, err)
	}
	if ownerMap.Owners == nil {
		ownerMap.Owners = []string{}
	}
	if ownerMap.Projects == nil {
		ownerMap.Projects = map[string]string{}
	}
	for _, owner := range ownerMap.Owners {
		if err := ValidOwner(owner); err != nil {
			return OwnerMap{}, "", fmt.Errorf("parse %s: %w", path, err)
		}
	}
	for _, project := range slices.Sorted(maps.Keys(ownerMap.Projects)) {
		if err := ValidProject(project); err != nil {
			return OwnerMap{}, "", fmt.Errorf("parse %s: %w", path, err)
		}
	}
	home, homeErr := os.UserHomeDir()
	paths := make(map[string]string, len(ownerMap.Paths))
	written := map[string]string{}
	for _, rule := range slices.Sorted(maps.Keys(ownerMap.Paths)) {
		directory := rule
		if rest, ok := strings.CutPrefix(rule, "~/"); ok && homeErr == nil {
			directory = filepath.Join(home, rest)
		}
		if !filepath.IsAbs(directory) {
			return OwnerMap{}, "", fmt.Errorf("parse %s: path rule %q is relative; give the directory's absolute path, or one starting ~/", path, rule)
		}
		directory = filepath.Clean(directory)
		if earlier, taken := written[directory]; taken {
			return OwnerMap{}, "", fmt.Errorf("parse %s: path rules %q and %q name one directory; keep one", path, earlier, rule)
		}
		written[directory] = rule
		paths[directory] = ownerMap.Paths[rule]
	}
	ownerMap.Paths = paths
	return ownerMap, fingerprint(content), nil
}

func Save(path string, ownerMap OwnerMap) error {
	ownerMap.Owners = slices.Sorted(slices.Values(ownerMap.Owners))
	if ownerMap.Owners == nil {
		ownerMap.Owners = []string{}
	}
	if ownerMap.Paths == nil {
		ownerMap.Paths = map[string]string{}
	}
	if ownerMap.Projects == nil {
		ownerMap.Projects = map[string]string{}
	}
	content, err := json.MarshalIndent(ownerMap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode owner map: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	_, err = temporary.Write(append(content, '\n'))
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary.Name(), path)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func ValidOwner(owner string) error {
	if !ownerShape.MatchString(owner) {
		return fmt.Errorf("owner %q must be letters, digits, - and _, starting with a letter or digit", owner)
	}
	return nil
}

func ValidProject(project string) error {
	if project == "" || strings.TrimSpace(project) != project || strings.ContainsRune(project, filepath.Separator) || !notes.ValidProject(project) {
		return fmt.Errorf("project %q must be a directory name, with no separator, control character or surrounding space", project)
	}
	return nil
}

func (m OwnerMap) Clone() OwnerMap {
	return OwnerMap{Owners: slices.Clone(m.Owners), Paths: maps.Clone(m.Paths), Projects: maps.Clone(m.Projects)}
}

// Owner returns "" when neither the project entry nor a path rule names a listed owner.
func (m OwnerMap) Owner(project string, directory string) string {
	if owner := m.ProjectOwner(project); owner != "" {
		return owner
	}
	if owner, ok := m.OwnerOfPath(directory); ok && slices.Contains(m.Owners, owner) {
		return owner
	}
	return ""
}

func (m OwnerMap) OwnerOfPath(directory string) (string, bool) {
	longest := ""
	for rule := range m.Paths {
		within := directory == rule || rule == string(filepath.Separator) || strings.HasPrefix(directory, rule+string(filepath.Separator))
		if within && len(rule) > len(longest) {
			longest = rule
		}
	}
	if longest == "" {
		return "", false
	}
	return m.Paths[longest], true
}

func (m OwnerMap) ProjectOwner(project string) string {
	if owner := m.Projects[project]; slices.Contains(m.Owners, owner) {
		return owner
	}
	return ""
}

func fingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
