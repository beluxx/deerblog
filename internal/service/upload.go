package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxUploadSize = 5 * 1024 * 1024 // 5MB
	UploadBaseDir = "static/uploads"
)

var allowedMIMEs = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
	"image/svg+xml": ".svg",
}

type UploadService struct {
	baseDir string
}

func NewUploadService(baseDir string) *UploadService {
	if baseDir == "" {
		baseDir = UploadBaseDir
	}
	return &UploadService{baseDir: baseDir}
}

// SaveImage 验证并持久化上传的图片
func (s *UploadService) SaveImage(fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader.Size > MaxUploadSize {
		return "", fmt.Errorf("文件大小超出限制 (最大 %d MB)", MaxUploadSize/(1024*1024))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("无法读取上传文件: %w", err)
	}
	defer file.Close()

	// 嗅探文件前 512 字节以识别真实 MIME 类型
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("读取文件头失败: %w", err)
	}

	mimeType := http.DetectContentType(buffer[:n])
	// 处理 SVG 特殊情况
	if strings.Contains(string(buffer[:n]), "<svg") {
		mimeType = "image/svg+xml"
	}

	ext, ok := allowedMIMEs[mimeType]
	if !ok {
		// 尝试根据原始文件名后缀回退验证
		origExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if origExt == ".svg" && strings.Contains(mimeType, "text/") {
			ext = ".svg"
		} else {
			return "", errors.New("仅支持上传 JPEG, PNG, WebP, GIF 或 SVG 格式的图片")
		}
	}

	// 重置文件指针到开头
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// 按年月分目录存储：static/uploads/2026/11/
	now := time.Now()
	subDir := fmt.Sprintf("%d/%02d", now.Year(), now.Month())
	targetDir := filepath.Join(s.baseDir, subDir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("创建存储目录失败: %w", err)
	}

	// 生成安全不可预测的唯一文件名
	randomBytes := make([]byte, 16)
	_, _ = rand.Read(randomBytes)
	fileName := hex.EncodeToString(randomBytes) + ext
	destPath := filepath.Join(targetDir, fileName)

	destFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		return "", fmt.Errorf("保存文件失败: %w", err)
	}

	// 返回浏览器访问路径
	publicURL := fmt.Sprintf("/static/uploads/%s/%s", subDir, fileName)
	return publicURL, nil
}
