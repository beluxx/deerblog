package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"blog/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByUsername(username string) (*domain.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, nickname, avatar_url, created_at, updated_at
		FROM users WHERE username = ?
	`, username)

	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByID(id int64) (*domain.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, password_hash, nickname, avatar_url, created_at, updated_at
		FROM users WHERE id = ?
	`, id)

	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) UpdatePassword(id int64, passwordHash string) error {
	_, err := r.db.Exec(`
		UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?
	`, passwordHash, time.Now(), id)
	return err
}

func (r *UserRepository) UpdateProfile(id int64, nickname, avatarURL string) error {
	_, err := r.db.Exec(`
		UPDATE users SET nickname = ?, avatar_url = ?, updated_at = ? WHERE id = ?
	`, nickname, avatarURL, time.Now(), id)
	return err
}
