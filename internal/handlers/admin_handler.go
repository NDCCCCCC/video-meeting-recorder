package handlers

import (
	"strings"

	"github.com/cpic/record_v2/internal/auth"
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	cfg    *config.Config
	logger *zap.Logger
}

func NewAdminHandler(cfg *config.Config, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		cfg:    cfg,
		logger: logger,
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
		Mode string             `json:"mode" binding:"required,oneof=local ad"`
		AD   auth.ADAuthConfig  `json:"ad"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	currentUserID := middleware.GetUserID(c)

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
	userID := middleware.GetUserID(c)
	username := middleware.GetUsername(c)

	response.GinSuccess(c, gin.H{
		"user_id":  userID,
		"username": username,
	})
}
