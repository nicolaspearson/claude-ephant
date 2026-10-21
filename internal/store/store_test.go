package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAppliesPragmasToEveryPath(t *testing.T) {
	tests := []struct {
		directory string
		name      string
	}{
		{directory: "a#b", name: "hash"},
		{directory: "a%20b", name: "percent"},
		{directory: "a?b", name: "question mark"},
		{directory: "Application Support", name: "space"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), test.directory)
			require.NoError(t, os.Mkdir(directory, 0o700))
			path := filepath.Join(directory, "ele.db")

			db, err := Open(path)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })

			var file string
			require.NoError(t, db.QueryRow(`SELECT file FROM pragma_database_list WHERE name = 'main'`).Scan(&file))
			resolved, err := filepath.EvalSymlinks(path)
			require.NoError(t, err)
			require.Equal(t, resolved, file)

			var timeout int
			require.NoError(t, db.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout))
			require.Equal(t, 5000, timeout)

			var foreignKeys int
			require.NoError(t, db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys))
			require.Equal(t, 1, foreignKeys)

			var mode string
			require.NoError(t, db.QueryRow(`PRAGMA journal_mode`).Scan(&mode))
			require.Equal(t, "wal", mode)
		})
	}
}

func TestOpenReadOnlyNeverWrites(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "snapshot.db")
	writer, err := Open(path)
	require.NoError(t, err)
	_, err = writer.Exec(`CREATE TABLE rows (body TEXT); INSERT INTO rows VALUES ('kept')`)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.NoError(t, os.Chmod(path, 0o400))

	db, err := OpenReadOnly(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	var body string
	require.NoError(t, db.QueryRow(`SELECT body FROM rows`).Scan(&body))
	require.Equal(t, "kept", body)
	_, err = db.Exec(`INSERT INTO rows VALUES ('lost')`)
	require.ErrorContains(t, err, "readonly")

	entries, err = os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestOpenResolvesRelativePath(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)

	db, err := Open("ele.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	var file string
	require.NoError(t, db.QueryRow(`SELECT file FROM pragma_database_list WHERE name = 'main'`).Scan(&file))
	resolved, err := filepath.EvalSymlinks(filepath.Join(directory, "ele.db"))
	require.NoError(t, err)
	require.Equal(t, resolved, file)
}

func TestOpenSupportsFTS5(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "ele.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, err = db.Exec(`CREATE VIRTUAL TABLE documents USING fts5(body)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO documents (body) VALUES ('freeze claude-mem before import'), ('pin pydantic below 2.14')`)
	require.NoError(t, err)

	var body string
	require.NoError(t, db.QueryRow(`SELECT body FROM documents WHERE documents MATCH 'pydantic'`).Scan(&body))
	require.Equal(t, "pin pydantic below 2.14", body)
}
