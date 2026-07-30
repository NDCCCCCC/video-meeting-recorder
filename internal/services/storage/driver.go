package storage

import "time"

// StorageDriver 接口定义已迁移到 file_service.go（STYLE-003：消费方包定义接口）。
// 本文件保留辅助类型 UploadResult / FileInfo。
//
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
