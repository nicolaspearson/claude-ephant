package notes

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const Day = 24 * time.Hour

// DueMinute is the layout of a due with a time, local like a date-only due, so the two compare as strings and a date sorts before its own times.
const DueMinute = "2006-01-02T15:04"

const (
	KindDecision = "decision"
	KindFact     = "fact"
	KindFlag     = "flag"
	KindHandoff  = "handoff"
	KindPlan     = "plan"
	KindQuestion = "question"
	KindReminder = "reminder"

	// An owner name cannot start with _, so no owner's folder clashes.
	OwnerEverywhere = "_all"

	ScopeEverywhere = "everywhere"
	ScopeGlobal     = "global"
	ScopeProject    = "project"

	VerdictBad  = "bad"
	VerdictGood = "good"
)

var (
	ErrInvalid = errors.New("invalid note")

	Kinds    = []string{KindDecision, KindFact, KindFlag, KindHandoff, KindPlan, KindQuestion, KindReminder}
	Verdicts = []string{VerdictBad, VerdictGood}
	scopes   = []string{ScopeEverywhere, ScopeGlobal, ScopeProject}

	// Five digits of days stay inside a time.Duration.
	everyShape = regexp.MustCompile(`^([1-9][0-9]{0,4})([dhm]?)$`)

	// A project name is one front matter value; directory names may hold spaces and accents.
	projectShape = regexp.MustCompile(`^\P{Cc}*$`)

	// A session is one front matter value and one `ele show` argument.
	sessionShape = regexp.MustCompile(`^[A-Za-z0-9_-]*$`)
)

type Note struct {
	Body       string
	Closed     string
	Date       string
	DoneFile   string
	Due        string
	Every      time.Duration
	ID         string
	Kind       string
	Owner      string
	Profile    string
	Project    string
	Scope      string
	Session    string
	Shown      string
	Supersedes []string
	Time       string
	Verdict    string
}

func ClearClosed(directory string, note Note) error {
	return clearKey(directory, note, "closed")
}

func ClearShown(directory string, note Note) error {
	return clearKey(directory, note, "shown")
}

func Format(note Note) []byte {
	var every string
	if note.Every != 0 {
		every = FormatEvery(note.Every)
	}
	var out bytes.Buffer
	out.WriteString("---\n")
	for _, field := range []struct{ key, value string }{
		{"closed", note.Closed},
		{"date", note.Date},
		{"done-file", note.DoneFile},
		{"due", note.Due},
		{"every", every},
		{"kind", note.Kind},
		{"owner", note.Owner},
		{"profile", note.Profile},
		{"project", note.Project},
		{"scope", note.Scope},
		{"session", note.Session},
		{"shown", note.Shown},
		{"supersedes", strings.Join(note.Supersedes, ", ")},
		{"time", note.Time},
		{"verdict", note.Verdict},
	} {
		if field.value != "" {
			fmt.Fprintf(&out, "%s: %s\n", field.key, field.value)
		}
	}
	out.WriteString("---\n")
	out.WriteString(strings.TrimSpace(note.Body))
	out.WriteString("\n")
	return out.Bytes()
}

// FormatEvery writes whole days bare, so a build that predates sub-day intervals still reads them.
func FormatEvery(every time.Duration) string {
	switch {
	case every%Day == 0:
		return strconv.Itoa(int(every / Day))
	case every%time.Hour == 0:
		return strconv.Itoa(int(every/time.Hour)) + "h"
	}
	return strconv.Itoa(int(every/time.Minute)) + "m"
}

func MarkClosed(directory string, note Note, when time.Time) error {
	return setKey(directory, note, "closed", when.UTC().Format(time.RFC3339))
}

func MarkShown(directory string, note Note, when time.Time) error {
	return setKey(directory, note, "shown", when.UTC().Format(time.RFC3339))
}

// Move rewrites only the owner line and refiles the note under its new owner, so a hand edit survives.
func Move(directory string, note Note, owner string) error {
	from := filepath.Join(directory, note.Owner, note.ID+".md")
	to := filepath.Join(directory, owner, note.ID+".md")
	failed := func(err error) error { return fmt.Errorf("move %s to %s: %w", from, owner, err) }
	info, err := os.Stat(from)
	if err != nil {
		return failed(err)
	}
	content, err := os.ReadFile(from)
	if err != nil {
		return failed(err)
	}
	lines := strings.SplitAfter(string(content), "\n")
	ownerIndex := -1
	for number := 1; number < len(lines) && strings.TrimSpace(lines[number]) != "---"; number++ {
		if key, _, _ := strings.Cut(lines[number], ":"); strings.TrimSpace(key) == "owner" {
			ownerIndex = number
			break
		}
	}
	if ownerIndex < 0 {
		return fmt.Errorf("%w %s: no owner line to rewrite", ErrInvalid, from)
	}
	lines[ownerIndex] = "owner: " + owner + "\n"
	if err = os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return failed(err)
	}
	if _, err = os.Lstat(to); err == nil {
		return failed(fmt.Errorf("%s already exists", to))
	}
	temporary, err := os.CreateTemp(filepath.Dir(to), note.ID+".md.*.tmp")
	if err != nil {
		return failed(err)
	}
	_, err = temporary.WriteString(strings.Join(lines, ""))
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(temporary.Name(), info.Mode().Perm())
	}
	if err == nil {
		err = os.Rename(temporary.Name(), to)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
		return failed(err)
	}
	if err = os.Remove(from); err != nil {
		return failed(err)
	}
	return nil
}

