package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cpic/record_v2/internal/auth"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 用于存储请求开始时间
const requestStartTimeKey = "api_key_request_start_time"

// logAPIKeyUsage 记录 API Key 使用日志
func logAPIKeyUsage(c *gin.Context, db *gorm.DB, apiKeyID, userID uint, statusCode int, duration time.Duration) {
	// 在 goroutine 启动前提取所有数据，避免在请求完成后访问被回收的 Context
	method := c.Request.Method
	path := c.Request.URL.Path
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	durationMs := int(duration.Milliseconds())

	go func() {
		log := &models.APIKeyUsageLog{
			APIKeyID:   apiKeyID,
			UserID:     userID,
			Method:     method,
			Path:       path,
			StatusCode: statusCode,
			ClientIP:   clientIP,
			UserAgent:  userAgent,
			Duration:   durationMs,
			Success:    statusCode < 400,
		}
		db.Create(log)
	}()
}

// responseWriter 用于捕获响应状态码
type responseWriter struct {
	gin.ResponseWriter
	status     int
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.status = statusCode
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) Status() int {
	return w.status
}

// getContextTime 辅助函数，从 context 获取时间
func getContextTime(c *gin.Context, key string) time.Time {
	if v, exists := c.Get(key); exists {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

// MultiAuth 支持多种认证方式（SM4 Token或API Key）
func MultiAuth(db *gorm.DB, tokenService *auth.SM4TokenService) gin.HandlerFunc {
	// 预创建 SM4 认证 handler，避免每次请求都分配新闭包
	sm4Handler := SM4Auth(tokenService)

	return func(c *gin.Context) {
		// 先尝试API Key认证
		if handleAPIKeyAuth(c, db) {
			return
		}

		// 再尝试SM4 Token认证（extractToken 统一处理 Authorization 头和下载端点的 ?token= 查询参数）
		// SM4Auth 内部会再次调用 extractToken 获取同样的 token
		if extractToken(c) != "" {
			sm4Handler(c)
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权"})
		c.Abort()
	}
}

// handleAPIKeyAuth 处理 API Key 认证，返回 true 表示已处理（成功或失败），false 表示跳过
func handleAPIKeyAuth(c *gin.Context, db *gorm.DB) bool {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		return false
	}

	var key models.APIKey
	clientIP := c.ClientIP()
	err := db.Preload("User").Preload("User.Role").Where("key = ?", apiKey).First(&key).Error
	if err != nil {
		// API Key 格式不对或不存在，继续尝试其他认证方式
		return false
	}

	if !key.IsActive || key.IsExpired() {
		c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "API Key已失效"})
		c.Abort()
		return true
	}
	if !key.User.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "用户已被禁用"})
		c.Abort()
		return true
	}
	if !key.IsIPAllowed(clientIP) {
		c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "IP地址不在白名单中"})
		c.Abort()
		return true
	}

	// 更新最后使用时间
	now := time.Now()
	key.LastUsedAt = &now
	db.Save(&key)

	// 将用户信息存入context
	c.Set("user_id", key.UserID)
	c.Set("username", key.User.Username)
	c.Set("role_id", key.User.RoleID)
	c.Set("api_key_id", key.ID)
	c.Set("auth_type", "apikey")

	// 设置作用域和权限继承信息
	c.Set("scopes", key.GetScopeList())
	c.Set("inherit_perms", key.InheritPerms)

	// 如果继承权限，设置用户权限信息
	if key.InheritPerms && key.User.Role != nil {
		var permissions []string
		for _, perm := range key.User.Role.Permissions {
			permStr := perm.Resource + ":" + perm.Action
			permissions = append(permissions, permStr)
		}
		c.Set("permissions", permissions)
		c.Set("is_admin", key.User.Role.Name == models.RoleAdmin)
	}

	// 记录请求开始时间
	c.Set(requestStartTimeKey, time.Now())

	// 使用响应写入器来捕获状态码
	writer := &responseWriter{ResponseWriter: c.Writer, status: http.StatusOK}
	c.Writer = writer

	c.Next()

	// 请求完成后记录使用日志
	logAPIKeyUsage(c, db, key.ID, key.UserID, writer.status, time.Since(getContextTime(c, requestStartTimeKey)))
	return true
}

