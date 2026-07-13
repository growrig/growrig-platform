package store

import (
	"fmt"
	"strings"
)

// DatabaseTable describes an application table, its current row count, and
// allocated SQLite pages. SizeBytes includes indexes owned by the table.
// Table names come exclusively from SQLite's own schema metadata before being
// quoted, so the dynamic COUNT query cannot contain user-provided SQL.
type DatabaseTable struct {
	Name      string `json:"name"`
	Rows      int64  `json:"rows"`
	SizeBytes int64  `json:"sizeBytes"`
}

func (s *Store) DatabaseTables() ([]DatabaseTable, error) {
	rows, err := s.db.Query(`SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	sizes := map[string]int64{}
	sizeRows, err := s.db.Query(`
		SELECT schema_object.tbl_name, COALESCE(SUM(dbstat.pgsize), 0)
		FROM dbstat
		JOIN sqlite_schema AS schema_object ON schema_object.name = dbstat.name
		WHERE schema_object.type IN ('table', 'index')
		  AND schema_object.tbl_name NOT LIKE 'sqlite_%'
		GROUP BY schema_object.tbl_name`)
	if err != nil {
		return nil, fmt.Errorf("read database page sizes: %w", err)
	}
	for sizeRows.Next() {
		var name string
		var size int64
		if err := sizeRows.Scan(&name, &size); err != nil {
			sizeRows.Close()
			return nil, err
		}
		sizes[name] = size
	}
	if err := sizeRows.Close(); err != nil {
		return nil, err
	}

	out := make([]DatabaseTable, 0, len(names))
	for _, name := range names {
		quoted := `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
		var count int64
		if err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", quoted)).Scan(&count); err != nil {
			return nil, fmt.Errorf("count %s: %w", name, err)
		}
		out = append(out, DatabaseTable{Name: name, Rows: count, SizeBytes: sizes[name]})
	}
	return out, nil
}
