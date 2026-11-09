package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port         string
	DataDir      string
	SqlitePath   string
	BoltPath     string
	AdminUser    string
	AdminPass    string
	SessionSecret string
	SiteTitle    string
	SiteSubtitle string
	AuthorName   string
	AuthorBio    string
}

func Load() *Config {
	dataDir := getEnv("DATA_DIR", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// 回退到当前目录
		dataDir = "."
	}

	return &Config{
		Port:          getEnv("PORT", "8080"),
		DataDir:       dataDir,
		SqlitePath:    filepath.Join(dataDir, getEnv("SQLITE_FILE", "blog.db")),
		BoltPath:      filepath.Join(dataDir, getEnv("BOLT_FILE", "cache.bolt")),
		AdminUser:     getEnv("ADMIN_USER", "admin"),
		AdminPass:     getEnv("ADMIN_PASS", "admin123"),
		SessionSecret: getEnv("SESSION_SECRET", "super-secret-key-change-in-prod-32bytes!"),
		SiteTitle:     getEnv("SITE_TITLE", "极客与远方"),
		SiteSubtitle:  getEnv("SITE_SUBTITLE", "思考、技术、架构与数字生活"),
		AuthorName:    getEnv("AUTHOR_NAME", "博主"),
		AuthorBio:     getEnv("AUTHOR_BIO", "一名热爱系统底层、纯粹架构与极简代码的全栈工程师。"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
