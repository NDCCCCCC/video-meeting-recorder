package storage

import (
	"context"
	"io"
	"mime/multipart"
	"time"
)

// StorageDriver 存储驱动接口
type StorageDriver interface {
	// 上传文件
	Upload(ctx context.Context, file *multipart.FileHeader, path string) (*UploadResult, error)

	// 下载文件
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// 删除文件
	Delete(ctx context.Context, path string) error

	// 检查文件是否存在
	Exists(ctx context.Context, path string) (bool, error)

	// 获取访问URL
	GetURL(ctx context.Context, path string, expires time.Duration) (string, error)

	// 获取文件信息
	GetInfo(ctx context.Context, path string) (*FileInfo, error)

	// 复制文件
	Copy(ctx context.Context, srcPath, destPath string) error

	// 移动文件
	Move(ctx context.Context, srcPath, destPath string) error

	// 列出文件
	List(ctx context.Context, prefix string, limit int) ([]*FileInfo, error)
}

// UploadResult 上传结果
type UploadResult struct {
	FilePath    string `json:"file_path"`
	StoragePath string `json:"storage_path"`
	FileSize    int64  `json:"file_size"`
	ETag        string `json:"etag,omitempty"`
	URL         string `json:"url,omitempty"`
}

// FileInfo 文件信息
type FileInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`
	ContentType string    `json:"content_type,omitempty"`
	ETag        string    `json:"etag,omitempty"`
}
