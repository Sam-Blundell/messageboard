package storage

import (
	"database/sql"
	"errors"
	"slices"
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

// appliedNames returns the ledger's recorded migration names in applied order.
func appliedNames(t *testing.T, conn *sql.DB) []string {
	t.Helper()
	rows, err := conn.Query("SELECT name FROM migration ORDER BY rowid")
	if err != nil {
		t.Fatalf("querying migration ledger: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scanning ledger row: %v", err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading ledger: %v", err)
	}
	return names
}

// A virgin database — no ledger table at all — is not an error: every
// migration in the list is pending, in order.
func TestPendingReportsEverythingOnVirginDatabase(t *testing.T) {
	conn := newTestDB(t)
	migrations := []Migration{
		{name: "create a", stmt: "CREATE TABLE a (id INTEGER)"},
		{name: "create b", stmt: "CREATE TABLE b (id INTEGER)"},
	}

	pending, err := Pending(conn, migrations)
	if err != nil {
		t.Fatalf("Pending on a virgin database: %v", err)
	}
	if !slices.Equal(pending, migrations) {
		t.Errorf("got %v, want the full list pending", pending)
	}
}

// Pending runs as the guard before every command, so it must never write. A
// virgin database is the scenario that tempts it: creating the ledger there
// would smuggle DDL into every read-only command.
func TestPendingIsReadOnly(t *testing.T) {
	conn := newTestDB(t)
	migrations := []Migration{
		{name: "create a", stmt: "CREATE TABLE a (id INTEGER)"},
	}

	if _, err := Pending(conn, migrations); err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if tableExists(t, conn, "migration") {
		t.Error("Pending created the ledger table")
	}
}

// Migrate records what it applies: after a run the ledger holds every migration
// name in list order, and Pending reports nothing left to do.
func TestMigrateRecordsHistory(t *testing.T) {
	conn := newTestDB(t)
	migrations := []Migration{
		{name: "create a", stmt: "CREATE TABLE a (id INTEGER)"},
		{name: "create b", stmt: "CREATE TABLE b (id INTEGER)"},
	}

	if err := Migrate(conn, migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	got := appliedNames(t, conn)
	want := []string{"create a", "create b"}
	if !slices.Equal(got, want) {
		t.Errorf("ledger: got %q, want %q", got, want)
	}

	pending, err := Pending(conn, migrations)
	if err != nil {
		t.Fatalf("Pending after Migrate: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("got %d pending after Migrate, want 0", len(pending))
	}
}

// These migrations are deliberately NOT idempotent (no IF NOT EXISTS), so
// re-executing one fails loudly. A rerun succeeding therefore proves the runner
// skips applied migrations because they're recorded — not because re-running
// happened to be harmless.
func TestMigrateSkipsAppliedMigrations(t *testing.T) {
	conn := newTestDB(t)
	base := []Migration{
		{name: "create a", stmt: "CREATE TABLE a (id INTEGER)"},
	}
	extended := []Migration{
		base[0],
		{name: "create b", stmt: "CREATE TABLE b (id INTEGER)"},
	}

	if err := Migrate(conn, base); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(conn, extended); err != nil {
		t.Fatalf("Migrate with the extended list should apply only the new tail, got: %v", err)
	}
	if !tableExists(t, conn, "b") {
		t.Error("the new migration was not applied")
	}
	if err := Migrate(conn, extended); err != nil {
		t.Errorf("rerun with everything applied should be a no-op, got: %v", err)
	}
}

// A failed migration must leave no trace of itself: the error names it, and it
// is not recorded — so a fixed binary can re-attempt it. Asserting that the
// earlier success IS recorded stops this test passing vacuously on an empty
// ledger.
func TestFailedMigrationIsNotRecorded(t *testing.T) {
	conn := newTestDB(t)
	migrations := []Migration{
		{name: "good", stmt: "CREATE TABLE good_table (id INTEGER)"},
		{name: "broken", stmt: "CREATE TABLE {'}"},
	}

	err := Migrate(conn, migrations)
	if err == nil {
		t.Fatal("expected the broken migration to error")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Errorf("error %q should name the broken migration", err)
	}

	got := appliedNames(t, conn)
	want := []string{"good"}
	if !slices.Equal(got, want) {
		t.Errorf("ledger: got %q, want %q (good recorded, broken absent)", got, want)
	}
}

// applyMigration is atomic: if recording fails, the applied statement is rolled
// back too — SQLite DDL participates in transactions, so even the CREATE TABLE
// un-happens. The recording failure is forced by pre-inserting the migration's
// name, so the ledger INSERT hits the primary-key constraint.
func TestApplyMigrationIsAtomic(t *testing.T) {
	conn := newTestDB(t)
	if err := Migrate(conn, nil); err != nil { // empty list: bootstraps the ledger only
		t.Fatalf("bootstrapping ledger: %v", err)
	}
	if _, err := conn.Exec("INSERT INTO migration (name, applied_at) VALUES ('sneaky', 0)"); err != nil {
		t.Fatalf("pre-recording name: %v", err)
	}

	err := applyMigration(conn, Migration{name: "sneaky", stmt: "CREATE TABLE sneaked (id INTEGER)"})
	if err == nil {
		t.Fatal("expected recording to fail on the duplicate name")
	}
	// The failure must come from the recording step: if it happened any earlier
	// the statement never ran, and the rollback assertion below proves nothing.
	if !strings.Contains(err.Error(), "recording") {
		t.Fatalf("got %v, want a recording-step failure", err)
	}
	if tableExists(t, conn, "sneaked") {
		t.Error("the statement's effect survived a failed recording — transaction is not atomic")
	}
}

// A recorded name that doesn't match the list at the same position means the
// database's history diverged from this binary's (wrong binary, wrong database,
// or an edited list). Pending must refuse, naming both sides.
func TestPendingRejectsDivergentHistory(t *testing.T) {
	conn := newTestDB(t)
	if err := Migrate(conn, []Migration{{name: "theirs", stmt: "CREATE TABLE x (id INTEGER)"}}); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	divergent := []Migration{{name: "ours", stmt: "CREATE TABLE y (id INTEGER)"}}
	_, err := Pending(conn, divergent)
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("got %v, want a history-mismatch error", err)
	}
	if !strings.Contains(err.Error(), "theirs") || !strings.Contains(err.Error(), "ours") {
		t.Errorf("error %q should name both the recorded and the expected migration", err)
	}
}

// A ledger with more entries than the binary's list was written by a newer
// binary; Pending must refuse rather than treat unknown history as fine.
func TestPendingRejectsNewerDatabase(t *testing.T) {
	conn := newTestDB(t)
	both := []Migration{
		{name: "first", stmt: "CREATE TABLE x (id INTEGER)"},
		{name: "second", stmt: "CREATE TABLE y (id INTEGER)"},
	}
	if err := Migrate(conn, both); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	_, err := Pending(conn, both[:1])
	if err == nil || !strings.Contains(err.Error(), "don't exist in the binary") {
		t.Errorf("got %v, want a database-newer-than-binary error", err)
	}
}

// A database that predates the ledger — schema present, no history — must be
// adopted by a single Migrate: the shipped creates are IF NOT EXISTS, so they
// no-op and everything gets recorded.
func TestMigrateAdoptsPreLedgerDatabase(t *testing.T) {
	conn := newTestDB(t)
	for _, m := range Migrations {
		if _, err := conn.Exec(m.stmt); err != nil {
			t.Fatalf("seeding old-style schema: %v", err)
		}
	}

	if err := Migrate(conn, Migrations); err != nil {
		t.Fatalf("Migrate on a pre-ledger database: %v", err)
	}
	want := make([]string, len(Migrations))
	for i, m := range Migrations {
		want[i] = m.name
	}
	if got := appliedNames(t, conn); !slices.Equal(got, want) {
		t.Errorf("ledger: got %q, want %q", got, want)
	}
}