func Relocate(fromDirectory string, toDirectory string, note Note) error {
	from := filepath.Join(fromDirectory, note.Owner, note.ID+".md")
	to := filepath.Join(toDirectory, note.Owner, note.ID+".md")
	if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return fmt.Errorf("relocate %s: %w", from, err)
	}
	if _, err := os.Lstat(to); err == nil {
		return fmt.Errorf("relocate %s: %s already exists", from, to)
	}
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("relocate %s: %w", from, err)
	}
	return nil
}

func NewID(date time.Time) (string, error) {
	suffix := make([]byte, 3)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("note id: %w", err)
	}
	return date.Format("20060102") + "-" + hex.EncodeToString(suffix), nil
}

// Parse takes the file name as the id; the owner in front matter must match the file's folder. Errors name path as given.
func Parse(path string, content []byte) (Note, error) {
	invalid := func(format string, arguments ...any) (Note, error) {
		return Note{}, fmt.Errorf("%w %s: %s", ErrInvalid, path, fmt.Sprintf(format, arguments...))
	}
	note := Note{ID: strings.TrimSuffix(filepath.Base(path), ".md")}
	var every, supersedes string
	scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))))
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return invalid("first line is not ---")
	}
	fields := map[string]*string{
		"closed": &note.Closed, "date": &note.Date, "done-file": &note.DoneFile, "due": &note.Due, "every": &every, "kind": &note.Kind, "owner": &note.Owner, "profile": &note.Profile,
		"project": &note.Project, "scope": &note.Scope, "session": &note.Session, "shown": &note.Shown, "supersedes": &supersedes, "time": &note.Time,
		"verdict": &note.Verdict,
	}
	seen := map[string]bool{}
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return invalid("front matter line %q is not key: value", line)
		}
		key = strings.TrimSpace(key)
		field, known := fields[key]
		if !known {
			return invalid("unknown front matter key %q", key)
		}
		if seen[key] {
			return invalid("front matter key %q appears twice", key)
		}
		seen[key] = true
		*field = strings.TrimSpace(value)
	}
	if !closed {
		return invalid("front matter never closes")
	}
	if every != "" {
		interval, err := ParseEvery(every)
		if err != nil {
			return invalid("%v", err)
		}
		note.Every = interval
	}
	for identifier := range strings.SplitSeq(supersedes, ",") {
		if identifier = strings.TrimSpace(identifier); identifier != "" {
			note.Supersedes = append(note.Supersedes, identifier)
		}
	}
	var body []string
	for scanner.Scan() {
		body = append(body, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return Note{}, fmt.Errorf("read %s: %w", path, err)
	}
	note.Body = strings.TrimSpace(strings.Join(body, "\n"))
	if owner := filepath.Base(filepath.Dir(path)); owner != note.Owner {
		return invalid("owner %q but filed under %q", note.Owner, owner)
	}
	if reason := problem(note); reason != "" {
		return invalid("%s", reason)
	}
	return note, nil
}

func ParseEvery(value string) (time.Duration, error) {
	match := everyShape.FindStringSubmatch(value)
	if match == nil {
		return 0, fmt.Errorf("every %q is not minutes, hours or days above 0, such as 5m, 2h or 7d", value)
	}
	count, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	unit := map[string]time.Duration{"": Day, "d": Day, "h": time.Hour, "m": time.Minute}[match[2]]
	return time.Duration(count) * unit, nil
}

// ReadFor reads the owner's notes and the everywhere notes; an empty owner reads only the latter.
func ReadFor(directory string, owner string) ([]Note, int, error) {
	var read []Note
	invalid := 0
	for _, folder := range []string{owner, OwnerEverywhere} {
		if folder == "" {
			continue
		}
		owned, count, err := ReadOwner(directory, folder)
		if err != nil {
			return nil, 0, err
		}
		read, invalid = append(read, owned...), invalid+count
	}
	return read, invalid, nil
}

