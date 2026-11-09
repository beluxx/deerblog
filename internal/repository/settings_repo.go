package repository

import (
	"encoding/json"
	"fmt"

	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/domain"
	bolt "go.etcd.io/bbolt"
)

const keySiteSettings = "global_settings"

type SettingsRepository struct {
	db  *bolt.DB
	cfg *config.Config
}

func NewSettingsRepository(db *bolt.DB, cfg *config.Config) *SettingsRepository {
	return &SettingsRepository{db: db, cfg: cfg}
}

// Get 获取当前站点配置 (优先从 BoltDB 读取，无则以 config 初始化并回填)
func (r *SettingsRepository) Get() (*domain.SiteSettings, error) {
	var settings domain.SiteSettings
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSiteMeta))
		if b == nil {
			return fmt.Errorf("site_meta bucket not found")
		}
		data := b.Get([]byte(keySiteSettings))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &settings)
	})
	if err != nil {
		return nil, err
	}

	// 若未在数据库中保存过，使用系统默认并持久化
	if settings.SiteTitle == "" {
		settings = domain.SiteSettings{
			SiteTitle:    r.cfg.SiteTitle,
			SiteSubtitle: r.cfg.SiteSubtitle,
			AuthorName:   r.cfg.AuthorName,
			AuthorBio:    r.cfg.AuthorBio,
			GithubURL:    "https://github.com",
			TwitterURL:   "https://twitter.com",
			Email:        "author@example.com",
		}
		_ = r.Save(&settings)
	}

	return &settings, nil
}

// Save 更新站点配置至 BoltDB
func (r *SettingsRepository) Save(s *domain.SiteSettings) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(database.BucketSiteMeta))
		if b == nil {
			return fmt.Errorf("site_meta bucket not found")
		}
		return b.Put([]byte(keySiteSettings), data)
	})
}
