package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shanth1/gotools/errs"
	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	_ "modernc.org/sqlite"
)

type Storage struct {
	db  *sql.DB
	log log.Logger
}

func New(path string, l log.Logger) (*Storage, error) {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, errs.Wrap(errs.ErrConnection, err.Error())
	}

	if err := db.Ping(); err != nil {
		return nil, errs.Wrap(errs.ErrConnection, "failed to ping db")
	}

	s := &Storage{db: db, log: l}
	if err := s.init(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS message_routes (
		admin_msg_id INTEGER PRIMARY KEY,
		user_chat_id INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_created_at ON message_routes(created_at);
	`
	_, err := s.db.Exec(query)
	return errs.Wrap(err, "failed to init schema")
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveRoute(ctx context.Context, adminMsgID int, userChatID int64) error {
	query := `INSERT INTO message_routes (admin_msg_id, user_chat_id, created_at) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, adminMsgID, userChatID, time.Now().Unix())
	if err != nil {
		return errs.Wrap(errs.ErrDBQuery, err.Error())
	}
	return nil
}

func (s *Storage) GetUserByAdminMsg(ctx context.Context, adminMsgID int) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx, "SELECT user_chat_id FROM message_routes WHERE admin_msg_id = ?", adminMsgID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errs.ErrNotFound
		}
		return 0, errs.Wrap(errs.ErrDBQuery, err.Error())
	}
	return userID, nil
}

func (s *Storage) StartCleaner(ctx context.Context, retentionDays int) {
	if retentionDays <= 0 {
		return
	}
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		defer ticker.Stop()
		s.log.Info().Int("retention_days", retentionDays).Msg("DB cleaner started")
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
				res, err := s.db.ExecContext(ctx, "DELETE FROM message_routes WHERE created_at < ?", cutoff)
				if err != nil {
					s.log.Error().Err(err).Msg("Failed to clean DB")
					continue
				}
				rows, _ := res.RowsAffected()
				if rows > 0 {
					s.log.Info().
						Int64(logkeys.DBRows, rows).
						Msg("DB cleanup complete")
				}
			}
		}
	}()
}