// ReadOwner counts the .md files Parse rejects; any other entry is skipped uncounted.
func ReadOwner(directory string, owner string) ([]Note, int, error) {
	folder := filepath.Join(directory, owner)
	entries, err := os.ReadDir(folder)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var read []Note
	invalid := 0
	for _, entry := range entries {
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		path := filepath.Join(folder, entry.Name())
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, 0, readErr
		}
		note, parseErr := Parse(path, content)
		switch {
		case errors.Is(parseErr, ErrInvalid):
			invalid++
		case parseErr != nil:
			return nil, 0, parseErr
		default:
			read = append(read, note)
		}
	}
	return read, invalid, nil
}

func SetDue(directory string, note Note, due string) error {
	if !ValidDue(due) {
		return fmt.Errorf("%w %s: due %q is not YYYY-MM-DD or YYYY-MM-DDTHH:MM", ErrInvalid, note.ID, due)
	}
	return setKey(directory, note, "due", due)
}

// ValidDue checks the width too: Parse takes a one-digit hour, which compares wrongly as a string.
func ValidDue(due string) bool {
	layout := map[int]string{len(time.DateOnly): time.DateOnly, len(DueMinute): DueMinute}[len(due)]
	_, err := time.Parse(layout, due)
	return layout != "" && err == nil
}

func ValidProject(project string) bool {
	return projectShape.MatchString(project)
}

func ValidSession(session string) bool {
	return sessionShape.MatchString(session)
}

func Validate(note Note) error {
	reason := problem(note)
	switch {
	case reason == "":
		return nil
	case note.ID == "":
		return fmt.Errorf("%w: %s", ErrInvalid, reason)
	}
	return fmt.Errorf("%w %s: %s", ErrInvalid, note.ID, reason)
}

// Write files a valid note under directory/<owner>/<id>.md, refusing to overwrite an existing note.
func Write(directory string, note Note) (string, error) {
	if err := Validate(note); err != nil {
		return "", err
	}
	folder := filepath.Join(directory, note.Owner)
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return "", fmt.Errorf("create %s: %w", folder, err)
	}
	path := filepath.Join(folder, note.ID+".md")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	if _, err = file.Write(Format(note)); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	if err = file.Close(); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

func clearKey(directory string, note Note, key string) error {
	return rewriteKey(directory, note, key, "clear", func(lines []string, _ int, keyIndex int) []string {
		if keyIndex < 0 {
			return nil
		}
		return slices.Concat(lines[:keyIndex], lines[keyIndex+1:])
	})
}

