package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
	"github.com/nicolaspearson/claude-ephant/internal/transcript"
)

// A line key is the line's id, '#' and a place; '$' sorts just after '#', so the range holds that line's keys.
const callQuery = `SELECT id, text FROM entries WHERE line_key >= ? AND line_key < ? AND kind = '` + transcript.KindTool + `' ORDER BY line_key LIMIT 1`

// A skill's last calls, any one closing a run: a prompt before it is the run's own, such as a typed approval of fixes.
// checkpoint-log.sh was checkpoint's before ele checkpoint log.
var closingCalls = map[string][]string{
	"checkpoint":   {"checkpoint-log.sh", "ele checkpoint log"},
	"plan-audit":   {"ele ticket ledger record"},
	"ticket-audit": {"ele ticket ledger record"},
}

// A subagent's skill call ends no run: its events fall inside its parent's. Each strftime bound is a second loose,
// whatever a timestamp's precision, so the index can narrow to it; julianday then decides.
func duringCondition(during string) (string, []any) {
	skill := during[strings.LastIndex(during, ":")+1:]
	names := []any{skill, len(skill) + 1, ":" + skill}
	runEvent := func(alias string) string {
		return alias + ".session = events.session AND " + alias + ".subagent = 0 AND " + alias + ".kind IN ('" + transcript.EventCommand + "', '" + transcript.EventSkill + "')"
	}
	named := func(alias string) string {
		return "(" + alias + ".name = ? OR substr(" + alias + ".name, -?) = ?)"
	}
	closesRun, closesArguments := "1", []any(nil)
	if calls := closingCalls[skill]; len(calls) > 0 {
		// A later run's closing call is not this one's.
		closing := `SELECT 1 FROM entries closing WHERE closing.session = events.session AND closing.subagent = 0 AND closing.kind = '` + transcript.KindTool + `'
			AND closing.text LIKE 'Bash: %' AND (` + strings.Repeat("instr(closing.text, ?) > 0 OR ", len(calls)) + `0) AND closing.timestamp > opener.timestamp
			AND NOT EXISTS (SELECT 1 FROM events reopener WHERE ` + runEvent("reopener") + ` AND ` + named("reopener") + `
				AND reopener.timestamp > opener.timestamp AND reopener.timestamp < closing.timestamp)`
		closesRun = `(EXISTS (` + closing + ` AND closing.timestamp < prompt.timestamp)
			OR NOT EXISTS (` + closing + ` AND closing.timestamp <= strftime('%Y-%m-%dT%H:%M:%SZ', opener.timestamp, '+2 hours', '+1 second')
				AND julianday(closing.timestamp) <= julianday(opener.timestamp) + 2.0 / 24))`
		for _, call := range calls {
			closesArguments = append(closesArguments, call)
		}
		closesArguments = append(closesArguments, names...)
		closesArguments = append(closesArguments, closesArguments...)
	}
	condition := `EXISTS (SELECT 1 FROM events opener WHERE ` + runEvent("opener") + ` AND ` + named("opener") + `
		AND opener.timestamp <= events.timestamp AND opener.timestamp >= strftime('%Y-%m-%dT%H:%M:%SZ', events.timestamp, '-2 hours', '-1 second')
		AND julianday(events.timestamp) <= julianday(opener.timestamp) + 2.0 / 24
		AND NOT EXISTS (SELECT 1 FROM events closer WHERE ` + runEvent("closer") + ` AND NOT ` + named("closer") + `
			AND closer.timestamp > opener.timestamp AND closer.timestamp <= events.timestamp)
		AND NOT EXISTS (SELECT 1 FROM entries prompt WHERE prompt.session = events.session AND prompt.subagent = 0 AND prompt.kind = '` + transcript.KindPrompt + `'
			AND prompt.timestamp > opener.timestamp AND prompt.timestamp <= events.timestamp AND ` + closesRun + `))`
	return condition, append(append(append([]any{}, names...), names...), closesArguments...)
}

// Events returns the newest matches up to the limit, oldest first, and how many matched in all.
func (x *Index) Events(ctx context.Context, query dto.EventQuery) ([]dto.Event, int, error) {
	var conditions []string
	var arguments []any
	for _, filter := range []struct {
		column string
		value  string
	}{
		{"kind", query.Kind},
		{"name", query.Name},
		{"owner", query.Owner},
		{"project", query.Project},
		{"session", query.Session},
	} {
		if filter.value != "" {
			conditions = append(conditions, filter.column+" = ?")
			arguments = append(arguments, filter.value)
		}
	}
	if query.Since != "" {
		conditions = append(conditions, "timestamp >= ?")
		arguments = append(arguments, query.Since)
	}
	if query.During != "" {
		condition, duringArguments := duringCondition(query.During)
		conditions = append(conditions, condition)
		arguments = append(arguments, duringArguments...)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int
	if err := x.db.QueryRowContext(ctx, `SELECT count(*) FROM events`+where, arguments...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count events: %w", err)
	}
	selected := `SELECT attributes, id, kind, name, owner, profile, project, session, source, subagent, timestamp FROM events` + where
	if query.Limit > 0 {
		selected = `SELECT * FROM (` + selected + ` ORDER BY timestamp DESC, id DESC LIMIT ?)`
		arguments = append(arguments, query.Limit)
	}
	rows, err := x.db.QueryContext(ctx, `SELECT attributes, kind, name, owner, profile, project, session, source, subagent, timestamp FROM (`+selected+`) ORDER BY timestamp, id`, arguments...)
	if err != nil {
		return nil, 0, fmt.Errorf("read events: %w", err)
	}
	defer rows.Close()
	var events []dto.Event
	for rows.Next() {
		var event dto.Event
		var attributes string
		if err = rows.Scan(&attributes, &event.Kind, &event.Name, &event.Owner, &event.Profile, &event.Project, &event.Session, &event.Source, &event.Subagent, &event.Timestamp); err != nil {
			return nil, 0, err
		}
		if err = event.DecodeAttributes(attributes); err != nil {
			return nil, 0, fmt.Errorf("read event attributes: %w", err)
		}
		events = append(events, event)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	for _, event := range events {
		// Only a Bash call's text is its whole input; another tool's is its target, the same across changed calls.
		if event.Kind != transcript.EventToolError || event.Name != "Bash" {
			continue
		}
		callLine, _ := event.Attributes["callLine"].(string)
		retried, known, err := x.retried(ctx, callLine, event.Timestamp)
		if err != nil {
			return nil, 0, err
		}
		if known {
			event.Attributes["retried"] = retried
		}
	}
	return events, total, nil
}

// Unknown, the second result false, until a call follows the failure: a call made before it is a sibling from the same
// response, and a cut call text hides what changed past the cut.
func (x *Index) retried(ctx context.Context, callLine string, failed string) (bool, bool, error) {
	var id int64
	var call string
	err := x.db.QueryRowContext(ctx, callQuery, callLine+"#", callLine+"$").Scan(&id, &call)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("find the call of a tool error: %w", err)
	}
	var next string
	err = x.db.QueryRowContext(ctx, `SELECT text FROM entries WHERE source = (SELECT source FROM entries WHERE id = ?) AND kind = ? AND id > ? AND timestamp >= ? ORDER BY id LIMIT 1`, id, transcript.KindTool, id, failed).Scan(&next)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("find the call after a failed one: %w", err)
	}
	if strings.HasSuffix(call, "…") || strings.HasSuffix(next, "…") {
		return false, false, nil
	}
	return next == call, true, nil
}
