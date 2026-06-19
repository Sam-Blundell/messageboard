package storage

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Migration struct {
	name string
	stmt string
}

var Migrations = []Migration{
	{name: "create post table", stmt: "CREATE TABLE IF NOT EXISTS post (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, body TEXT, created_at INTEGER)"},
}

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return db, nil
}

func Migrate(conn *sql.DB, migrations []Migration) error {
	for _, m := range migrations {
		_, err := conn.Exec(m.stmt)
		if err != nil {
			return fmt.Errorf("error running %s migration: %w", m.name, err)
		}
	}
	return nil
}
