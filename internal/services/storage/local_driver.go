package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// LocalStorageDriver 本地存储驱动
type LocalStorageDriver struct {
	basePath string
	baseURL  string
	logger   *zap.Logger
}

// NewLocalStorageDriver 创建本地存储驱动
func NewLocalStorageDriver(basePath, baseURL string, logger *zap.Logger) *LocalStorageDriver {
	// 确保目录存在
	os.MkdirAll(basePath, 0755)

	return &LocalStorageDriver{
		basePath: basePath,
		baseURL:  baseURL,
		logger:   logger,
	}
}

// Upload 上传文件
func (d *LocalStorageDriver) Upload(ctx context.Context, file *multipart.FileHeader, path string) (*UploadResult, error) {
	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	fullPath := filepath.Join(d.basePath, path)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	// 复制文件
	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}

	return &UploadResult{
		FilePath:    path,
		StoragePath: fullPath,
		FileSize:    file.Size,
		URL:         d.baseURL + "/" + path,
	}, nil
}

// Download 下载文件
func (d *LocalStorageDriver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(d.basePath, path)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	return file, nil
}

// Delete 删除文件
func (d *LocalStorageDriver) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(d.basePath, path)
	return os.Remove(fullPath)
}

// Exists 检查文件是否存在
func (d *LocalStorageDriver) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(d.basePath, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetURL 获取访问URL
func (d *LocalStorageDriver) GetURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	// 本地存储直接返回URL，expires参数忽略
	return d.baseURL + "/" + path, nil
}

// GetInfo 获取文件信息
func (d *LocalStorageDriver) GetInfo(ctx context.Context, path string) (*FileInfo, error) {
	fullPath := filepath.Join(d.basePath, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Path:    path,
		Name:    filepath.Base(path),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		IsDir:   info.IsDir(),
	}, nil
}

// Copy 复制文件
func (d *LocalStorageDriver) Copy(ctx context.Context, srcPath, destPath string) error {
	src := filepath.Join(d.basePath, srcPath)
	dst := filepath.Join(d.basePath, destPath)

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Move 移动文件
func (d *LocalStorageDriver) Move(ctx context.Context, srcPath, destPath string) error {
	src := filepath.Join(d.basePath, srcPath)
	dst := filepath.Join(d.basePath, destPath)

	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	return os.Rename(src, dst)
}

// List 列出文件
func (d *LocalStorageDriver) List(ctx context.Context, prefix string, limit int) ([]*FileInfo, error) {
	dir := filepath.Join(d.basePath, prefix)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := make([]*FileInfo, 0, len(entries))
	for _, entry := range entries {
		if limit > 0 && len(result) >= limit {
			break
		}

		info, _ := entry.Info()
		result = append(result, &FileInfo{
			Path:    filepath.Join(prefix, entry.Name()),
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   entry.IsDir(),
		})
	}

	return result, nil
}
