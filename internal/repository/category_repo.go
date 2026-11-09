package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"blog/internal/domain"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) ListAll() ([]domain.Category, error) {
	query := `
		SELECT c.id, c.name, c.slug, c.description, c.created_at,
		       (SELECT COUNT(*) FROM posts p WHERE p.category_id = c.id AND p.is_published = 1) as post_count
		FROM categories c
		ORDER BY post_count DESC, c.name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var list []domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt, &c.PostCount); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *CategoryRepository) GetBySlug(slug string) (*domain.Category, error) {
	row := r.db.QueryRow(`
		SELECT id, name, slug, description, created_at
		FROM categories WHERE slug = ?
	`, slug)
	var c domain.Category
	err := row.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) Create(name, slug, description string) error {
	_, err := r.db.Exec(`
		INSERT INTO categories (name, slug, description) VALUES (?, ?, ?)
	`, name, slug, description)
	return err
}

func (r *CategoryRepository) Update(id int64, name, slug, description string) error {
	_, err := r.db.Exec(`
		UPDATE categories SET name = ?, slug = ?, description = ? WHERE id = ?
	`, name, slug, description, id)
	return err
}

func (r *CategoryRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	return err
}

// TagRepository 处理文章标签
type TagRepository struct {
	db *sql.DB
}

func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) ListAll() ([]domain.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug,
		       (SELECT COUNT(*) FROM post_tags pt JOIN posts p ON p.id = pt.post_id WHERE pt.tag_id = t.id AND p.is_published = 1) as post_count
		FROM tags t
		ORDER BY post_count DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.PostCount); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}

func (r *TagRepository) GetOrCreate(name, slug string) (int64, error) {
	var id int64
	err := r.db.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	res, err := r.db.Exec("INSERT INTO tags (name, slug) VALUES (?, ?)", name, slug)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
