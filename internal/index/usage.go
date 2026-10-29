package index

import (
	"context"
	"fmt"
	"strings"

	"github.com/nicolaspearson/claude-ephant/internal/dto"
)

// Usage counts a session only where it has a prompt.
func (x *Index) Usage(ctx context.Context, query dto.UsageQuery) (dto.Usage, error) {
	var conditions []string
	var arguments []any
	if query.Owner != "" {
		conditions = append(conditions, "owner = ?")
		arguments = append(arguments, query.Owner)
	}
	if query.Project != "" {
		conditions = append(conditions, "project = ?")
		arguments = append(arguments, query.Project)
	}
	if query.Since != "" {
		conditions = append(conditions, "timestamp >= ?")
		arguments = append(arguments, query.Since)
	}
	whereKind := func(kind string) string {
		return " WHERE " + strings.Join(append([]string{kind}, conditions...), " AND ")
	}
	where := func(kind string) string {
		return whereKind("kind = '" + kind + "'")
	}

	var usage dto.Usage
	err := x.db.QueryRowContext(ctx, `SELECT count(*), count(DISTINCT session) FROM entries`+whereKind("kind IN "+promptKinds), arguments...).Scan(&usage.Prompts, &usage.Sessions)
	if err != nil {
		return dto.Usage{}, fmt.Errorf("count prompts: %w", err)
	}
	err = x.db.QueryRowContext(ctx, `SELECT count(*), coalesce(sum(json_extract(attributes, '$.characters')), 0) FROM events`+where("reply"), arguments...).Scan(&usage.Replies, &usage.ReplyCharacters)
	if err != nil {
		return dto.Usage{}, fmt.Errorf("count replies: %w", err)
	}
	rows, err := x.db.QueryContext(ctx, `SELECT subagent, sum(json_extract(attributes, '$.input')), sum(json_extract(attributes, '$.cacheWrite')), sum(json_extract(attributes, '$.cacheRead')), sum(json_extract(attributes, '$.output')) FROM events`+where("tokens")+` GROUP BY subagent`, arguments...)
	if err != nil {
		return dto.Usage{}, fmt.Errorf("count tokens: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var subagent bool
		var tokens dto.UsageTokens
		if err = rows.Scan(&subagent, &tokens.Input, &tokens.CacheWrite, &tokens.CacheRead, &tokens.Output); err != nil {
			return dto.Usage{}, fmt.Errorf("count tokens: %w", err)
		}
		if subagent {
			usage.SubagentTokens = tokens
		} else {
			usage.MainTokens = tokens
		}
	}
	if err = rows.Err(); err != nil {
		return dto.Usage{}, fmt.Errorf("count tokens: %w", err)
	}
	if usage.Tools, err = x.usageCounts(ctx, `SELECT substr(text, 1, instr(text, ':') - 1) AS name, sum(subagent = 0) AS main, sum(subagent = 1) AS subagents FROM entries`+where("tool")+` GROUP BY name`, arguments); err != nil {
		return dto.Usage{}, fmt.Errorf("count tool calls: %w", err)
	}
	if usage.Injections, err = x.usageCounts(ctx, `SELECT name, sum(iif(subagent = 0, json_extract(attributes, '$.bytes'), 0)) AS main, sum(iif(subagent = 1, json_extract(attributes, '$.bytes'), 0)) AS subagents FROM events`+where("injection")+` GROUP BY name`, arguments); err != nil {
		return dto.Usage{}, fmt.Errorf("count injections: %w", err)
	}
	return usage, nil
}

func (x *Index) usageCounts(ctx context.Context, selected string, arguments []any) ([]dto.UsageCount, error) {
	rows, err := x.db.QueryContext(ctx, selected+` ORDER BY main + subagents DESC, name`, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var counts []dto.UsageCount
	for rows.Next() {
		var count dto.UsageCount
		if err = rows.Scan(&count.Name, &count.Main, &count.Subagent); err != nil {
			return nil, err
		}
		counts = append(counts, count)
	}
	return counts, rows.Err()
}
