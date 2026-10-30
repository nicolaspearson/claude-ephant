package ticket

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/nicolaspearson/claude-ephant/internal/datadir"
)

var methods = []string{"artefacts", "contradiction", "decisions", "graph", "premises", "status"}

type ledgerLine struct {
	Audited string   `json:"audited"`
	Hash    string   `json:"hash"`
	Key     string   `json:"key"`
	Rulings []ruling `json:"rulings"`
}

type manifestEntry struct {
	Hash string `json:"hash"`
	Key  string `json:"key"`
}

type ruling struct {
	Finding string `json:"finding"`
	Ruling  string `json:"ruling"`
}

func LedgerPath() (string, error) {
	return datadir.LogPath(datadir.TicketLedger)
}

// Record writes nothing unless every record can be recorded: a partial set reads as a whole audit.
func Record(w io.Writer, manifestPath string, recordsPath string, ledgerPath string, now time.Time) error {
	hashes, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	records, err := readJSON(recordsPath)
	if err != nil {
		return err
	}
	if problems := recordProblems(records, hashes); len(problems) > 0 {
		return fmt.Errorf("nothing recorded:\n  %s", strings.Join(problems, "\n  "))
	}
	var listed []map[string]json.RawMessage
	if err = json.Unmarshal(records, &listed); err != nil {
		return err
	}
	audited := now.UTC().Format("2006-01-02T15:04:05+00:00")
	// json.dumps' ", " and ": " separators.
	var lines bytes.Buffer
	for _, record := range listed {
		key, _ := jsonString(record["key"])
		lines.WriteString(`{"audited": ` + dumpsString(audited) + `, "hash": ` + dumpsString(hashes[key]) + `, "key": ` + dumpsString(key) + `, "methods": [`)
		var named []string
		_ = json.Unmarshal(record["methods"], &named)
		for index, method := range named {
			lines.WriteString(separator(index) + dumpsString(method))
		}
		lines.WriteString(`], "rulings": [`)
		var rulings []json.RawMessage
		_ = json.Unmarshal(record["rulings"], &rulings)
		for index, raw := range rulings {
			lines.WriteString(separator(index) + dumpsRuling(raw))
		}
		lines.WriteString("]}\n")
	}
	if err = os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		return err
	}
	ledger, err := os.OpenFile(ledgerPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := ledger.Write(lines.Bytes()); err != nil {
		_ = ledger.Close()
		return err
	}
	if err := ledger.Close(); err != nil {
		return err
	}
	fmt.Fprintf(w, "recorded %d item(s) in %s\n", len(listed), ledgerPath)
	return nil
}

func WriteRulings(w io.Writer, manifestPath string, ledgerPath string) error {
	hashes, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	ledger, err := loadLedger(ledgerPath)
	if err != nil {
		return err
	}
	for _, line := range ledger {
		hash, ok := hashes[line.Key]
		if !ok {
			continue
		}
		since := "CHANGED"
		if line.Hash == hash {
			since = "unchanged"
		}
		audited := []rune(line.Audited)
		for _, earlier := range line.Rulings {
			fmt.Fprintf(w, "%s\n  ruled %s: %s — %s\n  text %s since\n", line.Key, string(audited[:min(10, len(audited))]), earlier.Finding, earlier.Ruling, since)
		}
	}
	return nil
}

func compact(raw json.RawMessage) string {
	var buffer bytes.Buffer
	if json.Compact(&buffer, raw) != nil {
		return string(raw)
	}
	return buffer.String()
}

// A ruling keeps the field order it was given in.
func dumpsRuling(raw json.RawMessage) string {
	var fields ruling
	_ = json.Unmarshal(raw, &fields)
	finding, rulingText := `"finding": `+dumpsString(fields.Finding), `"ruling": `+dumpsString(fields.Ruling)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	_, _ = decoder.Token()
	if first, _ := decoder.Token(); first == "ruling" {
		return "{" + rulingText + ", " + finding + "}"
	}
	return "{" + finding + ", " + rulingText + "}"
}

