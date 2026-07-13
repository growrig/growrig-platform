package store

import "testing"

func TestDatabaseTablesReturnsEveryApplicationTableWithCounts(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.db.Exec(`INSERT INTO settings(key,value) VALUES ('one','1'),('two','2')`); err != nil {
		t.Fatal(err)
	}
	tables, err := s.DatabaseTables()
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) == 0 {
		t.Fatal("expected database tables")
	}
	found := false
	for _, table := range tables {
		if table.Name == "settings" {
			found = true
			if table.Rows != 2 {
				t.Fatalf("settings rows = %d, want 2", table.Rows)
			}
			if table.SizeBytes <= 0 {
				t.Fatalf("settings size = %d, want allocated bytes", table.SizeBytes)
			}
		}
		if len(table.Name) >= 7 && table.Name[:7] == "sqlite_" {
			t.Fatalf("internal SQLite table returned: %s", table.Name)
		}
	}
	if !found {
		t.Fatal("settings table missing")
	}
}
