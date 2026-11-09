# 极速、极简、高质感个人博客系统

基于 **Golang (chi) + htmx + Tailwind CSS** 的现代化全栈个人博客系统。采用 **Zero CGO（纯 Go 实现）**，单二进制打包分发，具备毫秒级响应、沉浸式阅读审美与极低维护成本。

---

## 🌟 核心特性

- ⚡ **极致性能**：全站服务端渲染（SSR），TTFB（首字节时间）低于 5ms，原生 SEO 满分支持。
- 🪄 **htmx 驱动**：点赞、评论、分类过滤、实时搜索等局部动态刷新，免写前端状态管理，享受 SPA 级丝滑交互。
- 🎨 **Editorial 现代排版**：Tailwind CSS 精心调校的排版间距、暗黑模式支持与毛玻璃微交互。
- 📦 **纯 Go 单二进制交付（Zero CGO）**：
  - 关系型数据：**SQLite3**（纯 Go 驱动 `modernc.org/sqlite`，绝不依赖 gcc/CGO）。
  - KV 高速缓存：**BoltDB**（纯 Go `go.etcd.io/bbolt`，存储 Session、访问计数与防刷限流）。
  - `CGO_ENABLED=0` 静态编译，全站静态资源及模板内嵌（`//go:embed`），直接生成单一可执行文件。
- 🖋️ **服务端语法高亮**：采用 `goldmark` + `chroma/v2`，服务端直出 VS Code 级语法高亮，客户端 0KB 高亮 JS 负担。

---

## 📂 项目结构概览

```text
├── AGENTS.md                  # 开发者与智能体开发规范准则
├── docs/
│   └── ARCHITECTURE.md        # 详细系统架构设计与数据模型说明书
├── cmd/
│   └── server/main.go         # 应用程序入口
├── internal/
│   ├── config/                # 配置读取
│   ├── database/              # SQLite (modernc) 与 BoltDB 初始化
│   ├── domain/                # 领域实体模型 (Post, Category, Comment, User)
│   ├── repository/            # 数据访问层
│   ├── service/               # 业务逻辑 (Markdown 解析、Chroma 高亮、Auth)
│   ├── handler/               # Chi 路由与 htmx 处理器
│   └── views/                 # HTML 模板与组件
├── static/                    # 静态资源 (Tailwind CSS, htmx.min.js, 图标等)
├── Makefile                   # 一键编译与样式构建
├── go.mod
└── README.md
```

---

## 🚀 快速上手与验证

### 1. 编译样式 (Tailwind CSS)
```bash
npx tailwindcss -i ./web/css/input.css -o ./static/css/tailwind.min.css --minify
```

### 2. 本地开发运行
```bash
go run ./cmd/server/main.go
```

### 3. 生产静态单二进制打包 (验证 Zero CGO)
```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/blog-server ./cmd/server
```

详细架构规范请参阅 [ARCHITECTURE.md](file:///home/antigravity/docs/ARCHITECTURE.md) 与 [AGENTS.md](file:///home/antigravity/AGENTS.md)。
