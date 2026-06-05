package storage

import (
	"context"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

// NewStore creates the SQLite-backed whatsmeow store container.
func NewStore(dbPath string) (*sqlstore.Container, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	return sqlstore.New(context.Background(), "sqlite3", "file:"+dbPath+"?_foreign_keys=on", nil)
}
