package service

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
	"unicode/utf8"

	"blog/internal/domain"
	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

type MarkdownService struct {
	gm goldmark.Markdown
}

func NewMarkdownService() *MarkdownService {
	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Footnote,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			goldmarkhtml.WithHardWraps(),
			goldmarkhtml.WithXHTML(),
			goldmarkhtml.WithUnsafe(),
		),
	)
	return &MarkdownService{gm: gm}
}

// Render 将 Markdown 文本渲染为带有 Chroma 代码高亮、Mermaid 支持与 LaTeX 公式的 HTML
func (s *MarkdownService) Render(mdContent string) (template.HTML, []domain.TableOfContentItem, int) {
	// 1. 保护 LaTeX 数学公式（防止下划线 _ 和星号 * 被 Markdown 解析器误转为斜体/粗体）
	protectedMD, mathPlaceholders := s.protectMathFormulas(mdContent)

	var buf bytes.Buffer
	if err := s.gm.Convert([]byte(protectedMD), &buf); err != nil {
		return template.HTML(html.EscapeString(mdContent)), nil, 1
	}

	rawHTML := buf.String()

	// 2. 还原 LaTeX 数学公式
	for placeholder, originalMath := range mathPlaceholders {
		rawHTML = strings.ReplaceAll(rawHTML, placeholder, originalMath)
	}

	// 3. 提取目录 (h2, h3)
	toc := s.extractTOC(rawHTML)

	// 4. 处理代码块：区分 Mermaid 图表 与 Chroma 语法高亮
	highlightedHTML := s.highlightCodeBlocks(rawHTML)

	// 5. 计算阅读时间 (平均每分钟阅读 350 字)
	words := utf8.RuneCountInString(mdContent)
	minutes := words / 350
	if minutes < 1 {
		minutes = 1
	}

	return template.HTML(highlightedHTML), toc, minutes
}

// protectMathFormulas 提取并保护 $$...$$ 和 $...$ 避免 Markdown 解析器格式破坏
func (s *MarkdownService) protectMathFormulas(content string) (string, map[string]string) {
	// 1. 临时保护代码块与行内代码，防止代码中的符号 ($ 等) 被误当作公式
	codePlaceholders := make(map[string]string)
	codeCounter := 0
	codeRe := regexp.MustCompile("(?s)```[\\s\\S]*?```|`[^`\\n]+`")
	content = codeRe.ReplaceAllStringFunc(content, func(m string) string {
		p := fmt.Sprintf("@@CODE_TEMP_GUARD_%d@@", codeCounter)
		codePlaceholders[p] = m
		codeCounter++
		return p
	})

	mathPlaceholders := make(map[string]string)
	mathCounter := 0

	// 2. 匹配块级公式 $$...$$
	blockMathRe := regexp.MustCompile(`(?s)\$\$.*?\$\$`)
	content = blockMathRe.ReplaceAllStringFunc(content, func(m string) string {
		p := fmt.Sprintf("@@MATH_BLOCK_%d@@", mathCounter)
		mathPlaceholders[p] = m
		mathCounter++
		return p
	})

	// 3. 匹配行内公式 $...$ (非跨行)
	inlineMathRe := regexp.MustCompile(`\$[^\$\n\r]+?\$`)
	content = inlineMathRe.ReplaceAllStringFunc(content, func(m string) string {
		p := fmt.Sprintf("@@MATH_INLINE_%d@@", mathCounter)
		mathPlaceholders[p] = m
		mathCounter++
		return p
	})

	// 4. 还原代码块内容
	for p, code := range codePlaceholders {
		content = strings.ReplaceAll(content, p, code)
	}

	return content, mathPlaceholders
}

// 提取 TOC
func (s *MarkdownService) extractTOC(htmlContent string) []domain.TableOfContentItem {
	re := regexp.MustCompile(`(?i)<h([23])\s+id="([^"]+)">(.*?)</h[23]>`)
	tagRe := regexp.MustCompile(`<[^>]*>`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	var items []domain.TableOfContentItem
	for _, m := range matches {
		level := 2
		if m[1] == "3" {
			level = 3
		}
		cleanTitle := tagRe.ReplaceAllString(m[3], "")
		items = append(items, domain.TableOfContentItem{
			Level: level,
			ID:    m[2],
			Title: strings.TrimSpace(html.UnescapeString(cleanTitle)),
		})
	}
	return items
}

// 将 HTML 里的代码块替换为 Chroma 语法高亮或 Mermaid 图表容器
func (s *MarkdownService) highlightCodeBlocks(htmlContent string) string {
	codeBlockRe := regexp.MustCompile(`(?s)<pre[^>]*><code(?:\s+class="language-([a-zA-Z0-9_\+#\-]+)")?[^>]*>(.*?)</code></pre>`)

	return codeBlockRe.ReplaceAllStringFunc(htmlContent, func(match string) string {
		sub := codeBlockRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}

		lang := strings.ToLower(strings.TrimSpace(sub[1]))
		if lang == "" {
			lang = "text"
		}
		codeText := html.UnescapeString(sub[2])

		// 如果是 Mermaid 图表，生成专用的 Mermaid 容器，前端自动转换为 SVG 图表
		if lang == "mermaid" {
			return fmt.Sprintf(`
			<div class="my-6 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white/60 dark:bg-zinc-950/60 p-5 shadow-sm overflow-x-auto text-center">
				<div class="mermaid flex justify-center text-xs">%s</div>
			</div>`, html.EscapeString(codeText))
		}

		// 获取 Chroma 词法分析器
		lexer := lexers.Get(lang)
		if lexer == nil {
			lexer = lexers.Analyse(codeText)
		}
		if lexer == nil {
			lexer = lexers.Fallback
		}
		lexer = chroma.Coalesce(lexer)

		// 选用现代 VS Code 风格的高亮主题：dracula / monokai / github-dark
		style := styles.Get("dracula")
		if style == nil {
			style = styles.Fallback
		}

		// 格式化器：内联 CSS 样式，不需要外部额外加载 css
		formatter := chromahtml.New(
			chromahtml.WithClasses(false),
			chromahtml.TabWidth(2),
		)

		iterator, err := lexer.Tokenise(nil, codeText)
		if err != nil {
			return match
		}

		var b bytes.Buffer
		if err := formatter.Format(&b, style, iterator); err != nil {
			return match
		}

		// 外壳添加精致的 macOS 风格代码卡片顶栏（语言标签 + 装饰圆点 + 一键复制按钮）
		return fmt.Sprintf(`
		<div class="my-6 rounded-xl overflow-hidden border border-zinc-700/60 bg-zinc-950 shadow-lg text-sm font-mono">
			<div class="flex items-center justify-between px-4 py-2.5 bg-zinc-900 border-b border-zinc-800 text-xs text-zinc-400 select-none">
				<div class="flex items-center space-x-2">
					<span class="w-3 h-3 rounded-full bg-red-500/80 inline-block"></span>
					<span class="w-3 h-3 rounded-full bg-yellow-500/80 inline-block"></span>
					<span class="w-3 h-3 rounded-full bg-green-500/80 inline-block"></span>
				</div>
				<div class="flex items-center space-x-2">
					<span class="uppercase font-semibold tracking-wider text-zinc-400">%s</span>
					<button type="button" class="copy-code-btn px-2 py-0.5 rounded text-[10px] bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-white transition cursor-pointer" onclick="copyCode(this)">复制</button>
				</div>
			</div>
			<div class="p-4 overflow-x-auto text-[13px] leading-relaxed code-content">
				%s
			</div>
		</div>`, html.EscapeString(lang), b.String())
	})
}
