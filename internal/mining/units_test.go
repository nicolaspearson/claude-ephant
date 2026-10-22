package mining

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

func launch(session string, timestamp string, description string, reviewers ...any) dto.Event {
	return dto.Event{Attributes: map[string]any{"description": description, "reviewers": reviewers}, Kind: "agent", Name: "general-purpose", Project: "gateway", Session: session, Timestamp: timestamp}
}

func TestBoundRunsCloseEachRunAtTheNextCycleItDoesNotCover(t *testing.T) {
	launches := []dto.Event{
		launch("s1", "2026-10-06T08:20:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-06T09:00:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T08:01:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T08:04:00Z", "Re-review fixes", "bugs"),
		launch("s1", "2026-10-16T08:20:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T08:40:00Z", "Bugs review", "bugs"),
		launch("s2", "2026-10-16T08:10:00Z", "Bugs review", "bugs"),
	}
	runs, err := BoundRuns(launches, []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-06T08:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T08:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T09:00:00Z"},
		{Session: "s2", Timestamp: "2026-10-16T08:00:00Z"},
	})
	require.NoError(t, err)
	require.Equal(t, []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-06T08:00:00Z", Until: "2026-10-06T09:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T08:00:00Z", Until: "2026-10-16T08:20:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T09:00:00Z"},
		{Session: "s2", Timestamp: "2026-10-16T08:00:00Z"},
	}, runs, "a run's own first cycle and its re-review stay in it, in either era, and a run with no later cycle in its session is unbounded")
}

func TestCheckpointRunsMergesSkillAndCommandEventsOldestFirst(t *testing.T) {
	runs, err := CheckpointRuns([]dto.Event{
		{Kind: "skill", Name: "checkpoint", Owner: "corp", Project: "gateway", Session: "s1", Timestamp: "2026-10-16T09:00:00Z"},
		{Kind: "command", Name: "checkpoint", Owner: "corp", Project: "gateway", Session: "s1", Timestamp: "2026-10-16T08:00:00Z"},
		{Kind: "skill", Name: "checkpoint", Owner: "corp", Project: "gateway", Session: "s1", Timestamp: "2026-10-16T09:00:00Z"},
	})
	require.NoError(t, err)
	require.Equal(t, []dto.MiningUnit{
		{Owner: "corp", Project: "gateway", Session: "s1", Timestamp: "2026-10-16T08:00:00Z"},
		{Owner: "corp", Project: "gateway", Session: "s1", Timestamp: "2026-10-16T09:00:00Z"},
	}, runs)
}

func TestReviewCyclesGiveARereviewAfterARunItsOwnCycle(t *testing.T) {
	launches := []dto.Event{
		launch("s1", "2026-10-15T11:27:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-15T11:50:00Z", "Re-review bugs", "bugs"),
		launch("s1", "2026-10-15T12:33:00Z", "Re-review bugs", "bugs"),
		launch("s1", "2026-10-15T12:36:00Z", "Post-commit bugs review", "bugs"),
		launch("s1", "2026-10-15T12:53:00Z", "Re-review bugs", "bugs"),
		launch("s1", "2026-10-15T13:30:00Z", "Bugs review", "bugs"),
	}
	runs := []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-15T11:26:00Z"},
		{Session: "s1", Timestamp: "2026-10-15T12:32:00Z"},
	}
	bounded, err := BoundRuns(launches, runs)
	require.NoError(t, err)
	require.Equal(t, []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-15T11:26:00Z", Until: "2026-10-15T12:33:00Z"},
		{Session: "s1", Timestamp: "2026-10-15T12:32:00Z", Until: "2026-10-15T12:36:00Z"},
	}, bounded, "a run closes its session's cycle to re-reviews, which open their own; the next fresh launch closes that one")
}

