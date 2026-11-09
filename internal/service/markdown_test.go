package service

import (
	"strings"
	"testing"
)

func TestMarkdownService_Render(t *testing.T) {
	svc := NewMarkdownService()

	input := `# 标题一

这是一个测试段落。

## 章节一

这是章节内容包含数学公式：$a_1 + b_1 = c_1$，以及块级公式：
$$
\int_0^\infty e^{-x^2} dx = \frac{\sqrt{\pi}}{2}
$$

` + "```go" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello Go")
}
` + "```" + `

` + "```mermaid" + `
graph TD
    A[Client] --> B[Server]
` + "```" + `

### 子章节 ` + "`Deep`" + ` 架构

结尾总结。包含脚本变量：
` + "```bash" + `
export PATH=$PATH:$HOME/bin
echo "$VAR1 and $VAR2"
` + "```" + `
`

	htmlOutput, toc, readingTime := svc.Render(input)

	if readingTime < 1 {
		t.Errorf("expected reading time >= 1, got %d", readingTime)
	}

	htmlStr := string(htmlOutput)

	// 1. 检查普通代码块高亮与复制按钮
	if !strings.Contains(htmlStr, "Hello Go") {
		t.Errorf("rendered html does not contain code text: %s", htmlStr)
	}
	if !strings.Contains(htmlStr, "copy-code-btn") {
		t.Errorf("rendered html does not contain copy-code-btn: %s", htmlStr)
	}

	// 2. 检查 Mermaid 容器
	if !strings.Contains(htmlStr, "class=\"mermaid") || !strings.Contains(htmlStr, "graph TD") {
		t.Errorf("rendered html does not contain mermaid container: %s", htmlStr)
	}

	// 3. 检查 LaTeX 公式未被斜体下划线破坏
	if !strings.Contains(htmlStr, "$a_1 + b_1 = c_1$") {
		t.Errorf("inline latex math formula was distorted: %s", htmlStr)
	}
	if !strings.Contains(htmlStr, `\int_0^\infty`) {
		t.Errorf("block latex math formula was distorted: %s", htmlStr)
	}

	// 4. 检查 Bash 代码中的 $PATH / $VAR 未被误当作 LaTeX 公式替换
	if !strings.Contains(htmlStr, "PATH") || !strings.Contains(htmlStr, "VAR1") {
		t.Errorf("bash code variables distorted: %s", htmlStr)
	}

	// 5. 检查 TOC 提取，即使带行内 code 标签也能正确清洗出标题
	if len(toc) != 2 {
		t.Errorf("expected 2 TOC items (h2, h3), got %d", len(toc))
	} else {
		if toc[0].Title != "章节一" || toc[0].Level != 2 {
			t.Errorf("unexpected TOC item 0: %+v", toc[0])
		}
		if toc[1].Title != "子章节 Deep 架构" || toc[1].Level != 3 {
			t.Errorf("unexpected TOC item 1: %+v", toc[1])
		}
	}
}
