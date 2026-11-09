package repository

import (
	"database/sql"
	"fmt"
	"time"

	"blog/internal/domain"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// ListByPostID 获取指定文章下已审核通过的评论
func (r *CommentRepository) ListByPostID(postID int64) ([]domain.Comment, error) {
	query := `
		SELECT id, post_id, author_name, author_email, author_site, content, is_approved, created_at
		FROM comments
		WHERE post_id = ? AND is_approved = 1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(query, postID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	var list []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.AuthorEmail, &c.AuthorSite, &c.Content, &c.IsApproved, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

// Create 插入一条新评论
func (r *CommentRepository) Create(c *domain.Comment) (int64, error) {
	query := `
		INSERT INTO comments (post_id, author_name, author_email, author_site, content, is_approved, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	res, err := r.db.Exec(query, c.PostID, c.AuthorName, c.AuthorEmail, c.AuthorSite, c.Content, c.IsApproved, c.IPAddress, c.UserAgent, now)
	if err != nil {
		return 0, fmt.Errorf("failed to insert comment: %w", err)
	}
	return res.LastInsertId()
}

// ListAllForAdmin 后台查看所有评论
func (r *CommentRepository) ListAllForAdmin() ([]domain.Comment, error) {
	query := `
		SELECT c.id, c.post_id, p.title, c.author_name, c.author_email, c.author_site, c.content, c.is_approved, c.created_at
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		ORDER BY c.created_at DESC
		LIMIT 100
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.PostTitle, &c.AuthorName, &c.AuthorEmail, &c.AuthorSite, &c.Content, &c.IsApproved, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

// Approve 审核通过
func (r *CommentRepository) Approve(id int64) error {
	_, err := r.db.Exec("UPDATE comments SET is_approved = 1 WHERE id = ?", id)
	return err
}

// Delete 删除评论
func (r *CommentRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM comments WHERE id = ?", id)
	return err
}
