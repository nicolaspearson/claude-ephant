package mining

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

func TestTakeRaisesTheWatermarkOnlyPastJudgedUnits(t *testing.T) {
	cache := t.TempDir()
	units := []dto.MiningUnit{
		{Session: "s0", Timestamp: "2026-10-06T09:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T09:00:00Z"},
		{Session: "s2", Timestamp: "2026-10-16T10:00:00Z"},
		{Session: "s3", Timestamp: "2026-10-16T11:00:00Z"},
	}
	prompt := []byte("Classify the run.\n")
	digest := func(unit dto.MiningUnit) (string, error) { return "digest of " + unit.Session, nil }

	taken, err := Take(cache, "checkpoint-runs", prompt, digest, []byte(`{"session":"s0","timestamp":"2026-10-06T09:00:00Z","mode":"full"}`+"\n"+`{"session":"s1","timestamp":"2026-10-16T09:00:00Z","mode":"small"}`+"\n\n"+`{"session":"s3","timestamp":"2026-10-16T11:00:00Z","mode":"full"}`+"\n"), units)
	require.NoError(t, err)
	require.Equal(t, Taken{Added: 3, Classifier: Classifier(prompt), Judgments: 3, Watermark: time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)}, taken, "s2 is unjudged, so s3 stays ahead of the watermark")
	saved, err := os.ReadFile(filepath.Join(cache, "classifiers", Classifier(prompt)+".md"))
	require.NoError(t, err)
	require.Equal(t, prompt, saved)
	held, err := Load(cache, "checkpoint-runs")
	require.NoError(t, err)
	require.Equal(t, Judgment{Classifier: Classifier(prompt), Digest: fingerprint([]byte("digest of s0")), Era: "inline", Result: json.RawMessage(`{"session":"s0","timestamp":"2026-10-06T09:00:00Z","mode":"full"}`), Session: "s0", Timestamp: "2026-10-06T09:00:00Z"}, held[0])
	require.Equal(t, "subagent", held[1].Era)

	taken, err = Take(cache, "checkpoint-runs", prompt, digest, []byte(`{"session":"s2","timestamp":"2026-10-16T10:00:00Z","mode":"full"}`), units)
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 10, 16, 11, 0, 0, 0, time.UTC), taken.Watermark)
	watermark, err := Watermark(cache, "checkpoint-runs")
	require.NoError(t, err)
	require.Equal(t, taken.Watermark, watermark)
}

func TestTakeRejectsAJudgmentForNoKnownUnitOrOneJudgedTwice(t *testing.T) {
	cache := t.TempDir()
	units := []dto.MiningUnit{{Session: "s1", Timestamp: "2026-10-16T09:00:00Z"}}
	digest := func(dto.MiningUnit) (string, error) { return "", nil }

	_, err := Take(cache, "review-cycles", []byte("Classify the cycle.\n"), digest, []byte(`{"session":"s1","timestamp":"2026-10-16T09:00:00.000Z"}`), units)
	require.ErrorContains(t, err, `judgments line 1: no review-cycles unit has session "s1" and timestamp "2026-10-16T09:00:00.000Z"`)
	_, err = os.Stat(filepath.Join(cache, "review-cycles.jsonl"))
	require.ErrorIs(t, err, os.ErrNotExist, "a bad line stores nothing")

	_, err = Take(cache, "review-cycles", []byte("Classify the cycle.\n"), digest, []byte(`{"session":"s1","timestamp":"2026-10-16T09:00:00Z","verdict":"a"}`+"\n"+`{"session":"s1","timestamp":"2026-10-16T09:00:00Z","verdict":"b"}`), units)
	require.ErrorContains(t, err, `judgments line 2: line 1 already judges session "s1" at "2026-10-16T09:00:00Z"`)
	_, err = os.Stat(filepath.Join(cache, "review-cycles.jsonl"))
	require.ErrorIs(t, err, os.ErrNotExist, "a unit judged twice stores nothing")

	missing := func(dto.MiningUnit) (string, error) { return "", os.ErrNotExist }
	_, err = Take(cache, "review-cycles", []byte("Classify the cycle.\n"), missing, []byte(`{"session":"s1","timestamp":"2026-10-16T09:00:00Z"}`), units)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorContains(t, err, "judgments line 1: ")
	_, err = os.Stat(filepath.Join(cache, "review-cycles.jsonl"))
	require.ErrorIs(t, err, os.ErrNotExist, "a unit with no digest stores nothing")

	_, err = Take(cache, "review-cycles", []byte("Classify the cycle.\n"), digest, []byte("\n"), units)
	require.ErrorContains(t, err, "no judgments to store")
}
