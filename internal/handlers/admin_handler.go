package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	cfg           *config.Config
	logger        *zap.Logger
	configService *services.ConfigService
	authService   *auth.Service
	db            *gorm.DB
}

func NewAdminHandler(cfg *config.Config, logger *zap.Logger, configService *services.ConfigService, authService *auth.Service, db *gorm.DB) *AdminHandler {
	return &AdminHandler{
		cfg:           cfg,
		logger:        logger,
		configService: configService,
		authService:   authService,
		db:            db,
	}
}

// GetAuthConfig 获取认证配置
// @Summary 获取认证配置
// @Description 获取当前认证配置（隐藏敏感信息）
// @Tags 系统管理
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /api/v1/admin/auth/config [get]
func (h *AdminHandler) GetAuthConfig(c *gin.Context) {
	// Return sanitized config (hide password)
	sanitized := map[string]interface{}{
		"mode": h.cfg.Auth.Mode,
		"ad": map[string]interface{}{
			"server":               h.cfg.Auth.AD.Server,
			"bind_dn":              h.cfg.Auth.AD.BindDN,
			"base_dn":              h.cfg.Auth.AD.BaseDN,
			"use_tls":              h.cfg.Auth.AD.UseTLS,
			"pool_size":            h.cfg.Auth.AD.PoolSize,
			"dial_timeout":         h.cfg.Auth.AD.DialTimeout,
			"request_timeout":      h.cfg.Auth.AD.RequestTimeout,
			"insecure_skip_verify": h.cfg.Auth.AD.InsecureSkipVerify,
			"allow_auto_create":    h.cfg.Auth.AD.AllowAutoCreate, // Default: true
			// Password excluded for security
		},
	}
	response.GinSuccess(c, sanitized)
}

