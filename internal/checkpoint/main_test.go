package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
)

// The checkpoint state lives under the data directory, so a test that forgets its own ELE_HOME must not reach the real one.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "ele-checkpoint-test-")
	if err != nil {
		panic(err)
	}
	if err = os.Setenv("ELE_HOME", filepath.Join(home, "data")); err != nil {
		panic(err)
	}
	code := m.Run()
	_ = os.RemoveAll(home)
	os.Exit(code)
}
