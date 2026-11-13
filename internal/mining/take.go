package mining

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

type Taken struct {
	Added      int
	Classifier string
	Judgments  int
	LeftOut    []string
	Watermark  time.Time
}

// Take stops the watermark before the first unit still unjudged, so a unit the pass missed stays listed.
func Take(directory string, analysis string, prompt []byte, digest func(dto.MiningUnit) (string, error), judgments []byte, units []dto.MiningUnit) (Taken, error) {
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
	var leftOut []string
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
		if byKey[key].Open {
			return Taken{}, fmt.Errorf("judgments line %d: session %q is still live, so its unit at %q can still grow", line, record.Session, record.Timestamp)
		}
		named[key] = line
		if err := checkResult(scanner.Bytes()); err != nil {
			leftOut = append(leftOut, fmt.Sprintf("judgments line %d: %v", line, err))
			continue
		}
		content, err := digest(byKey[key])
		if err != nil {
			return Taken{}, fmt.Errorf("judgments line %d: %w", line, err)
		}
		judgment := Judgment{Classifier: classifier, Digest: fingerprint([]byte(content)), Era: eraAt(at), Result: bytes.Clone(scanner.Bytes()), Session: record.Session, Timestamp: record.Timestamp}
		parsed = append(parsed, judgment)
	}
	if err := scanner.Err(); err != nil {
		return Taken{}, err
	}
	if len(parsed) == 0 && len(leftOut) > 0 {
		return Taken{}, fmt.Errorf("no judgments to store; left out %s", strings.Join(leftOut, "; "))
	}
	if len(parsed) == 0 {
		return Taken{}, errors.New("no judgments to store")
	}
	if _, err := SavePrompt(directory, prompt); err != nil {
		return Taken{}, err
	}
	added, err := Store(directory, analysis, parsed)
	if err != nil {
		return Taken{}, err
	}
	watermark, err := Watermark(directory, analysis)
	if err != nil {
		return Taken{}, err
	}
	held, err := Load(directory, analysis)
	if err != nil {
		return Taken{}, err
	}
	judged := map[[2]string]bool{}
	for _, judgment := range held {
		judged[[2]string{judgment.Session, judgment.Timestamp}] = true
	}
	var firstPending time.Time
	for key, at := range known {
		if !judged[key] && at.After(watermark) && (firstPending.IsZero() || at.Before(firstPending)) {
			firstPending = at
		}
	}
	raised := watermark
	for _, at := range known {
		if at.After(raised) && (firstPending.IsZero() || at.Before(firstPending)) {
			raised = at
		}
	}
	if raised.After(watermark) {
		if err = Advance(directory, analysis, raised); err != nil {
			return Taken{}, err
		}
	}
	return Taken{Added: added, Classifier: classifier, Judgments: len(parsed), LeftOut: leftOut, Watermark: raised}, nil
}