// UpdateAuthConfig 更新认证配置
// @Summary 更新认证配置
// @Description 更新认证配置并验证（切换到AD模式前必须验证通过）
// @Tags 系统管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body object{mode=string,ad=auth.ADAuthConfig} true "认证配置"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/auth/config [put]
func (h *AdminHandler) UpdateAuthConfig(c *gin.Context) {
	var req struct {
		Mode string            `json:"mode" binding:"required,oneof=local ad"`
		AD   auth.ADAuthConfig `json:"ad"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	currentUserID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}

	// If switching to AD mode, validate AD config first (per D-17)
	if req.Mode == "ad" {
		validator := auth.NewADConfigValidator(h.logger)
		result := validator.Validate(&req.AD)

		if !result.Valid {
			response.GinError(c, response.CodeInvalidRequest,
				"AD配置验证失败: "+strings.Join(result.Errors, "; "))
			return
		}

		// Log warnings (including port 389 warning) to audit (per D-13)
		if len(result.Warnings) > 0 {
			h.logger.Warn("AD configuration warnings",
				zap.Uint("user_id", currentUserID),
				zap.Strings("warnings", result.Warnings),
			)
		}
	}

	// Log the configuration change
	h.logger.Info("Authentication mode changed",
		zap.Uint("user_id", currentUserID),
		zap.String("old_mode", h.cfg.Auth.Mode),
		zap.String("new_mode", req.Mode),
	)

	// Update configuration
	h.cfg.Auth.Mode = req.Mode
	if req.Mode == "ad" {
		// Convert auth.ADAuthConfig to config.ADAuthConfig
		h.cfg.Auth.AD.Server = req.AD.Server
		h.cfg.Auth.AD.BindDN = req.AD.BindDN
		h.cfg.Auth.AD.Password = req.AD.Password
		h.cfg.Auth.AD.BaseDN = req.AD.BaseDN
		h.cfg.Auth.AD.UseTLS = req.AD.UseTLS
		h.cfg.Auth.AD.PoolSize = req.AD.PoolSize
		h.cfg.Auth.AD.DialTimeout = req.AD.DialTimeout
		h.cfg.Auth.AD.RequestTimeout = req.AD.RequestTimeout
		h.cfg.Auth.AD.InsecureSkipVerify = req.AD.InsecureSkipVerify
		h.cfg.Auth.AD.AllowAutoCreate = req.AD.AllowAutoCreate
	}

	// Persist to database
	if h.configService != nil {
		if err := h.configService.SaveAuthConfig(req.Mode, &req.AD); err != nil {
			h.logger.Error("Failed to save auth config to database", zap.Error(err))
			response.GinError(c, response.CodeInternalError, "配置保存失败")
			return
		}
	}

	response.GinSuccess(c, gin.H{"message": "认证配置已更新"})
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 系统管理
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /api/v1/admin/auth/me [get]
func (h *AdminHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	username := middleware.GetUsername(c)

	response.GinSuccess(c, gin.H{
		"user_id":  userID,
		"username": username,
	})
}

// LookupADUser 查询AD用户信息
// @Summary 查询AD用户信息
// @Description 根据用户名从Active Directory查询用户信息
// @Tags 系统管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body object{username=string} true "用户名"
// @Success 200 {object} response.Response{data=auth.ADUserLookupResult}
// @Router /api/v1/admin/auth/lookup-ad-user [post]
func (h *AdminHandler) LookupADUser(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	// Check if current auth mode is AD
	if h.cfg.Auth.Mode != "ad" {
		response.GinError(c, response.CodeInvalidRequest, "当前认证模式不是AD域控模式")
		return
	}

	// Get AD authenticator from auth service
	adAuth := h.authService.GetADAuthenticator()
	if adAuth == nil {
		response.GinError(c, response.CodeInternalError, "AD认证器未初始化")
		return
	}

	// Perform AD lookup
	result, err := adAuth.LookupUser(req.Username)
	if err != nil {
		h.logger.Error("AD user lookup failed", zap.String("username", req.Username), zap.Error(err))
		response.GinError(c, response.CodeInternalError, "域控查询失败: "+err.Error())
		return
	}

	response.GinSuccess(c, result)
}

// MigrateInputConfigs 迁移华为配置到输入配置
// @Summary 迁移华为配置到输入配置
// @Description 将 huawei_configs 表数据迁移到 input_configs 表（可选操作）
// @Tags 系统管理
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /api/v1/admin/migrate-input-configs [post]
func (h *AdminHandler) MigrateInputConfigs(c *gin.Context) {
	h.logger.Info("Starting migration from huawei_configs to input_configs")

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Count existing huawei_configs
	var totalCount int64
	if err := tx.Table("huawei_configs").Count(&totalCount).Error; err != nil {
		h.logger.Error("Failed to count huawei_configs", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "迁移失败：无法读取源数据")
		return
	}

	// Fetch all huawei_configs
	type HuaweiConfigRow struct {
		ID                  uint
		Name                string
		Description         string
		Server              string
		Port                int
		Username            string
		Password            string
		TerminalNumber      string
		ConferenceNumber    string
		CameraBackend       string
		USBCameraName       string
		USBCameraDevice     string
		CameraBindingStatus string
		AudioBackend        string
		USBAudioName        string
		USBAudioDevice      string
		AudioBindingStatus  string
		OutputFormat        string
		StreamProtocol      string
		StreamURL           string
		StreamUsername      string
		StreamPassword      string
		StreamEnabled       bool
		IsActive            bool
		IsLocked            bool
		LockedBy            *uint
		LockedAt            *time.Time
		CreatedAt           time.Time
		UpdatedAt           time.Time
		DeletedAt           *time.Time
	}

	var huaweiConfigs []HuaweiConfigRow
	if err := tx.Table("huawei_configs").Find(&huaweiConfigs).Error; err != nil {
		h.logger.Error("Failed to fetch huawei_configs", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "迁移失败：无法读取源数据")
		return
	}

	migratedCount := 0
	skippedCount := 0

	// PERF-005/D-03.8: 将每条记录的迁移工作改为 bounded concurrency，缩短 handler 响应时间。
	// 注意：仍使用同一个 tx（共享事务），所有迁移在同一 SQL 事务内成功后才统一 Commit。
	concurrency := h.cfg.Admin.MigrationConcurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	if concurrency > len(huaweiConfigs) {
		concurrency = len(huaweiConfigs)
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, hc := range huaweiConfigs {
		wg.Add(1)
		sem <- struct{}{}
		go func(hc HuaweiConfigRow) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					h.logger.Error("admin migration worker panicked",
						zap.Any("recover", r), zap.Stack("stack"))
				}
			}()
			// 内联迁移：因 HuaweiConfigRow 是 handler 内本地类型，无法移到包级 helper。
			var existingCount int64
			if err := tx.Table("input_configs").
				Where("name = ? AND created_at = ?", hc.Name, hc.CreatedAt).
				Count(&existingCount).Error; err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			if existingCount > 0 {
				mu.Lock()
				skippedCount++
				mu.Unlock()
				return
			}
			configType := "usb"
			if hc.USBCameraDevice == "" && hc.StreamURL != "" {
				configType = "stream"
			}
			hwEnabled := hc.Server != "" && hc.Username != "" && hc.TerminalNumber != ""
			insertSQL := `
				INSERT INTO input_configs (
					created_at, updated_at, deleted_at,
					name, description, config_type, huawei_enabled,
					server, port, username, password, terminal_number, conference_number,
					camera_backend, usb_camera_name, usb_camera_device, camera_binding_status,
					audio_backend, usb_audio_name, usb_audio_device, audio_binding_status,
					output_format,
					stream_protocol, stream_url, stream_username, stream_password, stream_enabled,
					is_active, is_locked, locked_by, locked_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
			if err := tx.Exec(insertSQL,
				hc.CreatedAt, hc.UpdatedAt, hc.DeletedAt,
				hc.Name, hc.Description, configType, hwEnabled,
				hc.Server, hc.Port, hc.Username, hc.Password, hc.TerminalNumber, hc.ConferenceNumber,
				hc.CameraBackend, hc.USBCameraName, hc.USBCameraDevice, hc.CameraBindingStatus,
				hc.AudioBackend, hc.USBAudioName, hc.USBAudioDevice, hc.AudioBindingStatus,
				hc.OutputFormat,
				hc.StreamProtocol, hc.StreamURL, hc.StreamUsername, hc.StreamPassword, hc.StreamEnabled,
				hc.IsActive, hc.IsLocked, hc.LockedBy, hc.LockedAt,
			).Error; err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			migratedCount++
			mu.Unlock()
		}(hc)
	}
	wg.Wait()

	if firstErr != nil {
		h.logger.Error("Failed to migrate config", zap.Error(firstErr))
		tx.Rollback()
		response.GinError(c, response.CodeInternalError, "迁移失败：无法写入目标数据")
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		h.logger.Error("Failed to commit migration", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "迁移失败：无法提交事务")
		return
	}

	h.logger.Info("Migration completed",
		zap.Int64("total", totalCount),
		zap.Int("migrated", migratedCount),
		zap.Int("skipped", skippedCount),
	)

	response.GinSuccess(c, gin.H{
		"total":    totalCount,
		"migrated": migratedCount,
		"skipped":  skippedCount,
		"message":  fmt.Sprintf("成功迁移 %d 条配置（跳过 %d 条已存在记录）", migratedCount, skippedCount),
	})
}

// migrateOneHuaweiConfig 已被内联到 MigrateHuaweiConfigsHandler 的 PERF-005 闭包中，
// 因为 HuaweiConfigRow 是 handler 内本地类型，无法在包级 helper 中引用。