func TestReviewCyclesGroupLaunchesFiveMinutesApart(t *testing.T) {
	subagentLaunch := launch("s1", "2026-10-16T08:00:30Z", "Bugs review", "bugs")
	subagentLaunch.Subagent = true
	cycles, err := ReviewCycles([]dto.Event{
		launch("s1", "2026-10-16T08:04:00Z", "Prose review", "prose"),
		launch("s1", "2026-10-16T08:00:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T08:02:00Z", "Conventions review", "conventions"),
		subagentLaunch,
		launch("s1", "2026-10-16T08:03:00Z", "Explore the repository"),
		launch("s1", "2026-10-16T08:20:00Z", "Re-review fixes", "bugs"),
		launch("s1", "2026-10-16T09:00:00Z", "Bugs review", "bugs"),
		launch("s2", "2026-10-16T08:01:00Z", "Re-review fixes", "bugs"),
		launch("s2", "2026-10-16T08:01:30Z", "Re-review prose", "prose"),
		launch("s2", "2026-10-16T08:02:00Z", "Bugs review", "bugs"),
		launch("s3", "2026-10-16T10:00:00Z", "Bugs review", "bugs"),
		launch("s3", "2026-10-16T10:05:00Z", "Prose review", "prose"),
		launch("s3", "2026-10-16T10:08:00Z", "Re-review fixes", "bugs"),
		launch("s3", "2026-10-16T10:11:00Z", "Bugs review", "bugs"),
	}, nil)
	require.NoError(t, err)
	require.Equal(t, []dto.MiningUnit{
		{Project: "gateway", Rereviews: 1, Reviewers: 3, Session: "s1", Timestamp: "2026-10-16T08:00:00Z", Until: "2026-10-16T09:00:00Z"},
		{Project: "gateway", Rereviews: 1, Reviewers: 1, Session: "s2", Timestamp: "2026-10-16T08:01:00Z", Until: "2026-10-16T08:02:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s2", Timestamp: "2026-10-16T08:02:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s1", Timestamp: "2026-10-16T09:00:00Z"},
		{Project: "gateway", Rereviews: 1, Reviewers: 2, Session: "s3", Timestamp: "2026-10-16T10:00:00Z", Until: "2026-10-16T10:11:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s3", Timestamp: "2026-10-16T10:11:00Z"},
	}, cycles, "a re-review with no cycle open starts one that later re-reviews join and no fresh launch does; the gap runs from the last launch that is not a re-review")
}

func TestReviewCyclesSkipTheFirstCycleASubagentEraRunCovers(t *testing.T) {
	launches := []dto.Event{
		launch("s1", "2026-10-06T08:10:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T08:10:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T08:20:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T10:29:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-16T11:30:00Z", "Bugs review", "bugs"),
		launch("s3", "2026-10-07T11:50:00Z", "Bugs review", "bugs"),
		launch("s2", "2026-10-16T08:10:00Z", "Bugs review", "bugs"),
	}
	runs := []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-06T08:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T08:10:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T10:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-16T11:00:00Z"},
		{Session: "s3", Timestamp: "2026-10-07T11:48:00Z"},
	}
	cycles, err := ReviewCycles(launches, runs)
	require.NoError(t, err)
	var starts []string
	for _, cycle := range cycles {
		starts = append(starts, cycle.Session+" "+cycle.Timestamp)
	}
	require.Equal(t, []string{
		"s1 2026-10-06T08:10:00Z",
		"s2 2026-10-16T08:10:00Z",
		"s1 2026-10-16T08:20:00Z",
		"s1 2026-10-16T11:30:00Z",
	}, starts, "an inline-era run covers nothing, a run at the era's start does, a run covers only its first cycle and only within 30 minutes, and never another session's")
}

