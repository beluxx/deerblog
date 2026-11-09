package static

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed css js vendor
var embeddedFS embed.FS

// FileServer 返回一个智能静态资源处理器：
// 优先检查本地磁盘（支持用户上传文件 static/uploads/*），
// 本地不存在时无缝回退到嵌入式二进制 embed.FS，做到 100% 零外部依赖单二进制离线运行！
func FileServer(diskDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		relPath := strings.TrimPrefix(r.URL.Path, "/")

		// 1. 如果本地磁盘存在该文件（例如用户上传的图片 uploads/*），优先从磁盘提供
		if diskDir != "" {
			diskPath := diskDir + "/" + relPath
			if info, err := os.Stat(diskPath); err == nil && !info.IsDir() {
				http.ServeFile(w, r, diskPath)
				return
			}
		}

		// 2. 否则直接从编译进二进制的 embed.FS 中提供
		subFS, err := fs.Sub(embeddedFS, ".")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		http.FileServer(http.FS(subFS)).ServeHTTP(w, r)
	})
}
