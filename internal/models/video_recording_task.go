package models

import (
	"errors"
	"time"
)

// VideoRecordingTask 视频录制任务模型
type VideoRecordingTask struct {
	Base
	Name               string                   `gorm:"type:varchar(200);not null" json:"name"`
	Description        string                   `gorm:"type:text" json:"description"`
	StartTime          time.Time                `gorm:"not null;index" json:"start_time"`
	EndTime            time.Time                `gorm:"not null;index" json:"end_time"`
	PreJoinMinutes     int                      `gorm:"default:5" json:"pre_join_minutes"`
	RecordDelayMinutes int                      `gorm:"default:0" json:"record_delay_minutes"`
	ConferenceNumber   string                   `gorm:"type:varchar(50);not null;index" json:"conference_number"`
	HuaweiConfigID     uint                     `gorm:"not null;index" json:"huawei_config_id"`
	HuaweiConfig       *HuaweiConfig            `gorm:"foreignKey:HuaweiConfigID" json:"huawei_config,omitempty"`
	RTSPStreamURL      string                   `gorm:"type:varchar(500)" json:"rtsp_stream_url"` // RTSP流地址（可选，与USB设备同级）
	Status             VideoRecordingTaskStatus `gorm:"type:varchar(20);index" json:"status"`
	RecordingFile      string                   `gorm:"type:varchar(500)" json:"recording_file"` // 兼容旧字段，指向MKV文件
	RecordingDuration  int                      `json:"recording_duration"`                      // 秒
	ErrorMsg           string                   `gorm:"type:text" json:"error_msg,omitempty"`
	// MKV录制和MP4转换相关字段
	MKVFilePath           string           `gorm:"type:varchar(500)" json:"mkv_file_path"`                      // MKV文件路径
	HLSPreviewPath        string           `gorm:"type:varchar(500)" json:"hls_preview_path"`                   // HLS预览路径
	MP4FilePath           string           `gorm:"type:varchar(500)" json:"mp4_file_path"`                      // MP4文件路径（转换后）
	ConversionStatus      ConversionStatus `gorm:"type:varchar(20);default:'pending'" json:"conversion_status"` // 转换状态
	ConversionErrorMsg    string           `gorm:"type:text" json:"conversion_error_msg,omitempty"`             // 转换错误信息
	ConversionStartedAt   *time.Time       `json:"conversion_started_at,omitempty"`                             // 转换开始时间
	ConversionCompletedAt *time.Time       `json:"conversion_completed_at,omitempty"`                           // 转换完成时间
	ConversionRetryCount  int              `gorm:"default:0" json:"conversion_retry_count"`                     // 转换重试次数
	CreatedBy             uint             `gorm:"not null" json:"created_by"`
	Creator               *User            `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// VideoRecordingTaskStatus 任务状态枚举
type VideoRecordingTaskStatus string

const (
	VideoStatusPending    VideoRecordingTaskStatus = "pending"    // 待执行
	VideoStatusConnecting VideoRecordingTaskStatus = "connecting" // 连接会议中
	VideoStatusRecording  VideoRecordingTaskStatus = "recording"  // 录制中
	VideoStatusCompleted  VideoRecordingTaskStatus = "completed"  // 已完成
	VideoStatusFailed     VideoRecordingTaskStatus = "failed"     // 执行失败
	VideoStatusCancelled  VideoRecordingTaskStatus = "cancelled"  // 已取消
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
		return errors.New("开始时间不能晚于结束时间")
	}
	if t.PreJoinMinutes < 0 || t.PreJoinMinutes > 60 {
		return errors.New("提前进入时间必须在0-60分钟之间")
	}
	if t.ConferenceNumber == "" {
		return errors.New("会议号不能为空")
	}
	if t.HuaweiConfigID == 0 {
		return errors.New("必须指定华为配置")
	}
	return nil
}

// CanTransitionTo 检查状态转换是否合法
func (t *VideoRecordingTask) CanTransitionTo(newStatus VideoRecordingTaskStatus) bool {
	validTransitions := map[VideoRecordingTaskStatus][]VideoRecordingTaskStatus{
		VideoStatusPending:    {VideoStatusConnecting, VideoStatusFailed, VideoStatusCancelled}, // pending可直接转为failed（如触发时间已过期）
		VideoStatusConnecting: {VideoStatusRecording, VideoStatusFailed, VideoStatusCancelled},
		VideoStatusRecording:  {VideoStatusCompleted, VideoStatusFailed, VideoStatusCancelled},
		VideoStatusCompleted:  {},                   // 终态
		VideoStatusFailed:     {VideoStatusPending}, // 可重试
		VideoStatusCancelled:  {},                   // 终态
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
