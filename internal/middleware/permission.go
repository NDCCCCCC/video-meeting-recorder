package middleware

import (
	"net/http"

	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequirePermission 需要特定权限
func RequirePermission(db *gorm.DB, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权"})
			c.Abort()
			return
		}

		// 加载用户和权限
		var user models.User
		if err := db.Preload("Role").Preload("Role.Permissions").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "message": "加载用户信息失败"})
			c.Abort()
			return
		}

		// 检查权限
		if !user.HasPermission(resource, action) {
			c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole 需要特定角色
func RequireRole(db *gorm.DB, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权"})
			c.Abort()
			return
		}

		var user models.User
		if err := db.Preload("Role").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "message": "加载用户信息失败"})
			c.Abort()
			return
		}

		// 检查角色
		hasRole := false
		for _, role := range roles {
			if user.Role.Name == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOwnershipOrRole 要求资源所有者或特定角色
func RequireOwnershipOrRole(db *gorm.DB, ownerIDField string, roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权"})
			c.Abort()
			return
		}

		// 获取资源所有者ID（需要从context或参数中获取）
		resourceOwnerID := c.GetUint(ownerIDField)

		// 检查是否为所有者
		if resourceOwnerID == userID {
			c.Next()
			return
		}

		// 检查角色
		var user models.User
		if err := db.Preload("Role").First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": response.CodeInternalError, "message": "加载用户信息失败"})
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range roles {
			if user.Role.Name == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"code": response.CodeForbidden, "message": "权限不足"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireDataScope 数据范围权限中间件
// ownerIDField: 模型中所有者字段的名称，如 "created_by"
// 管理员可以访问所有数据，非管理员只能访问自己创建的数据
func RequireDataScope(ownerIDField string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		isAdmin := c.GetBool("is_admin")

		// 管理员可以访问所有数据
		if isAdmin {
			c.Next()
			return
		}

		// 非管理员只能访问自己创建的数据
		if userID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": response.CodeUnauthorized, "message": "未授权"})
			c.Abort()
			return
		}

		c.Set("data_scope_owner_id", userID)
		c.Set("data_scope_field", ownerIDField)
		c.Next()
	}
}

// GetDataScopeFilter 获取数据范围过滤条件
// 返回: (field, value, shouldApply)
// field: 所有者字段名
// value: 用户ID
// shouldApply: 是否应该应用过滤（管理员返回false）
func GetDataScopeFilter(c *gin.Context) (string, uint, bool) {
	field, exists := c.Get("data_scope_field")
	if !exists {
		return "", 0, false
	}
	ownerID, exists := c.Get("data_scope_owner_id")
	if !exists {
		return "", 0, false
	}
	return field.(string), ownerID.(uint), true
}

// SetUserAuthContext 从 JWT Claims 设置用户认证上下文
func SetUserAuthContext(c *gin.Context, claims interface{}) {
	c.Set("user_claims", claims)
}
