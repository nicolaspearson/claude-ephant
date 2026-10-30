package checkpoint

import (
	"bufio"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Finding struct {
	AlsoRaisedBy []string
	Category     string
	Origin       string
	Outcome      string
	Severity     string
	Shared       bool
	Source       string
	Stage        string
	Verdict      string
}

type Loaded struct {
	First      string
	Runs       []Run
	Unreadable int
}

type Run struct {
	ChangeLines   float64
	ChangeUnknown bool
	Date          string
	Dropped       map[string]float64
	Duration      int64
	Findings      []Finding
	Mode          string
	Mutations     map[string]float64
	Rereview      string
	Scope         string
	Timed         bool
}

type counted[K comparable] struct {
	count int
	key   K
}

// Load counts a line it cannot read as unreadable and skips it. A missing file loads as empty.
func Load(path string, since time.Time) (Loaded, error) {
	var loaded Loaded
	file, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return loaded, nil
	}
	if err != nil {
		return loaded, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			run, when, ok := parseRun(line)
			if !ok {
				loaded.Unreadable++
			} else {
				if loaded.First == "" {
					loaded.First = run.Date[:10]
				}
				if !when.Before(since) {
					loaded.Runs = append(loaded.Runs, run)
				}
			}
		}
		if errors.Is(err, io.EOF) {
			return loaded, nil
		}
		if err != nil {
			return loaded, err
		}
	}
}

func Median(values []float64) float64 {
	sorted := slices.Sorted(slices.Values(values))
	middle := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[middle]
	}
	return (sorted[middle-1] + sorted[middle]) / 2
}

