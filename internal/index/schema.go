package index

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// SchemaProblems names where the index's tables differed from this build's when it was opened; Reset clears them.
func (x *Index) SchemaProblems() []string {
	return x.problems
}

func columnList(columns []string) string {
	if len(columns) == 1 {
		return "column " + columns[0]
	}
	return "columns " + strings.Join(columns, ", ")
}

// No tables: a new index. Settings skipped: Reset keeps them, so a difference there would rebuild every sync.
func schemaProblems(ctx context.Context, db *sql.DB) ([]string, error) {
	have, err := tableColumns(ctx, db)
	if err != nil || len(have) == 0 {
		return nil, err
	}
	expected, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	defer expected.Close()
	// Each connection to :memory: is its own database.
	expected.SetMaxOpenConns(1)
	if _, err = expected.ExecContext(ctx, schema); err != nil {
		return nil, fmt.Errorf("build the expected schema: %w", err)
	}
	want, err := tableColumns(ctx, expected)
	if err != nil {
		return nil, err
	}
	// A constraint change keeps the column names; the stored CREATE statement shows it.
	haveSQL, err := tableDefinitions(ctx, db)
	if err != nil {
		return nil, err
	}
	wantSQL, err := tableDefinitions(ctx, expected)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, table := range slices.Sorted(maps.Keys(want)) {
		if table == "settings" {
			continue
		}
		columns, present := have[table]
		if !present {
			problems = append(problems, table+" is missing")
			continue
		}
		var missing, unexpected []string
		for _, column := range want[table] {
			if !slices.Contains(columns, column) {
				missing = append(missing, column)
			}
		}
		for _, column := range columns {
			if !slices.Contains(want[table], column) {
				unexpected = append(unexpected, column)
			}
		}
		if len(missing) > 0 {
			problems = append(problems, table+" lacks "+columnList(missing))
		}
		if len(unexpected) > 0 {
			problems = append(problems, table+" has "+columnList(unexpected)+" that this version does not")
		}
		if len(missing)+len(unexpected) == 0 && haveSQL[table] != wantSQL[table] {
			problems = append(problems, table+" has a different definition")
		}
	}
	return problems, nil
}

func tableColumns(ctx context.Context, db *sql.DB) (map[string][]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT m.name, p.name FROM sqlite_master m JOIN pragma_table_info(m.name) p
		WHERE m.type = 'table' AND m.name NOT LIKE 'sqlite_%' ORDER BY m.name, p.cid`)
	if err != nil {
		return nil, fmt.Errorf("read tables: %w", err)
	}
	defer rows.Close()
	tables := map[string][]string{}
	for rows.Next() {
		var table, column string
		if err = rows.Scan(&table, &column); err != nil {
			return nil, err
		}
		tables[table] = append(tables[table], column)
	}
	return tables, rows.Err()
}

func tableDefinitions(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT name, sql FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("read table definitions: %w", err)
	}
	defer rows.Close()
	definitions := map[string]string{}
	for rows.Next() {
		var name, definition string
		if err = rows.Scan(&name, &definition); err != nil {
			return nil, err
		}
		definitions[name] = definition
	}
	return definitions, rows.Err()
}
