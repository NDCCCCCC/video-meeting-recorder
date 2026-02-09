package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// parseUintParam 解析无符号整数参数
func parseUintParam(c *gin.Context, key string) (uint, error) {
	idStr := c.Param(key)
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		return 0, err
	}
	return id, nil
}
