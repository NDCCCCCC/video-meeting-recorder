package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// APIKeyAuth API Key认证中间件
func APIKeyAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权：缺少API Key"})
			c.Abort()
			return
		}

		// 查询API Key
		var key models.APIKey
		err := db.Preload("User").Preload("User.Role").Where("key = ?", apiKey).First(&key).Error
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "无效的API Key"})
			c.Abort()
			return
		}

		// 检查状态
		if !key.IsActive || key.IsExpired() {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "API Key已失效"})
			c.Abort()
			return
		}

		// 检查用户状态
		if !key.User.IsActive {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "用户已被禁用"})
			c.Abort()
			return
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

		c.Next()
	}
}

// MultiAuth 支持多种认证方式（JWT或API Key）
func MultiAuth(db *gorm.DB, jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先尝试API Key认证
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			APIKeyAuth(db)(c)
			return
		}

		// 再尝试JWT认证
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				JWTAuth(jwtService)(c)
				return
			}
		}

		c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权"})
		c.Abort()
	}
}
