package ticket

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	firstKey  = "repo/plan.md: Build the thing"
	secondKey = "repo/plan.md: Ship the thing"
)

var ledgerNow = time.Date(2026, 10, 21, 9, 30, 0, 0, time.FixedZone("SAST", 2*60*60))

func TestDumpsStringWritesASCIIEscapes(t *testing.T) {
	require.Equal(t, `"\u00e9 \"q\" \\ \n\t\b\f\r \u0001 \u007f \u2028 \ud83d\ude00 <&> /"`, dumpsString("\u00e9 \"q\" \\ \n\t\b\f\r \x01 \x7f \u2028 \U0001f600 <&> /"))
}

func TestRecordAppendsOneLinePerItemWithTheManifestHash(t *testing.T) {
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "logs", "ticket-audits.jsonl")
	records := `[{"key":"` + firstKey + `","methods":["contradiction","premises"],"rulings":[{"ruling":"not a contradiction","finding":"contradicts item 2"}]},` +
		`{"key":"` + secondKey + `","methods":["premises"],"rulings":[]}]`

	output, err := record(t, directory, records, ledgerPath)

	require.NoError(t, err)
	require.Equal(t, "recorded 2 item(s) in "+ledgerPath+"\n", output)
	written, err := os.ReadFile(ledgerPath)
	require.NoError(t, err)
	require.Equal(t, `{"audited": "2026-10-21T07:30:00+00:00", "hash": "aaa", "key": "`+firstKey+`", "methods": ["contradiction", "premises"], "rulings": [{"ruling": "not a contradiction", "finding": "contradicts item 2"}]}`+"\n"+
		`{"audited": "2026-10-21T07:30:00+00:00", "hash": "bbb", "key": "`+secondKey+`", "methods": ["premises"], "rulings": []}`+"\n", string(written), "a ruling keeps its field order")
}

func TestRecordRefusesWhatTheAuditDidNotReadOrRunWritingNothing(t *testing.T) {
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ticket-audits.jsonl")
	for records, problem := range map[string]string{
		`5`:      "records must be a list",
		`[null]`: "record 0: fields must be exactly ['key', 'methods', 'rulings']",
		`[{"key":"` + firstKey + `","method":[],"methods":[],"rulings":[]}]`:                                                                            "record 0: fields must be exactly ['key', 'methods', 'rulings']",
		`[{"key":"` + firstKey + `","methods":["vibes"],"rulings":[]}]`:                                                                                 `record 0: methods must be a list from ['artefacts', 'contradiction', 'decisions', 'graph', 'premises', 'status'], got ["vibes"]`,
		`[{"key":"` + firstKey + `","methods":[],"rulings":[]},{"key":"` + firstKey + `","methods":[],"rulings":[]}]`:                                   "record 1: key twice: " + firstKey,
		`[{"key":"` + firstKey + `","methods":[],"rulings":[{"finding":"\u001c ","ruling":"x"}]}]`:                                                      "record 0: each ruling needs exactly a non-empty finding and ruling",
		`[{"key":"` + firstKey + `","methods":[],"rulings":[{"ruling":"x"}]}]`:                                                                          "record 0: each ruling needs exactly a non-empty finding and ruling",
		`[{"key":"` + secondKey + `","methods":["premises"],"rulings":[]},{"key":"repo/plan.md: Never read","methods":{"premises":true},"rulings":[]}]`: "record 1: key not in the manifest: repo/plan.md: Never read\n  record 1: methods must be a list from ['artefacts', 'contradiction', 'decisions', 'graph', 'premises', 'status'], got {\"premises\":true}",
		`[{"key":null,"methods":[],"rulings":[]}]`:                                                                                                      "record 0: key must be a string, got null",
	} {
		_, err := record(t, directory, records, ledgerPath)

		require.EqualError(t, err, "nothing recorded:\n  "+problem, records)
		require.NoFileExists(t, ledgerPath, records)
	}
}

func TestWriteRulingsLabelsEachEarlierRulingByWhetherItsTextChanged(t *testing.T) {
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ticket-audits.jsonl")
	writeLedger(t, ledgerPath,
		`{"audited":"2026-10-01T10:00:00+00:00","hash":"aaa","key":"`+firstKey+`","methods":[],"rulings":[{"finding":"f1","ruling":"r1"},{"finding":"f2","ruling":"r2"}]}`,
		"  ",
		`{"audited":"2026-10-02","hash":"old","key":"`+secondKey+`","methods":[],"rulings":[{"finding":"f3","ruling":"r3"}]}`,
		`{"audited":"2026-10-03","hash":"aaa","key":"repo/plan.md: Elsewhere","methods":[],"rulings":[{"finding":"f4","ruling":"r4"}]}`,
	)
	var output bytes.Buffer

	require.NoError(t, WriteRulings(&output, writeManifest(t, directory), ledgerPath))

	require.Equal(t, firstKey+"\n  ruled 2026-10-01: f1 — r1\n  text unchanged since\n"+
		firstKey+"\n  ruled 2026-10-01: f2 — r2\n  text unchanged since\n"+
		secondKey+"\n  ruled 2026-10-02: f3 — r3\n  text CHANGED since\n", output.String())
}

func TestWriteRulingsRefusesABadLineRatherThanHidingARuling(t *testing.T) {
	directory := t.TempDir()
	ledgerPath := filepath.Join(directory, "ticket-audits.jsonl")
	good := `{"audited":"x","hash":"aaa","key":"` + firstKey + `","methods":[],"rulings":[]}`
	for line, problem := range map[string]string{
		`[1]`: ":2: needs fields ['audited', 'hash', 'key', 'methods', 'rulings']",
		`{"hash":"aaa","key":"` + firstKey + `"}`: ":2: needs fields ['audited', 'hash', 'key', 'methods', 'rulings']",
		`{not json`: ":2: not JSON: ",
	} {
		writeLedger(t, ledgerPath, good, line)

		err := WriteRulings(&bytes.Buffer{}, writeManifest(t, directory), ledgerPath)

		require.Error(t, err, line)
		require.True(t, strings.HasPrefix(err.Error(), ledgerPath+problem), err.Error())
	}
	require.NoError(t, WriteRulings(&bytes.Buffer{}, writeManifest(t, directory), filepath.Join(directory, "missing.jsonl")), "no ledger yet holds no rulings")
}

func record(t *testing.T, directory string, records string, ledgerPath string) (string, error) {
	t.Helper()
	recordsPath := filepath.Join(directory, "records.json")
	require.NoError(t, os.WriteFile(recordsPath, []byte(records), 0o600))
	var output bytes.Buffer
	err := Record(&output, writeManifest(t, directory), recordsPath, ledgerPath, ledgerNow)
	return output.String(), err
}

func writeLedger(t *testing.T, path string, lines ...string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600))
}

func writeManifest(t *testing.T, directory string) string {
	t.Helper()
	manifest, err := json.Marshal([]map[string]string{{"file": "01.md", "hash": "aaa", "key": firstKey}, {"file": "02.md", "hash": "bbb", "key": secondKey}})
	require.NoError(t, err)
	path := filepath.Join(directory, "manifest.json")
	require.NoError(t, os.WriteFile(path, manifest, 0o600))
	return path
}