// RequireScope 验证 API Key 是否具有所需作用域
// requiredScope: read, write, admin
// 如果用户使用 Token 认证，此中间件会跳过检查（由 RequirePermission 处理）
func RequireScope(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authType := c.GetString("auth_type")
		// 如果不是 API Key 认证，跳过检查
		if authType != "apikey" {
			c.Next()
			return
		}

		// 获取 API Key 信息
		_, exists := c.Get("api_key_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "无效的认证信息"})
			c.Abort()
			return
		}

		// 如果继承权限，检查用户权限
		inheritPerms := c.GetBool("inherit_perms")
		if inheritPerms {
			// 检查用户是否为管理员或具有相应权限
			permissions, exists := c.Get("permissions")
			if !exists {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
				c.Abort()
				return
			}

			permList, ok := permissions.([]string)
			if !ok {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限验证失败"})
				c.Abort()
				return
			}

			// 检查是否有 admin 权限
			hasAdmin := false
			for _, perm := range permList {
				if perm == "admin" || perm == "*" {
					hasAdmin = true
					break
				}
			}
			if hasAdmin {
				c.Next()
				return
			}

			// 根据作用域检查权限
			// write 权限包含 read，admin 包含所有
			hasPermission := false
			switch requiredScope {
			case models.ScopeRead:
				// 检查是否有任何资源的读取权限
				for _, perm := range permList {
					if parts := strings.Split(perm, ":"); len(parts) == 2 && parts[1] == "view" {
						hasPermission = true
						break
					}
				}
			case models.ScopeWrite:
				// 检查是否有任何资源的写入/编辑/创建权限
				for _, perm := range permList {
					if parts := strings.Split(perm, ":"); len(parts) == 2 &&
						(parts[1] == "edit" || parts[1] == "create" || parts[1] == "delete") {
						hasPermission = true
						break
					}
				}
			case models.ScopeAdmin:
				// admin 作用域需要管理员权限
				isAdmin := c.GetBool("is_admin")
				hasPermission = isAdmin
			}

			if !hasPermission {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
				c.Abort()
				return
			}
		} else {
			// 不继承权限，检查 API Key 自定义作用域
			scopes, exists := c.Get("scopes")
			if !exists {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "API Key 未配置权限"})
				c.Abort()
				return
			}

			scopeList, ok := scopes.([]string)
			if !ok {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限验证失败"})
				c.Abort()
				return
			}

			// 检查作用域
			if !checkScope(scopeList, requiredScope) {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// checkScope 检查作用域是否满足要求
func checkScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == models.ScopeAdmin {
			return true // admin 拥有所有权限
		}
		if scope == models.ScopeWrite && (required == models.ScopeRead || required == models.ScopeWrite) {
			return true // write 权限包含 read
		}
		if scope == required {
			return true
		}
	}
	return false
}

