package index

import (
	"context"
	"fmt"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

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
	return events, total, rows.Err()
}
