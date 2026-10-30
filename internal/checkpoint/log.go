package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
)

var (
	fullSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
	kebab   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	seconds = regexp.MustCompile(`^[0-9]+$`)
)

var reviewScopes = []string{"local-branch", "mr"}

func Append(ctx context.Context, root string, record map[string]any, logPath string, now time.Time) error {
	line := maps.Clone(record)
	head, _ := git(ctx, root, "rev-parse", "--verify", "--quiet", "HEAD")
	line["date"] = now.UTC().Format("2006-01-02T15:04:05Z")
	line["head"] = strings.TrimSpace(head)
	line["repository"] = root
	run, runErr := RunFile(ctx, root, Session())
	// A review logged mid-checkpoint leaves that run's clock running.
	timed := runErr == nil && !IsReview(record)
	if startedSeconds, ok := startedAt(run); timed && ok {
		line["duration"] = now.Unix() - startedSeconds
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(line); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(encoded.Bytes()); err != nil {
		_ = file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if !timed {
		return nil
	}
	if err = os.Remove(run + ".started"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func IsReview(record map[string]any) bool {
	scope, _ := record["scope"].(string)
	return isReviewScope(scope)
}

func LogPath() (string, error) {
	if path := os.Getenv("CHECKPOINT_LOG"); path != "" {
		return path, nil
	}
	return datadir.LogPath(datadir.CheckpointLog)
}

// Problems reads every value on input, so empty input and two records are refused, not logged as none or two runs.
func Problems(input io.Reader) (map[string]any, []string, error) {
	decoder := json.NewDecoder(input)
	decoder.UseNumber()
	var records []any
	for {
		var record any
		err := decoder.Decode(&record)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		records = append(records, record)
	}
	switch {
	case len(records) == 0:
		return nil, []string{"no record on stdin"}, nil
	case len(records) > 1:
		return nil, []string{fmt.Sprintf("%d records on stdin; send one per checkpoint", len(records))}, nil
	}
	record, ok := records[0].(map[string]any)
	if !ok {
		return nil, []string{"the record is not an object"}, nil
	}
	var problems []string
	if _, supplied := record["duration"]; supplied {
		problems = append(problems, "duration is derived from the first freeze; leave it out")
	}
	review := IsReview(record)
	if review {
		problems = append(problems, notOneOf("mode", valueAt(record, "mode"), "full")...)
	} else {
		problems = append(problems, notOneOf("mode", valueAt(record, "mode"), "full", "small")...)
	}
	problems = append(problems, notOneOf("scope", valueAt(record, "scope"), "branch", "delta", "local-branch", "mr", "uncommitted")...)
	counts := [][]string{{"change", "files"}, {"change", "lines"}}
	if review {
		counts = append(counts, []string{"dropped", "excluded"}, []string{"dropped", "minor"}, []string{"dropped", "raised"})
		for _, name := range []string{"mutations", "rereview"} {
			if _, present := record[name]; present {
				problems = append(problems, name+" is absent on a review, since none runs; leave it out")
			}
		}
		if target, _ := record["target"].(string); target == "" {
			problems = append(problems, "a review has no target")
		}
		if reviewed, _ := record["reviewed"].(string); !fullSHA.MatchString(reviewed) {
			problems = append(problems, fmt.Sprintf("reviewed %s is not a full commit SHA", toJSON(record["reviewed"])))
		}
	} else {
		counts = append(counts, []string{"mutations", "failed"}, []string{"mutations", "inconclusive"}, []string{"mutations", "run"})
		problems = append(problems, notOneOf("rereview", valueAt(record, "rereview"), "ran", "skipped")...)
		for _, name := range []string{"dropped", "reviewed", "target"} {
			if _, present := record[name]; present {
				problems = append(problems, name+" belongs to a review scope; leave it out")
			}
		}
		failed, failedOK := number(valueAt(record, "mutations", "failed"))
		inconclusive, inconclusiveOK := number(valueAt(record, "mutations", "inconclusive"))
		run, runOK := number(valueAt(record, "mutations", "run"))
		if failedOK && inconclusiveOK && runOK && failed+inconclusive > run {
			problems = append(problems, "mutations.failed plus mutations.inconclusive exceeds mutations.run")
		}
	}
	for _, path := range counts {
		if !wholeNumber(valueAt(record, path...)) {
			problems = append(problems, strings.Join(path, ".")+" is not a whole number")
		}
	}
	findings, ok := record["findings"].([]any)
	if !ok {
		return record, append(problems, "findings is not a list"), nil
	}
	for index, entry := range findings {
		finding, ok := entry.(map[string]any)
		if !ok {
			problems = append(problems, fmt.Sprintf("findings[%d] is not an object", index))
			continue
		}
		problems = append(problems, findingProblems(index, finding, review)...)
	}
	return record, problems, nil
}

func findingProblems(index int, finding map[string]any, review bool) []string {
	name := fmt.Sprintf("findings[%d]", index)
	var problems []string
	source, _ := finding["source"].(string)
	if raised, present := finding["also_raised_by"]; present {
		list, ok := raised.([]any)
		sources := make([]string, 0, len(list))
		for _, item := range list {
			if text, _ := item.(string); text != "" {
				sources = append(sources, text)
			}
		}
		switch {
		case !ok || len(list) == 0 || len(sources) != len(list):
			problems = append(problems, name+".also_raised_by is not a list of sources")
		case slices.Contains(sources, source):
			problems = append(problems, fmt.Sprintf("%s.also_raised_by repeats its source %s", name, toJSON(finding["source"])))
		case len(slices.Compact(slices.Sorted(slices.Values(sources)))) != len(sources):
			problems = append(problems, name+".also_raised_by names a source twice")
		}
	}
	problems = append(problems, notOneOf(name+".severity", finding["severity"], "blocker", "defect", "nit", "quality")...)
	if review {
		problems = append(problems, notOneOf(name+".origin", finding["origin"], "change")...)
		problems = append(problems, notOneOf(name+".stage", finding["stage"], "review")...)
		problems = append(problems, notOneOf(name+".verdict", finding["verdict"], "confirmed", "open", "refuted")...)
		outcome, present := finding["outcome"]
		switch verdict, _ := finding["verdict"].(string); {
		case verdict == "refuted" && present:
			problems = append(problems, name+".outcome is absent on a refuted finding, which is never shown; leave it out")
		case verdict == "confirmed" || verdict == "open":
			problems = append(problems, notOneOf(name+".outcome", outcome, "discarded", "published", "raw", "skipped", "unanswered")...)
		}
	} else {
		problems = append(problems, notOneOf(name+".origin", finding["origin"], "change", "earlier", "fix")...)
		problems = append(problems, notOneOf(name+".stage", finding["stage"], "gate", "mutation", "rereview", "review", "user")...)
		problems = append(problems, notOneOf(name+".verdict", finding["verdict"], "deliberate", "fixed", "open", "refuted")...)
		if _, present := finding["outcome"]; present {
			problems = append(problems, name+".outcome belongs to a review scope; leave it out")
		}
	}
	if category, ok := finding["category"].(string); !ok || !kebab.MatchString(category) {
		problems = append(problems, fmt.Sprintf("%s.category %s is not kebab-case", name, toJSON(finding["category"])))
	}
	if source == "" {
		problems = append(problems, name+" has no source")
	}
	if summary, _ := finding["summary"].(string); summary == "" {
		problems = append(problems, name+" has no summary")
	}
	return problems
}

func isReviewScope(scope string) bool {
	return slices.Contains(reviewScopes, scope)
}

func notOneOf(name string, value any, allowed ...string) []string {
	if text, ok := value.(string); ok && slices.Contains(allowed, text) {
		return nil
	}
	return []string{fmt.Sprintf("%s %s is not one of %s", name, toJSON(value), strings.Join(allowed, ", "))}
}

func number(value any) (float64, bool) {
	literal, ok := value.(json.Number)
	if !ok {
		return 0, false
	}
	parsed, err := literal.Float64()
	return parsed, err == nil
}

func startedAt(marker string) (int64, bool) {
	started, err := os.ReadFile(marker + ".started")
	text := strings.TrimRight(string(started), "\n")
	if err != nil || !seconds.MatchString(text) {
		return 0, false
	}
	startedSeconds, err := strconv.ParseInt(text, 10, 64)
	return startedSeconds, err == nil
}

func toJSON(value any) string {
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return fmt.Sprint(value)
	}
	return strings.TrimSuffix(encoded.String(), "\n")
}

func valueAt(record map[string]any, path ...string) any {
	var value any = record
	for _, key := range path {
		object, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = object[key]
	}
	return value
}

func wholeNumber(value any) bool {
	parsed, ok := number(value)
	return ok && parsed >= 0 && math.Floor(parsed) == parsed
}
