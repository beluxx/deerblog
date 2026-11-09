package database

import (
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	BucketSessions    = "sessions"
	BucketViewsBuffer = "views_buffer"
	BucketSiteMeta    = "site_meta"
)

// InitBolt 初始化纯 Go BoltDB 嵌入式 KV 数据库
func InitBolt(filepath string) (*bolt.DB, error) {
	db, err := bolt.Open(filepath, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open boltdb: %w", err)
	}

	// 预先创建所有基础存储桶
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := []string{BucketSessions, BucketViewsBuffer, BucketSiteMeta}
		for _, b := range buckets {
			if _, err := tx.CreateBucketIfNotExists([]byte(b)); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", b, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
