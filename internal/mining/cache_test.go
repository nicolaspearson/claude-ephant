package mining

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRejectsAMalformedLine(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "checkpoint-runs.jsonl"), []byte("{\"classifier\":\"a\"}\nnot json\n"), 0o600))

	_, err := Load(directory, "checkpoint-runs")
	require.ErrorContains(t, err, "line 2")
}

func TestLoadRejectsAPathAsAnalysis(t *testing.T) {
	_, err := Load(t.TempDir(), "../notes")
	require.ErrorContains(t, err, "not lowercase words")
}

func TestLoadSkipsABlankLine(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "checkpoint-runs.jsonl"), []byte("{\"classifier\":\"a\"}\n\n{\"classifier\":\"b\"}\n"), 0o600))

	held, err := Load(directory, "checkpoint-runs")
	require.NoError(t, err)
	require.Equal(t, []Judgment{{Classifier: "a"}, {Classifier: "b"}}, held)
}

func TestSavePromptKeepsThePromptUnderItsHash(t *testing.T) {
	directory := t.TempDir()

	classifier, err := SavePrompt(directory, []byte("Classify the run.\n"))
	require.NoError(t, err)
	require.Equal(t, Classifier([]byte("Classify the run.\n")), classifier)
	content, err := os.ReadFile(filepath.Join(directory, "classifiers", classifier+".md"))
	require.NoError(t, err)
	require.Equal(t, "Classify the run.\n", string(content))
}

func TestStoreKeepsAHeldLineAsWritten(t *testing.T) {
	directory := t.TempDir()
	held := `{"classifier":"v1","result":{"mode":"full"},"reviewed":true,"session":"s1","timestamp":"2026-10-07T11:15:42Z"}`
	require.NoError(t, os.WriteFile(filepath.Join(directory, "checkpoint-runs.jsonl"), []byte(held+"\n"), 0o600))

	added, err := Store(directory, "checkpoint-runs", []Judgment{{Classifier: "v1", Result: json.RawMessage(`{}`), Session: "s2", Timestamp: "2026-10-08T11:15:42Z"}})
	require.NoError(t, err)
	require.Equal(t, 1, added)
	content, err := os.ReadFile(filepath.Join(directory, "checkpoint-runs.jsonl"))
	require.NoError(t, err)
	require.Equal(t, held+"\n"+`{"classifier":"v1","result":{},"session":"s2","timestamp":"2026-10-08T11:15:42Z"}`+"\n", string(content), "a key this build does not declare survives")
}

func TestStoreKeepsOneJudgmentPerUnitClassifierAndDigest(t *testing.T) {
	directory := t.TempDir()
	first := Judgment{Classifier: "v1", Digest: "d1", Era: "inline", Result: json.RawMessage(`{"mode": "full"}`), Session: "s1", Timestamp: "2026-10-07T11:15:42Z"}
	rejudged := Judgment{Classifier: "v1", Digest: "d1", Result: json.RawMessage(`{"mode":"small"}`), Session: "s1", Timestamp: "2026-10-07T11:15:42Z"}
	later := Judgment{Classifier: "v3", Digest: "d1", Result: json.RawMessage(`{"mode":"small"}`), Session: "s1", Timestamp: "2026-10-07T11:15:42Z"}
	changed := Judgment{Classifier: "v1", Digest: "d2", Result: json.RawMessage(`{"mode":"small"}`), Session: "s1", Timestamp: "2026-10-07T11:15:42Z"}

	added, err := Store(directory, "checkpoint-runs", []Judgment{first, rejudged})
	require.NoError(t, err)
	require.Equal(t, 1, added)
	added, err = Store(directory, "checkpoint-runs", []Judgment{rejudged, later, changed})
	require.NoError(t, err)
	require.Equal(t, 2, added, "a judgment under another classifier, or of a changed digest, is kept beside the first")

	held, err := Load(directory, "checkpoint-runs")
	require.NoError(t, err)
	first.Result = json.RawMessage(`{"mode":"full"}`)
	require.Equal(t, []Judgment{first, later, changed}, held, "the first judgment stands, compacted to one line")
}

func TestStoreRejectsAJudgmentWithoutItsUnit(t *testing.T) {
	_, err := Store(t.TempDir(), "checkpoint-runs", []Judgment{{Classifier: "v1", Result: json.RawMessage(`{}`), Session: "s1"}})
	require.ErrorContains(t, err, "needs a classifier, a session and a timestamp")
}
