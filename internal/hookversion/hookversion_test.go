package hookversion

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAtGivesTheVersionRecordedLastBeforeTheTime(t *testing.T) {
	directory := filepath.Join(t.TempDir(), Directory)
	start := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	require.NoError(t, Record(directory, "s1", "v1.0.0", start))
	require.NoError(t, Record(directory, "s1", "devel", start.Add(time.Hour)))
	log := Open(directory)

	tests := []struct {
		name      string
		session   string
		timestamp string
		want      string
	}{
		{name: "a session with no file", session: "s2", timestamp: "2026-11-12T11:30:00Z", want: ""},
		{name: "a timestamp that does not parse", session: "s1", timestamp: "noon", want: ""},
		{name: "after the second record", session: "s1", timestamp: "2026-11-12T11:30:00.123Z", want: "devel"},
		{name: "at a record", session: "s1", timestamp: "2026-11-12T11:00:00Z", want: "devel"},
		{name: "before any record", session: "s1", timestamp: "2026-11-12T09:59:59Z", want: ""},
		{name: "between the records", session: "s1", timestamp: "2026-11-12T10:59:59.999Z", want: "v1.0.0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, log.At(test.session, test.timestamp))
		})
	}
}

func TestAtSkipsALineItCannotRead(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "s1"), []byte("half a line\n2026-11-12T10:00:00Z\tv1.0.0\n"), 0o600))

	require.Equal(t, "v1.0.0", Open(directory).At("s1", "2026-11-12T10:30:00Z"))
}

func TestRecordAppendsOnlyWhenTheVersionChanges(t *testing.T) {
	directory := filepath.Join(t.TempDir(), Directory)
	start := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	require.NoError(t, Record(directory, "s1", "v1.0.0", start))
	require.NoError(t, Record(directory, "s1", "v1.0.0", start.Add(time.Minute)))
	require.NoError(t, Record(directory, "s1", "devel", start.Add(2*time.Minute)))

	recorded, err := os.ReadFile(filepath.Join(directory, "s1"))
	require.NoError(t, err)
	require.Equal(t, "2026-11-12T10:00:00Z\tv1.0.0\n2026-11-12T10:02:00Z\tdevel\n", string(recorded))
}

func TestRecordRefusesAPathForASession(t *testing.T) {
	directory := filepath.Join(t.TempDir(), Directory)
	for _, session := range []string{"", "../s1", "a/b"} {
		require.Error(t, Record(directory, session, "v1.0.0", time.Now()), session)
	}
	require.NoDirExists(t, directory)
}
