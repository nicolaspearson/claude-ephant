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

	"github.com/nicolaspearson/claude-ephant/internal/atomicfile"
	"github.com/nicolaspearson/claude-ephant/internal/datadir"
)

var methods = []string{"artefacts", "contradiction", "decisions", "graph", "premises", "status"}

type ledgerLine struct {
	Audited string            `json:"audited"`
	Hash    string            `json:"hash"`
	Key     string            `json:"key"`
	Methods []string          `json:"methods"`
	Rulings []json.RawMessage `json:"rulings"`
}

type manifestEntry struct {
	Alias string `json:"alias"`
	Hash  string `json:"hash"`
	Key   string `json:"key"`
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
	hashes, aliases, err := loadManifest(manifestPath)
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
	target := ledgerTarget(ledgerPath)
	// Unlocked, two sessions recording at once each rewrite the ledger they read, losing one run's lines.
	unlock, err := atomicfile.Lock(target + ".lock")
	if err != nil {
		return fmt.Errorf("nothing recorded: %w", err)
	}
	defer unlock()
	ledger, err := loadLedger(w, ledgerPath)
	if err != nil {
		return fmt.Errorf("nothing recorded: %w", err)
	}
	renameAliases(ledger, aliases)
	audited := now.UTC().Format("2006-01-02T15:04:05+00:00")
	for _, record := range listed {
		line := ledgerLine{Audited: audited}
		line.Key, _ = jsonString(record["key"])
		line.Hash = hashes[line.Key]
		_ = json.Unmarshal(record["methods"], &line.Methods)
		_ = json.Unmarshal(record["rulings"], &line.Rulings)
		ledger = append(ledger, line)
	}
	var lines bytes.Buffer
	for _, line := range mergeLines(ledger) {
		lines.WriteString(dumpsLine(line))
	}
	if err = atomicfile.Write(target, lines.Bytes()); err != nil {
		return err
	}
	fmt.Fprintf(w, "recorded %d item(s) in %s\n", len(listed), ledgerPath)
	return nil
}

func WriteRulings(w io.Writer, manifestPath string, ledgerPath string) error {
	hashes, aliases, err := loadManifest(manifestPath)
	if err != nil {
		return err
	}
	ledger, err := loadLedger(w, ledgerPath)
	if err != nil {
		return err
	}
	renameAliases(ledger, aliases)
	type shown struct {
		audited string
		earlier ruling
		key     string
		since   string
	}
	var printed []shown
	at := map[[3]string]int{}
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
		for _, raw := range line.Rulings {
			entry := shown{audited: string(audited[:min(10, len(audited))]), earlier: parseRuling(raw), key: line.Key, since: since}
			identity := [3]string{line.Key, entry.earlier.Finding, entry.earlier.Ruling}
			index, seen := at[identity]
			switch {
			case !seen:
				at[identity] = len(printed)
				printed = append(printed, entry)
			case since == "unchanged" || printed[index].since == "CHANGED":
				printed[index] = entry
			}
		}
	}
	for _, entry := range printed {
		fmt.Fprintf(w, "%s\n  ruled %s: %s — %s\n  text %s since\n", entry.key, entry.audited, entry.earlier.Finding, entry.earlier.Ruling, entry.since)
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

// json.dumps' ", " and ": " separators.
func dumpsLine(line ledgerLine) string {
	var dumped strings.Builder
	dumped.WriteString(`{"audited": ` + dumpsString(line.Audited) + `, "hash": ` + dumpsString(line.Hash) + `, "key": ` + dumpsString(line.Key) + `, "methods": [`)
	for index, method := range line.Methods {
		dumped.WriteString(separator(index) + dumpsString(method))
	}
	dumped.WriteString(`], "rulings": [`)
	for index, raw := range line.Rulings {
		dumped.WriteString(separator(index) + dumpsRuling(raw))
	}
	dumped.WriteString("]}\n")
	return dumped.String()
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

// A rename over a symlink would replace the link, so the ledger is written, and locked, at its target.
func ledgerTarget(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return path
	}
	if resolved, resolveErr := filepath.EvalSymlinks(path); resolveErr == nil {
		return resolved
	}
	if !filepath.IsAbs(target) {
		return filepath.Join(filepath.Dir(path), target)
	}
	return target
}

// Refuses a bad line rather than skipping it: a skipped line is a hidden ruling.
// An unparsable last line with no newline is a write cut short, which never held a whole ruling.
func loadLedger(w io.Writer, path string) ([]ledgerLine, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	torn := !bytes.HasSuffix(content, []byte("\n"))
	split := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	var lines []ledgerLine
	for number, text := range split {
		if strings.TrimSpace(text) == "" {
			continue
		}
		where := fmt.Sprintf("%s:%d", path, number+1)
		if !json.Valid([]byte(text)) {
			if torn && number == len(split)-1 {
				fmt.Fprintf(w, "warning: %s is cut short, so it is skipped; the next record drops it\n", where)
				continue
			}
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

func loadManifest(path string) (map[string]string, map[string]string, error) {
	raw, err := readJSON(path)
	if err != nil {
		return nil, nil, err
	}
	var entries []manifestEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	hashes, aliases := map[string]string{}, map[string]string{}
	for _, entry := range entries {
		hashes[entry.Key] = entry.Hash
		if entry.Alias != "" {
			aliases[entry.Alias] = entry.Key
		}
	}
	return hashes, aliases, nil
}

// One line per key and text, so a ruling on older text keeps that text's hash and still reads as changed.
func mergeLines(lines []ledgerLine) []ledgerLine {
	var merged []ledgerLine
	at := map[[2]string]int{}
	for _, line := range lines {
		group := [2]string{line.Key, line.Hash}
		index, ok := at[group]
		if !ok {
			index = len(merged)
			at[group] = index
			merged = append(merged, ledgerLine{Hash: line.Hash, Key: line.Key})
		}
		target := &merged[index]
		target.Audited = line.Audited
		for _, method := range line.Methods {
			if !slices.Contains(target.Methods, method) {
				target.Methods = append(target.Methods, method)
			}
		}
		for _, raw := range line.Rulings {
			if !slices.ContainsFunc(target.Rulings, func(kept json.RawMessage) bool { return parseRuling(kept) == parseRuling(raw) }) {
				target.Rulings = append(target.Rulings, raw)
			}
		}
	}
	return merged
}

func parseRuling(raw json.RawMessage) ruling {
	var parsed ruling
	_ = json.Unmarshal(raw, &parsed)
	return parsed
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

func renameAliases(ledger []ledgerLine, aliases map[string]string) {
	for index := range ledger {
		if key, ok := aliases[ledger[index].Key]; ok {
			ledger[index].Key = key
		}
	}
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
