package database

import (
	"database/sql"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// InitSQLite 初始化纯 Go SQLite 数据库连接并执行自动迁移
func InitSQLite(filepath, defaultAdminUser, defaultAdminPass string) (*sql.DB, error) {
	// 连接参数：启用 WAL 模式、缓存共享、外键约束
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", filepath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// 设置连接池：由于是嵌入式 SQLite，设置适量的闲置与活跃连接
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	// 运行 DDL 迁移
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run sqlite migrations: %w", err)
	}

	// 检查并初始化默认管理员
	if err := ensureDefaultAdmin(db, defaultAdminUser, defaultAdminPass); err != nil {
		log.Printf("[Warning] Failed to ensure default admin: %v", err)
	}

	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		nickname TEXT NOT NULL,
		avatar_url TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		slug TEXT NOT NULL UNIQUE,
		description TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		slug TEXT NOT NULL UNIQUE
	);

	CREATE TABLE IF NOT EXISTS posts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		summary TEXT DEFAULT '',
		content_raw TEXT NOT NULL,
		content_html TEXT NOT NULL,
		cover_image TEXT DEFAULT '',
		category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
		is_published BOOLEAN DEFAULT FALSE,
		published_at DATETIME,
		views_count INTEGER DEFAULT 0,
		likes_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug);
	CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(is_published, published_at DESC);

	CREATE TABLE IF NOT EXISTS post_tags (
		post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (post_id, tag_id)
	);

	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		author_name TEXT NOT NULL,
		author_email TEXT NOT NULL,
		author_site TEXT DEFAULT '',
		content TEXT NOT NULL,
		is_approved BOOLEAN DEFAULT TRUE,
		ip_address TEXT DEFAULT '',
		user_agent TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id, is_approved);
	`

	_, err := db.Exec(schema)
	return err
}

func ensureDefaultAdmin(db *sql.DB, username, password string) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (username, password_hash, nickname, avatar_url)
		VALUES (?, ?, ?, ?)
	`, username, string(hash), "站长", "/static/images/avatar.png")

	if err == nil {
		log.Printf("==================================================================")
		log.Printf(" [Security] 已自动创建初始管理员账号: %s", username)
		log.Printf(" [Security] 初始密码: %s (请登录后台后及时修改！)", password)
		log.Printf("==================================================================")
	}
	return err
}
