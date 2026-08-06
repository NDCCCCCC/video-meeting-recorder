package models

import (
	"fmt"
	"time"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// VideoRecordingTask 视频录制任务模型
type VideoRecordingTask struct {
	Base
	Name               string    `gorm:"type:varchar(200);not null" json:"name"`
	Description        string    `gorm:"type:text" json:"description"`
	StartTime          time.Time `gorm:"not null;index" json:"start_time"`
	EndTime            time.Time `gorm:"not null;index" json:"end_time"`
	PreJoinMinutes     int       `gorm:"default:5" json:"pre_join_minutes"`
	RecordDelayMinutes int       `gorm:"default:0" json:"record_delay_minutes"`
	ConferenceNumber   string    `gorm:"type:varchar(50);not null;index" json:"conference_number"`
	// InputConfigID 已废弃，请使用 TaskInputConfigs 关联表
	// 保留此字段仅为兼容旧数据
	InputConfigID     *uint                    `gorm:"index" json:"input_config_id,omitempty"`
	InputConfig       *InputConfig             `gorm:"foreignKey:InputConfigID" json:"input_config,omitempty"`
	RTSPStreamURL     string                   `gorm:"type:varchar(500)" json:"rtsp_stream_url"` // RTSP流地址（可选，与USB设备同级）
	Status            VideoRecordingTaskStatus `gorm:"type:varchar(20);index" json:"status"`
	RecordingFile     string                   `gorm:"type:varchar(500)" json:"recording_file"` // 兼容旧字段，指向MKV文件
	RecordingDuration int                      `json:"recording_duration"`                      // 秒
	ErrorMsg          string                   `gorm:"type:text" json:"error_msg,omitempty"`
	// MKV录制和MP4转换相关字段
	MKVFilePath           string           `gorm:"type:varchar(500)" json:"mkv_file_path"`                      // MKV文件路径
	HLSPreviewPath        string           `gorm:"type:varchar(500)" json:"hls_preview_path"`                   // HLS预览路径
	MP4FilePath           string           `gorm:"type:varchar(500)" json:"mp4_file_path"`                      // MP4文件路径（转换后）
	ConversionStatus      ConversionStatus `gorm:"type:varchar(20);default:'pending'" json:"conversion_status"` // 转换状态
	ConversionErrorMsg    string           `gorm:"type:text" json:"conversion_error_msg,omitempty"`             // 转换错误信息
	ConversionStartedAt   *time.Time       `json:"conversion_started_at,omitempty"`                             // 转换开始时间
	ConversionCompletedAt *time.Time       `json:"conversion_completed_at,omitempty"`                           // 转换完成时间
	ConversionRetryCount  int              `gorm:"default:0" json:"conversion_retry_count"`                     // 转换重试次数

	// Smart-end audit fields (Phase 23 AUDIT-01). Written by service layer in Phase 25. Columns are added via AutoMigrate on next boot.
	ExtensionCount      int    `gorm:"not null;default:0" json:"extension_count"`
	LastExtensionReason string `gorm:"type:text;not null;default:''" json:"last_extension_reason,omitempty"`
	EndedEarly          bool   `gorm:"not null;default:false" json:"ended_early"`
	EndedEarlyReason    string `gorm:"type:text" json:"ended_early_reason,omitempty"`
	EndedByHuaWeAPI     bool   `gorm:"not null;default:false;column:ended_by_huawei_api" json:"ended_by_huawei_api"`

	CreatedBy uint  `gorm:"not null" json:"created_by"`
	Creator   *User `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`

	// TaskInputConfigs 任务关联的输入配置列表
	TaskInputConfigs []TaskInputConfig `gorm:"foreignKey:TaskID" json:"task_input_configs,omitempty"`
}

// VideoRecordingTaskStatus 任务状态枚举
type VideoRecordingTaskStatus string

const (
	VideoStatusPending    VideoRecordingTaskStatus = "pending"    // 待执行
	VideoStatusConnecting VideoRecordingTaskStatus = "connecting" // 连接会议中
	VideoStatusRecording  VideoRecordingTaskStatus = "recording"  // 录制中
	VideoStatusConverting VideoRecordingTaskStatus = "converting" // 转换中（MKV转MP4）
	VideoStatusCompleted  VideoRecordingTaskStatus = "completed"  // 已完成
	VideoStatusFailed     VideoRecordingTaskStatus = "failed"     // 执行失败
	VideoStatusCancelled  VideoRecordingTaskStatus = "canceled"   // 已取消
)

// ConversionStatus 转换状态枚举
type ConversionStatus string

const (
	ConversionStatusPending    ConversionStatus = "pending"    // 待转换
	ConversionStatusProcessing ConversionStatus = "processing" // 转换中
	ConversionStatusCompleted  ConversionStatus = "completed"  // 转换完成
	ConversionStatusFailed     ConversionStatus = "failed"     // 转换失败
)

// GetTriggerTime 返回实际触发时间
func (t *VideoRecordingTask) GetTriggerTime() time.Time {
	return t.StartTime.Add(-time.Duration(t.PreJoinMinutes) * time.Minute)
}

// GetRecordDelayMinutes 返回连接后延迟录制时间
func (t *VideoRecordingTask) GetRecordDelayMinutes() int {
	return t.RecordDelayMinutes
}

// IsExpired 检查任务是否已过期
func (t *VideoRecordingTask) IsExpired() bool {
	expiryTime := t.EndTime.Add(24 * time.Hour)
	return time.Now().After(expiryTime)
}

// IsValid 验证任务数据
func (t *VideoRecordingTask) IsValid() error {
	if t.StartTime.After(t.EndTime) {
		return fmt.Errorf("开始时间不能晚于结束时间: %w", apperrors.ErrInvalidInput)
	}
	if t.PreJoinMinutes < 0 || t.PreJoinMinutes > 60 {
		return fmt.Errorf("提前进入时间必须在0-60分钟之间: %w", apperrors.ErrInvalidInput)
	}
	if t.ConferenceNumber == "" {
		return fmt.Errorf("会议号不能为空: %w", apperrors.ErrInvalidInput)
	}

	// 至少需要指定一种输入配置
	// 注意：TaskInputConfigs 在任务创建后才填充，所以这里只检查 InputConfigID
	// 多配置验证在 CreateTask 服务层通过 InputConfigIDs 参数进行
	hasInputConfig := t.InputConfigID != nil && *t.InputConfigID > 0

	if !hasInputConfig {
		return fmt.Errorf("必须指定至少一种输入配置: %w", apperrors.ErrInvalidInput)
	}

	return nil
}

// CanTransitionTo 检查状态转换是否合法
func (t *VideoRecordingTask) CanTransitionTo(newStatus VideoRecordingTaskStatus) bool {
	validTransitions := map[VideoRecordingTaskStatus][]VideoRecordingTaskStatus{
		VideoStatusPending:    {VideoStatusConnecting, VideoStatusFailed, VideoStatusCancelled}, // pending可直接转为failed（如触发时间已过期）
		VideoStatusConnecting: {VideoStatusRecording, VideoStatusFailed, VideoStatusCancelled},  // 连接中可以转为录制、失败或取消
		VideoStatusRecording:  {VideoStatusConverting, VideoStatusFailed, VideoStatusCancelled}, // 录制中可以转为转换中、失败或取消
		VideoStatusConverting: {VideoStatusCompleted, VideoStatusFailed},                        // 转换中可以转为完成或失败
		VideoStatusCompleted:  {},                                                               // 终态
		VideoStatusFailed:     {VideoStatusPending},                                             // 可重试
		VideoStatusCancelled:  {},                                                               // 终态
	}

	allowed, ok := validTransitions[t.Status]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}
	return false
}

// TableName 指定表名
func (VideoRecordingTask) TableName() string {
	return "video_recording_tasks"
}
