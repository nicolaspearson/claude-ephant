package ownermap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

func TestMovesNamesEachProjectWhoseOwnerChanges(t *testing.T) {
	before := OwnerMap{Owners: []string{"corp", "personal"}, Paths: map[string]string{"/w": "corp"}, Projects: map[string]string{}}
	after := before.Clone()
	after.Paths["/w/mine"] = "personal"
	directories := []dto.ProjectDirectory{
		{Directory: "/w/gateway", Project: "gateway"},
		{Directory: "/w/mine/notes", Project: "notes"},
		{Directory: "/w/mine/notes/sub", Project: "notes"},
		{Directory: "/elsewhere/x", Project: "x"},
	}

	require.Equal(t, []dto.OwnerMove{{From: "corp", Project: "notes", To: "personal"}}, Moves(before, after, directories))
	require.Equal(t, map[string]string{"/w": "corp"}, before.Paths, "Clone leaves the original alone")
}

func TestOwnerPrefersProjectEntryThenLongestPathRule(t *testing.T) {
	ownerMap := OwnerMap{
		Owners:   []string{"corp", "personal"},
		Paths:    map[string]string{"/w": "personal", "/w/corp": "corp"},
		Projects: map[string]string{"pinned": "personal", "typo": "copr"},
	}

	tests := []struct {
		directory string
		name      string
		owner     string
		project   string
	}{
		{directory: "/w/corp/x", name: "longest rule", owner: "corp", project: "x"},
		{directory: "/elsewhere", name: "no rule", owner: "", project: "x"},
		{directory: "/w/corp/pinned", name: "project entry wins", owner: "personal", project: "pinned"},
		{directory: "/w/corp-extra/x", name: "rule boundary", owner: "personal", project: "x"},
		{directory: "/w/corp/typo", name: "unlisted entry falls to rule", owner: "corp", project: "typo"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.owner, ownerMap.Owner(test.project, test.directory))
		})
	}
}

func TestProjectOwnerIgnoresAnUnlistedOwner(t *testing.T) {
	ownerMap := OwnerMap{Owners: []string{"corp"}, Projects: map[string]string{"gateway": "corp", "typo": "copr"}}
	require.Equal(t, "corp", ownerMap.ProjectOwner("gateway"))
	require.Empty(t, ownerMap.ProjectOwner("typo"))
	require.Empty(t, ownerMap.ProjectOwner("absent"))
}

func TestReadExpandsAPathRuleUnderHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "owner-map.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"owners": ["corp"], "paths": {"~/code": "corp"}}`), 0o600))

	loaded, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, map[string]string{filepath.Join(home, "code"): "corp"}, loaded.Paths)
}

func TestReadRefusesWhatOwnerCommandsRefuse(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "owner-map.json")
	tests := []struct {
		content string
		message string
		name    string
	}{
		{content: `{"owners": ["../x"]}`, message: `owner "../x" must be letters, digits, - and _, starting with a letter or digit`, name: "owner leaving the notes folder"},
		{content: `{"owners": ["corp"], "projects": {"a/b": "corp"}}`, message: `project "a/b" must be a directory name, with no separator, control character or surrounding space`, name: "project with a separator"},
		{content: `{"owners": ["corp"], "paths": {"code": "corp"}}`, message: `path rule "code" is relative; give the directory's absolute path, or one starting ~/`, name: "relative rule"},
		{content: `{"owners": ["corp", "personal"], "paths": {"/a/b": "corp", "/a/b/": "personal"}}`, message: `path rules "/a/b" and "/a/b/" name one directory; keep one`, name: "rules cleaning to one key"},
		{content: `{"owners": ["corp"], "paths": {"~code": "corp"}}`, message: `path rule "~code" is relative; give the directory's absolute path, or one starting ~/`, name: "tilde without slash"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(path, []byte(test.content), 0o600))
			_, err := Load(path)
			require.EqualError(t, err, "parse "+path+": "+test.message)
		})
	}
}

func TestSaveSortsOwnersAndReadFingerprintsTheBytesItParsed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owner-map.json")
	require.NoError(t, Save(path, OwnerMap{Owners: []string{"personal", "corp"}, Paths: map[string]string{}, Projects: map[string]string{}}))

	loaded, fingerprinted, err := Read(path)
	require.NoError(t, err)
	require.Equal(t, []string{"corp", "personal"}, loaded.Owners)
	current, err := Fingerprint(path)
	require.NoError(t, err)
	require.Equal(t, current, fingerprinted)
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owner-map.json")
	saved := OwnerMap{Owners: []string{"corp"}, Paths: map[string]string{"/w/corp/": "corp"}, Projects: map[string]string{"a": "corp"}}

	require.NoError(t, Save(path, saved))
	loaded, err := Load(path)
	require.NoError(t, err)
	require.Equal(t, OwnerMap{Owners: []string{"corp"}, Paths: map[string]string{"/w/corp": "corp"}, Projects: map[string]string{"a": "corp"}}, loaded)
}

func TestSaveWritesEmptyListsAndLeavesNoTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "owner-map.json")
	require.NoError(t, os.WriteFile(path+".tmp", []byte("someone else's"), 0o600))

	require.NoError(t, Save(path, OwnerMap{}))
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.JSONEq(t, `{"owners": [], "paths": {}, "projects": {}}`, string(content))
	other, err := os.ReadFile(path + ".tmp")
	require.NoError(t, err)
	require.Equal(t, "someone else's", string(other), "a fixed temporary name would be another writer's")
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}
