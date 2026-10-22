package mining

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/checkpoint"
)

type after struct {
	Category                   string `json:"category"`
	ShouldHaveBeenCaughtBy     string `json:"should_have_been_caught_by"`
	UserDisputedAFindingOrFix  bool   `json:"user_disputed_a_finding_or_fix"`
	UserReportedProblem        bool   `json:"user_reported_problem"`
	UserSaidCheckpointUnwanted bool   `json:"user_said_checkpoint_unwanted"`
	What                       string `json:"what"`
}

type eraUnit struct {
	era string
	judged
}

type finding struct {
	AlsoRaisedBy         []string `json:"also_raised_by"`
	Category             string   `json:"category"`
	Origin               string   `json:"origin"`
	RereviewKind         string   `json:"rereview_kind"`
	ReraisesUserDecision bool     `json:"reraises_user_decision"`
	Severity             string   `json:"severity"`
	Source               string   `json:"source"`
	Stage                string   `json:"stage"`
	Verdict              string   `json:"verdict"`
}

type judged struct {
	at      time.Time
	date    string
	result  result
	session string
}

func (unit judged) checkpoint() bool {
	return unit.result.IsCheckpoint == nil || *unit.result.IsCheckpoint
}

type passCounts struct {
	Fixed         float64 `json:"fixed"`
	RefutedOrLeft float64 `json:"refuted_or_left"`
}

type result struct {
	After  after `json:"after"`
	Change struct {
		Lines *float64 `json:"lines"`
	} `json:"change"`
	Confidence            string          `json:"confidence"`
	CoveredShortCircuit   bool            `json:"covered_shortcircuit"`
	CreditedEarlierPasses bool            `json:"credited_earlier_passes"`
	FindingCategories     []string        `json:"finding_categories"`
	Findings              json.RawMessage `json:"findings"`
	Gate                  struct {
		FailedAfterReview bool `json:"failed_after_review"`
	} `json:"gate"`
	IsCheckpoint    *bool              `json:"is_checkpoint"`
	MemoryWouldHelp string             `json:"memory_would_help"`
	Mode            string             `json:"mode"`
	Mutations       map[string]float64 `json:"mutations"`
	Notes           map[string]float64 `json:"notes"`
	RepeatedWork    string             `json:"repeated_work"`
	ReportTimestamp string             `json:"report_ts"`
	Rereview        string             `json:"rereview"`
	Session         string             `json:"session"`
	StartedBy       string             `json:"started_by"`
}

// Report reads the last stored judgment per unit, and leaves out a review cycle starting inside a subagent-era run's window, which that run's judgment already covers.
func Report(w io.Writer, runs []Judgment, cycles []Judgment) error {
	runUnits, err := lastPerUnit(runs)
	if err != nil {
		return err
	}
	cycleUnits, err := lastPerUnit(cycles)
	if err != nil {
		return err
	}
	var inline, subagent []judged
	for _, unit := range runUnits {
		if unit.era == "inline" {
			inline = append(inline, unit.judged)
		} else {
			subagent = append(subagent, unit.judged)
		}
	}
	kept, err := uncovered(subagent, cycleUnits)
	if err != nil {
		return err
	}
	writeSubagentEra(w, subagent, kept, len(cycleUnits)-len(kept))
	writeInlineEra(w, inline)
	return nil
}

func cut(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}

func findingsOf(unit judged) []finding {
	var entries []json.RawMessage
	if json.Unmarshal(unit.result.Findings, &entries) != nil {
		return nil
	}
	var findings []finding
	for _, entry := range entries {
		var parsed finding
		if json.Unmarshal(entry, &parsed) == nil {
			findings = append(findings, parsed)
		}
	}
	return findings
}

func joinCounts(counts map[string]int) string {
	keys := slices.SortedFunc(maps.Keys(counts), func(a, b string) int {
		return cmp.Or(counts[b]-counts[a], strings.Compare(a, b))
	})
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", cmp.Or(key, "unset"), counts[key]))
	}
	if len(parts) == 0 {
		return "nothing"
	}
	return strings.Join(parts, ", ")
}

