// Copyright © 2021 Mikael Berthe <mikael@lilotux.net>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type sqlHandler struct {
	db *sql.DB
}

func dbOpen(dbPath string) (*sqlHandler, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}
	return &sqlHandler{db}, nil
}

func (s *sqlHandler) Close() {
	s.db.Close()
	s.db = nil
}

func dbCreate(dbPath string) error {
	h, err := dbOpen(dbPath)
	if err != nil {
		return err
	}
	defer h.Close()
	sqlStmt := "CREATE TABLE cache (id TEXT NOT NULL PRIMARY KEY, value TEXT, timestamp INTEGER);"
	if _, err := h.db.Exec(sqlStmt); err != nil {
		sqlStmt = "DELETE FROM cache;"
		if _, err := h.db.Exec(sqlStmt); err != nil {
			return fmt.Errorf("cannot create/delete DB table: %w", err)
		}
	}
	return nil
}

func (s *sqlHandler) GetItem(item string) (string, int64, error) {
	if s == nil {
		return "", 0, fmt.Errorf("no DB file")
	}
	stmt, err := s.db.Prepare("SELECT value,timestamp FROM cache WHERE id = ?")
	if err != nil {
		return "", 0, fmt.Errorf("cannot prepare query: %w", err)
	}
	defer stmt.Close()
	var value string
	var ts int64
	err = stmt.QueryRow(item).Scan(&value, &ts)
	if err != nil {
		return "", ts, fmt.Errorf("cannot get value: %w", err)
	}
	return value, ts, nil
}

func (s *sqlHandler) SetItem(item, value string) error {
	if s == nil {
		return fmt.Errorf("no DB file")
	}

	var ts int64 = time.Now().Unix()

	stmt, err := s.db.Prepare("INSERT INTO cache(id,value,timestamp) VALUES(?,?,?)")
	if err != nil {
		return fmt.Errorf("cannot prepare INSERT query: %w", err)
	}
	_, err = stmt.Exec(item, value, ts) // Try to insert
	stmt.Close()
	if err == nil {
		return nil
	}

	if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return fmt.Errorf("cannot insert item: %w", err)
	}

	// The item is already present; let's update it
	stmt, err = s.db.Prepare("UPDATE cache SET (value,timestamp) = (?,?) WHERE id = ?")
	if err != nil {
		return fmt.Errorf("cannot prepare UPDATE query: %w", err)
	}
	defer stmt.Close()
	if result, err := stmt.Exec(value, ts, item); err == nil {
		if n, _ := result.RowsAffected(); n == 0 {
			return fmt.Errorf("cannot update value (no row affected)")
		}
	} else {
		return fmt.Errorf("cannot update value: %w", err)
	}
	return nil
}

func (s *sqlHandler) Clean(ts int64) error {
	if s == nil {
		return fmt.Errorf("no DB file")
	}
	_, err := s.db.Exec(fmt.Sprintf("DELETE FROM cache WHERE timestamp < %d", ts))
	return err
}