func Report(w io.Writer, runs []Run) {
	modes := map[string]int{}
	scopes := map[string]int{}
	var sized []float64
	for _, run := range runs {
		modes[run.Mode]++
		scopes[run.Scope]++
		if !run.ChangeUnknown {
			sized = append(sized, run.ChangeLines)
		}
	}
	size := "No change size known"
	if len(sized) > 0 {
		size = fmt.Sprintf("Median change %.6g lines added plus removed", Median(sized))
	}
	if unsized := len(runs) - len(sized); unsized > 0 {
		size += fmt.Sprintf(", %d of unknown size left out", unsized)
	}
	fmt.Fprintf(w, "Mode: full %d, small %d. %s.\n", modes["full"], modes["small"], size)
	fmt.Fprintf(w, "Scope: uncommitted %d, delta %d, branch %d.\n\n", scopes["uncommitted"], scopes["delta"], scopes["branch"])

	type staged struct {
		Finding
		mode string
	}
	var findings []staged
	for _, run := range runs {
		for _, finding := range run.Findings {
			findings = append(findings, staged{finding, run.Mode})
		}
	}
	fmt.Fprintln(w, "Who found what (verdict counts; a merged duplicate counts only for its source)")
	fmt.Fprintf(w, "%-10s %-14s %-6s %6s %8s %5s %11s %10s\n", "stage", "source", "mode", "fixed", "refuted", "open", "deliberate", "refuted %")
	table := map[[3]string]map[string]int{}
	for _, finding := range findings {
		key := [3]string{finding.Stage, finding.Source, finding.mode}
		if table[key] == nil {
			table[key] = map[string]int{}
		}
		table[key][finding.Verdict]++
	}
	keys := slices.SortedFunc(maps.Keys(table), func(a, b [3]string) int {
		return cmp.Or(strings.Compare(a[0], b[0]), strings.Compare(a[1], b[1]), strings.Compare(a[2], b[2]))
	})
	for _, key := range keys {
		verdicts := table[key]
		total := 0
		for _, count := range verdicts {
			total += count
		}
		fmt.Fprintf(w, "%-10s %-14s %-6s %6d %8d %5d %11d %10s\n", key[0], key[1], key[2], verdicts["fixed"], verdicts["refuted"], verdicts["open"], verdicts["deliberate"], percent(float64(verdicts["refuted"]), float64(total)))
	}

	fmt.Fprintln(w, "\nConfirmed findings (fixed or deliberate) by source and severity; a merged duplicate counts for each source that raised it")
	fmt.Fprintf(w, "%-14s %8s %7s %8s %4s %8s %7s\n", "source", "blocker", "defect", "quality", "nit", "unrated", "shared")
	severities := map[string]map[string]int{}
	for _, finding := range findings {
		if !confirmed(finding.Finding) {
			continue
		}
		for _, source := range append([]string{finding.Source}, finding.AlsoRaisedBy...) {
			if severities[source] == nil {
				severities[source] = map[string]int{}
			}
			severities[source][finding.Severity]++
			if finding.Shared {
				severities[source]["shared"]++
			}
		}
	}
	for _, source := range slices.Sorted(maps.Keys(severities)) {
		counts := severities[source]
		fmt.Fprintf(w, "%-14s %8d %7d %8d %4d %8d %7d\n", source, counts["blocker"], counts["defect"], counts["quality"], counts["nit"], counts["unrated"], counts["shared"])
	}
	if len(severities) == 0 {
		fmt.Fprintln(w, "  nothing")
	}

	confirmedRereview := func(finding Finding) bool {
		return finding.Stage == "rereview" && confirmed(finding)
	}
	ran, useful, broken, missed := 0, 0, 0, 0
	for _, run := range runs {
		if run.Rereview != "ran" {
			continue
		}
		ran++
		if slices.ContainsFunc(run.Findings, confirmedRereview) {
			useful++
		}
	}
	for _, finding := range findings {
		switch {
		case !confirmedRereview(finding.Finding):
		case finding.Origin == "fix":
			broken++
		default:
			missed++
		}
	}
	fmt.Fprintf(w, "\nRe-reviews: ran in %d of %d checkpoints (skipped when nothing changed after the first review)\n", ran, len(runs))
	fmt.Fprintf(w, "  found something confirmed: %d of %d\n", useful, ran)
	fmt.Fprintf(w, "  broken by a fix: %d\n", broken)
	fmt.Fprintf(w, "  missed by the first review: %d\n", missed)

	fmt.Fprintln(w, "\nCaught after review (gate, mutation, user), by category")
	var late [][2]string
	var older []string
	for _, finding := range findings {
		if slices.Contains([]string{"gate", "mutation", "user"}, finding.Stage) {
			late = append(late, [2]string{finding.Stage, finding.Category})
		}
		if finding.Origin == "earlier" && confirmed(finding.Finding) {
			older = append(older, finding.Category)
		}
	}
	WriteMostCommon(w, late, func(key [2]string) string { return fmt.Sprintf("%-9s %-32s", key[0], key[1]) })
	fmt.Fprintln(w, "\nOlder than the change (origin earlier), by category")
	WriteMostCommon(w, older, func(category string) string { return fmt.Sprintf("%-32s", category) })

	mutations := map[string]float64{}
	for _, run := range runs {
		for key, count := range run.Mutations {
			mutations[key] += count
		}
	}
	fmt.Fprintf(w, "\nMutations: %s run, %s failed as meant, %s did not compile or did not land (%s)\n", whole(mutations["run"]), whole(mutations["failed"]), whole(mutations["inconclusive"]), percent(mutations["inconclusive"], mutations["run"]))

	var durations []float64
	byMode := map[string][]float64{}
	for _, run := range runs {
		if run.Timed {
			durations = append(durations, float64(run.Duration))
			byMode[run.Mode] = append(byMode[run.Mode], float64(run.Duration))
		}
	}
	if len(durations) == 0 {
		fmt.Fprintln(w, "Duration: no checkpoint carries one")
		return
	}
	var medians []string
	for _, mode := range []string{"full", "small"} {
		if len(byMode[mode]) > 0 {
			medians = append(medians, fmt.Sprintf("%s %.1fm", mode, Median(byMode[mode])/60))
		}
	}
	fmt.Fprintf(w, "Duration, first freeze to log line: median %.1fm over %d of %d checkpoints (%s); longest %.1fm\n", Median(durations)/60, len(durations), len(runs), strings.Join(medians, ", "), slices.Max(durations)/60)
}

