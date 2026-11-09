package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"blog/internal/config"
	"blog/internal/domain"
	"blog/internal/repository"
	"blog/internal/service"
	"blog/internal/views"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	cfg          *config.Config
	views        *views.ViewEngine
	authSvc      *service.AuthService
	postSvc      *service.PostService
	uploadSvc    *service.UploadService
	postRepo     *repository.PostRepository
	catRepo      *repository.CategoryRepository
	tagRepo      *repository.TagRepository
	commentRepo  *repository.CommentRepository
	settingsRepo *repository.SettingsRepository
}

func NewAdminHandler(
	cfg *config.Config,
	views *views.ViewEngine,
	authSvc *service.AuthService,
	postSvc *service.PostService,
	uploadSvc *service.UploadService,
	postRepo *repository.PostRepository,
	catRepo *repository.CategoryRepository,
	tagRepo *repository.TagRepository,
	commentRepo *repository.CommentRepository,
	settingsRepo *repository.SettingsRepository,
) *AdminHandler {
	return &AdminHandler{
		cfg:          cfg,
		views:        views,
		authSvc:      authSvc,
		postSvc:      postSvc,
		uploadSvc:    uploadSvc,
		postRepo:     postRepo,
		catRepo:      catRepo,
		tagRepo:      tagRepo,
		commentRepo:  commentRepo,
		settingsRepo: settingsRepo,
	}
}

// LoginPage 登录页
func (h *AdminHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	// 如果已经登录，直接跳仪表盘
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		if _, err := h.authSvc.ValidateSession(cookie.Value); err == nil {
			http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
			return
		}
	}

	_ = h.views.RenderPage(w, "admin/login.html", map[string]any{
		"Site": h.cfg,
	})
}

// LoginSubmit 登录处理
func (h *AdminHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	token, _, err := h.authSvc.Login(username, password)
	if err != nil {
		_ = h.views.RenderPage(w, "admin/login.html", map[string]any{
			"Site":  h.cfg,
			"Error": "账号或密码错误，请重新输入",
		})
		return
	}

	// 写入 HttpOnly 会话 Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}

// Logout 注销退出
func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		_ = h.authSvc.Logout(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   SessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// Dashboard 仪表盘
func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.postRepo.GetDashboardStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = h.views.RenderPage(w, "admin/dashboard.html", map[string]any{
		"Title": "控制台仪表盘",
		"Stats": stats,
	})
}

// PostsList 文章管理表格
func (h *AdminHandler) PostsList(w http.ResponseWriter, r *http.Request) {
	posts, err := h.postRepo.ListAllForAdmin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = h.views.RenderPage(w, "admin/posts.html", map[string]any{
		"Title": "文章列表",
		"Posts": posts,
	})
}

// PostNew 新建文章页面
func (h *AdminHandler) PostNew(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catRepo.ListAll()
	if err != nil {
		log.Printf("[Admin] PostNew 获取分类列表失败: %v", err)
	}

	_ = h.views.RenderPage(w, "admin/post_edit.html", map[string]any{
		"Title":      "撰写新文章",
		"Post":       domain.Post{IsPublished: true},
		"Categories": categories,
		"TagNames":   "",
	})
}

// PostEdit 编辑已有文章页面
func (h *AdminHandler) PostEdit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的文章ID", http.StatusBadRequest)
		return
	}

	post, err := h.postRepo.GetByID(id)
	if err != nil || post == nil {
		http.NotFound(w, r)
		return
	}

	categories, err := h.catRepo.ListAll()
	if err != nil {
		log.Printf("[Admin] PostEdit 获取分类列表失败: %v", err)
	}

	var tagNames []string
	for _, t := range post.Tags {
		tagNames = append(tagNames, t.Name)
	}

	_ = h.views.RenderPage(w, "admin/post_edit.html", map[string]any{
		"Title":      "编辑文章: " + post.Title,
		"Post":       *post,
		"Categories": categories,
		"TagNames":   strings.Join(tagNames, ", "),
	})
}

