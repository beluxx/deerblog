package handler

import (
	"fmt"
	"net/http"
	"time"

	"blog/internal/config"
	"blog/internal/repository"
	"blog/internal/service"
	"blog/internal/views"
	"github.com/go-chi/chi/v5"
)

type BlogHandler struct {
	cfg          *config.Config
	views        *views.ViewEngine
	postSvc      *service.PostService
	feedSvc      *service.FeedService
	postRepo     *repository.PostRepository
	catRepo      *repository.CategoryRepository
	tagRepo      *repository.TagRepository
	commentRepo  *repository.CommentRepository
	settingsRepo *repository.SettingsRepository
}

func NewBlogHandler(
	cfg *config.Config,
	views *views.ViewEngine,
	postSvc *service.PostService,
	feedSvc *service.FeedService,
	postRepo *repository.PostRepository,
	catRepo *repository.CategoryRepository,
	tagRepo *repository.TagRepository,
	commentRepo *repository.CommentRepository,
	settingsRepo *repository.SettingsRepository,
) *BlogHandler {
	return &BlogHandler{
		cfg:          cfg,
		views:        views,
		postSvc:      postSvc,
		feedSvc:      feedSvc,
		postRepo:     postRepo,
		catRepo:      catRepo,
		tagRepo:      tagRepo,
		commentRepo:  commentRepo,
		settingsRepo: settingsRepo,
	}
}

// 基础模板上下文
func (h *BlogHandler) baseData(title, desc string) map[string]any {
	settings, err := h.settingsRepo.Get()
	siteData := any(h.cfg)
	if err == nil && settings != nil {
		siteData = settings
	}

	return map[string]any{
		"Site":            siteData,
		"Title":           title,
		"MetaDescription": desc,
		"CurrentYear":     time.Now().Year(),
	}
}

// Home 首页
func (h *BlogHandler) Home(w http.ResponseWriter, r *http.Request) {
	posts, total, err := h.postRepo.ListPublished(20, 0, "", "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categories, _ := h.catRepo.ListAll()
	tags, _ := h.tagRepo.ListAll()

	data := h.baseData("", h.cfg.SiteSubtitle)
	data["Posts"] = posts
	data["TotalPosts"] = total
	data["Categories"] = categories
	data["Tags"] = tags

	_ = h.views.RenderPage(w, "home.html", data)
}

// CategoryPosts 按分类查看文章
func (h *BlogHandler) CategoryPosts(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	cat, err := h.catRepo.GetBySlug(slug)
	if err != nil || cat == nil {
		http.NotFound(w, r)
		return
	}

	posts, total, err := h.postRepo.ListPublished(20, 0, slug, "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categories, _ := h.catRepo.ListAll()
	tags, _ := h.tagRepo.ListAll()

	data := h.baseData(cat.Name, cat.Description)
	data["Posts"] = posts
	data["TotalPosts"] = total
	data["ActiveCategory"] = cat
	data["Categories"] = categories
	data["Tags"] = tags

	_ = h.views.RenderPage(w, "home.html", data)
}

// TagPosts 按标签查看文章
func (h *BlogHandler) TagPosts(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	posts, total, err := h.postRepo.ListPublished(20, 0, "", slug, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	categories, _ := h.catRepo.ListAll()
	tags, _ := h.tagRepo.ListAll()

	data := h.baseData("#"+slug, "标签为 #"+slug+" 的文章合集")
	data["Posts"] = posts
	data["TotalPosts"] = total
	data["Categories"] = categories
	data["Tags"] = tags

	_ = h.views.RenderPage(w, "home.html", data)
}

// PostDetail 文章详情页
func (h *BlogHandler) PostDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	post, toc, err := h.postSvc.GetPostDetail(slug)
	if err != nil || post == nil {
		http.NotFound(w, r)
		return
	}

	comments, _ := h.commentRepo.ListByPostID(post.ID)

	data := h.baseData(post.Title, post.Summary)
	data["Post"] = post
	data["TOC"] = toc
	data["Comments"] = comments
	data["IsArticle"] = true

	_ = h.views.RenderPage(w, "post.html", data)
}

// Archive 归档页面
func (h *BlogHandler) Archive(w http.ResponseWriter, r *http.Request) {
	posts, _, err := h.postRepo.ListPublished(100, 0, "", "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := h.baseData("文章归档", "历年发表的技术博客归档")
	data["Posts"] = posts

	_ = h.views.RenderPage(w, "archive.html", data)
}

// Feed 输出 RSS 2.0 XML
func (h *BlogHandler) Feed(w http.ResponseWriter, r *http.Request) {
	posts, _, err := h.postRepo.ListPublished(20, 0, "", "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	xmlData, err := h.feedSvc.GenerateRSS(baseURL, posts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(xmlData)
}

// Sitemap 输出符合标准的 sitemap.xml
func (h *BlogHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	posts, _, err := h.postRepo.ListPublished(500, 0, "", "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	xmlData, err := h.feedSvc.GenerateSitemap(baseURL, posts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write(xmlData)
}

// Robots 输出爬虫 robots.txt
func (h *BlogHandler) Robots(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	content := fmt.Sprintf("User-agent: *\nAllow: /\nDisallow: /admin/\n\nSitemap: %s://%s/sitemap.xml\n", scheme, r.Host)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(content))
}
