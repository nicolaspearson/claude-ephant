package mining

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeRun(t *testing.T) string {
	t.Helper()
	run := t.TempDir()
	work := filepath.Join(run, "work")
	require.NoError(t, os.MkdirAll(filepath.Join(work, "class"), 0o700))
	files := map[string]string{
		"class/K00.jsonl":  `{"id": "C000", "session": "b91f003a", "mode": "full"}` + "\n",
		"class/N00.jsonl":  `{"id": "000", "session": "2cb6fb01"}` + "\n\n",
		"class/R00.jsonl":  `{"id":"376","mode":"small"}` + "\n",
		"class/S00.jsonl":  `{"id":"376","mode":"full"}` + "\n",
		"classify-cyc.md":  "cycles\n",
		"classify-inl.md":  "inline\n",
		"classify-sub.md":  "subagent v1\n",
		"classify-sub3.md": "subagent v3\n",
		"cycles.json":      `[{"cid":"C000","session":"b91f003a-full","start":"2026-10-07T11:16:21.171Z"}]`,
		"runs_index.json":  `[{"id":"000","session":"2cb6fb01-full","ts":"2026-09-09T18:26:13.370Z","era":"inline"},{"id":"377","session":"b91f003a-full","ts":"2026-10-15T16:58:57.664Z","era":"subagent"},{"id":"376","session":"b91f003a-full","ts":"2026-10-07T11:15:42.261Z","era":"subagent"}]`,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(work, name), []byte(content), 0o600))
	}
	return run
}

func TestImportKeysEachJudgmentByItsInventoryUnit(t *testing.T) {
	run := writeRun(t)
	cache := t.TempDir()

	imported, err := Import(run, cache)
	require.NoError(t, err)
	require.Equal(t, []Imported{
		{Added: 1, Analysis: "checkpoint-runs", Classifier: Classifier([]byte("inline\n")), Judgments: 1, Prompt: "classify-inl.md"},
		{Added: 1, Analysis: "checkpoint-runs", Classifier: Classifier([]byte("subagent v3\n")), Judgments: 1, Prompt: "classify-sub3.md"},
		{Added: 1, Analysis: "checkpoint-runs", Classifier: Classifier([]byte("subagent v1\n")), Judgments: 1, Prompt: "classify-sub.md"},
		{Added: 1, Analysis: "review-cycles", Classifier: Classifier([]byte("cycles\n")), Judgments: 1, Prompt: "classify-cyc.md"},
	}, imported)

	runs, err := Load(cache, "checkpoint-runs")
	require.NoError(t, err)
	require.Len(t, runs, 3)
	require.Equal(t, Judgment{Classifier: Classifier([]byte("inline\n")), Era: "inline", Result: []byte(`{"id":"000","session":"2cb6fb01"}`), Session: "2cb6fb01-full", Timestamp: "2026-09-09T18:26:13.370Z"}, runs[0])
	require.Equal(t, runs[1].Timestamp, runs[2].Timestamp, "both passes over run 376 are kept")
	require.JSONEq(t, `{"id":"376","mode":"small"}`, string(runs[1].Result))
	cycles, err := Load(cache, "review-cycles")
	require.NoError(t, err)
	require.Equal(t, Judgment{Classifier: Classifier([]byte("cycles\n")), Result: []byte(`{"id":"C000","session":"b91f003a","mode":"full"}`), Session: "b91f003a-full", Timestamp: "2026-10-07T11:16:21.171Z"}, cycles[0])
	watermark, err := Watermark(cache, "checkpoint-runs")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 10, 15, 16, 58, 57, 664_000_000, time.UTC), watermark, "run 377 is unjudged but in the inventory")
	watermark, err = Watermark(cache, "review-cycles")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 10, 7, 11, 16, 21, 171_000_000, time.UTC), watermark)

	again, err := Import(run, cache)
	require.NoError(t, err)
	for _, family := range again {
		require.Zero(t, family.Added, family.Prompt)
	}
}

func TestImportRefusesAnIdOutsideTheInventory(t *testing.T) {
	run := writeRun(t)
	require.NoError(t, os.WriteFile(filepath.Join(run, "work", "class", "N01.jsonl"), []byte(`{"id":"999"}`+"\n"), 0o600))

	_, err := Import(run, t.TempDir())
	require.ErrorContains(t, err, `id "999" is not in the run's inventory`)
}

func TestImportRefusesANumericId(t *testing.T) {
	run := writeRun(t)
	require.NoError(t, os.WriteFile(filepath.Join(run, "work", "class", "R01.jsonl"), []byte(`{"id":376}`+"\n"), 0o600))

	_, err := Import(run, t.TempDir())
	require.ErrorContains(t, err, "R01.jsonl line 1")
}
