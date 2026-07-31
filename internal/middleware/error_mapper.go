package middleware

import (
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorMapper 是 backstop 错误映射中间件（STYLE-001 决策 3 组件 C）。
//
// c.Next() 后，若响应尚未写入且 c.Errors 非空，则把最后一个 error 通过
// response.HandleError 映射为 HTTP 响应，避免客户端因 handler 只 c.Error(err)
// 未写响应而收到空体。
//
// 零行为风险保证：此中间件不改变任何现有 handler 行为——handler 仍可自行 c.JSON
// 错误响应；仅当 handler 通过 c.Error(err) 记录错误但未写入响应时，本中间件兜底。
// c.Writer.Written() 守卫防止与 handler 自身响应双写。
func ErrorMapper(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// handler 已写入响应 → 不干预，防双写。
		if c.Writer.Written() {
			return
		}
		if len(c.Errors) == 0 {
			return
		}

		lastErr := c.Errors.Last().Err
		if lastErr == nil {
			return
		}

		recognized := response.HandleError(c, lastErr)
		if recognized {
			logger.Warn("未处理的 service 错误被 backstop 映射（已知 sentinel）",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Error(lastErr))
		} else {
			logger.Warn("未处理的 service 错误被 backstop 映射（未知 → 500）",
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.Error(lastErr))
		}
	}
}
