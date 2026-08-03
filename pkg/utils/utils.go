package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateToken 生成随机令牌
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileSize 获取文件大小
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// RemoveAll 删除目录及其内容
func RemoveAll(dir string) error {
	return os.RemoveAll(dir)
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destination.Close() }()

	_, err = io.Copy(destination, source)
	return err
}

// GetFileExtension 获取文件扩展名
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.ToLower(ext)
}

// FormatDuration 格式化时长
func FormatDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	secs := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}

// ParseDuration 解析时长字符串
func ParseDuration(s string) (int, error) {
	// 解析 "HH:MM:SS" 或 "MM:SS" 格式
	parts := strings.Split(s, ":")
	var seconds int

	switch len(parts) {
	case 3: // HH:MM:SS
		hours := parseInt(parts[0])
		minutes := parseInt(parts[1])
		secs := parseInt(parts[2])
		seconds = hours*3600 + minutes*60 + secs
	case 2: // MM:SS
		minutes := parseInt(parts[0])
		secs := parseInt(parts[1])
		seconds = minutes*60 + secs
	default:
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	return seconds, nil
}

func parseInt(s string) int {
	var val int
	_, _ = fmt.Sscanf(s, "%d", &val)
	return val
}
