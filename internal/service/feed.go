package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"time"

	"blog/internal/config"
	"blog/internal/domain"
)

type FeedService struct {
	cfg *config.Config
}

func NewFeedService(cfg *config.Config) *FeedService {
	return &FeedService{cfg: cfg}
}

// RSS 2.0 数据结构
type rssChannel struct {
	XMLName       xml.Name  `xml:"channel"`
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Generator     string    `xml:"generator"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Guid        string `xml:"guid"`
}

type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

// GenerateRSS 生成符合 RSS 2.0 标准规范的 XML 字符串
func (s *FeedService) GenerateRSS(baseURL string, posts []domain.Post) ([]byte, error) {
	var items []rssItem
	for _, p := range posts {
		pubTime := p.CreatedAt
		if p.PublishedAt != nil {
			pubTime = *p.PublishedAt
		}
		itemURL := fmt.Sprintf("%s/posts/%s", baseURL, p.Slug)
		items = append(items, rssItem{
			Title:       p.Title,
			Link:        itemURL,
			Description: html.EscapeString(p.Summary),
			PubDate:     pubTime.Format(time.RFC1123Z),
			Guid:        itemURL,
		})
	}

	feed := rssRoot{
		Version: "2.0",
		Channel: rssChannel{
			Title:         s.cfg.SiteTitle,
			Link:          baseURL,
			Description:   s.cfg.SiteSubtitle,
			Language:      "zh-CN",
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Generator:     "Golang Chi Pure-Go Blog Engine",
			Items:         items,
		},
	}

	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Sitemap 结构
type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemapIndex struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// GenerateSitemap 生成标准搜索引擎 Sitemap.xml
func (s *FeedService) GenerateSitemap(baseURL string, posts []domain.Post) ([]byte, error) {
	now := time.Now().Format("2006-01-02")
	urls := []sitemapURL{
		{
			Loc:        baseURL + "/",
			LastMod:    now,
			ChangeFreq: "daily",
			Priority:   "1.0",
		},
		{
			Loc:        baseURL + "/archive",
			LastMod:    now,
			ChangeFreq: "weekly",
			Priority:   "0.8",
		},
	}

	for _, p := range posts {
		modTime := p.UpdatedAt.Format("2006-01-02")
		urls = append(urls, sitemapURL{
			Loc:        fmt.Sprintf("%s/posts/%s", baseURL, p.Slug),
			LastMod:    modTime,
			ChangeFreq: "weekly",
			Priority:   "0.9",
		})
	}

	root := sitemapIndex{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
