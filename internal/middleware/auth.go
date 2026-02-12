package middleware

import (
	"net/http"
	"strings"

	"github.com/cpic/record_v2/internal/auth"
	"github.com/cpic/record_v2/pkg/response"
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

// JWTAuth JWT认证中间件
// 支持 Authorization 头和 token 查询参数（用于视频播放等场景）
func JWTAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 尝试从多个来源获取token
		var tokenString string

		// 优先从 Authorization 头获取
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权：无效的认证格式"})
				c.Abort()
				return
			}
		}

		// 如果 Header 没有，尝试从查询参数获取（用于视频播放）
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		// 2. 检查是否获取到token
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权：缺少认证令牌"})
			c.Abort()
			return
		}

		// 3. 验证token
		claims, err := jwtService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeInvalidToken, "message": "Token无效或已过期"})
			c.Abort()
			return
		}

		// 4. 将用户信息存入context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role_id", claims.RoleID)
		c.Set("is_admin", claims.IsAdmin)
		c.Set("permissions", claims.Permissions)
		c.Set("token", tokenString)

		c.Next()
	}
}

// OptionalAuth 可选认证中间件（允许未认证用户访问）
func OptionalAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		if claims, err := jwtService.ValidateToken(parts[1]); err == nil {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role_id", claims.RoleID)
			c.Set("is_admin", claims.IsAdmin)
			c.Set("permissions", claims.Permissions)
			c.Set("token", parts[1])
		}

		c.Next()
	}
}
