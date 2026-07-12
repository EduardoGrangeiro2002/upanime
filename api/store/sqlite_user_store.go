package store

import (
	"context"
	"database/sql"
	"time"

	"upanime/api/model"
)

type SQLiteUserStore struct {
	db *sql.DB
}

func NewSQLiteUserStore(db *sql.DB) *SQLiteUserStore {
	return &SQLiteUserStore{db: db}
}

func (s *SQLiteUserStore) Create(ctx context.Context, user *model.User) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, must_change_password, is_admin) VALUES (?, ?, ?, ?)`,
		user.Email, user.PasswordHash, boolToInt(user.MustChangePassword), boolToInt(user.IsAdmin),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

func (s *SQLiteUserStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, must_change_password, is_admin, last_ip, last_location, last_mfa_at
		 FROM users WHERE email = ?`, email,
	)

	var user model.User
	var mustChange, isAdmin int
	var lastMFA string
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &mustChange, &isAdmin, &user.LastIP, &user.LastLocation, &lastMFA); err != nil {
		return nil, err
	}
	user.MustChangePassword = mustChange == 1
	user.IsAdmin = isAdmin == 1
	if lastMFA != "" {
		parsed, err := time.Parse(time.RFC3339, lastMFA)
		if err == nil {
			user.LastMFAAt = parsed
		}
	}
	return &user, nil
}

func (s *SQLiteUserStore) List(ctx context.Context) ([]model.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, email, must_change_password, is_admin FROM users ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		var mustChange, isAdmin int
		if err := rows.Scan(&user.ID, &user.Email, &mustChange, &isAdmin); err != nil {
			return nil, err
		}
		user.MustChangePassword = mustChange == 1
		user.IsAdmin = isAdmin == 1
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *SQLiteUserStore) UpdatePassword(ctx context.Context, email, passwordHash string, mustChange bool) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, must_change_password = ?, updated_at = datetime('now') WHERE email = ?`,
		passwordHash, boolToInt(mustChange), email,
	)
	return err
}

func (s *SQLiteUserStore) UpdateMFAContext(ctx context.Context, email, ip, location string, at time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET last_ip = ?, last_location = ?, last_mfa_at = ?, updated_at = datetime('now') WHERE email = ?`,
		ip, location, at.UTC().Format(time.RFC3339), email,
	)
	return err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
