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

// JWTAuth JWT认证中间件
func JWTAuth(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从Header获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权：缺少认证令牌"})
			c.Abort()
			return
		}

		// 2. 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权：无效的认证格式"})
			c.Abort()
			return
		}

		// 3. 验证token
		claims, err := jwtService.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeInvalidToken, "message": "Token无效或已过期"})
			c.Abort()
			return
		}

		// 4. 将用户信息存入context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role_id", claims.RoleID)
		c.Set("token", parts[1])

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
			c.Set("token", parts[1])
		}

		c.Next()
	}
}
