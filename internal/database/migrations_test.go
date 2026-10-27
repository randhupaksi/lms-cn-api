package database

import "testing"

func TestSchoolRoleConstraintDialect(t *testing.T) {
	statement := "ALTER TABLE users DROP CHECK chk_users_role"
	if migrationForEngine(statement, "8.0.40") != statement {
		t.Fatal("MySQL statement changed")
	}
	if migrationForEngine(statement, "10.4.32-MariaDB") != "ALTER TABLE users DROP CONSTRAINT chk_users_role" {
		t.Fatal("MariaDB constraint syntax not adapted")
	}
}

func TestSplitMigrationStatements(t *testing.T) {
	statements := splitMigrationStatements("CREATE TABLE users (id CHAR(36));\n\nALTER TABLE users ADD COLUMN status VARCHAR(20);")
	if len(statements) != 2 {
		t.Fatalf("expected two statements, got %d", len(statements))
	}
	if statements[0] != "CREATE TABLE users (id CHAR(36))" {
		t.Fatalf("unexpected first statement: %q", statements[0])
	}
}
