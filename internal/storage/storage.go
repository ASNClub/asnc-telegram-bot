package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

var migrations = []string{
	`ALTER TABLE reputation ADD COLUMN positive_given INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE reputation ADD COLUMN negative_given INTEGER NOT NULL DEFAULT 0`,
}

type User struct {
	ChatID        int64
	UserID        int64
	Username      string
	DisplayName   string
	Score         int64
	PositiveGiven int64
	NegativeGiven int64
}

type KnownUser struct {
	UserID    int64
	Username  string
	FirstName string
	LastName  string
	IsBot     bool
}

type Store interface {
	Init(ctx context.Context) error
	ApplyDelta(ctx context.Context, chatID, userID int64, username, displayName string, delta int, at time.Time) (User, error)
	GetUser(ctx context.Context, chatID, userID int64) (User, bool, error)
	GetScore(ctx context.Context, chatID, userID int64) (int64, error)
	Top(ctx context.Context, chatID int64, limit int) ([]User, error)
	GetLastChange(ctx context.Context, chatID, fromID, toID int64) (time.Time, bool, error)
	TouchCooldown(ctx context.Context, chatID, fromID, toID int64, at time.Time) error
	RememberUser(ctx context.Context, chatID int64, u KnownUser, at time.Time) error
	FindByUsername(ctx context.Context, chatID int64, username string) (KnownUser, bool, error)
	Close() error
}

type sqliteStore struct {
	db *sql.DB
}

func NewSQLite(path string) (Store, error) {

	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("mkdir %s: %w", dir, err)
			}
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return &sqliteStore{db: db}, nil
}

func (s *sqliteStore) Init(ctx context.Context) error {

	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}
	for _, m := range migrations {
		if _, err := s.db.ExecContext(ctx, m); err != nil {
			if strings.Contains(err.Error(), "duplicate column") {
				continue
			}
			return fmt.Errorf("migration %q: %w", m, err)
		}
	}
	return nil
}

func (s *sqliteStore) ApplyDelta(ctx context.Context, chatID, userID int64, username, displayName string, delta int, at time.Time) (User, error) {

	var posDelta, negDelta int
	if delta >= 0 {
		posDelta = delta
	} else {
		negDelta = -delta
	}

	const q = `
INSERT INTO reputation (chat_id, user_id, username, display_name, score, positive_given, negative_given, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chat_id, user_id) DO UPDATE SET
    score          = reputation.score + excluded.score,
    positive_given = reputation.positive_given + excluded.positive_given,
    negative_given = reputation.negative_given + excluded.negative_given,
    username       = excluded.username,
    display_name   = excluded.display_name,
    updated_at     = excluded.updated_at
RETURNING score, positive_given, negative_given;
`
	u := User{
		ChatID:      chatID,
		UserID:      userID,
		Username:    username,
		DisplayName: displayName,
	}
	row := s.db.QueryRowContext(ctx, q,
		chatID, userID, username, displayName,
		delta, posDelta, negDelta, at.Unix())
	if err := row.Scan(&u.Score, &u.PositiveGiven, &u.NegativeGiven); err != nil {
		return User{}, fmt.Errorf("apply delta: %w", err)
	}
	return u, nil
}

func (s *sqliteStore) GetUser(ctx context.Context, chatID, userID int64) (User, bool, error) {

	const q = `
SELECT chat_id, user_id, username, display_name, score, positive_given, negative_given
FROM reputation
WHERE chat_id = ? AND user_id = ?`
	var u User
	err := s.db.QueryRowContext(ctx, q, chatID, userID).
		Scan(&u.ChatID, &u.UserID, &u.Username, &u.DisplayName, &u.Score, &u.PositiveGiven, &u.NegativeGiven)
	if err == sql.ErrNoRows {
		return User{ChatID: chatID, UserID: userID}, false, nil
	}
	if err != nil {
		return User{}, false, fmt.Errorf("get user: %w", err)
	}
	return u, true, nil
}

func (s *sqliteStore) GetScore(ctx context.Context, chatID, userID int64) (int64, error) {
	const q = `SELECT score FROM reputation WHERE chat_id = ? AND user_id = ?`
	var score int64
	err := s.db.QueryRowContext(ctx, q, chatID, userID).Scan(&score)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get score: %w", err)
	}
	return score, nil
}

func (s *sqliteStore) Top(ctx context.Context, chatID int64, limit int) ([]User, error) {
	const q = `
SELECT chat_id, user_id, username, display_name, score, positive_given, negative_given
FROM reputation
WHERE chat_id = ?
ORDER BY score DESC, updated_at ASC
LIMIT ?;
`
	rows, err := s.db.QueryContext(ctx, q, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("top query: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ChatID, &u.UserID, &u.Username, &u.DisplayName, &u.Score, &u.PositiveGiven, &u.NegativeGiven); err != nil {
			return nil, fmt.Errorf("top scan: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *sqliteStore) GetLastChange(ctx context.Context, chatID, fromID, toID int64) (time.Time, bool, error) {

	const q = `SELECT last_change_at FROM cooldown WHERE chat_id = ? AND from_user_id = ? AND to_user_id = ?`
	var ts int64
	err := s.db.QueryRowContext(ctx, q, chatID, fromID, toID).Scan(&ts)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("get last change: %w", err)
	}
	return time.Unix(ts, 0), true, nil
}

func (s *sqliteStore) TouchCooldown(ctx context.Context, chatID, fromID, toID int64, at time.Time) error {
	const q = `
INSERT INTO cooldown (chat_id, from_user_id, to_user_id, last_change_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(chat_id, from_user_id, to_user_id) DO UPDATE SET
    last_change_at = excluded.last_change_at;
`
	if _, err := s.db.ExecContext(ctx, q, chatID, fromID, toID, at.Unix()); err != nil {
		return fmt.Errorf("touch cooldown: %w", err)
	}
	return nil
}

func (s *sqliteStore) RememberUser(ctx context.Context, chatID int64, u KnownUser, at time.Time) error {

	const q = `
INSERT INTO known_users (chat_id, user_id, username, first_name, last_name, is_bot, seen_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chat_id, user_id) DO UPDATE SET
    username   = excluded.username,
    first_name = excluded.first_name,
    last_name  = excluded.last_name,
    is_bot     = excluded.is_bot,
    seen_at    = excluded.seen_at;
`
	var isBot int
	if u.IsBot {
		isBot = 1
	}
	if _, err := s.db.ExecContext(ctx, q,
		chatID, u.UserID, u.Username, u.FirstName, u.LastName, isBot, at.Unix()); err != nil {
		return fmt.Errorf("remember user: %w", err)
	}
	return nil
}

func (s *sqliteStore) FindByUsername(ctx context.Context, chatID int64, username string) (KnownUser, bool, error) {

	const q = `
SELECT user_id, username, first_name, last_name, is_bot
FROM known_users
WHERE chat_id = ? AND username != '' AND lower(username) = lower(?)
ORDER BY seen_at DESC
LIMIT 1;
`
	var (
		u     KnownUser
		isBot int
	)
	err := s.db.QueryRowContext(ctx, q, chatID, username).
		Scan(&u.UserID, &u.Username, &u.FirstName, &u.LastName, &isBot)
	if err == sql.ErrNoRows {
		return KnownUser{}, false, nil
	}
	if err != nil {
		return KnownUser{}, false, fmt.Errorf("find by username: %w", err)
	}
	u.IsBot = isBot != 0
	return u, true, nil
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
