package index

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenNamesWhatDiffersAndResetRepairsIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ele.db")
	fresh, err := Open(path)
	require.NoError(t, err)
	require.Empty(t, fresh.SchemaProblems(), "a new index matches")
	_, err = fresh.db.Exec(`DROP TABLE notes;
		DROP TABLE events;
		CREATE TABLE events (attributes TEXT NOT NULL, id INTEGER PRIMARY KEY, name TEXT NOT NULL, owner TEXT NOT NULL, profile TEXT NOT NULL, project TEXT NOT NULL, session TEXT NOT NULL, source TEXT NOT NULL, stray TEXT, subagent INTEGER NOT NULL, timestamp TEXT NOT NULL, uuid TEXT NOT NULL);
		DROP TABLE archive_files;
		CREATE TABLE archive_files (archive_size INTEGER NOT NULL, indexed_bytes INTEGER NOT NULL, path TEXT);
		ALTER TABLE settings ADD COLUMN stray TEXT`)
	require.NoError(t, err)
	require.NoError(t, fresh.Close())

	reopened, err := Open(path)
	require.NoError(t, err, "a mismatched index still opens, so sync can repair it")
	require.Equal(t, []string{"archive_files has a different definition", "events lacks columns kind, line_key", "events has column stray that this version does not", "notes is missing"}, reopened.SchemaProblems())

	require.NoError(t, reopened.Reset(context.Background()))
	require.Empty(t, reopened.SchemaProblems())
	require.NoError(t, reopened.Close())
	repaired, err := Open(path)
	require.NoError(t, err)
	require.Empty(t, repaired.SchemaProblems())
	require.NoError(t, repaired.Close())
}
