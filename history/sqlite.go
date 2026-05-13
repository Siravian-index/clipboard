package history

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

type SQLiteHistory struct {
	db      *sql.DB
	maxSize int
}

// NewSQLiteHistory opens (or creates) a SQLite database at dbPath and returns
// a History backed by it. The schema is created if it does not exist.
func NewSQLiteHistory(dbPath string, maxSize int) (*SQLiteHistory, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			content   TEXT    NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteHistory{db: db, maxSize: maxSize}, nil
}

func (h *SQLiteHistory) Add(entry string) {
	// Remove any existing occurrence so the entry moves to the top.
	if _, err := h.db.Exec(`DELETE FROM entries WHERE content = ?`, entry); err != nil {
		log.Printf("sqlite: failed to remove duplicate: %v", err)
		return
	}

	if _, err := h.db.Exec(`INSERT INTO entries (content) VALUES (?)`, entry); err != nil {
		log.Printf("sqlite: failed to insert entry: %v", err)
		return
	}

	// Trim oldest entries beyond maxSize.
	_, err := h.db.Exec(`
		DELETE FROM entries WHERE id IN (
			SELECT id FROM entries ORDER BY id DESC LIMIT -1 OFFSET ?
		)
	`, h.maxSize)
	if err != nil {
		log.Printf("sqlite: failed to trim entries: %v", err)
	}
}

func (h *SQLiteHistory) List() []string {
	rows, err := h.db.Query(`SELECT content FROM entries ORDER BY id DESC`)
	if err != nil {
		log.Printf("sqlite: failed to query entries: %v", err)
		return nil
	}
	defer rows.Close()

	var entries []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			continue
		}
		entries = append(entries, content)
	}
	return entries
}

func (h *SQLiteHistory) Clear() {
	if _, err := h.db.Exec(`DELETE FROM entries`); err != nil {
		log.Printf("sqlite: failed to clear entries: %v", err)
	}
}

func (h *SQLiteHistory) Close() error {
	return h.db.Close()
}
