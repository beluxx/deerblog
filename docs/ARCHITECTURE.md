# 个人博客系统架构设计说明书 (Architecture Design Document)

## 1. 核心设计原则与哲学

本博客系统致力于打造一个**极致性能（High Performance）**、**现代高质感视觉（Aesthetic UI）**、**极简运维（Single Binary）**的全栈内容管理与发布平台。

核心架构哲学遵循：
1. **Zero CGO 铁律**：所有 Go 依赖库必须为 100% 纯 Go（Pure Go）实现。编译时保证 `CGO_ENABLED=0`，实现完全静态链接，零外部 libc 依赖，可跨平台任意分发部署。
2. **GOTH 范式（Go + Templ/Tailwind + Chi + htmx）**：
   - 彻底摆脱传统前端 SPA 庞大的运行时与构建复杂性。
   - 页面首次加载为**完整服务端直出 HTML（SSR）**，TTFB < 5ms，SEO 天然满分。
   - 页面内部交互（点赞、评论、搜索过滤、后台草稿保存）通过 **htmx** 驱动 DOM 局部替换，保持 SPA 级丝滑无感交互。
3. **单二进制交付（Single Binary Distribution）**：
   - 利用 Go 的 `//go:embed` 技术，将编译生成的 Tailwind CSS、静态资源与 HTML 模板完全打包进单一可执行文件。
   - 生产环境不需要安装 Node.js、不需要配置反向代理静态目录、不需要额外安装外部数据库服务。

---

## 2. 核心技术栈清单

| 分层维度 | 技术选型 | 纯 Go / 零 CGO 验证 | 核心用途与选型理由 |
| :--- | :--- | :--- | :--- |
| **HTTP 路由** | `go-chi/chi/v5` | ✅ 纯 Go | 极轻量、零内存分配、100% 兼容 Go 原生 `net/http` 中间件体系。 |
| **关系型主数据库** | `modernc.org/sqlite` | ✅ 纯 Go | 无需 gcc/CGO 的 SQLite 纯 Go 驱动，用于存储文章、分类、标签、评论等结构化业务数据。 |
| **KV 高速缓存/统计**| `go.etcd.io/bbolt` | ✅ 纯 Go | 纯 Go 实现的单文件嵌入式 ACID KV 数据库，用于存储访问计数、防爆破速率限制、会话 Session 与临时缓存。 |
| **页面交互** | `htmx v2.x` | ✅ 仅 14KB gzip | HTML 属性驱动异步通信与 DOM Swap，无需编写 Fetch 与前端状态管理。 |
| **样式与视觉设计** | `Tailwind CSS v3/v4` | 编译期工具 | 现代原子化设计系统，深色模式采用 OKLCH 色彩空间，打造 Editorial / Craft 顶级质感。 |
| **Markdown 与高亮** | `goldmark` + `chroma/v2` | ✅ 纯 Go | Hugo 同款引擎。在 Go 服务端毫秒级完成 Markdown 转换与语法高亮，客户端 0KB 高亮 JS 负担。 |
| **安全与认证** | 纯 Go 原生 Cookie + HMAC-SHA256 / bbolt session | ✅ 纯 Go | HttpOnly + SameSite 安全会话，防 XSS/CSRF。 |

---

## 3. 总体系统拓扑与请求流转

```mermaid
flowchart TD
    subgraph Client ["浏览器客户端 (Browser)"]
        HTML[纯 HTML + 内联/缓存 CSS]
        HTMX[htmx (仅 14KB)]
    end

    subgraph GoApp ["Golang 单二进制博客服务 (CGO_ENABLED=0)"]
        subgraph ChiRouter ["Chi HTTP 路由 & 中间件"]
            MW[Logger / Recover / RateLimit / SecurityHeaders]
            AuthMW[Admin 鉴权中间件 (Cookie + Session)]
        end

        subgraph Handlers ["业务控制层 (Handlers)"]
            BlogH[前台页面 Handler: 首页 / 文章 / 归档 / 搜索]
            AdminH[后台管理 Handler: 文章 CRUD / 评论审核 / 设置]
            HtmxH[htmx 片段 Handler: 点赞 / 评论增量追加 / 实时搜索]
        end

        subgraph Service ["业务领域层 (Service)"]
            MDService[Markdown 解析 (goldmark) + 服务端语法高亮 (chroma)]
            StatsService[访问热度与点赞计数]
        end

        subgraph StorageLayer ["存储层 (Pure Go Storage)"]
            SqliteDB[(SQLite3: modernc.org/sqlite<br>文章/分类/标签/评论/用户)]
            BoltKV[(BoltDB: go.etcd.io/bbolt<br>访问日志/KV缓存/Session)]
        end

        subgraph EmbedFS ["嵌入静态资源 (go:embed)"]
            Assets[Tailwind CSS / htmx.js / Favicon]
        end
    end

    Client -- "首次访问 (GET)" --> MW --> BlogH --> MDService --> SqliteDB
    BlogH -- "完整 HTML 直出 (SSR, TTFB < 5ms)" --> Client
    
    HTMX -- "局部交互 (hx-post / hx-get)" --> MW --> HtmxH --> StorageLayer
    HtmxH -- "HTML 片段 (hx-swap outerHTML/beforeend)" --> HTMX

    AdminH -- "受保护路由" --> AuthMW --> SqliteDB
```

