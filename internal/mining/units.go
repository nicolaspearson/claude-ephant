package mining

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

const (
	coverWindow = 30 * time.Minute
	cycleGap    = 5 * time.Minute
)

var (
	rereview      = regexp.MustCompile(`(?i)re-?review|round 2|second round`)
	subagentEraAt = time.Date(2026, 10, 7, 11, 48, 0, 0, time.UTC)
)

type timedUnit struct {
	at          time.Time
	description string
	unit        dto.MiningUnit
}

// BoundRuns ends each run at its session's next cycle after the one it covers: a later checkpoint runs on the loaded skill with no invocation. An inline-era run covers one here too.
func BoundRuns(launches []dto.Event, runs []dto.MiningUnit) ([]dto.MiningUnit, error) {
	cycles, err := reviewCycles(launches, runs)
	if err != nil {
		return nil, err
	}
	covers, err := coverage(cycles, runs, time.Time{})
	if err != nil {
		return nil, err
	}
	bounded := slices.Clone(runs)
	for position, run := range bounded {
		ranAt, err := parseTimestamp(run.Session, run.Timestamp)
		if err != nil {
			return nil, err
		}
		next := -1
		for index, cycle := range cycles {
			if cycle.unit.Session != run.Session || cycle.at.Before(ranAt) || index == covers[position] {
				continue
			}
			if next < 0 || cycle.at.Before(cycles[next].at) {
				next = index
			}
		}
		if next >= 0 {
			bounded[position].Until = cycles[next].unit.Timestamp
		}
	}
	return bounded, nil
}

func CheckpointRuns(events []dto.Event) ([]dto.MiningUnit, error) {
	seen := map[[2]string]bool{}
	var runs []timedUnit
	for _, event := range events {
		key := [2]string{event.Session, event.Timestamp}
		if seen[key] {
			continue
		}
		seen[key] = true
		at, err := parseTimestamp(event.Session, event.Timestamp)
		if err != nil {
			return nil, err
		}
		runs = append(runs, timedUnit{at: at, unit: dto.MiningUnit{Owner: event.Owner, Project: event.Project, Session: event.Session, Source: event.Source, Timestamp: event.Timestamp}})
	}
	return sortedUnits(runs), nil
}

// ReviewCycles groups main-thread reviewer launches, then drops the cycle each subagent-era run covers: that run's judgment is the cycle's.
func ReviewCycles(launches []dto.Event, runs []dto.MiningUnit) ([]dto.MiningUnit, error) {
	cycles, err := reviewCycles(launches, runs)
	if err != nil {
		return nil, err
	}
	covers, err := coverage(cycles, runs, subagentEraAt)
	if err != nil {
		return nil, err
	}
	var uncovered []timedUnit
	for index, cycle := range cycles {
		if !slices.Contains(covers, index) {
			uncovered = append(uncovered, cycle)
		}
	}
	return sortedUnits(uncovered), nil
}

