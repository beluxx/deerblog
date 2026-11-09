package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"blog/internal/domain"
)

type PostRepository struct {
	db *sql.DB
}

func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

// ListPublished 获取已发布的文章列表 (支持分页、分类筛选、标签筛选、关键词搜索)
func (r *PostRepository) ListPublished(limit, offset int, categorySlug, tagSlug, searchQuery string) ([]domain.Post, int, error) {
	where := []string{"p.is_published = 1"}
	var args []any

	if categorySlug != "" {
		where = append(where, "c.slug = ?")
		args = append(args, categorySlug)
	}

	if tagSlug != "" {
		where = append(where, "EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id = pt.tag_id WHERE pt.post_id = p.id AND t.slug = ?)")
		args = append(args, tagSlug)
	}

	if searchQuery != "" {
		where = append(where, "(p.title LIKE ? OR p.summary LIKE ? OR p.content_raw LIKE ?)")
		pattern := "%" + searchQuery + "%"
		args = append(args, pattern, pattern, pattern)
	}

	whereClause := strings.Join(where, " AND ")

	// 统计总数
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM posts p
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE %s
	`, whereClause)
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count published posts: %w", err)
	}

	// 查询数据列表
	query := fmt.Sprintf(`
		SELECT p.id, p.title, p.slug, p.summary, p.cover_image, p.category_id,
		       p.is_published, p.published_at, p.views_count, p.likes_count, p.created_at, p.updated_at,
		       c.id, c.name, c.slug, c.description
		FROM posts p
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE %s
		ORDER BY p.published_at DESC, p.id DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	argsWithPaging := append(args, limit, offset)
	rows, err := r.db.Query(query, argsWithPaging...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []domain.Post
	for rows.Next() {
		var p domain.Post
		var catID sql.NullInt64
		var cID sql.NullInt64
		var cName, cSlug, cDesc sql.NullString
		var pubAt sql.NullTime

		err := rows.Scan(
			&p.ID, &p.Title, &p.Slug, &p.Summary, &p.CoverImage, &catID,
			&p.IsPublished, &pubAt, &p.ViewsCount, &p.LikesCount, &p.CreatedAt, &p.UpdatedAt,
			&cID, &cName, &cSlug, &cDesc,
		)
		if err != nil {
			return nil, 0, err
		}

		if catID.Valid {
			p.CategoryID = &catID.Int64
		}
		if pubAt.Valid {
			p.PublishedAt = &pubAt.Time
		}
		if cID.Valid {
			p.Category = &domain.Category{
				ID:          cID.Int64,
				Name:        cName.String,
				Slug:        cSlug.String,
				Description: cDesc.String,
			}
		}

		posts = append(posts, p)
	}

	return posts, total, nil
}

// GetBySlug 获取单篇文章详情 (包含分类、标签与预渲染的 HTML 内容)
func (r *PostRepository) GetBySlug(slug string, onlyPublished bool) (*domain.Post, error) {
	where := "p.slug = ?"
	if onlyPublished {
		where += " AND p.is_published = 1"
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.title, p.slug, p.summary, p.content_raw, p.content_html, p.cover_image, p.category_id,
		       p.is_published, p.published_at, p.views_count, p.likes_count, p.created_at, p.updated_at,
		       c.id, c.name, c.slug, c.description
		FROM posts p
		LEFT JOIN categories c ON c.id = p.category_id
		WHERE %s
	`, where)

	row := r.db.QueryRow(query, slug)
	var p domain.Post
	var htmlStr string
	var catID sql.NullInt64
	var cID sql.NullInt64
	var cName, cSlug, cDesc sql.NullString
	var pubAt sql.NullTime

	err := row.Scan(
		&p.ID, &p.Title, &p.Slug, &p.Summary, &p.ContentRaw, &htmlStr, &p.CoverImage, &catID,
		&p.IsPublished, &pubAt, &p.ViewsCount, &p.LikesCount, &p.CreatedAt, &p.UpdatedAt,
		&cID, &cName, &cSlug, &cDesc,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post by slug: %w", err)
	}

	p.ContentHTML = template.HTML(htmlStr)
	if catID.Valid {
		p.CategoryID = &catID.Int64
	}
	if pubAt.Valid {
		p.PublishedAt = &pubAt.Time
	}
	if cID.Valid {
		p.Category = &domain.Category{
			ID:          cID.Int64,
			Name:        cName.String,
			Slug:        cSlug.String,
			Description: cDesc.String,
		}
	}

	// 加载关联的标签
	tags, err := r.getTagsForPost(p.ID)
	if err == nil {
		p.Tags = tags
	}

	return &p, nil
}

// GetByID 后台编辑获取文章
func (r *PostRepository) GetByID(id int64) (*domain.Post, error) {
	query := `
		SELECT p.id, p.title, p.slug, p.summary, p.content_raw, p.content_html, p.cover_image, p.category_id,
		       p.is_published, p.published_at, p.views_count, p.likes_count, p.created_at, p.updated_at
		FROM posts p
		WHERE p.id = ?
	`
	row := r.db.QueryRow(query, id)
	var p domain.Post
	var htmlStr string
	var catID sql.NullInt64
	var pubAt sql.NullTime

	err := row.Scan(
		&p.ID, &p.Title, &p.Slug, &p.Summary, &p.ContentRaw, &htmlStr, &p.CoverImage, &catID,
		&p.IsPublished, &pubAt, &p.ViewsCount, &p.LikesCount, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.ContentHTML = template.HTML(htmlStr)
	if catID.Valid {
		p.CategoryID = &catID.Int64
	}
	if pubAt.Valid {
		p.PublishedAt = &pubAt.Time
	}

	tags, _ := r.getTagsForPost(p.ID)
	p.Tags = tags
	return &p, nil
}

func (r *PostRepository) getTagsForPost(postID int64) ([]domain.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug
		FROM tags t
		JOIN post_tags pt ON pt.tag_id = t.id
		WHERE pt.post_id = ?
		ORDER BY t.name ASC
	`
	rows, err := r.db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err == nil {
			tags = append(tags, t)
		}
	}
	return tags, nil
}

// Create 创建新文章
func (r *PostRepository) Create(p *domain.Post, tagIDs []int64) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO posts (title, slug, summary, content_raw, content_html, cover_image, category_id, is_published, published_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	var pubAt *time.Time
	if p.IsPublished {
		pubAt = &now
	}

	res, err := tx.Exec(query, p.Title, p.Slug, p.Summary, p.ContentRaw, string(p.ContentHTML), p.CoverImage, p.CategoryID, p.IsPublished, pubAt, now, now)
	if err != nil {
		return 0, fmt.Errorf("failed to insert post: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, tid := range tagIDs {
		if _, err := tx.Exec("INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)", id, tid); err != nil {
			return 0, err
		}
	}

	return id, tx.Commit()
}

// Update 更新文章
func (r *PostRepository) Update(p *domain.Post, tagIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	var pubAt *time.Time
	if p.IsPublished {
		if p.PublishedAt != nil {
			pubAt = p.PublishedAt
		} else {
			pubAt = &now
		}
	}

	query := `
		UPDATE posts SET title = ?, slug = ?, summary = ?, content_raw = ?, content_html = ?, cover_image = ?, category_id = ?, is_published = ?, published_at = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = tx.Exec(query, p.Title, p.Slug, p.Summary, p.ContentRaw, string(p.ContentHTML), p.CoverImage, p.CategoryID, p.IsPublished, pubAt, now, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	// 重新关联标签
	if _, err := tx.Exec("DELETE FROM post_tags WHERE post_id = ?", p.ID); err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if _, err := tx.Exec("INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)", p.ID, tid); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete 删除文章
func (r *PostRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM posts WHERE id = ?", id)
	return err
}

// IncrementLikes 点赞递增并返回最新点赞数
func (r *PostRepository) IncrementLikes(slug string) (int, error) {
	_, err := r.db.Exec("UPDATE posts SET likes_count = likes_count + 1 WHERE slug = ?", slug)
	if err != nil {
		return 0, err
	}
	var likes int
	err = r.db.QueryRow("SELECT likes_count FROM posts WHERE slug = ?", slug).Scan(&likes)
	return likes, err
}

// IncrementViews 浏览量递增
func (r *PostRepository) IncrementViews(id int64) error {
	_, err := r.db.Exec("UPDATE posts SET views_count = views_count + 1 WHERE id = ?", id)
	return err
}

// ListAllForAdmin 后台获取全部文章 (包含草稿)
func (r *PostRepository) ListAllForAdmin() ([]domain.Post, error) {
	query := `
		SELECT p.id, p.title, p.slug, p.is_published, p.published_at, p.views_count, p.likes_count, p.created_at,
		       c.name
		FROM posts p
		LEFT JOIN categories c ON c.id = p.category_id
		ORDER BY p.id DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []domain.Post
	for rows.Next() {
		var p domain.Post
		var cName sql.NullString
		var pubAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.Title, &p.Slug, &p.IsPublished, &pubAt, &p.ViewsCount, &p.LikesCount, &p.CreatedAt, &cName); err != nil {
			return nil, err
		}
		if pubAt.Valid {
			p.PublishedAt = &pubAt.Time
		}
		if cName.Valid {
			p.Category = &domain.Category{Name: cName.String}
		}
		posts = append(posts, p)
	}
	return posts, nil
}

// GetDashboardStats 获取仪表盘统计数据
func (r *PostRepository) GetDashboardStats() (*domain.DashboardStats, error) {
	var s domain.DashboardStats
	err := r.db.QueryRow(`
		SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN is_published = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(views_count), 0),
			COALESCE(SUM(likes_count), 0)
		FROM posts
	`).Scan(&s.TotalPosts, &s.PublishedPosts, &s.TotalViews, &s.TotalLikes)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`
		SELECT 
			COUNT(*),
			COALESCE(SUM(CASE WHEN is_approved = 0 THEN 1 ELSE 0 END), 0)
		FROM comments
	`).Scan(&s.TotalComments, &s.PendingComments)
	if err != nil {
		return nil, err
	}

	return &s, nil
}