// json.dumps' form: ASCII only, non-BMP as surrogate pairs.
func dumpsString(text string) string {
	var quoted strings.Builder
	quoted.WriteByte('"')
	for _, character := range text {
		switch {
		case character == '"' || character == '\\':
			quoted.WriteString(`\` + string(character))
		case character == '\b':
			quoted.WriteString(`\b`)
		case character == '\f':
			quoted.WriteString(`\f`)
		case character == '\n':
			quoted.WriteString(`\n`)
		case character == '\r':
			quoted.WriteString(`\r`)
		case character == '\t':
			quoted.WriteString(`\t`)
		case character >= ' ' && character <= '~':
			quoted.WriteRune(character)
		case character > 0xffff:
			high, low := utf16.EncodeRune(character)
			fmt.Fprintf(&quoted, `\u%04x\u%04x`, high, low)
		default:
			fmt.Fprintf(&quoted, `\u%04x`, character)
		}
	}
	quoted.WriteByte('"')
	return quoted.String()
}

func everyElement(raw json.RawMessage, valid func(json.RawMessage) bool) bool {
	var elements []json.RawMessage
	if !startsWith(raw, '[') || json.Unmarshal(raw, &elements) != nil {
		return false
	}
	return !slices.ContainsFunc(elements, func(element json.RawMessage) bool { return !valid(element) })
}

func exactFields(fields map[string]json.RawMessage, names ...string) bool {
	if len(fields) != len(names) {
		return false
	}
	return !slices.ContainsFunc(names, func(name string) bool { _, ok := fields[name]; return !ok })
}

func isMethod(raw json.RawMessage) bool {
	method, ok := jsonString(raw)
	return ok && slices.Contains(methods, method)
}

func jsonString(raw json.RawMessage) (string, bool) {
	var text string
	if !startsWith(raw, '"') || json.Unmarshal(raw, &text) != nil {
		return "", false
	}
	return text, true
}

// Refuses a bad line rather than skipping it: a skipped line is a hidden ruling.
func loadLedger(path string) ([]ledgerLine, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var lines []ledgerLine
	for number, text := range strings.Split(strings.TrimSuffix(string(content), "\n"), "\n") {
		if strings.TrimSpace(text) == "" {
			continue
		}
		where := fmt.Sprintf("%s:%d", path, number+1)
		if !json.Valid([]byte(text)) {
			var probe any
			return nil, fmt.Errorf("%s: not JSON: %w", where, json.Unmarshal([]byte(text), &probe))
		}
		var fields map[string]json.RawMessage
		if !startsWith([]byte(text), '{') || json.Unmarshal([]byte(text), &fields) != nil || slices.ContainsFunc([]string{"audited", "hash", "key", "methods", "rulings"}, func(name string) bool { _, ok := fields[name]; return !ok }) {
			return nil, fmt.Errorf("%s: needs fields ['audited', 'hash', 'key', 'methods', 'rulings']", where)
		}
		var line ledgerLine
		if err := json.Unmarshal([]byte(text), &line); err != nil {
			return nil, fmt.Errorf("%s: %w", where, err)
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func loadManifest(path string) (map[string]string, error) {
	raw, err := readJSON(path)
	if err != nil {
		return nil, err
	}
	var entries []manifestEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	hashes := map[string]string{}
	for _, entry := range entries {
		hashes[entry.Key] = entry.Hash
	}
	return hashes, nil
}

func readJSON(path string) (json.RawMessage, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	if !json.Valid(content) {
		var probe any
		return nil, fmt.Errorf("cannot read %s: %w", path, json.Unmarshal(content, &probe))
	}
	return bytes.TrimSpace(content), nil
}

func recordProblems(records json.RawMessage, hashes map[string]string) []string {
	var listed []json.RawMessage
	if !startsWith(records, '[') || json.Unmarshal(records, &listed) != nil {
		return []string{"records must be a list"}
	}
	var problems []string
	seen := map[string]bool{}
	for index, raw := range listed {
		where := fmt.Sprintf("record %d", index)
		var fields map[string]json.RawMessage
		if !startsWith(raw, '{') || json.Unmarshal(raw, &fields) != nil || !exactFields(fields, "key", "methods", "rulings") {
			problems = append(problems, where+": fields must be exactly ['key', 'methods', 'rulings']")
			continue
		}
		key, ok := jsonString(fields["key"])
		if !ok {
			problems = append(problems, fmt.Sprintf("%s: key must be a string, got %s", where, compact(fields["key"])))
			continue
		}
		if _, ok := hashes[key]; !ok {
			problems = append(problems, fmt.Sprintf("%s: key not in the manifest: %s", where, key))
		}
		if seen[key] {
			problems = append(problems, fmt.Sprintf("%s: key twice: %s", where, key))
		}
		seen[key] = true
		if !everyElement(fields["methods"], isMethod) {
			problems = append(problems, fmt.Sprintf("%s: methods must be a list from ['artefacts', 'contradiction', 'decisions', 'graph', 'premises', 'status'], got %s", where, compact(fields["methods"])))
		}
		if !everyElement(fields["rulings"], wellFormed) {
			problems = append(problems, where+": each ruling needs exactly a non-empty finding and ruling")
		}
	}
	return problems
}

func separator(index int) string {
	if index == 0 {
		return ""
	}
	return ", "
}

func startsWith(raw []byte, opening byte) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == opening
}

// Python's str.strip also drops \x1c to \x1f, which unicode.IsSpace does not.
func wellFormed(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if !startsWith(raw, '{') || json.Unmarshal(raw, &fields) != nil || !exactFields(fields, "finding", "ruling") {
		return false
	}
	for _, value := range fields {
		text, ok := jsonString(value)
		if !ok || strings.TrimFunc(text, func(character rune) bool { return unicode.IsSpace(character) || character >= 0x1c && character <= 0x1f }) == "" {
			return false
		}
	}
	return true
}