func ReportReviews(w io.Writer, runs []Run) {
	scopes := map[string]int{}
	dropped := map[string]float64{}
	var sized []float64
	for _, run := range runs {
		scopes[run.Scope]++
		sized = append(sized, run.ChangeLines)
		for key, count := range run.Dropped {
			dropped[key] += count
		}
	}
	fmt.Fprintf(w, "Reviews: mr %d, local-branch %d. Median change %.6g lines added plus removed.\n", scopes["mr"], scopes["local-branch"], Median(sized))
	fmt.Fprintf(w, "Dropped before verification: %s excluded, %s minor, %s already raised\n\n", whole(dropped["excluded"]), whole(dropped["minor"]), whole(dropped["raised"]))

	fmt.Fprintln(w, "Review findings by source; skip rate is skipped of those answered (published, raw, discarded, skipped)")
	fmt.Fprintf(w, "%-14s %9s %5s %8s %9s %4s %9s %7s %10s %9s\n", "source", "confirmed", "open", "refuted", "published", "raw", "discarded", "skipped", "unanswered", "skip rate")
	bySource := map[string]map[string]int{}
	total := map[string]int{}
	for _, run := range runs {
		for _, finding := range run.Findings {
			if bySource[finding.Source] == nil {
				bySource[finding.Source] = map[string]int{}
			}
			for _, counts := range []map[string]int{bySource[finding.Source], total} {
				counts[finding.Verdict]++
				counts[finding.Outcome]++
			}
		}
	}
	row := func(source string, counts map[string]int) {
		answered := counts["published"] + counts["raw"] + counts["discarded"] + counts["skipped"]
		fmt.Fprintf(w, "%-14s %9d %5d %8d %9d %4d %9d %7d %10d %9s\n", source, counts["confirmed"], counts["open"], counts["refuted"], counts["published"], counts["raw"], counts["discarded"], counts["skipped"], counts["unanswered"], percent(float64(counts["skipped"]), float64(answered)))
	}
	for _, source := range slices.Sorted(maps.Keys(bySource)) {
		row(source, bySource[source])
	}
	if len(bySource) == 0 {
		fmt.Fprintln(w, "  nothing")
		return
	}
	row("total", total)
}

func WriteMostCommon[K comparable](w io.Writer, keys []K, format func(K) string) {
	var counts []counted[K]
	at := map[K]int{}
	for _, key := range keys {
		if _, seen := at[key]; !seen {
			at[key] = len(counts)
			counts = append(counts, counted[K]{key: key})
		}
		counts[at[key]].count++
	}
	slices.SortStableFunc(counts, func(a, b counted[K]) int { return b.count - a.count })
	for _, entry := range counts[:min(15, len(counts))] {
		fmt.Fprintf(w, "  %s %d\n", format(entry.key), entry.count)
	}
	if len(counts) == 0 {
		fmt.Fprintln(w, "  nothing")
	}
}

func WriteStats(w io.Writer, logPath string, since time.Time) error {
	logged, err := Load(logPath, since)
	if err != nil {
		return err
	}
	var checkpoints, reviews []Run
	for _, run := range logged.Runs {
		if isReviewScope(run.Scope) {
			reviews = append(reviews, run)
		} else {
			checkpoints = append(checkpoints, run)
		}
	}
	starts := "The log has no readable line."
	if logged.First != "" {
		starts = "The log starts " + logged.First + "."
	}
	runs := fmt.Sprintf("%d checkpoint%s", len(checkpoints), plural(len(checkpoints)))
	if len(reviews) > 0 {
		runs += fmt.Sprintf(" and %d review%s", len(reviews), plural(len(reviews)))
	}
	fmt.Fprintf(w, "Since %s: %s logged. %s\n\n", since.UTC().Format("2006-01-02"), runs, starts)
	if logged.Unreadable > 0 {
		fmt.Fprintf(w, "%d log line%s unreadable or missing a field the report reads; skipped.\n\n", logged.Unreadable, plural(logged.Unreadable))
	}
	if len(checkpoints) > 0 {
		Report(w, checkpoints)
	}
	if len(reviews) > 0 {
		if len(checkpoints) > 0 {
			fmt.Fprintln(w)
		}
		ReportReviews(w, reviews)
	}
	if len(logged.Runs) == 0 {
		fmt.Fprintf(w, "Nothing logged in the window. The log is %s.\n", logPath)
	}
	return nil
}

