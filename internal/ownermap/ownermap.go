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
	"slices"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

// Fingerprint of a missing map, so creating one counts as a change.
const absent = "absent"

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
	paths := make(map[string]string, len(ownerMap.Paths))
	for rule, owner := range ownerMap.Paths {
		paths[filepath.Clean(rule)] = owner
	}
	ownerMap.Paths = paths
	return ownerMap, fingerprint(content), nil
}

func Save(path string, ownerMap OwnerMap) error {
	ownerMap.Owners = slices.Sorted(slices.Values(ownerMap.Owners))
	content, err := json.MarshalIndent(ownerMap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode owner map: %w", err)
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, append(content, '\n'), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("write %s: %w", path, err)
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
