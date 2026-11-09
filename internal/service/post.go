package service

import (
	"strings"
	"time"

	"blog/internal/domain"
	"blog/internal/repository"
)

type PostService struct {
	postRepo    *repository.PostRepository
	markdownSvc *MarkdownService
}

func NewPostService(postRepo *repository.PostRepository, markdownSvc *MarkdownService) *PostService {
	return &PostService{
		postRepo:    postRepo,
		markdownSvc: markdownSvc,
	}
}

// GetPostDetail 获取单篇文章前台阅读详情
func (s *PostService) GetPostDetail(slug string) (*domain.Post, []domain.TableOfContentItem, error) {
	post, err := s.postRepo.GetBySlug(slug, true)
	if err != nil {
		return nil, nil, err
	}
	if post == nil {
		return nil, nil, nil
	}

	// 重新计算目录 TOC 与阅读时长
	_, toc, minutes := s.markdownSvc.Render(post.ContentRaw)
	post.ReadingMinutes = minutes

	// 异步更新阅读量
	go func(id int64) {
		_ = s.postRepo.IncrementViews(id)
	}(post.ID)

	return post, toc, nil
}

// SavePost 保存或更新文章（管理员）
func (s *PostService) SavePost(p *domain.Post, tagIDs []int64) (int64, error) {
	// 渲染 Markdown 为 HTML 缓存
	htmlContent, _, minutes := s.markdownSvc.Render(p.ContentRaw)
	p.ContentHTML = htmlContent
	p.ReadingMinutes = minutes

	// 若未填写摘要，自动截取纯文本前 150 个字符
	if strings.TrimSpace(p.Summary) == "" {
		cleaned := strings.ReplaceAll(p.ContentRaw, "\n", " ")
		cleaned = strings.ReplaceAll(cleaned, "#", "")
		cleaned = strings.TrimSpace(cleaned)
		runes := []rune(cleaned)
		if len(runes) > 150 {
			p.Summary = string(runes[:150]) + "..."
		} else {
			p.Summary = string(runes)
		}
	}

	if p.ID == 0 {
		return s.postRepo.Create(p, tagIDs)
	}

	err := s.postRepo.Update(p, tagIDs)
	return p.ID, err
}

// LikePost 点赞
func (s *PostService) LikePost(slug string) (int, error) {
	return s.postRepo.IncrementLikes(slug)
}

// PreviewMarkdown 用于后台编辑实时预览
func (s *PostService) PreviewMarkdown(rawMD string) domain.Post {
	htmlContent, _, minutes := s.markdownSvc.Render(rawMD)
	now := time.Now()
	return domain.Post{
		Title:          "实时预览",
		ContentHTML:    htmlContent,
		ReadingMinutes: minutes,
		CreatedAt:      now,
	}
}
