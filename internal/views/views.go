package views

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

// ViewEngine 模板渲染引擎
type ViewEngine struct {
	templates map[string]*template.Template
}

func NewViewEngine() (*ViewEngine, error) {
	ve := &ViewEngine{
		templates: make(map[string]*template.Template),
	}

	funcMap := template.FuncMap{
		"formatDate": func(t time.Time, layout string) string {
			return t.Format(layout)
		},
		"isSameCategory": func(catID *int64, targetID int64) bool {
			if catID == nil {
				return false
			}
			return *catID == targetID
		},
	}

	// 1. 前台页面列表（包装在 base.html 中）
	baseLayout := "templates/layouts/base.html"
	partials := []string{
		"templates/partials/like_button.html",
		"templates/partials/comment_item.html",
		"templates/partials/search_results.html",
	}

	blogPages := []string{
		"home.html",
		"post.html",
		"archive.html",
	}

	for _, page := range blogPages {
		pagePath := path.Join("templates/pages", page)
		files := append([]string{baseLayout, pagePath}, partials...)
		tmpl, err := template.New(page).Funcs(funcMap).ParseFS(templateFS, files...)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", page, err)
		}
		ve.templates[page] = tmpl
	}

	// 2. 后台页面列表（包装在 admin.html 中）
	adminLayout := "templates/layouts/admin.html"
	adminPages := []string{
		"admin/dashboard.html",
		"admin/posts.html",
		"admin/post_edit.html",
		"admin/comments.html",
		"admin/categories.html",
		"admin/settings.html",
	}

	for _, page := range adminPages {
		pagePath := path.Join("templates/pages", page)
		files := []string{adminLayout, pagePath}
		tmpl, err := template.New(path.Base(page)).Funcs(funcMap).ParseFS(templateFS, files...)
		if err != nil {
			return nil, fmt.Errorf("failed to parse admin template %s: %w", page, err)
		}
		ve.templates[page] = tmpl
	}

	// 3. 独立登录页面 (无 layout)
	loginTmpl, err := template.New("login.html").Funcs(funcMap).ParseFS(templateFS, "templates/pages/admin/login.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse login template: %w", err)
	}
	ve.templates["admin/login.html"] = loginTmpl

	// 4. htmx 独立片段
	partialFiles := []string{
		"like_button.html",
		"comment_item.html",
		"search_results.html",
		"markdown_preview.html",
	}
	for _, p := range partialFiles {
		pPath := path.Join("templates/partials", p)
		tmpl, err := template.New(p).Funcs(funcMap).ParseFS(templateFS, pPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse partial %s: %w", p, err)
		}
		ve.templates[p] = tmpl
	}

	return ve, nil
}

// RenderPage 渲染完整页面
func (ve *ViewEngine) RenderPage(w http.ResponseWriter, name string, data any) error {
	tmpl, ok := ve.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 如果是 layout 包装的页面，执行根模板（base.html 或 admin.html），否则执行 name
	if name == "admin/login.html" {
		return tmpl.Execute(w, data)
	}
	if tmpl.Lookup("base.html") != nil {
		return tmpl.ExecuteTemplate(w, "base.html", data)
	}
	if tmpl.Lookup("admin.html") != nil {
		return tmpl.ExecuteTemplate(w, "admin.html", data)
	}
	return tmpl.Execute(w, data)
}

// RenderPartial 渲染 htmx 局部片段
func (ve *ViewEngine) RenderPartial(w io.Writer, name string, data any) error {
	tmpl, ok := ve.templates[name]
	if !ok {
		return fmt.Errorf("partial template %s not found", name)
	}
	return tmpl.Execute(w, data)
}
