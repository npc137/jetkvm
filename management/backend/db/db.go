// Package db manages the SQLite database connection and schema migrations.
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database at the given path and applies
// all pending schema migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
  entra_oid    TEXT PRIMARY KEY,
  email        TEXT NOT NULL,
  display_name TEXT NOT NULL,
  role         TEXT NOT NULL DEFAULT 'user',
  last_login_at DATETIME
);

CREATE TABLE IF NOT EXISTS devices (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  ip_address  TEXT NOT NULL,
  location    TEXT NOT NULL DEFAULT '',
  password    TEXT NOT NULL DEFAULT '',
  last_seen_at DATETIME,
  fw_version   TEXT NOT NULL DEFAULT '',
  status      TEXT NOT NULL DEFAULT 'unknown'
);

CREATE TABLE IF NOT EXISTS assignments (
  id          TEXT PRIMARY KEY,
  user_oid    TEXT NOT NULL REFERENCES users(entra_oid) ON DELETE CASCADE,
  device_id   TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  valid_from  DATETIME NOT NULL,
  valid_until DATETIME NOT NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for efficient active-assignment lookups.
CREATE INDEX IF NOT EXISTS idx_assignments_user    ON assignments(user_oid, valid_until);
CREATE INDEX IF NOT EXISTS idx_assignments_device  ON assignments(device_id, valid_until);

CREATE TABLE IF NOT EXISTS connection_tokens (
  token               TEXT PRIMARY KEY,
  user_oid            TEXT NOT NULL REFERENCES users(entra_oid) ON DELETE CASCADE,
  device_id           TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  device_ip           TEXT NOT NULL,
  jetkvm_auth_token   TEXT NOT NULL,
  expires_at          DATETIME NOT NULL,
  used_at             DATETIME
);

CREATE INDEX IF NOT EXISTS idx_tokens_expires ON connection_tokens(expires_at);
`
