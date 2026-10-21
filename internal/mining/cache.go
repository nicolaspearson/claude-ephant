package mining

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/nicolaspearson/claude-ephant/internal/archive"
)

// Judgment cannot be rebuilt from the archive, so the cache is a source of truth, not an index.
type Judgment struct {
	Classifier string          `json:"classifier"`
	Digest     string          `json:"digest,omitempty"`
	Era        string          `json:"era,omitempty"`
	Result     json.RawMessage `json:"result"`
	Session    string          `json:"session"`
	Timestamp  string          `json:"timestamp"`
}

var analysisName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func checkAnalysis(analysis string) error {
	if !analysisName.MatchString(analysis) {
		return fmt.Errorf("analysis %q is not lowercase words joined by hyphens", analysis)
	}
	return nil
}

func Classifier(prompt []byte) string {
	return fingerprint(prompt)
}

func fingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func Load(directory string, analysis string) ([]Judgment, error) {
	if err := checkAnalysis(analysis); err != nil {
		return nil, err
	}
	path := filepath.Join(directory, analysis+".jsonl")
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var judgments []Judgment
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(nil, len(content)+1)
	for line := 1; scanner.Scan(); line++ {
		var judgment Judgment
		if err = json.Unmarshal(scanner.Bytes(), &judgment); err != nil {
			return nil, fmt.Errorf("%s line %d: %w", path, line, err)
		}
		judgments = append(judgments, judgment)
	}
	return judgments, scanner.Err()
}

// SavePrompt keeps the prompt beside the judgments, so a classifier hash stays readable after its source is gone.
func SavePrompt(directory string, prompt []byte) (string, error) {
	classifier := Classifier(prompt)
	path := filepath.Join(directory, "classifiers", classifier+".md")
	if _, err := os.Stat(path); err == nil {
		return classifier, nil
	}
	if err := writeAtomically(path, prompt); err != nil {
		return "", err
	}
	return classifier, nil
}

// Store skips a judgment already held for its unit, classifier and digest.
func Store(directory string, analysis string, judgments []Judgment) (int, error) {
	unlock, err := archive.Lock(directory)
	if err != nil {
		return 0, err
	}
	defer unlock()
	held, err := Load(directory, analysis)
	if err != nil {
		return 0, err
	}
	type key struct{ classifier, digest, session, timestamp string }
	seen := make(map[key]bool, len(held))
	for _, judgment := range held {
		seen[key{judgment.Classifier, judgment.Digest, judgment.Session, judgment.Timestamp}] = true
	}
	var content bytes.Buffer
	for _, judgment := range held {
		if err = appendLine(&content, judgment); err != nil {
			return 0, err
		}
	}
	added := 0
	for _, judgment := range judgments {
		unit := key{judgment.Classifier, judgment.Digest, judgment.Session, judgment.Timestamp}
		if unit.classifier == "" || unit.session == "" || unit.timestamp == "" {
			return 0, fmt.Errorf("a judgment needs a classifier, a session and a timestamp: %+v", unit)
		}
		if seen[unit] {
			continue
		}
		seen[unit] = true
		if err = appendLine(&content, judgment); err != nil {
			return 0, err
		}
		added++
	}
	if added == 0 {
		return 0, nil
	}
	return added, writeAtomically(filepath.Join(directory, analysis+".jsonl"), content.Bytes())
}

func appendLine(content *bytes.Buffer, judgment Judgment) error {
	var result bytes.Buffer
	if err := json.Compact(&result, judgment.Result); err != nil {
		return fmt.Errorf("result for %s at %s: %w", judgment.Session, judgment.Timestamp, err)
	}
	judgment.Result = result.Bytes()
	line, err := json.Marshal(judgment)
	if err != nil {
		return err
	}
	content.Write(line)
	content.WriteByte('\n')
	return nil
}

func writeAtomically(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, err = temporary.Write(content)
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(temporary.Name(), path)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
	}
	return err
}
