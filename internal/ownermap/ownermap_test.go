package ownermap

import (
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
