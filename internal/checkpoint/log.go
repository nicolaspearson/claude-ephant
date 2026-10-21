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
)

var (
	kebab   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	seconds = regexp.MustCompile(`^[0-9]+$`)
)

func Append(ctx context.Context, root string, record map[string]any, logPath string, now time.Time) error {
	line := maps.Clone(record)
	head, _ := git(ctx, root, "rev-parse", "--verify", "--quiet", "HEAD")
	line["date"] = now.UTC().Format("2006-01-02T15:04:05Z")
	line["head"] = strings.TrimSpace(head)
	line["repository"] = root
	marker, markerErr := ReviewedFile(ctx, root)
	if startedSeconds, ok := startedAt(marker); markerErr == nil && ok {
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
	if markerErr != nil {
		return nil
	}
	if err = os.Remove(marker + ".started"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func LogPath() (string, error) {
	if path := os.Getenv("CHECKPOINT_LOG"); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "logs", "checkpoints.jsonl"), nil
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
	problems = append(problems, notOneOf("mode", valueAt(record, "mode"), "full", "small")...)
	problems = append(problems, notOneOf("rereview", valueAt(record, "rereview"), "ran", "skipped")...)
	problems = append(problems, notOneOf("scope", valueAt(record, "scope"), "branch", "delta", "uncommitted")...)
	for _, path := range [][]string{{"change", "files"}, {"change", "lines"}, {"mutations", "failed"}, {"mutations", "inconclusive"}, {"mutations", "run"}, {"notes", "approved"}, {"notes", "proposed"}} {
		if !wholeNumber(valueAt(record, path...)) {
			problems = append(problems, strings.Join(path, ".")+" is not a whole number")
		}
	}
	failed, failedOK := number(valueAt(record, "mutations", "failed"))
	inconclusive, inconclusiveOK := number(valueAt(record, "mutations", "inconclusive"))
	run, runOK := number(valueAt(record, "mutations", "run"))
	if failedOK && inconclusiveOK && runOK && failed+inconclusive > run {
		problems = append(problems, "mutations.failed plus mutations.inconclusive exceeds mutations.run")
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
		problems = append(problems, findingProblems(index, finding)...)
	}
	return record, problems, nil
}

func findingProblems(index int, finding map[string]any) []string {
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
	problems = append(problems, notOneOf(name+".origin", finding["origin"], "change", "earlier", "fix")...)
	problems = append(problems, notOneOf(name+".severity", finding["severity"], "blocker", "defect", "nit", "quality")...)
	problems = append(problems, notOneOf(name+".stage", finding["stage"], "gate", "mutation", "rereview", "review", "user")...)
	problems = append(problems, notOneOf(name+".verdict", finding["verdict"], "deliberate", "fixed", "open", "refuted")...)
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