func problem(note Note) string {
	if _, err := time.Parse(time.DateOnly, note.Date); err != nil {
		return fmt.Sprintf("date %q is not YYYY-MM-DD", note.Date)
	}
	switch {
	case note.Body == "":
		return "empty body"
	case note.ID == "":
		return "no id"
	case !slices.Contains(Kinds, note.Kind):
		return fmt.Sprintf("kind %q is not one of %s", note.Kind, strings.Join(Kinds, ", "))
	case note.Owner == "":
		return "no owner"
	case !slices.Contains(scopes, note.Scope):
		return fmt.Sprintf("scope %q is not one of %s", note.Scope, strings.Join(scopes, ", "))
	case note.Scope == ScopeProject && note.Project == "":
		return "project scope with no project"
	case note.Scope == ScopeEverywhere && note.Owner != OwnerEverywhere:
		return fmt.Sprintf("everywhere scope but owner %q, not %s", note.Owner, OwnerEverywhere)
	case note.Owner == OwnerEverywhere && note.Scope != ScopeEverywhere:
		return fmt.Sprintf("owner %s with scope %q, not %s", OwnerEverywhere, note.Scope, ScopeEverywhere)
	case note.Kind == KindFlag && note.Scope != ScopeProject:
		return fmt.Sprintf("flag with %s scope", note.Scope)
	case note.Kind == KindFlag && (note.Session == "" || note.Time == ""):
		return "flag with no session or time"
	case note.Kind == KindFlag && !slices.Contains(Verdicts, note.Verdict):
		return fmt.Sprintf("flag verdict %q is not one of %s", note.Verdict, strings.Join(Verdicts, ", "))
	case note.Kind != KindFlag && note.Verdict != "":
		return fmt.Sprintf("%s with a verdict, which only a flag takes", note.Kind)
	case note.Kind == KindHandoff && note.Scope != ScopeProject:
		return fmt.Sprintf("handoff with %s scope", note.Scope)
	case note.Kind == KindPlan && note.Scope != ScopeProject:
		return fmt.Sprintf("plan with %s scope", note.Scope)
	case note.Kind == KindPlan && (strings.Contains(note.Body, "\n") || !filepath.IsLocal(filepath.FromSlash(note.Body))):
		return fmt.Sprintf("plan body %q is not one path inside the project", note.Body)
	case note.Kind == KindReminder && note.Due == "":
		return "reminder with no due date"
	case note.Kind != KindReminder && (note.Due != "" || note.Closed != "" || note.DoneFile != "" || note.Every != 0):
		return fmt.Sprintf("%s with a closed, done-file, due or every key, which only a reminder takes", note.Kind)
	case note.Every < 0 || note.Every%time.Minute != 0:
		return fmt.Sprintf("every %s is not a whole number of minutes above 0", note.Every)
	case note.DoneFile != "" && note.Every == 0:
		return "done-file with no every"
	case !projectShape.MatchString(note.DoneFile):
		return fmt.Sprintf("done-file %q holds a control character", note.DoneFile)
	case note.DoneFile != "" && !filepath.IsAbs(note.DoneFile):
		return fmt.Sprintf("done-file %q is not an absolute path", note.DoneFile)
	case note.Due != "" && !ValidDue(note.Due):
		return fmt.Sprintf("due %q is not YYYY-MM-DD or YYYY-MM-DDTHH:MM", note.Due)
	case note.Closed != "" && !validTime(note.Closed):
		return fmt.Sprintf("closed %q is not an RFC 3339 timestamp", note.Closed)
	case !ValidProject(note.Project):
		return fmt.Sprintf("project %q holds a control character", note.Project)
	case !ValidSession(note.Session):
		return fmt.Sprintf("session %q holds characters other than letters, digits, - and _", note.Session)
	case slices.Contains(note.Supersedes, note.ID):
		return "supersedes itself"
	case repeated(note.Supersedes) != "":
		return fmt.Sprintf("supersedes names %q twice", repeated(note.Supersedes))
	case note.Shown != "" && !validTime(note.Shown):
		return fmt.Sprintf("shown %q is not an RFC 3339 timestamp", note.Shown)
	case note.Time != "" && !validTime(note.Time):
		return fmt.Sprintf("time %q is not an RFC 3339 timestamp", note.Time)
	}
	return ""
}

func repeated(identifiers []string) string {
	for at, identifier := range identifiers {
		if slices.Contains(identifiers[:at], identifier) {
			return identifier
		}
	}
	return ""
}

// change gets the index of the closing --- and of the key's line (-1 if absent); returning nil leaves the file alone.
func rewriteKey(directory string, note Note, key string, action string, change func(lines []string, closing int, keyIndex int) []string) error {
	path := filepath.Join(directory, note.Owner, note.ID+".md")
	failed := func(err error) error { return fmt.Errorf("%s %s %s: %w", action, path, key, err) }
	info, err := os.Stat(path)
	if err != nil {
		return failed(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return failed(err)
	}
	lines := strings.SplitAfter(string(content), "\n")
	closing := -1
	for number, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			closing = number + 1
			break
		}
	}
	if strings.TrimSpace(strings.TrimPrefix(lines[0], "\xef\xbb\xbf")) != "---" || closing < 0 {
		return fmt.Errorf("%w %s: no front matter to %s %s", ErrInvalid, path, action, key)
	}
	keyIndex := -1
	for number := 1; number < closing; number++ {
		if name, _, _ := strings.Cut(lines[number], ":"); strings.TrimSpace(name) == key {
			keyIndex = number
			break
		}
	}
	rewritten := change(lines, closing, keyIndex)
	if rewritten == nil {
		return nil
	}
	// Renamed into place, so a session reading the note sees it whole; named uniquely, as two starts may rewrite it at once.
	temporary, err := os.CreateTemp(filepath.Dir(path), note.ID+".md.*.tmp")
	if err != nil {
		return failed(err)
	}
	_, err = temporary.WriteString(strings.Join(rewritten, ""))
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Chmod(temporary.Name(), info.Mode().Perm())
	}
	if err == nil {
		err = os.Rename(temporary.Name(), path)
	}
	if err != nil {
		_ = os.Remove(temporary.Name())
		return failed(err)
	}
	return nil
}

// setKey sets one key and leaves every other line as written, so a hand edit survives.
func setKey(directory string, note Note, key string, value string) error {
	line := key + ": " + value + "\n"
	return rewriteKey(directory, note, key, "set", func(lines []string, closing int, keyIndex int) []string {
		if keyIndex < 0 {
			return slices.Concat(lines[:closing], []string{line}, lines[closing:])
		}
		return slices.Concat(lines[:keyIndex], []string{line}, lines[keyIndex+1:])
	})
}

func validTime(value string) bool {
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}
