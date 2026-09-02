package db

import (
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
)

type DB struct {
  conn *sql.DB
}

func New(path string) (*DB, error) {
  conn, err := sql.Open("sqlite3", path)
  if err != nil {
    return nil, err
  }
  return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
  return db.conn.Close()
}
