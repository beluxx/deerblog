package service

import (
	"strings"
	"testing"
	"time"

	"blog/internal/config"
	"blog/internal/domain"
)

func TestFeedService_GenerateRSS(t *testing.T) {
	cfg := &config.Config{
		SiteTitle:    "极客博客",
		SiteSubtitle: "技术与架构",
	}
	svc := NewFeedService(cfg)

	posts := []domain.Post{
		{
			Title:     "测试文章1",
			Slug:      "test-post-1",
			Summary:   "文章简介内容",
			CreatedAt: time.Now(),
		},
	}

	xmlBytes, err := svc.GenerateRSS("https://example.com", posts)
	if err != nil {
		t.Fatalf("failed to generate RSS: %v", err)
	}

	content := string(xmlBytes)
	if !strings.Contains(content, "<title>极客博客</title>") {
		t.Errorf("RSS missing title: %s", content)
	}
	if !strings.Contains(content, "<link>https://example.com/posts/test-post-1</link>") {
		t.Errorf("RSS missing post link: %s", content)
	}
}

func TestFeedService_GenerateSitemap(t *testing.T) {
	cfg := &config.Config{SiteTitle: "极客博客"}
	svc := NewFeedService(cfg)

	posts := []domain.Post{
		{
			Title:     "测试文章2",
			Slug:      "test-post-2",
			UpdatedAt: time.Now(),
		},
	}

	xmlBytes, err := svc.GenerateSitemap("https://example.com", posts)
	if err != nil {
		t.Fatalf("failed to generate sitemap: %v", err)
	}

	content := string(xmlBytes)
	if !strings.Contains(content, "http://www.sitemaps.org/schemas/sitemap/0.9") {
		t.Errorf("Sitemap missing xmlns schema: %s", content)
	}
	if !strings.Contains(content, "https://example.com/posts/test-post-2") {
		t.Errorf("Sitemap missing post loc: %s", content)
	}
}