func lastPerUnit(judgments []Judgment) ([]eraUnit, error) {
	at := map[[2]string]int{}
	var units []eraUnit
	for _, judgment := range judgments {
		var parsed result
		if err := json.Unmarshal(judgment.Result, &parsed); err != nil {
			return nil, fmt.Errorf("judgment of %s at %s: %w", judgment.Session, judgment.Timestamp, err)
		}
		when, err := parseTimestamp(judgment.Session, judgment.Timestamp)
		if err != nil {
			return nil, err
		}
		unit := eraUnit{era: judgment.Era, judged: judged{at: when, date: when.Format(time.DateOnly), result: parsed, session: judgment.Session}}
		key := [2]string{judgment.Session, judgment.Timestamp}
		if position, seen := at[key]; seen {
			units[position] = unit
			continue
		}
		at[key] = len(units)
		units = append(units, unit)
	}
	slices.SortStableFunc(units, func(a, b eraUnit) int {
		return cmp.Or(a.at.Compare(b.at), strings.Compare(a.session, b.session))
	})
	return units, nil
}

func shortSession(session string) string {
	return session[:min(8, len(session))]
}

func uncovered(runs []judged, cycles []eraUnit) ([]judged, error) {
	var kept []judged
	for _, cycle := range cycles {
		covered := false
		for _, run := range runs {
			if run.session != cycle.session || run.result.ReportTimestamp == "" || cycle.at.Before(run.at) {
				continue
			}
			reported, err := parseTimestamp(run.session, run.result.ReportTimestamp)
			if err != nil {
				return nil, err
			}
			if !cycle.at.After(reported) {
				covered = true
				break
			}
		}
		if !covered {
			kept = append(kept, cycle.judged)
		}
	}
	return kept, nil
}

func writeInlineEra(w io.Writer, units []judged) {
	var read, checkpoints []judged
	var passes []map[string]passCounts
	confidence := map[string]int{}
	for _, unit := range units {
		var unitPasses map[string]passCounts
		if json.Unmarshal(unit.result.Findings, &unitPasses) != nil {
			continue
		}
		read = append(read, unit)
		confidence[unit.result.Confidence]++
		if unit.checkpoint() {
			checkpoints = append(checkpoints, unit)
			passes = append(passes, unitPasses)
		}
	}
	fmt.Fprintf(w, "\n\nInline era: run judgments %d, checkpoints %d. Confidence high %d, medium %d, low %d.\n", len(read), len(checkpoints), confidence["high"], confidence["medium"], confidence["low"])
	if otherShape := len(units) - len(read); otherShape > 0 {
		fmt.Fprintf(w, "Inline-era runs judged under the subagent-era prompt, with no per-pass counts, left out: %d.\n", otherShape)
	}
	fixed := map[string]float64{}
	left := map[string]float64{}
	credited, noneFixed := 0, 0
	mutations := map[string]float64{}
	var categories []string
	for position, unit := range checkpoints {
		if unit.result.CreditedEarlierPasses {
			credited++
		}
		unitFixed := 0.0
		for pass, counts := range passes[position] {
			fixed[pass] += counts.Fixed
			left[pass] += counts.RefutedOrLeft
			unitFixed += counts.Fixed
		}
		if unitFixed == 0 {
			noneFixed++
		}
		categories = append(categories, unit.result.FindingCategories...)
		for key, count := range unit.result.Mutations {
			mutations[key] += count
		}
	}
	fmt.Fprintf(w, "Credited passes run earlier in the session: %d. Fixed nothing: %d.\n", credited, noneFixed)
	fmt.Fprintln(w, "\nFindings by pass")
	fmt.Fprintf(w, "%-14s %6s %16s\n", "pass", "fixed", "refuted or left")
	for _, pass := range slices.Sorted(maps.Keys(fixed)) {
		fmt.Fprintf(w, "%-14s %6.0f %16.0f\n", pass, fixed[pass], left[pass])
	}
	fmt.Fprintln(w, "\nFindings by category")
	checkpoint.WriteMostCommon(w, categories, func(category string) string { return fmt.Sprintf("%-32s", category) })
	fmt.Fprintf(w, "\nMutations: %.0f run, %.0f failed as meant\n", mutations["run"], mutations["failed"])
	writeOutcomes(w, checkpoints, read)
}

