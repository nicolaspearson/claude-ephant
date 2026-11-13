package mining

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdvanceNeverMovesAWatermarkBack(t *testing.T) {
	directory := t.TempDir()
	earlier := time.Date(2026, 10, 15, 16, 58, 57, 664_000_000, time.UTC)
	later := earlier.Add(time.Hour)

	require.NoError(t, Advance(directory, "checkpoint-runs", later))
	require.NoError(t, Advance(directory, "checkpoint-runs", earlier))
	require.NoError(t, Advance(directory, "review-cycles", earlier))

	watermark, err := Watermark(directory, "checkpoint-runs")
	require.NoError(t, err)
	require.Equal(t, later, watermark)
	watermark, err = Watermark(directory, "review-cycles")
	require.NoError(t, err)
	require.Equal(t, earlier, watermark, "each analysis keeps its own")
}

func TestAdvanceRejectsAPathAsAnalysis(t *testing.T) {
	require.ErrorContains(t, Advance(t.TempDir(), "../notes", time.Now()), "not lowercase words")
}

func TestWatermarkIsZeroForAnAnalysisNeverMined(t *testing.T) {
	watermark, err := Watermark(t.TempDir(), "checkpoint-runs")
	require.NoError(t, err)
	require.True(t, watermark.IsZero())
}

func TestWatermarksIgnoreAndKeepAKeyTheyDoNotKnow(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, watermarksFile), []byte(`{"review-cycles":"2026-10-15T16:58:57Z","version":2}`), 0o600))

	watermark, err := Watermark(directory, "review-cycles")
	require.NoError(t, err)
	require.Equal(t, time.Date(2026, 10, 15, 16, 58, 57, 0, time.UTC), watermark)
	require.NoError(t, Advance(directory, "checkpoint-runs", watermark))
	content, err := os.ReadFile(filepath.Join(directory, watermarksFile))
	require.NoError(t, err)
	require.Contains(t, string(content), `"version": 2`)
}
