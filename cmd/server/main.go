package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"blog/internal/config"
	"blog/internal/database"
	"blog/internal/domain"
	"blog/internal/handler"
	"blog/internal/repository"
	"blog/internal/service"
	"blog/internal/views"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

var Version = "dev"

func main() {
	cfg := config.Load()
	log.Printf("[Init] DeerBlog 版本: %s, 监听端口 :%s", Version, cfg.Port)

	// 1. 初始化纯 Go SQLite 数据库
	sqliteDB, err := database.InitSQLite(cfg.SqlitePath, cfg.AdminUser, cfg.AdminPass)
	if err != nil {
		log.Fatalf("[Fatal] SQLite 初始化失败: %v", err)
	}
	defer sqliteDB.Close()
	log.Printf("[Storage] SQLite (Zero CGO) 已就绪: %s", cfg.SqlitePath)

	// 2. 初始化纯 Go BoltDB KV 存储
	boltDB, err := database.InitBolt(cfg.BoltPath)
	if err != nil {
		log.Fatalf("[Fatal] BoltDB 初始化失败: %v", err)
	}
	defer boltDB.Close()
	log.Printf("[Storage] BoltDB KV 缓存已就绪: %s", cfg.BoltPath)

	// 3. 初始化 Repository
	userRepo := repository.NewUserRepository(sqliteDB)
	catRepo := repository.NewCategoryRepository(sqliteDB)
	tagRepo := repository.NewTagRepository(sqliteDB)
	postRepo := repository.NewPostRepository(sqliteDB)
	commentRepo := repository.NewCommentRepository(sqliteDB)
	sessionRepo := repository.NewSessionRepository(boltDB)
	settingsRepo := repository.NewSettingsRepository(boltDB, cfg)

	// 4. 初始化 Service
	markdownSvc := service.NewMarkdownService()
	authSvc := service.NewAuthService(userRepo, sessionRepo)
	postSvc := service.NewPostService(postRepo, markdownSvc)
	feedSvc := service.NewFeedService(cfg)
	uploadSvc := service.NewUploadService("static/uploads")

	// 5. 初始化 View Engine
	viewEngine, err := views.NewViewEngine()
	if err != nil {
		log.Fatalf("[Fatal] 模板引擎初始化失败: %v", err)
	}

	// 6. 填充初始演示文章 (如果数据库尚无文章)
	seedInitialData(postRepo, catRepo, tagRepo, postSvc)

	// 7. 初始化 Handlers 与中间件
	authMW := handler.NewAuthMiddleware(authSvc)
	blogH := handler.NewBlogHandler(cfg, viewEngine, postSvc, feedSvc, postRepo, catRepo, tagRepo, commentRepo, settingsRepo)
	htmxH := handler.NewHtmxHandler(viewEngine, postSvc, postRepo, commentRepo)
	adminH := handler.NewAdminHandler(cfg, viewEngine, authSvc, postSvc, uploadSvc, postRepo, catRepo, tagRepo, commentRepo, settingsRepo)

	// 8. 配置 Chi 路由体系
	r := chi.NewRouter()

	// 核心全局中间件
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(handler.SecurityHeaders)

	// 静态文件服务
	workDir, _ := os.Getwd()
	filesDir := http.Dir(workDir + "/static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(filesDir)))

	// 博客前台路由
	r.Get("/", blogH.Home)
	r.Get("/posts/{slug}", blogH.PostDetail)
	r.Get("/category/{slug}", blogH.CategoryPosts)
	r.Get("/tag/{slug}", blogH.TagPosts)
	r.Get("/archive", blogH.Archive)

	// RSS 订阅源与 SEO 路由
	r.Get("/feed.xml", blogH.Feed)
	r.Get("/rss.xml", blogH.Feed)
	r.Get("/sitemap.xml", blogH.Sitemap)
	r.Get("/robots.txt", blogH.Robots)

	// htmx 异步交互 API
	r.Route("/api", func(api chi.Router) {
		api.Post("/posts/{slug}/like", htmxH.LikePost)
		api.Post("/posts/{slug}/comments", htmxH.CreateComment)
		api.Get("/search", htmxH.Search)
	})

	// 管理后台路由
	r.Route("/admin", func(admin chi.Router) {
		// 登录认证
		admin.Get("/login", adminH.LoginPage)
		admin.Post("/login", adminH.LoginSubmit)
		admin.Post("/logout", adminH.Logout)

		// 受保护后台管理路由组
		admin.Group(func(protected chi.Router) {
			protected.Use(authMW.RequireAuth)

			// 仪表盘
			protected.Get("/dashboard", adminH.Dashboard)

			// 文章管理与实时预览
			protected.Get("/posts", adminH.PostsList)
			protected.Get("/posts/new", adminH.PostNew)
			protected.Post("/posts/new", adminH.PostSave)
			protected.Get("/posts/{id}/edit", adminH.PostEdit)
			protected.Post("/posts/{id}/edit", adminH.PostSave)
			protected.Post("/posts/{id}/delete", adminH.PostDelete)
			protected.Post("/posts/preview", adminH.PostPreview)

			// 本地图片媒体上传接口
			protected.Post("/api/upload", adminH.UploadImage)

			// 分类管理
			protected.Get("/categories", adminH.CategoriesList)
			protected.Post("/categories/new", adminH.CategoryCreate)
			protected.Post("/categories/{id}/delete", adminH.CategoryDelete)

			// 评论管理
			protected.Get("/comments", adminH.CommentsList)
			protected.Post("/comments/{id}/approve", adminH.CommentApprove)
			protected.Post("/comments/{id}/delete", adminH.CommentDelete)

			// 动态站点设置
			protected.Get("/settings", adminH.SettingsPage)
			protected.Post("/settings", adminH.SettingsSave)
		})
	})

	// 9. 启动服务与优雅停机
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("==================================================================")
		log.Printf(" 🚀 博客系统已启动成功！")
		log.Printf(" • 博客前台地址: http://localhost:%s", cfg.Port)
		log.Printf(" • 管理后台地址: http://localhost:%s/admin/login", cfg.Port)
		log.Printf(" • 纯 Go 零 CGO 模式运行中...")
		log.Printf("==================================================================")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[Error] HTTP server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[Shutdown] 正在优雅关闭服务...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[Error] Server forced to shutdown: %v", err)
	}
	log.Println("[Shutdown] 服务已安全退出。")
}