// RequireAPIKeyResourcePermission 验证 API Key 是否具有特定资源的操作权限
// 将作用域（read/write/admin）映射到具体资源操作
func RequireAPIKeyResourcePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authType := c.GetString("auth_type")
		// 如果不是 API Key 认证，跳过检查
		if authType != "apikey" {
			c.Next()
			return
		}

		inheritPerms := c.GetBool("inherit_perms")
		if inheritPerms {
			// 继承权限，检查用户是否有特定资源权限
			permissions, exists := c.Get("permissions")
			if !exists {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
				c.Abort()
				return
			}

			permList, ok := permissions.([]string)
			if !ok {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限验证失败"})
				c.Abort()
				return
			}

			// 检查是否有 admin 权限
			hasAdmin := false
			for _, perm := range permList {
				if perm == "admin" || perm == "*" {
					hasAdmin = true
					break
				}
			}
			if hasAdmin {
				c.Next()
				return
			}

			// 检查特定资源权限
			requiredPerm := resource + ":" + action
			hasPermission := false
			for _, perm := range permList {
				if perm == requiredPerm {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
				c.Abort()
				return
			}
		} else {
			// 不继承权限，将资源操作映射到作用域
			// view -> read, create/edit/delete -> write
			requiredScope := models.ScopeRead
			if action == "create" || action == "edit" || action == "delete" {
				requiredScope = models.ScopeWrite
			}

			scopes, exists := c.Get("scopes")
			if !exists {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "API Key 未配置权限"})
				c.Abort()
				return
			}

			scopeList, ok := scopes.([]string)
			if !ok {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限验证失败"})
				c.Abort()
				return
			}

			if !checkScope(scopeList, requiredScope) {
				c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// RateLimitMiddleware API Key 速率限制中间件
// 需要在 APIKeyAuth 或 MultiAuth 之后使用
func RateLimitMiddleware(limiter interface {
	CheckRateLimit(uint, services.RateLimitConfig) (bool, int64, time.Time)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		authType := c.GetString("auth_type")
		// 只对 API Key 认证进行速率限制
		if authType != "apikey" {
			c.Next()
			return
		}

		apiKeyID, exists := c.Get("api_key_id")
		if !exists {
			c.Next()
			return
		}

		keyID, ok := apiKeyID.(uint)
		if !ok {
			c.Next()
			return
		}

		// 使用默认配置检查速率限制
		allowed, remaining, resetTime := limiter.CheckRateLimit(keyID, services.DefaultRateLimitConfig)

		// 设置响应头
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    response.CodeTooManyRequests,
				"message": "请求过于频繁，请稍后再试",
				"data": gin.H{
					"reset_time": resetTime.Format(time.RFC3339),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitByScope 根据作用域应用不同的速率限制
func RateLimitByScope(limiter interface {
	CheckRateLimit(uint, services.RateLimitConfig) (bool, int64, time.Time)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		authType := c.GetString("auth_type")
		if authType != "apikey" {
			c.Next()
			return
		}

		apiKeyID, exists := c.Get("api_key_id")
		if !exists {
			c.Next()
			return
		}

		keyID, ok := apiKeyID.(uint)
		if !ok {
			c.Next()
			return
		}

		// 根据作用域确定速率限制
		config := services.DefaultRateLimitConfig

		// 检查是否为只读作用域
		scopes, hasScope := c.Get("scopes")
		inheritPerms := c.GetBool("inherit_perms")

		if inheritPerms {
			// 继承权限，使用较高限制
			config = services.RateLimitConfig{
				RequestsPerMinute: 120,
				RequestsPerHour:   2000,
				RequestsPerDay:    20000,
			}
		} else if hasScope {
			scopeList, ok := scopes.([]string)
			if ok {
				hasWrite := false
				hasAdmin := false
				for _, scope := range scopeList {
					if scope == models.ScopeAdmin {
						hasAdmin = true
						break
					}
					if scope == models.ScopeWrite {
						hasWrite = true
					}
				}

				if hasAdmin {
					// 管理员作用域，更高限制
					config = services.RateLimitConfig{
						RequestsPerMinute: 200,
						RequestsPerHour:   5000,
						RequestsPerDay:    50000,
					}
				} else if hasWrite {
					// 写入作用域，中等限制
					config = services.RateLimitConfig{
						RequestsPerMinute: 100,
						RequestsPerHour:   1500,
						RequestsPerDay:    15000,
					}
				} else {
					// 只读作用域，较低限制
					config = services.RateLimitConfig{
						RequestsPerMinute: 30,
						RequestsPerHour:   500,
						RequestsPerDay:    5000,
					}
				}
			}
		}

		allowed, remaining, resetTime := limiter.CheckRateLimit(keyID, config)

		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", resetTime.Format(time.RFC3339))

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    response.CodeTooManyRequests,
				"message": "请求过于频繁，请稍后再试",
				"data": gin.H{
					"reset_time": resetTime.Format(time.RFC3339),
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