func confirmed(finding Finding) bool {
	return finding.Verdict == "deliberate" || finding.Verdict == "fixed"
}

func integral(value any) (int64, bool) {
	literal, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	parsed, err := literal.Int64()
	return parsed, err == nil
}

func parseCounts(value any, required string) (map[string]float64, bool) {
	object, ok := value.(map[string]any)
	if _, present := object[required]; !ok || !present {
		return nil, false
	}
	counts := map[string]float64{}
	for key, count := range object {
		parsed, ok := number(count)
		if !ok {
			return nil, false
		}
		counts[key] = parsed
	}
	return counts, true
}

func parseFinding(value any) (Finding, bool) {
	object, ok := value.(map[string]any)
	if !ok {
		return Finding{}, false
	}
	var finding Finding
	for key, field := range map[string]*string{"category": &finding.Category, "origin": &finding.Origin, "source": &finding.Source, "stage": &finding.Stage, "verdict": &finding.Verdict} {
		if *field, ok = object[key].(string); !ok {
			return Finding{}, false
		}
	}
	finding.Outcome, _ = object["outcome"].(string)
	finding.Severity = "unrated"
	if severity, present := object["severity"]; present {
		finding.Severity, _ = severity.(string)
	}
	if others, ok := object["also_raised_by"].([]any); ok {
		finding.Shared = len(others) > 0
		for _, other := range others {
			if source, ok := other.(string); ok {
				finding.AlsoRaisedBy = append(finding.AlsoRaisedBy, source)
			}
		}
	}
	return finding, true
}

func parseRun(line string) (Run, time.Time, bool) {
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil || decoder.Decode(new(any)) != io.EOF {
		return Run{}, time.Time{}, false
	}
	object, ok := value.(map[string]any)
	if !ok {
		return Run{}, time.Time{}, false
	}
	var run Run
	if run.Date, ok = object["date"].(string); !ok {
		return Run{}, time.Time{}, false
	}
	when, err := time.Parse(time.RFC3339, run.Date)
	if err != nil {
		return Run{}, time.Time{}, false
	}
	mode, modeOK := object["mode"].(string)
	lines, linesOK := number(valueAt(object, "change", "lines"))
	entries, findingsOK := object["findings"].([]any)
	if !modeOK || !linesOK || !findingsOK {
		return Run{}, time.Time{}, false
	}
	run.ChangeLines, run.Mode = lines, mode
	if IsReview(object) {
		if run.Dropped, ok = parseCounts(object["dropped"], "excluded"); !ok {
			return Run{}, time.Time{}, false
		}
	} else {
		var mutationsOK, rereviewOK bool
		run.Mutations, mutationsOK = parseCounts(object["mutations"], "run")
		run.Rereview, rereviewOK = object["rereview"].(string)
		if !mutationsOK || !rereviewOK {
			return Run{}, time.Time{}, false
		}
	}
	for _, entry := range entries {
		finding, ok := parseFinding(entry)
		if !ok {
			return Run{}, time.Time{}, false
		}
		run.Findings = append(run.Findings, finding)
	}
	run.ChangeUnknown, _ = object["change_unknown"].(bool)
	run.Duration, run.Timed = integral(object["duration"])
	run.Scope = "uncommitted"
	if scope, present := object["scope"]; present {
		run.Scope, _ = scope.(string)
	}
	return run, when, true
}

func percent(part float64, total float64) string {
	if total == 0 {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", math.Floor(100*part/total))
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func whole(count float64) string {
	return strconv.FormatFloat(count, 'f', -1, 64)
}