// writeOutcomes counts unwanted and a saved note's help over every unit: a unit judged no checkpoint can carry either.
func writeOutcomes(w io.Writer, checkpoints []judged, units []judged) {
	var reported []judged
	caughtBy := map[string]int{}
	disputed, failedGate, repeated := 0, 0, 0
	for _, unit := range checkpoints {
		if unit.result.After.UserReportedProblem {
			reported = append(reported, unit)
			caughtBy[unit.result.After.ShouldHaveBeenCaughtBy]++
		}
		if unit.result.After.UserDisputedAFindingOrFix {
			disputed++
		}
		if unit.result.Gate.FailedAfterReview {
			failedGate++
		}
		if unit.result.RepeatedWork != "" {
			repeated++
		}
	}
	memory, unwanted := 0, 0
	for _, unit := range units {
		if unit.result.MemoryWouldHelp != "" {
			memory++
		}
		if unit.result.After.UserSaidCheckpointUnwanted {
			unwanted++
		}
	}
	fmt.Fprintf(w, "\nUser reported a problem after %d of %d checkpoints; should have been caught by %s\n", len(reported), len(checkpoints), joinCounts(caughtBy))
	for _, unit := range reported {
		fmt.Fprintf(w, "  - %s %s [%s/%s] %s\n", shortSession(unit.session), unit.date, unit.result.After.Category, unit.result.After.ShouldHaveBeenCaughtBy, cut(unit.result.After.What, 160))
	}
	fmt.Fprintf(w, "User disputed a finding or fix: checkpoints %d. Said a checkpoint was unwanted: units %d.\n", disputed, unwanted)
	fmt.Fprintf(w, "Gate failed after review: checkpoints %d.\n", failedGate)
	fmt.Fprintf(w, "Repeated work noted: checkpoints %d. A saved note would have helped: units %d.\n", repeated, memory)
}

