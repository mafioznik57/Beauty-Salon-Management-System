package db

import (
	"database/sql"
	"errors"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct{ SQL *sql.DB }

var ErrUnsupportedDSN = errors.New("unsupported DSN: only postgres:// or postgresql:// URLs or libpq key=value DSNs are accepted")

func Open(dsn string) (*DB, error) {
	if !strings.HasPrefix(dsn, "postgres://") && !strings.HasPrefix(dsn, "postgresql://") {
		if !strings.Contains(dsn, "=") {
			return nil, ErrUnsupportedDSN
		}
	}
	s, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := s.Ping(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return &DB{SQL: s}, nil
}

func (d *DB) Close() error { return d.SQL.Close() }

func (d *DB) Migrate() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
                id SERIAL PRIMARY KEY,
                email TEXT NOT NULL UNIQUE,
                password_hash TEXT NOT NULL,
                role TEXT NOT NULL CHECK(role IN ('admin','user')),
                created_at TIMESTAMP WITH TIME ZONE NOT NULL
            );`,
		`CREATE TABLE IF NOT EXISTS cells (
                id SERIAL PRIMARY KEY,
                name TEXT NOT NULL,
                lat DOUBLE PRECISION NOT NULL,
                lng DOUBLE PRECISION NOT NULL,
                h3_index TEXT NOT NULL,
                resolution INTEGER NOT NULL,
                created_by INTEGER NOT NULL REFERENCES users(id),
                created_at TIMESTAMP WITH TIME ZONE NOT NULL
            );`,
		`CREATE INDEX IF NOT EXISTS idx_cells_h3 ON cells(h3_index);`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
                id SERIAL PRIMARY KEY,
                actor_user_id INTEGER REFERENCES users(id),
                action TEXT NOT NULL,
                target TEXT NOT NULL,
                created_at TIMESTAMP WITH TIME ZONE NOT NULL
            );`,
		`CREATE TABLE IF NOT EXISTS cell_analytics (
                h3_parent TEXT NOT NULL,
                resolution INTEGER NOT NULL,
                cells_count INTEGER NOT NULL,
                updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
                PRIMARY KEY(h3_parent, resolution)
            );`,
		`CREATE TABLE IF NOT EXISTS sim_runs (
                id SERIAL PRIMARY KEY,
                requested_by INTEGER NOT NULL REFERENCES users(id),
                status TEXT NOT NULL,
                events_total INTEGER NOT NULL,
                events_processed INTEGER NOT NULL,
                created_at TIMESTAMP WITH TIME ZONE NOT NULL,
                updated_at TIMESTAMP WITH TIME ZONE NOT NULL
            );`,
	}
	for _, q := range schema {
		if _, err := d.SQL.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
