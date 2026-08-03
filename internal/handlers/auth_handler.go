package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *auth.Service
	logger      *zap.Logger
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authService *auth.Service, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户名密码登录，返回SM4-GCM加密Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body auth.LoginRequest true "登录请求"
// @Success 200 {object} response.Response{data=auth.LoginResponse}
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取客户端信息
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// 调用登录服务
	result, err := h.authService.Login(c.Request.Context(), &req, ipAddress, userAgent)
	if err != nil {
		h.logger.Warn("Login failed",
			zap.String("username", req.Username),
			zap.Error(err),
			response.SentinelField(err),
		)
		// Phase 20 (20-02): Login 错误统一走 response.HandleError；mapping.go 通过
		// errors.Is 链自动识别 sentinel → 对应 401/403/404/503/500 状态码。
		//   - ErrADUserNotRegistered → 403 (R-3 要求)。
		//   - ErrADConfigError / ErrADUnreachable → 503 (R-4: 500 → 503)。
		response.HandleError(c, err)
		return
	}

	h.logger.Info("User logged in",
		zap.String("username", req.Username),
		zap.Uint("user_id", result.User.ID),
	)

	response.GinSuccess(c, result)
}

// RefreshToken 刷新Token
// @Summary 刷新Token
// @Description 使用Refresh Token刷新Access Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body object{refreshToken=string} true "刷新Token请求"
// @Success 200 {object} response.Response{data=auth.RefreshTokenResponse}
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	result, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		response.GinError(c, response.CodeInvalidToken, err.Error())
		return
	}

	response.GinSuccess(c, result)
}

// Logout 用户登出
// @Summary 用户登出
// @Description 登出当前用户，撤销当前Token
// @Tags 认证
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetString("token")
	if token == "" {
		response.GinError(c, response.CodeInvalidRequest, "无效的请求")
		return
	}

	if err := h.authService.Logout(token); err != nil {
		h.logger.Error("Logout failed", zap.Error(err), response.SentinelField(err))
	}

	response.GinSuccess(c, gin.H{
		"message": "登出成功",
	})
}

// LogoutAll 登出所有设备
// @Summary 登出所有设备
// @Description 撤销用户所有活动会话
// @Tags 认证
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}

	if err := h.authService.LogoutAll(userID); err != nil {
		h.logger.Error("Logout all failed",
			zap.Uint("user_id", userID),
			zap.Error(err),
			response.SentinelField(err),
		)
		response.GinError(c, response.CodeInternalError, "操作失败")
		return
	}

	response.GinSuccess(c, gin.H{
		"message": "已登出所有设备",
	})
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前用户密码
// @Tags 认证
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body auth.ChangePasswordRequest true "修改密码请求"
// @Success 200 {object} response.Response
// @Router /api/v1/auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req auth.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		response.GinError(c, response.CodeInvalidPassword, err.Error())
		return
	}

	h.logger.Info("User changed password", zap.Uint("user_id", userID))

	response.GinSuccess(c, gin.H{
		"message": "密码修改成功",
	})
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户信息
// @Description 获取当前登录用户的详细信息
// @Tags 认证
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response{data=auth.UserDTO}
// @Router /api/v1/auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}

	user, err := h.authService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "用户不存在")
		return
	}

	response.GinSuccess(c, user)
}

// ValidatePassword 验证密码强度
// @Summary 验证密码强度
// @Description 验证密码是否符合安全策略
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body object{password=string} true "密码"
// @Success 200 {object} response.Response{data=auth.ValidationResult}
// @Router /api/v1/auth/validate-password [post]
func (h *AuthHandler) ValidatePassword(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	result := h.authService.ValidatePassword(req.Password)
	response.GinSuccess(c, result)
}

// TestADConnection 测试AD连接
// @Summary 测试AD域控连接
// @Description 验证AD配置是否正确（四层验证，per Spike 005）
// @Tags 认证
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body auth.ADAuthConfig true "AD配置"
// @Success 200 {object} response.Response{data=auth.ADConfigValidationResult}
// @Router /api/v1/auth/ad/test-connection [post]
func (h *AuthHandler) TestADConnection(c *gin.Context) {
	var req auth.ADAuthConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	validator := auth.NewADConfigValidator(h.logger)
	result := validator.Validate(&req)

	// Always return consistent format with response.GinSuccess
	// Validation result (success/failure) is indicated by result.Valid field
	response.GinSuccess(c, result)
}
