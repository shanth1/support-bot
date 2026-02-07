package storage

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func New(path string) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			admin_msg_id INTEGER PRIMARY KEY,
			user_chat_id INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS projects (
			name TEXT PRIMARY KEY,
			topic_id INTEGER
		);
	`)
	return &Storage{db: db}, err
}

func (s *Storage) SaveMessage(adminMsgID int, userChatID int64) error {
	_, err := s.db.Exec("INSERT INTO messages (admin_msg_id, user_chat_id) VALUES (?, ?)", adminMsgID, userChatID)
	return err
}

func (s *Storage) GetUserByMsg(adminMsgID int) (int64, error) {
	var chatID int64
	err := s.db.QueryRow("SELECT user_chat_id FROM messages WHERE admin_msg_id = ?", adminMsgID).Scan(&chatID)
	return chatID, err
}
