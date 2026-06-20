package storage

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	return conn
}

// tableExists reports whether a table of the given name is present, by asking
// SQLite's schema catalogue.
func tableExists(t *testing.T, conn *sql.DB, name string) bool {
	t.Helper()
	var found string
	err := conn.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", name,
	).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatalf("querying sqlite_master for %q: %v", name, err)
	}
	return true
}

// Happy path: every migration in the list is applied, so each table exists
// afterward.
func TestMigrateCreatesSchema(t *testing.T) {
	conn := newTestDB(t)
	migrations := []Migration{
		{name: "create a", stmt: "CREATE TABLE a (id INTEGER)"},
		{name: "create b", stmt: "CREATE TABLE b (id INTEGER)"},
	}

	if err := Migrate(conn, migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, name := range []string{"a", "b"} {
		if !tableExists(t, conn, name) {
			t.Errorf("table %q was not created", name)
		}
	}
}

// The app runs migrations on every startup, so the real Migrations must be safe
// to apply repeatedly (they use CREATE TABLE IF NOT EXISTS). Running the shipped
// list also smoke-tests that it's valid SQL.
func TestMigrateIsIdempotent(t *testing.T) {
	conn := newTestDB(t)

	if err := Migrate(conn, Migrations); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(conn, Migrations); err != nil {
		t.Fatalf("second Migrate should be a no-op, got: %v", err)
	}
}

// A failing migration must surface its name in the error, so a broken migration
// is identifiable. The valid migration runs first, so the error can only name
// "broken" — which also proves migrations run in order.
func TestMigrateNamesFailure(t *testing.T) {
	conn := newTestDB(t)
	bad := []Migration{
		{name: "valid", stmt: "CREATE TABLE IF NOT EXISTS valid (id INTEGER)"},
		{name: "broken", stmt: "CREATE TABLE {'}"},
	}

	err := Migrate(conn, bad)
	if err == nil {
		t.Fatal("expected an error from the broken migration, got nil")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Errorf("error %q should name the failing migration %q", err, "broken")
	}
}

// Migrate must stop at the first failure — migrations after a broken one must
// not run, so the schema can't be left half-applied past the error.
func TestMigrateStopsAtFailure(t *testing.T) {
	conn := newTestDB(t)
	migrations := []Migration{
		{name: "broken", stmt: "CREATE TABLE {'}"},
		{name: "never runs", stmt: "CREATE TABLE later (id INTEGER)"},
	}

	if err := Migrate(conn, migrations); err == nil {
		t.Fatal("expected an error from the broken migration, got nil")
	}
	if tableExists(t, conn, "later") {
		t.Error("a migration after the failure should not have run")
	}
}