func TestReviewCyclesSplitOffAnyLaunchARunPrecedes(t *testing.T) {
	cycles, err := ReviewCycles([]dto.Event{
		launch("s1", "2026-10-06T08:00:00Z", "Bugs review", "bugs"),
		launch("s1", "2026-10-06T08:10:00Z", "Re-review bugs", "bugs"),
		launch("s1", "2026-10-06T08:30:00Z", "Re-review bugs", "bugs"),
		launch("s2", "2026-10-06T08:00:00Z", "Bugs review", "bugs"),
		launch("s2", "2026-10-06T08:03:00Z", "Re-review bugs", "bugs"),
		launch("s3", "2026-10-06T08:00:00Z", "Bugs review", "bugs"),
		launch("s3", "2026-10-06T08:04:00Z", "Bugs review 2", "bugs"),
	}, []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-06T08:20:00Z"},
		{Session: "s2", Timestamp: "2026-10-06T08:02:00Z"},
		{Session: "s3", Timestamp: "2026-10-06T08:03:00Z"},
	})
	require.NoError(t, err)
	require.Equal(t, []dto.MiningUnit{
		{Project: "gateway", Rereviews: 1, Reviewers: 1, Session: "s1", Timestamp: "2026-10-06T08:00:00Z", Until: "2026-10-06T08:30:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s2", Timestamp: "2026-10-06T08:00:00Z", Until: "2026-10-06T08:03:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s3", Timestamp: "2026-10-06T08:00:00Z", Until: "2026-10-06T08:04:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s2", Timestamp: "2026-10-06T08:03:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s3", Timestamp: "2026-10-06T08:04:00Z"},
		{Project: "gateway", Reviewers: 1, Session: "s1", Timestamp: "2026-10-06T08:30:00Z"},
	}, cycles, "a run between two launches closes the cycle to the second, re-review or fresh, however soon; a re-review a run closed out never joins as a fresh launch")
}

func TestPendingListsUnjudgedUnitsPastTheWatermarkAndCountsTheRest(t *testing.T) {
	units := []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-15T10:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-15T11:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-15T12:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-15T13:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-15T14:00:00Z"},
	}
	judgments := []Judgment{
		{Classifier: "a", Result: json.RawMessage(`{}`), Session: "s1", Timestamp: "2026-10-15T10:00:00Z"},
		{Classifier: "b", Result: json.RawMessage(`{}`), Session: "s1", Timestamp: "2026-10-15T13:00:00Z"},
	}
	pending, behind, err := Pending(units, judgments, time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Equal(t, []dto.MiningUnit{{Session: "s1", Timestamp: "2026-10-15T14:00:00Z"}}, pending, "a judgment under any classifier counts")
	require.Equal(t, 2, behind, "the unit at the watermark is behind it")
}

func TestPendingRefusesAnUnparseableTimestamp(t *testing.T) {
	_, _, err := Pending([]dto.MiningUnit{{Session: "s1", Timestamp: "yesterday"}}, nil, time.Time{})
	require.ErrorContains(t, err, "s1")
}

func TestNamedListsTheUnitsALineNamesJudgedOrNot(t *testing.T) {
	units := []dto.MiningUnit{
		{Session: "s1", Timestamp: "2026-10-15T10:00:00Z"},
		{Session: "s1", Timestamp: "2026-10-15T11:00:00Z"},
		{Session: "s2", Timestamp: "2026-10-15T12:00:00Z"},
	}
	named, err := Named(units, []byte(`{"session":"s2","timestamp":"2026-10-15T12:00:00Z","digest":"/old/path.txt"}`+"\n\n"+`{"session":"s1","timestamp":"2026-10-15T10:00:00Z"}`+"\n"))
	require.NoError(t, err)
	require.Equal(t, []dto.MiningUnit{units[0], units[2]}, named, "in unit order, a listing line naming its unit as well as a bare one")

	_, err = Named(units, []byte(`{"session":"s1","timestamp":"2026-10-15T10:00:00Z"}`+"\n"+`{"session":"s1","timestamp":"2026-10-15T10:00:00.000Z"}`))
	require.ErrorContains(t, err, `units line 2: no unit has session "s1" and timestamp "2026-10-15T10:00:00.000Z"`)
	_, err = Named(units, []byte("\n"))
	require.ErrorContains(t, err, "no units named")
}
