package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MessageStore persists a minimal local history for text chats.
type MessageStore struct {
	db *sql.DB
}

// StoredMessage is a locally persisted text message entry.
type StoredMessage struct {
	ID            string
	JID           string
	FromMe        bool
	Text          string
	Timestamp     int64
	SenderJID     string
	SenderName    string
	IsGroup       bool
	Receipt       int
	HasMedia      bool
	MediaType     string
	MediaMIME     string
	MediaSize     int64
	LocalPath     string
	MediaKey      []byte
	DirectPath    string
	MediaURL      string
	FileEncSHA256 []byte
	FileSHA256    []byte
	IsRead        bool
	URLPreview    string
}

// NewMessageStore opens or creates the local messages database.
func NewMessageStore(dbPath string) (*MessageStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &MessageStore{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *MessageStore) init() error {
	const schema = `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    jid TEXT NOT NULL,
    from_me INTEGER NOT NULL,
    text TEXT NOT NULL DEFAULT '',
    timestamp INTEGER NOT NULL,
    has_media INTEGER NOT NULL DEFAULT 0,
    media_type TEXT NOT NULL DEFAULT '',
    media_mime TEXT NOT NULL DEFAULT '',
    media_size INTEGER NOT NULL DEFAULT 0,
    local_path TEXT NOT NULL DEFAULT '',
    media_key BLOB,
    direct_path TEXT NOT NULL DEFAULT '',
    media_url TEXT NOT NULL DEFAULT '',
    file_enc_sha256 BLOB,
    file_sha256 BLOB,
    is_read INTEGER NOT NULL DEFAULT 0,
    url_preview TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_messages_jid_timestamp
    ON messages(jid, timestamp DESC);
`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	return s.migrateColumns()
}

// Close closes the underlying database.
func (s *MessageStore) Close() error {
	return s.db.Close()
}

// Save inserts or updates one locally stored message.
func (s *MessageStore) Save(msg StoredMessage) error {
	if msg.URLPreview == "" {
		msg.URLPreview = extractFirstURL(msg.Text)
	}

	_, err := s.db.Exec(
		`INSERT INTO messages(
			id, jid, from_me, text, timestamp, sender_jid, sender_name, is_group, receipt,
			has_media, media_type, media_mime, media_size, local_path,
			media_key, direct_path, media_url, file_enc_sha256, file_sha256,
			is_read, url_preview
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			jid = excluded.jid,
			from_me = excluded.from_me,
			text = excluded.text,
			timestamp = excluded.timestamp,
			sender_jid = CASE WHEN excluded.sender_jid != '' THEN excluded.sender_jid ELSE messages.sender_jid END,
			sender_name = CASE WHEN excluded.sender_name != '' THEN excluded.sender_name ELSE messages.sender_name END,
			is_group = CASE WHEN excluded.is_group != 0 THEN excluded.is_group ELSE messages.is_group END,
			receipt = CASE WHEN excluded.receipt > messages.receipt THEN excluded.receipt ELSE messages.receipt END,
			has_media = CASE WHEN excluded.has_media != 0 THEN excluded.has_media ELSE messages.has_media END,
			media_type = CASE WHEN excluded.media_type != '' THEN excluded.media_type ELSE messages.media_type END,
			media_mime = CASE WHEN excluded.media_mime != '' THEN excluded.media_mime ELSE messages.media_mime END,
			media_size = CASE WHEN excluded.media_size != 0 THEN excluded.media_size ELSE messages.media_size END,
			local_path = CASE WHEN excluded.local_path != '' THEN excluded.local_path ELSE messages.local_path END,
			media_key = CASE WHEN excluded.media_key IS NOT NULL AND length(excluded.media_key) > 0 THEN excluded.media_key ELSE messages.media_key END,
			direct_path = CASE WHEN excluded.direct_path != '' THEN excluded.direct_path ELSE messages.direct_path END,
			media_url = CASE WHEN excluded.media_url != '' THEN excluded.media_url ELSE messages.media_url END,
			file_enc_sha256 = CASE WHEN excluded.file_enc_sha256 IS NOT NULL AND length(excluded.file_enc_sha256) > 0 THEN excluded.file_enc_sha256 ELSE messages.file_enc_sha256 END,
			file_sha256 = CASE WHEN excluded.file_sha256 IS NOT NULL AND length(excluded.file_sha256) > 0 THEN excluded.file_sha256 ELSE messages.file_sha256 END,
			is_read = CASE WHEN excluded.is_read != 0 THEN 1 ELSE messages.is_read END,
			url_preview = CASE WHEN excluded.url_preview != '' THEN excluded.url_preview ELSE messages.url_preview END`,
		msg.ID,
		msg.JID,
		boolToInt(msg.FromMe),
		msg.Text,
		msg.Timestamp,
		msg.SenderJID,
		msg.SenderName,
		boolToInt(msg.IsGroup),
		msg.Receipt,
		boolToInt(msg.HasMedia),
		msg.MediaType,
		msg.MediaMIME,
		msg.MediaSize,
		msg.LocalPath,
		msg.MediaKey,
		msg.DirectPath,
		msg.MediaURL,
		msg.FileEncSHA256,
		msg.FileSHA256,
		boolToInt(msg.IsRead),
		msg.URLPreview,
	)
	return err
}

// GetByJID returns recent messages for one chat ordered newest-first.
func (s *MessageStore) GetByJID(jid string, limit int, offset int) ([]StoredMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, jid, from_me, text, timestamp, sender_jid, sender_name, is_group, receipt,
		        has_media, media_type, media_mime, media_size, local_path,
		        media_key, direct_path, media_url, file_enc_sha256, file_sha256,
		        is_read, url_preview
		 FROM messages
		 WHERE jid = ?
		 ORDER BY timestamp DESC
		 LIMIT ? OFFSET ?`,
		jid,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// SearchByJID returns recent matching messages for one chat ordered newest-first.
func (s *MessageStore) SearchByJID(jid string, query string, limit int) ([]StoredMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, jid, from_me, text, timestamp, sender_jid, sender_name, is_group, receipt,
		        has_media, media_type, media_mime, media_size, local_path,
		        media_key, direct_path, media_url, file_enc_sha256, file_sha256,
		        is_read, url_preview
		 FROM messages
		 WHERE jid = ? AND LOWER(text) LIKE LOWER(?)
		 ORDER BY timestamp DESC
		 LIMIT ?`,
		jid,
		"%"+query+"%",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetRecentChats returns the newest stored message for every distinct chat.
func (s *MessageStore) GetRecentChats() ([]StoredMessage, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.jid, m.from_me, m.text, m.timestamp, m.sender_jid, m.sender_name, m.is_group, m.receipt,
		        m.has_media, m.media_type, m.media_mime, m.media_size, m.local_path,
		        m.media_key, m.direct_path, m.media_url, m.file_enc_sha256, m.file_sha256,
		        m.is_read, m.url_preview
		 FROM messages m
		 INNER JOIN (
		     SELECT jid, MAX(timestamp) AS max_timestamp
		     FROM messages
		     GROUP BY jid
		 ) latest ON latest.jid = m.jid AND latest.max_timestamp = m.timestamp
		 ORDER BY m.timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMessages(rows)
}

// GetByID returns a single stored message by its message ID.
func (s *MessageStore) GetByID(id string) (*StoredMessage, error) {
	row := s.db.QueryRow(
		`SELECT id, jid, from_me, text, timestamp, sender_jid, sender_name, is_group, receipt,
		        has_media, media_type, media_mime, media_size, local_path,
		        media_key, direct_path, media_url, file_enc_sha256, file_sha256,
		        is_read, url_preview
		 FROM messages
		 WHERE id = ?`,
		id,
	)

	msg, err := scanMessage(row)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// UpdateLocalPath stores the local file path after media download.
func (s *MessageStore) UpdateLocalPath(id string, localPath string) error {
	_, err := s.db.Exec(`UPDATE messages SET local_path = ? WHERE id = ?`, localPath, id)
	return err
}

// UpdateReceipt stores the highest-known receipt state for an outgoing message.
func (s *MessageStore) UpdateReceipt(id string, receipt int) error {
	_, err := s.db.Exec(`UPDATE messages SET receipt = MAX(receipt, ?) WHERE id = ?`, receipt, id)
	return err
}

// MarkJIDAsRead marks all received messages in a chat as read.
func (s *MessageStore) MarkJIDAsRead(jid string) error {
	_, err := s.db.Exec(`UPDATE messages SET is_read = 1 WHERE jid = ? AND from_me = 0`, jid)
	return err
}

// GetUnreadCount returns the unread count for a chat.
func (s *MessageStore) GetUnreadCount(jid string) (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM messages WHERE jid = ? AND from_me = 0 AND is_read = 0`, jid)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *MessageStore) migrateColumns() error {
	columns := map[string]string{
		"sender_jid":      "TEXT NOT NULL DEFAULT ''",
		"sender_name":     "TEXT NOT NULL DEFAULT ''",
		"is_group":        "INTEGER NOT NULL DEFAULT 0",
		"receipt":         "INTEGER NOT NULL DEFAULT 0",
		"has_media":       "INTEGER NOT NULL DEFAULT 0",
		"media_type":      "TEXT NOT NULL DEFAULT ''",
		"media_mime":      "TEXT NOT NULL DEFAULT ''",
		"media_size":      "INTEGER NOT NULL DEFAULT 0",
		"local_path":      "TEXT NOT NULL DEFAULT ''",
		"media_key":       "BLOB",
		"direct_path":     "TEXT NOT NULL DEFAULT ''",
		"media_url":       "TEXT NOT NULL DEFAULT ''",
		"file_enc_sha256": "BLOB",
		"file_sha256":     "BLOB",
		"is_read":         "INTEGER NOT NULL DEFAULT 0",
		"url_preview":     "TEXT NOT NULL DEFAULT ''",
	}

	existingColumns := make(map[string]bool)
	rows, err := s.db.Query(`PRAGMA table_info(messages)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existingColumns[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for name, definition := range columns {
		if existingColumns[name] {
			continue
		}
		if _, err := s.db.Exec(fmt.Sprintf("ALTER TABLE messages ADD COLUMN %s %s", name, definition)); err != nil {
			return err
		}
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMessages(rows *sql.Rows) ([]StoredMessage, error) {
	var messages []StoredMessage
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func scanMessage(scanner rowScanner) (StoredMessage, error) {
	var msg StoredMessage
	var fromMe int
	var hasMedia int
	var isRead int
	var isGroup int
	if err := scanner.Scan(
		&msg.ID, &msg.JID, &fromMe, &msg.Text, &msg.Timestamp, &msg.SenderJID, &msg.SenderName, &isGroup, &msg.Receipt,
		&hasMedia, &msg.MediaType, &msg.MediaMIME, &msg.MediaSize, &msg.LocalPath,
		&msg.MediaKey, &msg.DirectPath, &msg.MediaURL, &msg.FileEncSHA256, &msg.FileSHA256,
		&isRead, &msg.URLPreview,
	); err != nil {
		return StoredMessage{}, err
	}
	msg.FromMe = fromMe == 1
	msg.HasMedia = hasMedia == 1
	msg.IsRead = isRead == 1
	msg.IsGroup = isGroup == 1
	return msg, nil
}

var urlRegexp = regexp.MustCompile(`https?://[^\s]+`)

func extractFirstURL(text string) string {
	match := urlRegexp.FindString(text)
	return strings.TrimRight(match, ".,;:!?)]}\"'")
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
