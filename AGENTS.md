# AGENTS.md — 开发者与智能体开发协作规范

> 本文件是本项目智能体（Agent）和人类开发者的**最高工程规范指南**。在阅读、设计、修改或提交任何代码前，必须严格遵守本文件定义的规范。

---

## 1. 项目核心概述

这是一个基于 **Go (Chi) + htmx + Tailwind CSS** 的高性能全栈现代个人博客系统。
- **定位**：极致性能、毫秒级响应、顶级视觉质感、极简单二进制（Single Binary）零依赖部署。
- **架构**：服务端渲染（SSR）+ htmx 局部动态刷新，彻底告别沉重前端单页应用（SPA）依赖与构建地狱。

---

## 2. 绝对不可逾越的“铁律” (Hard Rules)

### 🔴 铁律一：严格禁止引入 CGO 依赖（Zero CGO）
- **禁止使用**：`github.com/mattn/go-sqlite3` 以及任何需要依赖本地 C 编译器（gcc/clang）的外部库。
- **必须使用**：**`modernc.org/sqlite`**（纯 Go 实现的 SQLite 驱动）；KV 存储使用 **`go.etcd.io/bbolt`**。
- **验证标准**：任何提交必须能够通过 `CGO_ENABLED=0 go build ./...` 编译成功！

### 🔴 铁律二：坚守轻量前端原则
- **严禁引入**：庞大的前端重型框架（如 React/Vue SPA 运行时、Webpack、外部重量级图表库等）。
- **交互边界**：所有动态异步交互（点赞、发表评论、列表过滤搜索、后台保存）必须优先使用 **htmx** 属性（`hx-post`, `hx-get`, `hx-target`, `hx-swap` 等）。
- **极简依赖**：前端仅允许引入微型库：`htmx.min.js` (~14KB) 及必要的轻量微交互脚本（如深色模式切换、纯原生少量脚本）。

### 🔴 铁律三：代码高亮一律在服务端完成
- **禁止**：向浏览器发送代码高亮客户端 JS（如 highlight.js / prism.js 运行时）。
- **必须**：在 Go 后端通过纯 Go 编写的 **`goldmark` + `chroma/v2`** 将 Markdown 和代码块预先渲染为携带色彩样式的纯 HTML。

### 🔴 铁律四：坚持高质感审美与设计细节 (Aesthetic UI)
- 所有界面使用 **Tailwind CSS**，遵循现代精致设计（Craft / Editorial 风格）：
  - 舒适正文行高、中西文混排留白、细腻圆角（`rounded-xl` / `rounded-2xl`）。
  - 暗黑模式必须支持并采用沉浸式配色（推荐中性灰/深黑 `zinc-900`/`zinc-950`，禁止刺眼纯黑或发灰混浊）。
  - 毛玻璃（`backdrop-blur-md`）、细边框（`border-zinc-200 dark:border-zinc-800`）。

---

## 3. 分层架构规范与代码组织

所有代码严格遵守 Clean Architecture 简化分层，禁止反向依赖：

```
internal/
├── domain/       # 领域核心：基础实体结构体 (Post, Category, Comment, User)
├── config/       # 配置层：从环境变量与默认值读取配置
├── database/     # 纯 Go SQLite (modernc) 与 BoltDB 实例初始化及 Migration
├── repository/   # 数据访问层：执行具体的 SQL / KV 操作，返回 domain 实体
├── service/      # 业务逻辑层：Markdown 解析渲染、密码 Hash、JWT/Session 签发
├── handler/      # 表现层：Chi 路由、参数绑定、调用 Service 并返回 HTML 模板/片段
└── views/        # 页面模板与 htmx 片段
```

### 规则：
1. **Handler 不得直接执行 SQL 查询**，必须通过 Repository 或 Service。
2. **错误处理必须规范**：所有 I/O、数据库操作必须显式处理错误，禁止吞掉 `err`，日志记录必须携带上下文。

---

## 4. 常用开发与构建指令

```bash
# 1. 下载 Go 依赖
go mod tidy

# 2. 编译 Tailwind CSS (开发监听模式)
npx tailwindcss -i ./web/css/input.css -o ./static/css/tailwind.css --watch

# 3. 编译 Tailwind CSS (生产压缩模式)
npx tailwindcss -i ./web/css/input.css -o ./static/css/tailwind.min.css --minify

# 4. 本地运行服务器
go run ./cmd/server/main.go

# 5. 生产静态单二进制编译 (零 CGO 验证)
CGO_ENABLED=0 go build -ldflags="-s -w" -o ./bin/blog ./cmd/server
```

---

## 5. Agent 编码自检清单（Checklist）

每次完成代码编写或重构后，Agent 必须对照此清单自检：
- [ ] 运行 `CGO_ENABLED=0 go test ./...` 或 `CGO_ENABLED=0 go build ./...` 是否 100% 成功通过？
- [ ] 是否无意中引入了任何包含 C 代码或外部 C 库依赖的包？
- [ ] 模板中新增的交互是否遵循 htmx 规范？
- [ ] 前端是否有意料之外的外部资源请求（字体、高亮 JS 等）？
- [ ] 暗色模式和移动端响应式布局是否完备？
