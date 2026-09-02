package database

import "testing"

func TestSplitMigrationStatements(t *testing.T) {
	statements := splitMigrationStatements("CREATE TABLE users (id CHAR(36));\n\nALTER TABLE users ADD COLUMN status VARCHAR(20);")
	if len(statements) != 2 {
		t.Fatalf("expected two statements, got %d", len(statements))
	}
	if statements[0] != "CREATE TABLE users (id CHAR(36))" {
		t.Fatalf("unexpected first statement: %q", statements[0])
	}
}