func Named(units []dto.MiningUnit, lines []byte) ([]dto.MiningUnit, error) {
	wanted := make(map[[2]string]bool, len(units))
	for _, unit := range units {
		wanted[[2]string{unit.Session, unit.Timestamp}] = false
	}
	scanner := bufio.NewScanner(bytes.NewReader(lines))
	scanner.Buffer(nil, len(lines)+1)
	for line := 1; scanner.Scan(); line++ {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var record struct {
			Session   string `json:"session"`
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("units line %d: %w", line, err)
		}
		key := [2]string{record.Session, record.Timestamp}
		if _, ok := wanted[key]; !ok {
			return nil, fmt.Errorf("units line %d: no unit has session %q and timestamp %q", line, record.Session, record.Timestamp)
		}
		wanted[key] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	var named []dto.MiningUnit
	for _, unit := range units {
		if wanted[[2]string{unit.Session, unit.Timestamp}] {
			named = append(named, unit)
		}
	}
	if len(named) == 0 {
		return nil, errors.New("no units named")
	}
	return named, nil
}

// Pending returns the unjudged units past the watermark and counts those at or before it.
func Pending(units []dto.MiningUnit, judgments []Judgment, watermark time.Time) ([]dto.MiningUnit, int, error) {
	judged := map[[2]string]bool{}
	for _, judgment := range judgments {
		judged[[2]string{judgment.Session, judgment.Timestamp}] = true
	}
	var pending []dto.MiningUnit
	behind := 0
	for _, unit := range units {
		if judged[[2]string{unit.Session, unit.Timestamp}] {
			continue
		}
		at, err := parseTimestamp(unit.Session, unit.Timestamp)
		if err != nil {
			return nil, 0, err
		}
		if at.After(watermark) {
			pending = append(pending, unit)
		} else {
			behind++
		}
	}
	return pending, behind, nil
}

// coverage returns, per run from since on, its session's first cycle within coverWindow after it, or -1.
func coverage(cycles []timedUnit, runs []dto.MiningUnit, since time.Time) ([]int, error) {
	covers := make([]int, len(runs))
	for position, run := range runs {
		covers[position] = -1
		ranAt, err := parseTimestamp(run.Session, run.Timestamp)
		if err != nil {
			return nil, err
		}
		if ranAt.Before(since) {
			continue
		}
		for index, cycle := range cycles {
			if cycle.unit.Session != run.Session || cycle.at.Before(ranAt) || cycle.at.Sub(ranAt) >= coverWindow {
				continue
			}
			if covers[position] < 0 || cycle.at.Before(cycles[covers[position]].at) {
				covers[position] = index
			}
		}
	}
	return covers, nil
}

func namesReviewers(launch dto.Event) bool {
	reviewers, _ := launch.Attributes["reviewers"].([]any)
	return len(reviewers) > 0
}

func parseTimestamp(session string, timestamp string) (time.Time, error) {
	at, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("unit %s: %w", session, err)
	}
	return at, nil
}

// reviewCycles closes a cycle to re-reviews once a run follows its last launch, and a cycle opened on a re-review takes only re-reviews.
func reviewCycles(launches []dto.Event, runs []dto.MiningUnit) ([]timedUnit, error) {
	bySession := map[string][]timedUnit{}
	for _, launch := range launches {
		if launch.Subagent || !namesReviewers(launch) {
			continue
		}
		at, err := parseTimestamp(launch.Session, launch.Timestamp)
		if err != nil {
			return nil, err
		}
		description, _ := launch.Attributes["description"].(string)
		bySession[launch.Session] = append(bySession[launch.Session], timedUnit{at: at, description: description, unit: dto.MiningUnit{Owner: launch.Owner, Project: launch.Project, Session: launch.Session, Source: launch.Source, Timestamp: launch.Timestamp}})
	}
	runsBySession := map[string][]time.Time{}
	for _, run := range runs {
		ranAt, err := parseTimestamp(run.Session, run.Timestamp)
		if err != nil {
			return nil, err
		}
		runsBySession[run.Session] = append(runsBySession[run.Session], ranAt)
	}
	var cycles []timedUnit
	for session, sessionLaunches := range bySession {
		slices.SortFunc(sessionLaunches, func(a, b timedUnit) int { return a.at.Compare(b.at) })
		open := -1
		openedOnRereview := false
		var last time.Time
		for _, launch := range sessionLaunches {
			isRereview := rereview.MatchString(launch.description)
			runBetween := slices.ContainsFunc(runsBySession[session], func(ranAt time.Time) bool { return ranAt.After(last) && ranAt.Before(launch.at) })
			switch {
			case isRereview && open >= 0 && !runBetween:
				cycles[open].unit.Rereviews++
			case !isRereview && open >= 0 && !openedOnRereview && launch.at.Sub(last) <= cycleGap:
				cycles[open].unit.Reviewers++
				last = launch.at
			default:
				launch.unit.Reviewers = 1
				if open >= 0 {
					cycles[open].unit.Until = launch.unit.Timestamp
				}
				cycles = append(cycles, launch)
				open = len(cycles) - 1
				openedOnRereview = isRereview
				last = launch.at
			}
		}
	}
	return cycles, nil
}

func sortedUnits(timed []timedUnit) []dto.MiningUnit {
	slices.SortFunc(timed, func(a, b timedUnit) int {
		return cmp.Or(a.at.Compare(b.at), cmp.Compare(a.unit.Session, b.unit.Session))
	})
	units := make([]dto.MiningUnit, 0, len(timed))
	for _, entry := range timed {
		units = append(units, entry.unit)
	}
	return units
}
