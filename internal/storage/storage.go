package storage

import (
	"database/sql"
	"fmt"

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

	query := `
	CREATE TABLE IF NOT EXISTS messages (
		admin_msg_id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS users (
		user_id INTEGER PRIMARY KEY,
		project_id TEXT NOT NULL
	);`

	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания таблиц: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveMessage(adminMsgID int, userID int64) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO messages (admin_msg_id, user_id) VALUES (?, ?)", adminMsgID, userID)
	return err
}

func (s *Storage) GetUserByMsg(adminMsgID int) (int64, error) {
	var userID int64
	err := s.db.QueryRow("SELECT user_id FROM messages WHERE admin_msg_id = ?", adminMsgID).Scan(&userID)
	return userID, err
}

func (s *Storage) SetUserProject(userID int64, projectID string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO users (user_id, project_id) VALUES (?, ?)", userID, projectID)
	return err
}

func (s *Storage) GetUserProject(userID int64) (string, error) {
	var projectID string
	err := s.db.QueryRow("SELECT project_id FROM users WHERE user_id = ?", userID).Scan(&projectID)
	if err != nil {
		return "default", nil
	}
	return projectID, nil
}
