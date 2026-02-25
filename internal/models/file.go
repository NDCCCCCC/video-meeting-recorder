package models

import (
	"time"
)

// StorageType 存储类型
type StorageType string

const (
	StorageLocal StorageType = "local"
	StorageOSS   StorageType = "oss"
	StorageS3    StorageType = "s3"
	StorageCOS   StorageType = "cos"
)

// FileStatus 文件状态
type FileStatus string

const (
	FileStatusActive   FileStatus = "active"
	FileStatusDeleted  FileStatus = "deleted"
	FileStatusArchived FileStatus = "archived"
)

// UploadedFile 上传文件记录
type UploadedFile struct {
	ID uint `gorm:"primarykey" json:"id"`

	// 文件信息
	FileName     string `gorm:"type:varchar(255);not null" json:"file_name"`
	OriginalName string `gorm:"type:varchar(255);not null" json:"original_name"`
	FilePath     string `gorm:"type:varchar(500);not null;index" json:"file_path"`
	FileSize     int64  `gorm:"not null" json:"file_size"`
	MimeType     string `gorm:"type:varchar(100)" json:"mime_type"`
	FileMD5      string `gorm:"type:varchar(32);index" json:"file_md5,omitempty"`

	// 存储信息
	StorageType string `gorm:"type:varchar(20);index" json:"storage_type"` // local, oss, s3
	StoragePath string `gorm:"type:varchar(500)" json:"storage_path"`
	BucketName  string `gorm:"type:varchar(100)" json:"bucket_name,omitempty"`

	// 访问控制
	IsPublic    bool   `gorm:"default:false;index" json:"is_public"`
	AccessURL   string `gorm:"type:varchar(500)" json:"access_url,omitempty"`
	AccessToken string `gorm:"type:varchar(64);uniqueIndex" json:"access_token,omitempty"`

	// 过期控制
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
	IsExpired bool       `gorm:"default:false;index" json:"is_expired"`

	// 上传信息
	UploadedBy uint      `gorm:"index" json:"uploaded_by"`
	UploadedAt time.Time `gorm:"index" json:"uploaded_at"`

	// 关联信息
	RelatedID   *uint  `json:"related_id,omitempty"`
	RelatedType string `gorm:"type:varchar(50)" json:"related_type,omitempty"`

	// 状态
	Status string `gorm:"type:varchar(20);default:active" json:"status"` // active, deleted

	Base
}

// FileShare 文件分享
type FileShare struct {
	ID     uint          `gorm:"primarykey" json:"id"`
	FileID uint          `gorm:"index;not null" json:"file_id"`
	File   *UploadedFile `gorm:"foreignKey:FileID" json:"file,omitempty"`

	// 分享信息
	ShareToken string `gorm:"type:varchar(64);uniqueIndex;not null" json:"share_token"`
	SharedBy   uint   `gorm:"index" json:"shared_by"`

	// 访问控制
	Password    string `gorm:"type:varchar(100)" json:"password,omitempty"`
	MaxAccess   int    `gorm:"default:0" json:"max_access"` // 0表示无限制
	AccessCount int    `gorm:"default:0" json:"access_count"`

	// 过期时间
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	IsExpired bool      `gorm:"default:false;index" json:"is_expired"`

	CreatedAt time.Time `json:"created_at"`
}

// UserStorageQuota 用户存储配额
type UserStorageQuota struct {
	ID     uint  `gorm:"primarykey" json:"id"`
	UserID uint  `gorm:"uniqueIndex;not null" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// 配额设置
	TotalQuota int64 `gorm:"default:10737418240" json:"total_quota"` // 10GB
	UsedQuota  int64 `gorm:"default:0" json:"used_quota"`
	FileCount  int   `gorm:"default:0" json:"file_count"`

	// 通知阈值
	AlertThreshold float64    `gorm:"default:0.9" json:"alert_threshold"` // 90%
	LastAlertAt    *time.Time `json:"last_alert_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (UserStorageQuota) TableName() string {
	return "user_storage_quotas"
}
