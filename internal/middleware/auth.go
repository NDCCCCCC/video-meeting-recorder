package middleware

import (
	"net/http"
	"strings"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetUserID 从context获取用户ID
func GetUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint)
	return userID, ok
}

// GetUsername 从context获取用户名
func GetUsername(c *gin.Context) string {
	if value, exists := c.Get("username"); exists {
		if typed, ok := value.(string); ok {
			return typed
		}
	}
	return ""
}

// GetRoleID 从context获取角色ID
func GetRoleID(c *gin.Context) uint {
	if value, exists := c.Get("role_id"); exists {
		if typed, ok := value.(uint); ok {
			return typed
		}
	}
	return 0
}

// GetRoleIDs 从context获取角色ID列表
func GetRoleIDs(c *gin.Context) []uint {
	if value, exists := c.Get("role_ids"); exists {
		if typed, ok := value.([]uint); ok {
			return typed
		}
	}
	return nil
}

// GetIsAdmin 从context获取是否管理员
func GetIsAdmin(c *gin.Context) bool {
	if value, exists := c.Get("is_admin"); exists {
		if typed, ok := value.(bool); ok {
			return typed
		}
	}
	return false
}

// GetHasSharedViewer 从context检查用户是否有 shared_viewer 角色
// RoleSharedViewer ID: 5
func GetHasSharedViewer(c *gin.Context) bool {
	roleIDs := GetRoleIDs(c)
	for _, roleID := range roleIDs {
		if roleID == 5 { // RoleSharedViewer ID
			return true
		}
	}
	return false
}

// CanAccessAllData 检查用户是否可以访问所有数据
// 管理员和 shared_viewers 可以看到所有用户创建的数据
func CanAccessAllData(c *gin.Context) bool {
	return GetIsAdmin(c) || GetHasSharedViewer(c)
}

// SM4Auth SM4-GCM Token认证中间件
// 支持 Authorization 头和 token 查询参数（用于视频播放等场景）
func SM4Auth(tokenService *auth.SM4TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c, tokenService.AllowedTokenURLPrefixes())
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    response.CodeUnauthorized,
				"message": "未授权：缺少认证令牌",
			})
			c.Abort()
			return
		}

		// 验证token
		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			// Phase 19 D7: 走 response.HandleError 让 sentinel 决定 HTTP 状态码；
			// 4 token sentinel 全部映射 401 + 中文化 message，frontend 由 respCode 1002 + 业务
			// 上下文分辨具体原因。Abort 在写响应之后调用，保证后续 handler 不再触发。
			response.HandleError(c, err)
			c.Abort()
			return
		}

		// 将用户信息存入context
		setUserContext(c, claims, tokenString)
		c.Next()
	}
}

// OptionalSM4Auth 可选认证中间件（允许未认证用户访问）
func OptionalSM4Auth(tokenService *auth.SM4TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenFromHeader(c)
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			// Token 验证失败，允许继续（未认证场景）
			c.Next()
			return
		}

		setUserContext(c, claims, tokenString)
		c.Next()
	}
}

// isVideoDownloadEndpoint 检查是否是文件/PPT下载端点
// 支持通过URL查询参数传递token（用于文件下载）
func isAllowedTokenURL(path string, allowedPrefixes []string) bool {
	for _, prefix := range allowedPrefixes {
		if prefix != "" && strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// extractToken 从多个来源提取token
// 注意：token 查询参数仅允许用于视频下载端点（用于 <video> 标签播放）
func extractToken(c *gin.Context, allowedPrefixes []string) string {
	// 优先从 Authorization 头获取
	tokenString := extractTokenFromHeader(c)

	// 如果 Header 没有，且是视频下载端点，才允许从查询参数获取
	// 警告：通过 URL 传递 token 会被记录在服务器日志和浏览器历史中
	if tokenString == "" && isAllowedTokenURL(c.Request.URL.Path, allowedPrefixes) {
		c.Set("token_via_query", true) // 标记使用了查询参数传递 token
		tokenString = c.Query("token")
	}

	return tokenString
}

// extractTokenFromHeader 从 Authorization 头提取token
func extractTokenFromHeader(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// setUserContext 将用户信息存入context
func setUserContext(c *gin.Context, claims *auth.Claims, tokenString string) {
	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("role_ids", claims.RoleIDs)
	c.Set("is_admin", claims.IsAdmin)
	c.Set("permissions", claims.Permissions)
	c.Set("token", tokenString)
}
