package history

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

type SQLiteHistory struct {
	db       *sql.DB
	imageDir string
	maxSize  int
	mu       sync.Mutex
}

// NewSQLiteHistory opens (or creates) a SQLite database at dbPath and returns
// a History backed by it. imageDir is where image files are stored.
func NewSQLiteHistory(dbPath, imageDir string, maxSize int) (*SQLiteHistory, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			content    TEXT    NOT NULL,
			type       TEXT    NOT NULL DEFAULT 'text',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Migrate existing databases that lack the type column.
	_, _ = db.Exec(`ALTER TABLE entries ADD COLUMN type TEXT NOT NULL DEFAULT 'text'`)

	return &SQLiteHistory{db: db, imageDir: imageDir, maxSize: maxSize}, nil
}

func (h *SQLiteHistory) ImageDir() string {
	return h.imageDir
}

func (h *SQLiteHistory) Add(entry ClipboardEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if entry.Type == EntryTypeText {
		entry.Content = strings.TrimSpace(entry.Content)
		if entry.Content == "" {
			return
		}
	}

	// Remove existing occurrence so the entry moves to the top.
	if _, err := h.db.Exec(`DELETE FROM entries WHERE content = ? AND type = ?`, entry.Content, string(entry.Type)); err != nil {
		log.Printf("sqlite: failed to remove duplicate: %v", err)
		return
	}

	if _, err := h.db.Exec(`INSERT INTO entries (content, type) VALUES (?, ?)`, entry.Content, string(entry.Type)); err != nil {
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

func (h *SQLiteHistory) List() []ClipboardEntry {
	rows, err := h.db.Query(`SELECT id, content, type FROM entries ORDER BY id DESC`)
	if err != nil {
		log.Printf("sqlite: failed to query entries: %v", err)
		return nil
	}
	defer rows.Close()

	var entries []ClipboardEntry
	for rows.Next() {
		var e ClipboardEntry
		var typ string
		if err := rows.Scan(&e.ID, &e.Content, &typ); err != nil {
			continue
		}
		e.Type = EntryType(typ)
		entries = append(entries, e)
	}
	return entries
}

func (h *SQLiteHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Collect image file paths before deleting rows.
	rows, err := h.db.Query(`SELECT content FROM entries WHERE type = 'image'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var path string
			if rows.Scan(&path) == nil {
				_ = os.Remove(path)
			}
		}
	}

	if _, err := h.db.Exec(`DELETE FROM entries`); err != nil {
		log.Printf("sqlite: failed to clear entries: %v", err)
	}
}

func (h *SQLiteHistory) SetMaxSize(maxSize int) {
	h.mu.Lock()
	h.maxSize = maxSize
	h.mu.Unlock()
}

func (h *SQLiteHistory) Close() error {
	return h.db.Close()
}

// EnsureImageDir creates the image storage directory and returns its path.
func EnsureImageDir(dataDir string) (string, error) {
	dir := filepath.Join(dataDir, "images")
	return dir, os.MkdirAll(dir, 0700)
}