// PostSave 保存文章 (新建或更新)
func (h *AdminHandler) PostSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	idStr := chi.URLParam(r, "id")
	var id int64
	if idStr != "" {
		id, _ = strconv.ParseInt(idStr, 10, 64)
	}

	title := strings.TrimSpace(r.FormValue("title"))
	slug := strings.TrimSpace(r.FormValue("slug"))
	contentRaw := r.FormValue("content_raw")
	isPublished := r.FormValue("is_published") == "true"
	catIDStr := r.FormValue("category_id")

	var catID *int64
	if catIDStr != "" {
		if cid, err := strconv.ParseInt(catIDStr, 10, 64); err == nil {
			catID = &cid
		}
	}

	// 标签处理
	tagInput := r.FormValue("tags")
	var tagIDs []int64
	if tagInput != "" {
		for _, raw := range strings.Split(tagInput, ",") {
			tagName := strings.TrimSpace(raw)
			if tagName != "" {
				tagSlug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))
				tid, err := h.tagRepo.GetOrCreate(tagName, tagSlug)
				if err == nil {
					tagIDs = append(tagIDs, tid)
				}
			}
		}
	}

	post := &domain.Post{
		ID:          id,
		Title:       title,
		Slug:        slug,
		ContentRaw:  contentRaw,
		CategoryID:  catID,
		IsPublished: isPublished,
	}

	_, err := h.postSvc.SavePost(post, tagIDs)
	if err != nil {
		http.Error(w, fmt.Sprintf("保存失败: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
}

// PostPreview 实时 Markdown 高亮预览 (由 htmx 触发)
func (h *AdminHandler) PostPreview(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	rawMD := r.FormValue("content_raw")

	preview := h.postSvc.PreviewMarkdown(rawMD)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.views.RenderPartial(w, "markdown_preview.html", map[string]any{
		"Post": preview,
	})
}

// PostDelete 删除文章
func (h *AdminHandler) PostDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的文章ID", http.StatusBadRequest)
		return
	}

	if err := h.postRepo.Delete(id); err != nil {
		log.Printf("[Admin] 删除文章 %d 失败: %v", id, err)
		http.Error(w, "删除文章失败", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
}

// CommentsList 评论列表
func (h *AdminHandler) CommentsList(w http.ResponseWriter, r *http.Request) {
	comments, err := h.commentRepo.ListAllForAdmin()
	if err != nil {
		log.Printf("[Admin] 获取评论列表失败: %v", err)
	}

	_ = h.views.RenderPage(w, "admin/comments.html", map[string]any{
		"Title":    "评论审核",
		"Comments": comments,
	})
}

// CommentApprove 批准评论
func (h *AdminHandler) CommentApprove(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的评论ID", http.StatusBadRequest)
		return
	}

	if err := h.commentRepo.Approve(id); err != nil {
		log.Printf("[Admin] 批准评论 %d 失败: %v", id, err)
		http.Error(w, "审核操作失败", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/comments", http.StatusSeeOther)
}

// CommentDelete 删除评论
func (h *AdminHandler) CommentDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的评论ID", http.StatusBadRequest)
		return
	}

	if err := h.commentRepo.Delete(id); err != nil {
		log.Printf("[Admin] 删除评论 %d 失败: %v", id, err)
		http.Error(w, "删除评论失败", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/comments", http.StatusSeeOther)
}

// UploadImage 异步上传图片接口
func (h *AdminHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// 最大解析 10MB 的表单数据
	if err := r.ParseMultipartForm(10 * 1024 * 1024); err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "无法解析上传数据: " + err.Error()})
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "未找到上传的文件"})
		return
	}
	defer file.Close()

	publicURL, err := h.uploadSvc.SaveImage(fileHeader)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"url":      publicURL,
		"filename": fileHeader.Filename,
	})
}

// CategoriesList 分类管理页
func (h *AdminHandler) CategoriesList(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catRepo.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = h.views.RenderPage(w, "admin/categories.html", map[string]any{
		"Title":      "分类管理",
		"Categories": categories,
	})
}

// CategoryCreate 创建新分类
func (h *AdminHandler) CategoryCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	slug := strings.TrimSpace(r.FormValue("slug"))
	desc := strings.TrimSpace(r.FormValue("description"))

	if name != "" && slug != "" {
		if err := h.catRepo.Create(name, slug, desc); err != nil {
			log.Printf("[Admin] 创建分类 %s 失败: %v", name, err)
			http.Error(w, "创建分类失败: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// CategoryDelete 删除分类
func (h *AdminHandler) CategoryDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的分类ID", http.StatusBadRequest)
		return
	}

	if err := h.catRepo.Delete(id); err != nil {
		log.Printf("[Admin] 删除分类 %d 失败: %v", id, err)
		http.Error(w, "删除分类失败", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
}

// SettingsPage 站点设置页
func (h *AdminHandler) SettingsPage(w http.ResponseWriter, r *http.Request) {
	settings, err := h.settingsRepo.Get()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = h.views.RenderPage(w, "admin/settings.html", map[string]any{
		"Title":    "站点设置",
		"Settings": settings,
	})
}

// SettingsSave 保存站点设置
func (h *AdminHandler) SettingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	settings := &domain.SiteSettings{
		SiteTitle:    strings.TrimSpace(r.FormValue("site_title")),
		SiteSubtitle: strings.TrimSpace(r.FormValue("site_subtitle")),
		AuthorName:   strings.TrimSpace(r.FormValue("author_name")),
		AuthorBio:    strings.TrimSpace(r.FormValue("author_bio")),
		GithubURL:    strings.TrimSpace(r.FormValue("github_url")),
		TwitterURL:   strings.TrimSpace(r.FormValue("twitter_url")),
		Email:        strings.TrimSpace(r.FormValue("email")),
	}

	if err := h.settingsRepo.Save(settings); err != nil {
		http.Error(w, fmt.Sprintf("保存失败: %v", err), http.StatusInternalServerError)
		return
	}

	_ = h.views.RenderPage(w, "admin/settings.html", map[string]any{
		"Title":    "站点设置",
		"Settings": settings,
		"Message":  "站点设置已成功保存并即时全站生效！",
	})
}
