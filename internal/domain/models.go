package domain

import (
	"html/template"
	"time"
)

// User 管理员实体
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Nickname     string    `json:"nickname"`
	AvatarURL    string    `json:"avatar_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Category 文章分类
type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	PostCount   int       `json:"post_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Tag 文章标签
type Tag struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	PostCount int    `json:"post_count,omitempty"`
}

// Post 文章实体
type Post struct {
	ID          int64         `json:"id"`
	Title       string        `json:"title"`
	Slug        string        `json:"slug"`
	Summary     string        `json:"summary"`
	ContentRaw  string        `json:"content_raw"`
	ContentHTML template.HTML `json:"content_html"` // 已经过服务端 Chroma 高亮和 Goldmark 渲染的安全 HTML
	CoverImage  string        `json:"cover_image"`
	CategoryID  *int64        `json:"category_id"`
	Category    *Category     `json:"category,omitempty"`
	Tags        []Tag         `json:"tags,omitempty"`
	IsPublished bool          `json:"is_published"`
	PublishedAt *time.Time    `json:"published_at"`
	ViewsCount  int           `json:"views_count"`
	LikesCount  int           `json:"likes_count"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	// 阅读统计计算字段
	ReadingMinutes int `json:"reading_minutes"`
}

// Comment 评论实体
type Comment struct {
	ID          int64     `json:"id"`
	PostID      int64     `json:"post_id"`
	PostTitle   string    `json:"post_title,omitempty"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	AuthorSite  string    `json:"author_site"`
	Content     string    `json:"content"`
	IsApproved  bool      `json:"is_approved"`
	IPAddress   string    `json:"ip_address,omitempty"`
	UserAgent   string    `json:"user_agent,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableOfContentItem 目录大纲
type TableOfContentItem struct {
	Level int    `json:"level"` // 2 for h2, 3 for h3
	Title string `json:"title"`
	ID    string `json:"id"`
}

// DashboardStats 后台仪表盘统计
type DashboardStats struct {
	TotalPosts     int `json:"total_posts"`
	PublishedPosts int `json:"published_posts"`
	TotalComments  int `json:"total_comments"`
	PendingComments int `json:"pending_comments"`
	TotalViews     int `json:"total_views"`
	TotalLikes     int `json:"total_likes"`
}

// SiteSettings 动态站点全局配置
type SiteSettings struct {
	SiteTitle    string `json:"site_title"`
	SiteSubtitle string `json:"site_subtitle"`
	AuthorName   string `json:"author_name"`
	AuthorBio    string `json:"author_bio"`
	GithubURL    string `json:"github_url"`
	TwitterURL   string `json:"twitter_url"`
	Email        string `json:"email"`
}
