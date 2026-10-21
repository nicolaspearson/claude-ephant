package mining

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

type Taken struct {
	Added      int
	Classifier string
	Judgments  int
	Watermark  time.Time
}

// Take stops the watermark before the first unit still unjudged, so a unit the pass missed stays listed.
func Take(cache string, analysis string, prompt []byte, digest func(dto.MiningUnit) (string, error), judgments []byte, units []dto.MiningUnit) (Taken, error) {
	known := map[[2]string]time.Time{}
	byKey := map[[2]string]dto.MiningUnit{}
	for _, unit := range units {
		at, err := parseTimestamp(unit.Session, unit.Timestamp)
		if err != nil {
			return Taken{}, err
		}
		known[[2]string{unit.Session, unit.Timestamp}] = at
		byKey[[2]string{unit.Session, unit.Timestamp}] = unit
	}
	classifier := Classifier(prompt)
	var parsed []Judgment
	named := map[[2]string]int{}
	scanner := bufio.NewScanner(bytes.NewReader(judgments))
	scanner.Buffer(nil, len(judgments)+1)
	for line := 1; scanner.Scan(); line++ {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var record struct {
			Session   string `json:"session"`
			Timestamp string `json:"timestamp"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return Taken{}, fmt.Errorf("judgments line %d: %w", line, err)
		}
		key := [2]string{record.Session, record.Timestamp}
		at, ok := known[key]
		if !ok {
			return Taken{}, fmt.Errorf("judgments line %d: no %s unit has session %q and timestamp %q", line, analysis, record.Session, record.Timestamp)
		}
		if first, ok := named[key]; ok {
			return Taken{}, fmt.Errorf("judgments line %d: line %d already judges session %q at %q", line, first, record.Session, record.Timestamp)
		}
		named[key] = line
		content, err := digest(byKey[key])
		if err != nil {
			return Taken{}, fmt.Errorf("judgments line %d: %w", line, err)
		}
		judgment := Judgment{Classifier: classifier, Digest: fingerprint([]byte(content)), Result: bytes.Clone(scanner.Bytes()), Session: record.Session, Timestamp: record.Timestamp}
		if analysis == "checkpoint-runs" {
			judgment.Era = "subagent"
			if at.Before(subagentEraAt) {
				judgment.Era = "inline"
			}
		}
		parsed = append(parsed, judgment)
	}
	if err := scanner.Err(); err != nil {
		return Taken{}, err
	}
	if len(parsed) == 0 {
		return Taken{}, errors.New("no judgments to store")
	}
	if _, err := SavePrompt(cache, prompt); err != nil {
		return Taken{}, err
	}
	added, err := Store(cache, analysis, parsed)
	if err != nil {
		return Taken{}, err
	}
	watermark, err := Watermark(cache, analysis)
	if err != nil {
		return Taken{}, err
	}
	held, err := Load(cache, analysis)
	if err != nil {
		return Taken{}, err
	}
	pending, _, err := Pending(units, held, watermark)
	if err != nil {
		return Taken{}, err
	}
	var firstPending time.Time
	if len(pending) > 0 {
		firstPending = known[[2]string{pending[0].Session, pending[0].Timestamp}]
	}
	raised := watermark
	for _, at := range known {
		if at.After(raised) && (firstPending.IsZero() || at.Before(firstPending)) {
			raised = at
		}
	}
	if raised.After(watermark) {
		if err = Advance(cache, analysis, raised); err != nil {
			return Taken{}, err
		}
	}
	return Taken{Added: added, Classifier: classifier, Judgments: len(parsed), Watermark: raised}, nil
}
