package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"blog/internal/database"
	bolt "go.etcd.io/bbolt"
)

type SessionData struct {
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
}

type SessionRepository struct {
	db *bolt.DB
}

func NewSessionRepository(db *bolt.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Save(token string, data *SessionData) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSessions))
		if b == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		return b.Put([]byte(token), bytes)
	})
}

func (r *SessionRepository) Get(token string) (*SessionData, error) {
	var data SessionData
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSessions))
		if b == nil {
			return fmt.Errorf("sessions bucket not found")
		}
		v := b.Get([]byte(token))
		if v == nil {
			return fmt.Errorf("session not found")
		}
		return json.Unmarshal(v, &data)
	})
	if err != nil {
		return nil, err
	}

	// 检查过期时间
	if time.Now().After(data.ExpiresAt) {
		_ = r.Delete(token)
		return nil, fmt.Errorf("session expired")
	}

	return &data, nil
}

func (r *SessionRepository) Delete(token string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSessions))
		if b == nil {
			return nil
		}
		return b.Delete([]byte(token))
	})
}