---

## 4. 存储架构与数据模型设计

### 4.1 关系型数据（SQLite3 - `modernc.org/sqlite`）

```sql
-- 1. 用户表 (管理员)
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nickname TEXT NOT NULL,
    avatar_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 2. 分类表
CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 3. 标签表
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE
);

-- 4. 文章表
CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    summary TEXT,
    content_raw TEXT NOT NULL,       -- Markdown 原文
    content_html TEXT NOT NULL,      -- goldmark + chroma 预渲染 HTML 缓存
    cover_image TEXT,
    category_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
    is_published BOOLEAN DEFAULT FALSE,
    published_at DATETIME,
    views_count INTEGER DEFAULT 0,
    likes_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug);
CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(is_published, published_at DESC);

-- 5. 文章与标签多对多关联表
CREATE TABLE IF NOT EXISTS post_tags (
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

-- 6. 评论表
CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_site TEXT,
    content TEXT NOT NULL,
    is_approved BOOLEAN DEFAULT TRUE,
    ip_address TEXT,
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id, is_approved);
```

### 4.2 KV 存储（BoltDB - `go.etcd.io/bbolt`）

Bucket 规划：
* `sessions`：管理员登录 Session（Token -> UserID, Expiration）
* `rate_limits`：IP 防刷防垃圾评论频次控制
* `post_views`：未入库的并发阅读量内存计数器（定期批量写回 SQLite）
* `site_meta`：全局站点设置（站点标题、副标题、关于我、社交链接等）

---

## 5. 项目目录分层设计

```text
.
├── cmd/
│   └── server/
│       └── main.go              # 程序入口、依赖注入、优雅退出
├── internal/
│   ├── config/                  # 配置加载 (环境变量/默认配置)
│   ├── database/                # SQLite (modernc) 与 BoltDB 实例管理与 Migration
│   ├── domain/                  # 核心实体结构体定义 (Post, User, Comment 等)
│   ├── repository/              # 数据库数据访问层 (Sqlite & Bolt 实现)
│   ├── service/                 # 业务逻辑层 (Markdown/Chroma 高亮、Auth 鉴权)
│   ├── handler/                 # Chi HTTP 路由处理
│   │   ├── blog.go              # 前台页面处理 (首页, 文章详情, 分类, 搜索)
│   │   ├── admin.go             # 后台 CMS 处理 (管理面板, 文章编辑, 评论)
│   │   ├── htmx.go              # 针对 htmx 的 HTML 片段接口 (点赞, 局部搜索, 异步评论)
│   │   └── middleware.go        # 鉴权、安全标头、防刷中间件
│   └── views/                   # 视图模板层 (包含基础布局、组件、完整页面)
├── static/                      # 静态资源 (输出的 tailwind.css, htmx.min.js, 图标等)
├── docs/                        # 架构与开发文档
│   ├── ARCHITECTURE.md          # 本架构设计文档
│   └── AGENTS.md                # AI 与开发者工程规范手册
├── Makefile                     # 构建脚本 (CSS 编译, 纯 Go 编译)
├── go.mod
└── go.sum
```

---

## 6. 构建与生产部署流水线

### 编译指令（严格遵循 Zero CGO）

```bash
# 1. 编译 Tailwind CSS (生成最小化样式文件至 static/css/tailwind.min.css)
npx tailwindcss -i ./web/css/input.css -o ./static/css/tailwind.min.css --minify

# 2. 纯 Go 静态编译可执行文件 (零 C 库依赖)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/blog-server ./cmd/server
```

生成结果：只有一个独立的二进制文件 `./bin/blog-server`。
运行内存：约 **15MB ~ 25MB**。
启动时间：**< 10ms**。
部署方式：直接复制到 Linux 服务器，配置一个 `systemd` 服务即可常年免维护运行。
