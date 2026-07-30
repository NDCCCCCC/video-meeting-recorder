package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuditMiddleware 审计日志中间件
type AuditMiddleware struct {
	auditService *audit.AuditLogService
	logger       *zap.Logger
	sanitizer    *audit.Sanitizer
}

// NewAuditMiddleware 创建审计日志中间件
// sanitizer 用于在 NewData 落库前脱敏（密码/token/手机号/邮箱等敏感字段），
// 为 nil 时退化为不过滤（仅用于测试场景）。
func NewAuditMiddleware(auditService *audit.AuditLogService, logger *zap.Logger, sanitizer *audit.Sanitizer) *AuditMiddleware {
	return &AuditMiddleware{
		auditService: auditService,
		logger:       logger,
		sanitizer:    sanitizer,
	}
}

// AuditOperation 审计操作
// module: 模块名称 (如 "user", "role", "task" 等)
// action: 操作类型 (如 "create", "update", "delete" 等)
func (m *AuditMiddleware) AuditOperation(module, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 读取请求体（用于记录请求数据）
		var requestBody map[string]interface{}
		if c.Request.Method != "GET" && c.Request.ContentLength > 0 {
			body, err := io.ReadAll(c.Request.Body)
			if err == nil && len(body) > 0 {
				// 重新设置请求体，以便后续处理器可以读取
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
				if err := json.Unmarshal(body, &requestBody); err != nil {
					m.logger.Warn("请求体 JSON 解析失败", zap.Error(err))
					requestBody = nil
				}
			}
		}

		// 处理请求
		c.Next()

		// 构建审计日志请求
		req := &audit.LogOperationRequest{
			UserID:    0, // 将从上下文获取
			Username:  "",
			RoleID:    0,
			RoleName:  "",
			Action:    action,
			Module:    module,
			RequestID: c.GetString("request_id"),
			TraceID:   c.GetString("trace_id"),
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			IPAddress: c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Duration:  time.Since(start).Milliseconds(),
		}

		// 敏感字段脱敏（密码/token/手机号/邮箱/身份证等），避免凭据类写操作
		// 把明文写进 audit_logs 表。Sanitizer 可能返回 map 或其它类型（取决于
		// 请求体结构），仅当返回 map 时才填入 NewData。
		if m.sanitizer != nil && requestBody != nil {
			sanitized := m.sanitizer.Sanitize(requestBody)
			if sanitizedMap, ok := sanitized.(map[string]interface{}); ok {
				req.NewData = sanitizedMap
			} else {
				// 非 map 类型（如纯字符串、数组），NewData 留空，绝不写入原文
				req.NewData = nil
			}
		}

		// 从上下文获取用户信息
		if userID, exists := c.Get("user_id"); exists {
			if id, ok := userID.(uint); ok {
				req.UserID = id
			}
		}
		if username, exists := c.Get("username"); exists {
			if name, ok := username.(string); ok {
				req.Username = name
			}
		}
		if roleID, exists := c.Get("role_id"); exists {
			if id, ok := roleID.(uint); ok {
				req.RoleID = id
			}
		}
		if roleName, exists := c.Get("role_name"); exists {
			if name, ok := roleName.(string); ok {
				req.RoleName = name
			}
		}

		// 根据响应状态设置结果
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			req.Status = models.StatusFailure
			req.ErrorMsg = "请求失败"
			req.ErrorCode = ""
		} else {
			req.Status = models.StatusSuccess
		}

		// 异步记录（不阻塞响应）
		if err := m.auditService.LogOperation(c.Request.Context(), req); err != nil {
			m.logger.Warn("记录审计日志失败",
				zap.Error(err),
				zap.String("module", module),
				zap.String("action", action),
			)
		}
	}
}

// AuditLogin 审计登录操作
func (m *AuditMiddleware) AuditLogin(c *gin.Context) {
	start := time.Now()

	c.Next()

	req := &audit.LogOperationRequest{
		Action:    models.ActionLogin,
		Module:    models.ModuleUser,
		RequestID: c.GetString("request_id"),
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Duration:  time.Since(start).Milliseconds(),
	}

	// 从上下文获取用户信息
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			req.UserID = id
		}
	}
	if username, exists := c.Get("username"); exists {
		if name, ok := username.(string); ok {
			req.Username = name
		}
	}

	// 登录成功/失败判断
	if c.Writer.Status() >= 400 {
		req.Status = models.StatusFailure
		req.ErrorMsg = "登录失败"
	} else {
		req.Status = models.StatusSuccess
	}

	m.auditService.LogOperation(c.Request.Context(), req)
}

// AuditLogout 审计登出操作
func (m *AuditMiddleware) AuditLogout(c *gin.Context) {
	start := time.Now()

	// 登出前先获取用户信息
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	c.Next()

	req := &audit.LogOperationRequest{
		Action:    models.ActionLogout,
		Module:    models.ModuleUser,
		RequestID: c.GetString("request_id"),
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Duration:  time.Since(start).Milliseconds(),
		Status:    models.StatusSuccess,
	}

	// 从上下文获取用户信息
	if id, ok := userID.(uint); ok {
		req.UserID = id
	}
	if name, ok := username.(string); ok {
		req.Username = name
	}

	m.auditService.LogOperation(c.Request.Context(), req)
}
