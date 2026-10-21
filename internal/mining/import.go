package mining

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Imported struct {
	Added      int
	Analysis   string
	Classifier string
	Judgments  int
	Prompt     string
}

type unit struct {
	era       string
	session   string
	timestamp string
}

// Reruns of one unit by a later prompt carry their own hash, so both passes are kept.
var families = []struct {
	analysis string
	pattern  string
	prompt   string
	units    string
}{
	{analysis: "checkpoint-runs", pattern: "N*.jsonl", prompt: "classify-inl.md", units: "runs"},
	{analysis: "checkpoint-runs", pattern: "R*.jsonl", prompt: "classify-sub3.md", units: "runs"},
	{analysis: "checkpoint-runs", pattern: "S*.jsonl", prompt: "classify-sub.md", units: "runs"},
	{analysis: "review-cycles", pattern: "K*.jsonl", prompt: "classify-cyc.md", units: "cycles"},
}

// Import reads the 2026-10-15 run's layout. Its ids are positions in that run's inventory, so units are keyed by the session and time the inventory recorded.
func Import(run string, cache string) ([]Imported, error) {
	work := filepath.Join(run, "work")
	units := map[string]map[string]unit{}
	var runs []struct {
		Era       string `json:"era"`
		ID        string `json:"id"`
		Session   string `json:"session"`
		Timestamp string `json:"ts"`
	}
	if err := readJSON(filepath.Join(work, "runs_index.json"), &runs); err != nil {
		return nil, err
	}
	units["runs"] = map[string]unit{}
	for _, record := range runs {
		units["runs"][record.ID] = unit{era: record.Era, session: record.Session, timestamp: record.Timestamp}
	}
	var cycles []struct {
		CycleID string `json:"cid"`
		Session string `json:"session"`
		Start   string `json:"start"`
	}
	if err := readJSON(filepath.Join(work, "cycles.json"), &cycles); err != nil {
		return nil, err
	}
	units["cycles"] = map[string]unit{}
	for _, record := range cycles {
		units["cycles"][record.CycleID] = unit{session: record.Session, timestamp: record.Start}
	}

	var imported []Imported
	for _, family := range families {
		prompt, err := os.ReadFile(filepath.Join(work, family.prompt))
		if err != nil {
			return nil, err
		}
		classifier, err := SavePrompt(cache, prompt)
		if err != nil {
			return nil, err
		}
		files, err := filepath.Glob(filepath.Join(work, "class", family.pattern))
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no %s in %s", family.pattern, filepath.Join(work, "class"))
		}
		sort.Strings(files)
		var judgments []Judgment
		for _, path := range files {
			read, readErr := readJudgments(path, units[family.units], classifier)
			if readErr != nil {
				return nil, readErr
			}
			judgments = append(judgments, read...)
		}
		added, err := Store(cache, family.analysis, judgments)
		if err != nil {
			return nil, err
		}
		imported = append(imported, Imported{Added: added, Analysis: family.analysis, Classifier: classifier, Judgments: len(judgments), Prompt: family.prompt})
	}
	// Inventory end, not newest judgment: units the run left unjudged on purpose stay behind it.
	for _, family := range families {
		newest, err := newestUnit(units[family.units])
		if err != nil {
			return nil, err
		}
		if err = Advance(cache, family.analysis, newest); err != nil {
			return nil, err
		}
	}
	return imported, nil
}

func newestUnit(units map[string]unit) (time.Time, error) {
	var newest time.Time
	for id, found := range units {
		timestamp, err := time.Parse(time.RFC3339Nano, found.timestamp)
		if err != nil {
			return time.Time{}, fmt.Errorf("unit %s: %w", id, err)
		}
		if timestamp.After(newest) {
			newest = timestamp
		}
	}
	return newest, nil
}

func readJSON(path string, into any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(content, into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func readJudgments(path string, units map[string]unit, classifier string) ([]Judgment, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var judgments []Judgment
	scanner := bufio.NewScanner(file)
	scanner.Buffer(nil, 16<<20)
	for line := 1; scanner.Scan(); line++ {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var record struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, line, err)
		}
		found, ok := units[record.ID]
		if !ok {
			return nil, fmt.Errorf("%s line %d: id %q is not in the run's inventory", path, line, record.ID)
		}
		result := make(json.RawMessage, len(scanner.Bytes()))
		copy(result, scanner.Bytes())
		judgments = append(judgments, Judgment{Classifier: classifier, Era: found.era, Result: result, Session: found.session, Timestamp: found.timestamp})
	}
	return judgments, scanner.Err()
}
