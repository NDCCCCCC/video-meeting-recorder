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

	// bitSize 取 strconv.IntSize（== uint 位宽）：ParseUint 保证结果落在 uint 范围内，
	// 既避免 32 位平台 uint(id64) 截断，也被 CodeQL 识别为 sanitizer（整数转换告警）。
	id64, err := strconv.ParseUint(idStr, 10, strconv.IntSize)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
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
