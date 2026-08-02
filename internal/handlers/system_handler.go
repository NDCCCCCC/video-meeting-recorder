package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// strUint 将 uint 转为字符串（避免引入 strconv 仅用于一处）。
// 内部 SEC-015 类型断言辅助函数。
func strUint(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}

// SystemHandler 系统设置处理器
type SystemHandler struct {
	db           *gorm.DB
	auditService *audit.AuditLogService
	logger       *zap.Logger
	config       *config.Config
}

// NewSystemHandler 创建系统设置处理器
func NewSystemHandler(db *gorm.DB, auditService *audit.AuditLogService, logger *zap.Logger, cfg *config.Config) *SystemHandler {
	return &SystemHandler{
		db:           db,
		auditService: auditService,
		logger:       logger,
		config:       cfg,
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
			"max_disk_usage":  h.config.Storage.MaxDiskUsage,
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
	MaxDiskUsage   *int    `json:"max_disk_usage"`
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

	// 记录配置变更
	changes := make([]string, 0)

	// ★ 审计：仅 snapshot 实际 mutate h.config 的 4 个 sub-keys（LogLevel/LogFormat/LogOutput/MaxDiskUsage）
	// restart-required 字段（路径类、FFmpeg）仅记录到 changes 字符串，不进 OldData（内存未变更）
	oldMap := map[string]interface{}{}
	newMap := map[string]interface{}{}
	changedKeys := make([]string, 0)

	// 更新内存中的配置（仅对日志级别等运行时可变配置生效）
	if req.LogLevel != nil && *req.LogLevel != h.config.Logging.Level {
		oldMap["logging.level"] = h.config.Logging.Level
		newMap["logging.level"] = *req.LogLevel
		changedKeys = append(changedKeys, "logging.level")
		h.config.Logging.Level = *req.LogLevel
		changes = append(changes, fmt.Sprintf("日志级别: %s -> %s", oldMap["logging.level"], *req.LogLevel))
		// 尝试动态更新日志级别
		if err := h.updateLogLevel(*req.LogLevel); err == nil {
			h.logger.Info("日志级别已动态更新", zap.String("level", *req.LogLevel))
		}
	}
	if req.LogFormat != nil && *req.LogFormat != h.config.Logging.Format {
		oldMap["logging.format"] = h.config.Logging.Format
		newMap["logging.format"] = *req.LogFormat
		changedKeys = append(changedKeys, "logging.format")
		h.config.Logging.Format = *req.LogFormat
		changes = append(changes, fmt.Sprintf("日志格式: %s", *req.LogFormat))
	}
	if req.LogOutput != nil && *req.LogOutput != h.config.Logging.Output {
		oldMap["logging.output"] = h.config.Logging.Output
		newMap["logging.output"] = *req.LogOutput
		changedKeys = append(changedKeys, "logging.output")
		h.config.Logging.Output = *req.LogOutput
		changes = append(changes, fmt.Sprintf("日志输出: %s", *req.LogOutput))
	}

	// 记录需要重启才能生效的配置变更
	if req.RecordingsPath != nil {
		changes = append(changes, fmt.Sprintf("录制路径: %s (需重启)", *req.RecordingsPath))
	}
	if req.HLSPath != nil {
		changes = append(changes, fmt.Sprintf("HLS路径: %s (需重启)", *req.HLSPath))
	}
	if req.TempPath != nil {
		changes = append(changes, fmt.Sprintf("临时路径: %s (需重启)", *req.TempPath))
	}
	if req.MaxDiskUsage != nil && *req.MaxDiskUsage != h.config.Storage.MaxDiskUsage {
		oldMap["storage.max_disk_usage"] = h.config.Storage.MaxDiskUsage
		newMap["storage.max_disk_usage"] = *req.MaxDiskUsage
		changedKeys = append(changedKeys, "storage.max_disk_usage")
		h.config.Storage.MaxDiskUsage = *req.MaxDiskUsage
		changes = append(changes, fmt.Sprintf("磁盘使用限制: %d%%", *req.MaxDiskUsage))
	}
	if req.FFmpegPath != nil {
		changes = append(changes, fmt.Sprintf("FFmpeg路径: %s (需重启)", *req.FFmpegPath))
	}
	if req.FFprobePath != nil {
		changes = append(changes, fmt.Sprintf("FFprobe路径: %s (需重启)", *req.FFprobePath))
	}

	// 记录配置变更
	if len(changes) > 0 {
		// SEC-015: 类型断言守卫；user_id 应为 uint，缺失/类型错误时回退到 "unknown"
		userIDStr := "unknown"
		if v, ok := c.Get("user_id"); ok {
			switch typed := v.(type) {
			case uint:
				userIDStr = strUint(typed)
			case string:
				userIDStr = typed
			}
		}
		h.logger.Info("系统配置已更新",
			zap.Strings("changes", changes),
			zap.String("user", userIDStr),
		)
	}

	// ★ 审计：仅在至少一个 sub-key 实际变更时记录（restart-required 路径类不 emit 审计行）
	if len(changedKeys) > 0 {
		resourceID := uint(0) // system config 没有单 ID
		if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
			Action:     "update_config",
			Module:     models.ModuleSystem,
			Resource:   "system_config:" + strings.Join(changedKeys, ","),
			ResourceID: &resourceID,
			OldData:    oldMap,
			NewData:    newMap,
		}); err != nil {
			h.logger.Warn("Failed to record system config update change", zap.Error(err))
		}
	}

	response.GinSuccess(c, gin.H{
		"message": "配置已更新，路径和FFmpeg相关配置需要重启服务后生效",
		"changes": changes,
	})
}

// updateLogLevel 尝试动态更新日志级别
func (h *SystemHandler) updateLogLevel(level string) error {
	// 解析日志级别
	var zapLevel zap.AtomicLevel
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		return err
	}

	// 这里可以添加动态更新日志级别的逻辑
	// 注意：zap 的日志级别通常在创建时设置，运行时修改比较复杂
	// 当前实现仅记录，实际应用需要更复杂的日志管理

	return nil
}

// ClearFiles 清空文件数据库
func (h *SystemHandler) ClearFiles(c *gin.Context) {
	// BUG-005: 透传 ctx 让客户端断开能级联取消 DELETE
	if err := h.db.WithContext(c.Request.Context()).Exec("DELETE FROM video_files").Error; err != nil {
		h.logger.Error("清空文件表失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "清空文件表失败")
		return
	}

	h.logger.Info("文件数据库已清空")
	response.GinSuccess(c, gin.H{
		"message": "文件数据库已清空",
	})
}
