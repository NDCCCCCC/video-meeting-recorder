package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AdminHandler 管理员后台 handler：负责认证配置查询/更新、华为配置迁移、
// 用户查找、配置 diff 与审计可视化等管理操作。鉴权由 RequireRole("admin") 中间件保证。
type AdminHandler struct {
	cfg           *config.Config
	logger        *zap.Logger
	configService *services.ConfigService
	authService   *auth.Service
	db            *gorm.DB
	encryptor     *services.CredentialEncryptor // Phase 18: 用于 MigrateInputConfigs 加密历史明文
}

// NewAdminHandler 构造 AdminHandler,集中处理 admin 路由组的配置/用户/迁移相关 handler。
// cfg + authService 用于 /config 端点读写;db 直接用于管理级 SQL;encryptor 负责
// 迁移期间的凭据信封封装(参见 SubmitAdminMigration)。
func NewAdminHandler(cfg *config.Config, logger *zap.Logger, configService *services.ConfigService, authService *auth.Service, db *gorm.DB, encryptor *services.CredentialEncryptor) *AdminHandler {
	return &AdminHandler{
		cfg:           cfg,
		logger:        logger,
		configService: configService,
		authService:   authService,
		db:            db,
		encryptor:     encryptor,
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
		if err := h.configService.SaveAuthConfig(c.Request.Context(), req.Mode, &req.AD); err != nil {
			h.logger.Error("Failed to save auth config to database", zap.Error(err), response.SentinelField(err))
			response.HandleError(c, err)
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
		h.logger.Error("AD user lookup failed", zap.String("username", req.Username), zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
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

	// Count existing huawei_configs（BUG-005: 透传 ctx 让客户端断开能级联取消）
	var totalCount int64
	if err := tx.WithContext(c.Request.Context()).Table("huawei_configs").Count(&totalCount).Error; err != nil {
		h.logger.Error("Failed to count huawei_configs", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
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
	if err := tx.WithContext(c.Request.Context()).Table("huawei_configs").Limit(1000).Find(&huaweiConfigs).Error; err != nil {
		h.logger.Error("Failed to fetch huawei_configs", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
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

			// Phase 18: 在 INSERT 前用 encryptor 把明文 Password / StreamPassword 包成 envelope。
			// 不在 INSERT 前重置 hc.Password，因为 tx.Exec 会用 hc.Password 直接写入 DB。
			pw := hc.Password
			if pw != "" && h.encryptor != nil {
				enc, eerr := h.encryptor.Encrypt(pw)
				if eerr != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("encrypt password (name=%s): %w", hc.Name, eerr)
					}
					mu.Unlock()
					return
				}
				pw = enc
			}
			spw := hc.StreamPassword
			if spw != "" && h.encryptor != nil {
				enc, eerr := h.encryptor.Encrypt(spw)
				if eerr != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("encrypt stream_password (name=%s): %w", hc.Name, eerr)
					}
					mu.Unlock()
					return
				}
				spw = enc
			}

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
				hc.Server, hc.Port, hc.Username, pw, hc.TerminalNumber, hc.ConferenceNumber,
				hc.CameraBackend, hc.USBCameraName, hc.USBCameraDevice, hc.CameraBindingStatus,
				hc.AudioBackend, hc.USBAudioName, hc.USBAudioDevice, hc.AudioBindingStatus,
				hc.OutputFormat,
				hc.StreamProtocol, hc.StreamURL, hc.StreamUsername, spw, hc.StreamEnabled,
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
		h.logger.Error("Failed to migrate config", zap.Error(firstErr), response.SentinelField(firstErr))
		tx.Rollback()
		response.HandleError(c, firstErr)
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		h.logger.Error("Failed to commit migration", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
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

// ============================================================================
// PERF-005/PR-F: 异步 admin migration
// ============================================================================
//
// 历史：MigrateInputConfigs 同事务内扫描全表 huawei_configs 并 worker pool 加密+写入
// input_configs,典型 50+ 行配置一次会阻塞 HTTP 线程 10-60 秒;client timeout 与中间件
// recover 都会触发,运维复测 "再迁一次" 会发现连接被服务端主动断开。
//
// 现拆为：
//   POST /api/v1/admin/migrate-input-configs        → SubmitAdminMigration (202 + job_id)
//   GET  /api/v1/admin/migrate-input-configs/:job_id → GetAdminMigrationStatus (200 + 进度)
// 实现保持单一事务语义（进 goroutine 后重新开 tx），避免把 admin_handler 业务大幅改写。

// SubmitAdminMigration 同步返回 202 + job_id,后台 goroutine 执行迁移。
// 复制原 MigrateInputConfigs 的核心流程(并发 worker + CredentialEncryptor)到 runMigrationJob 内。
//
// @Summary 提交华为配置迁移任务
// @Tags 系统管理
// @Security Bearer
// @Success 202 {object} response.Response{data=map[string]interface{}}
// @Router /api/v1/admin/migrate-input-configs [post]
func (h *AdminHandler) SubmitAdminMigration(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}

	// 立即插入 job 行 (Status=pending),让 progress endpoint 有数据可读。
	job := models.AdminMigrationJob{
		Status:      models.AdminMigrationStatusPending,
		RequestedBy: userID,
		StartedAt:   time.Now(),
	}
	// BUG-005: 透传请求 ctx
	if err := h.db.WithContext(c.Request.Context()).Create(&job).Error; err != nil {
		h.logger.Error("create admin migration job failed", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
		return
	}

	// 派生子 ctx：5 分钟超时；与请求 ctx 解耦（避免客户端断开取消整个迁移）。
	jobCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	go func(jobID uint) {
		defer cancel()
		// panic 防护 — 与 BUG-002 一致。
		defer func() {
			if r := recover(); r != nil {
				now := time.Now()
				h.db.WithContext(jobCtx).Model(&models.AdminMigrationJob{}).Where("id = ?", jobID).Updates(map[string]interface{}{
					"status":      models.AdminMigrationStatusFailed,
					"finished_at": &now,
					"error_msg":   fmt.Sprintf("panic: %v", r),
				})
				h.logger.Error("admin migration panicked",
					zap.Uint("job_id", jobID),
					zap.Any("recover", r),
					zap.Stack("stack"))
			}
		}()

		h.db.WithContext(jobCtx).Model(&models.AdminMigrationJob{}).Where("id = ?", jobID).Updates(map[string]interface{}{
			"status": models.AdminMigrationStatusRunning,
		})

		total, migrated, skipped, err := h.runMigrationJob(jobCtx)
		finishedAt := time.Now()
		updates := map[string]interface{}{
			"finished_at": &finishedAt,
			"total":       total,
			"migrated":    migrated,
			"skipped":     skipped,
		}
		if err != nil {
			updates["status"] = models.AdminMigrationStatusFailed
			updates["error_msg"] = err.Error()
		} else {
			updates["status"] = models.AdminMigrationStatusCompleted
		}
		h.db.WithContext(jobCtx).Model(&models.AdminMigrationJob{}).Where("id = ?", jobID).Updates(updates)

		h.logger.Info("admin migration finished",
			zap.Uint("job_id", jobID),
			zap.Int("total", total),
			zap.Int("migrated", migrated),
			zap.Int("skipped", skipped),
			zap.Error(err),
		)
	}(job.ID)

	c.Header("Location", fmt.Sprintf("/api/v1/admin/migrate-input-configs/%d", job.ID))
	c.JSON(202, gin.H{
		"code":    response.CodeSuccess,
		"message": "已提交,异步执行中",
		"data": gin.H{
			"job_id":      job.ID,
			"status":      models.AdminMigrationStatusPending,
			"status_url":  fmt.Sprintf("/api/v1/admin/migrate-input-configs/%d", job.ID),
		},
	})
}

// GetAdminMigrationStatus 查询异步迁移进度
//
// @Summary 查询 admin migration 进度
// @Tags 系统管理
// @Security Bearer
// @Param job_id path int true "任务 ID"
// @Success 200 {object} response.Response{data=models.AdminMigrationJob}
// @Router /api/v1/admin/migrate-input-configs/{job_id} [get]
func (h *AdminHandler) GetAdminMigrationStatus(c *gin.Context) {
	idStr := c.Param("job_id")
	jobID, err := parseUintParam(c, "job_id")
	if err != nil || jobID == 0 || idStr == "" {
		response.GinError(c, response.CodeInvalidRequest, "无效的 job_id")
		return
	}

	var job models.AdminMigrationJob
	// BUG-005: 透传 ctx
	if err := h.db.WithContext(c.Request.Context()).First(&job, jobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.GinError(c, response.CodeNotFound, "job not found")
			return
		}
		response.HandleError(c, err)
		return
	}
	response.GinSuccess(c, job)
}

// runMigrationJob 抽离出原 MigrateInputConfigs 的核心事务逻辑,接收 ctx 便于 cancel。
// 保留原 bounded-concurrency (cfg.Admin.MigrationConcurrency)。
//
// 实现要点（PERF-005 真实 worker pool 修复）：
//   - 进 goroutine 后**重新开 tx**（request ctx 已断开，使用 jobCtx）
//   - worker 数 = min(cfg.Admin.MigrationConcurrency, len(rows))，默认 4
//   - 每 worker 独立 defer recover — panic 不会拖垮整个 job
//   - 分发前检查 ctx.Err()，避免在 ctx 取消后还启动新 goroutine
//   - first-error-wins via mu+firstErr；其他 worker 完成后会被 wg.Wait 收尾
//   - 复用原 MigrateInputConfigs 的 CredentialEncryptor 信封（Phase 18）
func (h *AdminHandler) runMigrationJob(ctx context.Context) (total, migrated, skipped int, err error) {
	if cerr := ctx.Err(); cerr != nil {
		return 0, 0, 0, cerr
	}

	// 在 goroutine 内新开 tx：避免使用外层 request tx（request ctx 已断开）。
	tx := h.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, 0, 0, tx.Error
	}
	committed := false
	defer func() {
		// 任何 panic 路径都触发 Rollback（commit 未执行时）；callSite 自行
		// 在 SubmitAdminMigration 的外层 defer recover 写 status=failed。
		if !committed {
			tx.Rollback()
		}
	}()

	// 1. 计数
	var totalCount int64
	if e := tx.WithContext(ctx).Table("huawei_configs").Count(&totalCount).Error; e != nil {
		return 0, 0, 0, e
	}

	if totalCount == 0 {
		if e := tx.Commit().Error; e != nil {
			return 0, 0, 0, e
		}
		committed = true
		return 0, 0, 0, nil
	}

	// 2. 拉取源行（沿用原 Limit(1000) — admin 一次迁移 < 1000 行可控）
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
	if e := tx.WithContext(ctx).Table("huawei_configs").Limit(1000).Find(&huaweiConfigs).Error; e != nil {
		return 0, 0, 0, e
	}

	// 3. bounded concurrency worker pool
	concurrency := h.cfg.Admin.MigrationConcurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	if concurrency > len(huaweiConfigs) {
		concurrency = len(huaweiConfigs)
	}

	var migratedCount, skippedCount int
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	sem := make(chan struct{}, concurrency)
	for _, hc := range huaweiConfigs {
		// 分发前再 check ctx — 已被取消时不再启动新 worker。
		if cerr := ctx.Err(); cerr != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = cerr
			}
			mu.Unlock()
			break
		}
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

			// 已有同 name+created_at 记录 → 跳过
			var existingCount int64
			if e := tx.Table("input_configs").
				Where("name = ? AND created_at = ?", hc.Name, hc.CreatedAt).
				Count(&existingCount).Error; e != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = e
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

			// Phase 18: INSERT 前用 encryptor 把明文 Password/StreamPassword 包成 envelope
			pw := hc.Password
			if pw != "" && h.encryptor != nil {
				enc, eerr := h.encryptor.Encrypt(pw)
				if eerr != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("encrypt password (name=%s): %w", hc.Name, eerr)
					}
					mu.Unlock()
					return
				}
				pw = enc
			}
			spw := hc.StreamPassword
			if spw != "" && h.encryptor != nil {
				enc, eerr := h.encryptor.Encrypt(spw)
				if eerr != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("encrypt stream_password (name=%s): %w", hc.Name, eerr)
					}
					mu.Unlock()
					return
				}
				spw = enc
			}

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
			if e := tx.Exec(insertSQL,
				hc.CreatedAt, hc.UpdatedAt, hc.DeletedAt,
				hc.Name, hc.Description, configType, hwEnabled,
				hc.Server, hc.Port, hc.Username, pw, hc.TerminalNumber, hc.ConferenceNumber,
				hc.CameraBackend, hc.USBCameraName, hc.USBCameraDevice, hc.CameraBindingStatus,
				hc.AudioBackend, hc.USBAudioName, hc.USBAudioDevice, hc.AudioBindingStatus,
				hc.OutputFormat,
				hc.StreamProtocol, hc.StreamURL, hc.StreamUsername, spw, hc.StreamEnabled,
				hc.IsActive, hc.IsLocked, hc.LockedBy, hc.LockedAt,
			).Error; e != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = e
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
		return int(totalCount), migratedCount, skippedCount, firstErr
	}

	if e := tx.Commit().Error; e != nil {
		return int(totalCount), migratedCount, skippedCount, e
	}
	committed = true

	return int(totalCount), migratedCount, skippedCount, nil
}
