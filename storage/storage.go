package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Migration struct {
	name string
	stmt string
}

// Migrations is the application's schema history, applied in order by Migrate
// and recorded by name in the migration ledger table.
//
// The list is append-only. A migration that has shipped must never be edited,
// renamed, or reordered — its name is its identity in every existing database,
// and each database's applied history must remain a prefix of this list. To fix
// a past migration, append a new one that corrects it.
var Migrations = []Migration{
	{name: "create board table", stmt: "CREATE TABLE IF NOT EXISTS board (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE CHECK (name <> '' AND length(name) <= 24))"},
	{name: "create thread table", stmt: "CREATE TABLE IF NOT EXISTS thread (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL CHECK (title <> ''), board_id INTEGER NOT NULL REFERENCES board(id) ON DELETE CASCADE, created_at INTEGER NOT NULL, bumped_at INTEGER NOT NULL)"},
	{name: "create post table", stmt: "CREATE TABLE IF NOT EXISTS post (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, body TEXT NOT NULL CHECK (body <> ''), thread_id INTEGER NOT NULL REFERENCES thread(id) ON DELETE CASCADE, created_at INTEGER NOT NULL)"},
}

func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return db, nil
}

// Pending reports which of the given migrations have not yet been applied to
// this database. It errors if the database's history has diverged from the
// list, or records migrations the list doesn't contain (a newer binary's work).
//
// Pending never writes — it is safe as a startup guard. A database with no
// migration history simply has every migration pending.
func Pending(conn *sql.DB, migrations []Migration) ([]Migration, error) {
	var migrationTable string

	err := conn.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", "migration",
	).Scan(&migrationTable)
	if errors.Is(err, sql.ErrNoRows) {
		return migrations, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying sqlite_master for migration table: %w", err)
	}

	rows, err := conn.Query("SELECT name FROM migration ORDER BY rowid")
	if err != nil {
		return nil, fmt.Errorf("error accessing migrations table: %w", err)
	}
	defer rows.Close()

	var dbMigrations []string
	for rows.Next() {
		rowName := ""
		err := rows.Scan(&rowName)
		if err != nil {
			return nil, fmt.Errorf("error reading migration table: %w", err)
		}
		dbMigrations = append(dbMigrations, rowName)
	}
	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error reading migration table: %w", err)
	}

	if len(dbMigrations) > len(migrations) {
		return nil, errors.New("there are migrations in the DB that don't exist in the binary")
	}

	for i, name := range dbMigrations {
		if name != migrations[i].name {
			return nil, fmt.Errorf(
				"migration history mismatch at %d: database has %q, binary has %q",
				i,
				name,
				migrations[i].name,
			)
		}
	}

	return migrations[len(dbMigrations):], nil
}

// applyMigration applies and records one migration atomically: both or neither.
func applyMigration(conn *sql.DB, migration Migration) error {
	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction for %s: %w", migration.name, err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(migration.stmt)
	if err != nil {
		return fmt.Errorf("running %s migration: %w", migration.name, err)
	}

	_, err = tx.Exec(
		"INSERT INTO migration (name, applied_at) VALUES (?, ?)",
		migration.name,
		time.Now().UTC().Unix(),
	)
	if err != nil {
		return fmt.Errorf("recording %s migration: %w", migration.name, err)
	}

	return tx.Commit()
}

// Migrate brings the database up to date, applying unapplied migrations in
// list order. A failure stops the run with an error naming the migration;
// migrations applied before the failure stay applied, so a rerun resumes
// rather than restarts.
func Migrate(conn *sql.DB, migrations []Migration) error {
	createMigrationsTable := "CREATE TABLE IF NOT EXISTS migration (name TEXT NOT NULL PRIMARY KEY, applied_at INTEGER NOT NULL)"

	_, err := conn.Exec(createMigrationsTable)
	if err != nil {
		return fmt.Errorf("error creating migrations table: %w", err)
	}

	toApply, err := Pending(conn, migrations)
	if err != nil {
		return err
	}
	for _, m := range toApply {
		err = applyMigration(conn, m)
		if err != nil {
			return err
		}
	}

	return nil
}
