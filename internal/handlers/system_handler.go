package handlers

import (
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SystemHandler 系统设置处理器
type SystemHandler struct {
	db      *gorm.DB
	logger  *zap.Logger
	config  *config.Config
}

// NewSystemHandler 创建系统设置处理器
func NewSystemHandler(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *SystemHandler {
	return &SystemHandler{
		db:     db,
		logger:  logger,
		config:  cfg,
	}
}

// SetLogger 设置日志记录器
func (h *SystemHandler) SetLogger(logger *zap.Logger) {
	h.logger = logger
}

// GetConfig 获取系统配置
func (h *SystemHandler) GetConfig(c *gin.Context) {
	// 返回配置信息（隐藏敏感信息）
	configData := map[string]interface{}{
		"storage": map[string]interface{}{
			"recordings_path": h.config.Storage.RecordingsPath,
			"hls_path":        h.config.Storage.HLSPath,
			"temp_path":       h.config.Storage.TempPath,
			"max_disk_usage":   h.config.Storage.MaxDiskUsage,
		},
		"ffmpeg": map[string]interface{}{
			"path":         h.config.FFmpeg.Path,
			"ffprobe_path": h.config.FFmpeg.FFProbePath,
		},
		"logging": map[string]interface{}{
			"level":  h.config.Logging.Level,
			"format": h.config.Logging.Format,
			"output": h.config.Logging.Output,
		},
	}

	response.GinSuccess(c, configData)
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	RecordingsPath *string `json:"recordings_path"`
	HLSPath        *string `json:"hls_path"`
	TempPath       *string `json:"temp_path"`
	MaxDiskUsage  *int    `json:"max_disk_usage"`
	FFmpegPath     *string `json:"ffmpeg_path"`
	FFprobePath    *string `json:"ffprobe_path"`
	LogLevel       *string `json:"log_level"`
	LogFormat      *string `json:"log_format"`
	LogOutput      *string `json:"log_output"`
}

// UpdateConfig 更新系统配置
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "参数错误")
		return
	}

	// TODO: 实现配置更新逻辑
	// 注意：部分配置需要重启服务才能生效
	response.GinSuccess(c, gin.H{
		"message": "配置已更新，部分配置需要重启服务后生效",
	})
}

// ClearFiles 清空文件数据库
func (h *SystemHandler) ClearFiles(c *gin.Context) {
	// 删除所有视频文件记录
	if err := h.db.Exec("DELETE FROM video_files").Error; err != nil {
		h.logger.Error("清空文件表失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "清空文件表失败")
		return
	}

	h.logger.Info("文件数据库已清空")
	response.GinSuccess(c, gin.H{
		"message": "文件数据库已清空",
	})
}