func writeSubagentEra(w io.Writer, runs []judged, cycles []judged, coveredCycles int) {
	units := append(slices.Clone(runs), cycles...)
	var checkpoints []judged
	confidence := map[string]int{}
	for _, unit := range units {
		confidence[unit.result.Confidence]++
		if unit.checkpoint() {
			checkpoints = append(checkpoints, unit)
		}
	}
	fmt.Fprintf(w, "Subagent era: run judgments %d, review-cycle judgments %d, cycles inside a judged run's window left out %d, checkpoints %d. Confidence high %d, medium %d, low %d.\n", len(runs), len(cycles), coveredCycles, len(checkpoints), confidence["high"], confidence["medium"], confidence["low"])
	rereviews := map[string]int{}
	for _, unit := range checkpoints {
		rereviews[unit.result.Rereview]++
	}
	coveredShortCircuits := 0
	for _, unit := range units {
		if unit.result.CoveredShortCircuit {
			coveredShortCircuits++
		}
	}
	startedBy := map[string]int{}
	for _, cycle := range cycles {
		startedBy[cycle.result.StartedBy]++
	}
	fmt.Fprintf(w, "Re-review ran %d, skipped %d, not done %d. Covered short-circuit %d. Review cycles started by %s.\n\n", rereviews["ran"], rereviews["skipped"], rereviews["not-done"], coveredShortCircuits, joinCounts(startedBy))

	reportRuns := make([]checkpoint.Run, 0, len(checkpoints))
	var findings []finding
	for _, unit := range checkpoints {
		run := checkpoint.Run{Confidence: unit.result.Confidence, Mode: unit.result.Mode, Mutations: unit.result.Mutations, Notes: unit.result.Notes, Rereview: unit.result.Rereview}
		if unit.result.Change.Lines == nil {
			run.ChangeUnknown = true
		} else {
			run.ChangeLines = *unit.result.Change.Lines
		}
		for _, entry := range findingsOf(unit) {
			findings = append(findings, entry)
			severity := cmp.Or(entry.Severity, "unrated")
			run.Findings = append(run.Findings, checkpoint.Finding{AlsoRaisedBy: entry.AlsoRaisedBy, Category: entry.Category, Origin: entry.Origin, Severity: severity, Shared: len(entry.AlsoRaisedBy) > 0, Source: entry.Source, Stage: entry.Stage, Verdict: entry.Verdict})
		}
		reportRuns = append(reportRuns, run)
	}
	checkpoint.Report(w, reportRuns)

	categories := make([]string, 0, len(findings))
	var kinds []string
	var sourceKinds [][2]string
	brokenCategories, missedCategories := []string{}, []string{}
	for _, entry := range findings {
		categories = append(categories, entry.Category)
		if entry.Stage != "rereview" {
			continue
		}
		kinds = append(kinds, entry.RereviewKind)
		sourceKinds = append(sourceKinds, [2]string{entry.Source, entry.RereviewKind})
		switch entry.RereviewKind {
		case "broken-by-fix":
			brokenCategories = append(brokenCategories, entry.Category)
		case "missed-first-round":
			missedCategories = append(missedCategories, entry.Category)
		}
	}
	fmt.Fprintf(w, "\nFindings by category, %d in all\n", len(findings))
	checkpoint.WriteMostCommon(w, categories, func(category string) string { return fmt.Sprintf("%-32s", category) })
	ran, foundNothing, brokenIn, missedIn := 0, 0, 0, 0
	for _, unit := range checkpoints {
		if unit.result.Rereview != "ran" {
			continue
		}
		ran++
		unitFindings := findingsOf(unit)
		if !slices.ContainsFunc(unitFindings, func(entry finding) bool { return entry.Stage == "rereview" }) {
			foundNothing++
		}
		if slices.ContainsFunc(unitFindings, func(entry finding) bool { return entry.RereviewKind == "broken-by-fix" }) {
			brokenIn++
		}
		if slices.ContainsFunc(unitFindings, func(entry finding) bool { return entry.RereviewKind == "missed-first-round" }) {
			missedIn++
		}
	}
	fmt.Fprintf(w, "\nRe-review findings by the classifier's kind, every verdict, %d in all; of %d re-reviews that ran, %d found nothing, %d something a fix broke, %d something the first round missed\n", len(kinds), ran, foundNothing, brokenIn, missedIn)
	checkpoint.WriteMostCommon(w, kinds, func(kind string) string { return fmt.Sprintf("%-32s", kind) })
	fmt.Fprintln(w, "By source and kind")
	checkpoint.WriteMostCommon(w, sourceKinds, func(key [2]string) string { return fmt.Sprintf("%-14s %-20s", key[0], key[1]) })
	fmt.Fprintln(w, "Broken by a fix, by category")
	checkpoint.WriteMostCommon(w, brokenCategories, func(category string) string { return fmt.Sprintf("%-32s", category) })
	fmt.Fprintln(w, "Missed by the first round, by category")
	checkpoint.WriteMostCommon(w, missedCategories, func(category string) string { return fmt.Sprintf("%-32s", category) })

	var pairs, sharedCategories []string
	reviewed := 0
	for _, entry := range findings {
		if entry.Stage == "review" || entry.Stage == "rereview" {
			reviewed++
		}
		if len(entry.AlsoRaisedBy) == 0 {
			continue
		}
		sources := slices.Compact(slices.Sorted(slices.Values(append([]string{entry.Source}, entry.AlsoRaisedBy...))))
		pairs = append(pairs, strings.Join(sources, "+"))
		sharedCategories = append(sharedCategories, entry.Category)
	}
	fmt.Fprintf(w, "\nRaised by more than one reviewer: %d of %d review and re-review findings, by reviewers\n", len(pairs), reviewed)
	checkpoint.WriteMostCommon(w, pairs, func(sources string) string { return fmt.Sprintf("%-32s", sources) })
	fmt.Fprintln(w, "By category")
	checkpoint.WriteMostCommon(w, sharedCategories, func(category string) string { return fmt.Sprintf("%-32s", category) })

	refuted := map[string]int{}
	var refutedCategories []string
	reraised := 0
	for _, entry := range findings {
		if entry.Verdict == "refuted" {
			refuted[entry.Source]++
			refutedCategories = append(refutedCategories, entry.Category)
		}
		if entry.ReraisesUserDecision {
			reraised++
		}
	}
	reraisedIn := 0
	for _, unit := range checkpoints {
		if slices.ContainsFunc(findingsOf(unit), func(entry finding) bool { return entry.ReraisesUserDecision }) {
			reraisedIn++
		}
	}
	fmt.Fprintf(w, "\nRefuted: %d of %d findings; by source %s; by category\n", len(refutedCategories), len(findings), joinCounts(refuted))
	checkpoint.WriteMostCommon(w, refutedCategories, func(category string) string { return fmt.Sprintf("%-32s", category) })
	fmt.Fprintf(w, "Re-raised a decision the user had made: findings %d, checkpoints %d.\n", reraised, reraisedIn)

	writeOutcomes(w, checkpoints, units)
}
