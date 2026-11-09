package handler

import (
	"html"
	"net/http"
	"strings"
	"time"

	"blog/internal/domain"
	"blog/internal/repository"
	"blog/internal/service"
	"blog/internal/views"
	"github.com/go-chi/chi/v5"
)

type HtmxHandler struct {
	views       *views.ViewEngine
	postSvc     *service.PostService
	postRepo    *repository.PostRepository
	commentRepo *repository.CommentRepository
}

func NewHtmxHandler(
	views *views.ViewEngine,
	postSvc *service.PostService,
	postRepo *repository.PostRepository,
	commentRepo *repository.CommentRepository,
) *HtmxHandler {
	return &HtmxHandler{
		views:       views,
		postSvc:     postSvc,
		postRepo:    postRepo,
		commentRepo: commentRepo,
	}
}

// LikePost 处理文章点赞，返回局部按钮 HTML 片段
func (h *HtmxHandler) LikePost(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	likes, err := h.postSvc.LikePost(slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := domain.Post{
		Slug:       slug,
		LikesCount: likes,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.views.RenderPartial(w, "like_button.html", p)
}

// CreateComment 发表评论，返回新评论条目 HTML 片段 (直接追加在评论列表末尾)
func (h *HtmxHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post, err := h.postRepo.GetBySlug(slug, true)
	if err != nil || post == nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	authorName := strings.TrimSpace(r.FormValue("author_name"))
	authorEmail := strings.TrimSpace(r.FormValue("author_email"))
	authorSite := strings.TrimSpace(r.FormValue("author_site"))
	content := strings.TrimSpace(r.FormValue("content"))

	if authorName == "" || authorEmail == "" || content == "" {
		http.Error(w, "必填项不能为空", http.StatusBadRequest)
		return
	}

	// XSS 安全过滤
	c := &domain.Comment{
		PostID:      post.ID,
		AuthorName:  html.EscapeString(authorName),
		AuthorEmail: html.EscapeString(authorEmail),
		AuthorSite:  html.EscapeString(authorSite),
		Content:     html.EscapeString(content),
		IsApproved:  true, // 默认直接通过展示
		CreatedAt:   time.Now(),
		IPAddress:   r.RemoteAddr,
		UserAgent:   r.UserAgent(),
	}

	id, err := h.commentRepo.Create(c)
	if err != nil {
		http.Error(w, "评论保存失败", http.StatusInternalServerError)
		return
	}
	c.ID = id

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.views.RenderPartial(w, "comment_item.html", c)
}

// Search 实时关键词防抖搜索，返回搜索结果列表 HTML 片段
func (h *HtmxHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var posts []domain.Post
	if q != "" {
		p, _, err := h.postRepo.ListPublished(10, 0, "", "", q)
		if err == nil {
			posts = p
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.views.RenderPartial(w, "search_results.html", map[string]any{
		"Posts": posts,
	})
}