// 自动写入一篇精彩的演示文章
func seedInitialData(postRepo *repository.PostRepository, catRepo *repository.CategoryRepository, tagRepo *repository.TagRepository, postSvc *service.PostService) {
	posts, _, err := postRepo.ListPublished(1, 0, "", "", "")
	if err == nil && len(posts) > 0 {
		return
	}

	_ = catRepo.Create("系统架构", "architecture", "关于分布式、单机高性能与极简架构设计的思考")
	_ = catRepo.Create("Go语言", "golang", "Go 语言核心深度剖析与最佳工程实践")
	cats, _ := catRepo.ListAll()
	var catID *int64
	if len(cats) > 0 {
		catID = &cats[0].ID
	}

	t1, _ := tagRepo.GetOrCreate("Golang", "golang")
	t2, _ := tagRepo.GetOrCreate("htmx", "htmx")
	t3, _ := tagRepo.GetOrCreate("Zero-CGO", "zero-cgo")

	demoContent := `# 构建现代高质感博客：GOTH 架构实战

在这篇文章中，我们将探讨为什么 **Go (Chi) + htmx + Tailwind CSS** 是当下独立开发者和全栈工程师构建高性能博客的最佳组合。

## 为什么告别沉重的单页应用 (SPA)？

在过去的十年里，前端生态经历了空前的膨胀。为了实现简单的页面更新，我们不得不引入：
- 数百兆的 node_modules 依赖；
- 复杂的 Webpack / Vite 打包配置；
- 双重路由与客户端水合（Hydration）不匹配；
- 臃肿的客户端运行时 JavaScript。

> 真正的工程智慧不是在架构中不断增加复杂度，而是用最克制的代码实现最极致的体验。

## 纯 Go 服务端代码高亮 (Chroma)

借助纯 Go 实现的 Chroma 引擎，高亮在服务端毫秒级直出，浏览器收到的是纯静态的带颜色 HTML，**客户端加载 0KB 高亮 JS 负担**：

` + "```go" + `
package main

import (
	"fmt"
	"net/http"
	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, High Performance Pure Go Blog!"))
	})
	http.ListenAndServe(":8080", r)
}
` + "```" + `

## 零 CGO 单二进制交付

通过纯 Go 编写的 SQLite 驱动 modernc.org/sqlite 以及嵌入式 KV bbolt，我们可以直接使用：

` + "```bash" + `
CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/blog ./cmd/server
` + "```" + `

直接编译生成一个独立的二进制文件，完全摆脱了本地 libc 和外部 gcc/clang 的依赖束缚，随时随地一键分发！

## 总结

精简的工程链路带来了飞快的响应速度、更少的维护心智负担与极其愉悦的写作体验。欢迎在下方点赞或留下你的真知灼见！`

	demoPost := &domain.Post{
		Title:       "构建现代高质感博客：GOTH 架构实战",
		Slug:        "modern-goth-blog-architecture",
		ContentRaw:  demoContent,
		CategoryID:  catID,
		IsPublished: true,
	}

	_, _ = postSvc.SavePost(demoPost, []int64{t1, t2, t3})
	log.Printf("[Init] 已自动初始化首篇示范博文: %s", demoPost.Title)
}
