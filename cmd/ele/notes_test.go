package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/notes"
)

func TestNewestOfKindSortsByTimeNotTheLocalDate(t *testing.T) {
	east := notes.Note{Date: "2026-11-02", ID: "20261102-aaaaaa", Kind: notes.KindHandoff, Project: "gateway", Time: "2026-11-01T22:00:00Z"}
	west := notes.Note{Date: "2026-11-01", ID: "20261101-bbbbbb", Kind: notes.KindHandoff, Project: "gateway", Time: "2026-11-01T23:00:00Z"}
	require.Equal(t, []notes.Note{west, east}, newestOfKind([]notes.Note{east, west}, notes.KindHandoff, "gateway"))
}
