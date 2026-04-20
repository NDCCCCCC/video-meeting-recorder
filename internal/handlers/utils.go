package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// parseUintParam 解析无符号整数参数
func parseUintParam(c *gin.Context, key string) (uint, error) {
	idStr := c.Param(key)
	if idStr == "" {
		return 0, fmt.Errorf("parameter %s is empty", key)
	}

	// 使用 ParseUint 而不是 Sscanf，更安全和准确
	id64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	// 检查是否超出 uint 范围
	if id64 > ^uint64(0)>>1 {
		return 0, fmt.Errorf("%s value too large", key)
	}

	return uint(id64), nil
}

// trimString 去除字符串首尾空白字符
func trimString(s string) string {
	return strings.TrimSpace(s)
}

// containsPathSeparator 检查字符串是否包含路径分隔符
func containsPathSeparator(s string) bool {
	return strings.ContainsAny(s, "/\\")
}
