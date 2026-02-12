package middleware

import (
	"net/http"
	"strings"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
)

// GetUserID 从context获取用户ID
func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(uint)
	}
	return 0
}

// GetUsername 从context获取用户名
func GetUsername(c *gin.Context) string {
	if username, exists := c.Get("username"); exists {
		return username.(string)
	}
	return ""
}

// GetRoleID 从context获取角色ID
func GetRoleID(c *gin.Context) uint {
	if roleID, exists := c.Get("role_id"); exists {
		return roleID.(uint)
	}
	return 0
}

// GetIsAdmin 从context获取是否管理员
func GetIsAdmin(c *gin.Context) bool {
	if isAdmin, exists := c.Get("is_admin"); exists {
		return isAdmin.(bool)
	}
	return false
}

// JWTAuth JWT认证中间件
// 支持 Authorization 头和 token 查询参数（用于视频播放等场景）
func JWTAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": response.CodeUnauthorized,
				"message": "未授权：缺少认证令牌",
			})
			c.Abort()
			return
		}

		// 验证token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": response.CodeInvalidToken,
				"message": "Token无效或已过期",
			})
			c.Abort()
			return
		}

		// 将用户信息存入context
		setUserContext(c, claims, tokenString)
		c.Next()
	}
}

// OptionalAuth 可选认证中间件（允许未认证用户访问）
func OptionalAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractTokenFromHeader(c)
		if tokenString == "" {
			c.Next()
			return
		}

		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			// Token 验证失败，允许继续（未认证场景）
			c.Next()
			return
		}

		setUserContext(c, claims, tokenString)
		c.Next()
	}
}

// isVideoDownloadEndpoint 检查是否是视频下载端点
func isVideoDownloadEndpoint(path string) bool {
	return strings.Contains(path, "/files/") && strings.Contains(path, "/download")
}

// extractToken 从多个来源提取token
// 注意：token 查询参数仅允许用于视频下载端点（用于 <video> 标签播放）
func extractToken(c *gin.Context) string {
	// 优先从 Authorization 头获取
	tokenString := extractTokenFromHeader(c)

	// 如果 Header 没有，且是视频下载端点，才允许从查询参数获取
	// 警告：通过 URL 传递 token 会被记录在服务器日志和浏览器历史中
	if tokenString == "" && isVideoDownloadEndpoint(c.Request.URL.Path) {
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
	c.Set("role_id", claims.RoleID)
	c.Set("is_admin", claims.IsAdmin)
	c.Set("permissions", claims.Permissions)
	c.Set("token", tokenString)
}
