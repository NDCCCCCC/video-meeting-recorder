package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CSRF cookie / header 名称常量化,便于前端 / 测试对接。
//
// Double-Submit Cookie 模式:
//  1. 服务端在第一个被允许的请求上设置 _csrf cookie (HttpOnly=false,
//     SameSite=Strict, Secure 推荐) 含 32 字节随机 base64 token。
//  2. 客户端 (浏览器 JS) 读取该 cookie 后,把同值回填到 X-CSRF-Token
//     请求头中提交给服务端。
//  3. 服务端对写请求 (POST/PUT/PATCH/DELETE) 比对 cookie 与 header,任一不一致 -> 403。
//
// 参考:OWASP CSRF Protection Cheat Sheet "Double Submit Cookie"。
const (
	csrfCookieName = "_csrf"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 32
)

// safeMethods 是豁免 CSRF 校验的 HTTP 方法 (无副作用)。
// OPTIONS 始终返回 204 不带 body,无须验证。
var csrfSafeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

// CSRF 返回一个 gin.HandlerFunc,在 cookie 缺失时下发 _csrf,并对写请求执行
// double-submit 比对。CSRFEnabled=false 时不挂载 (工厂 nil-safe, 调用方可直接判空)。
//
// 安全 vs Bearer 认证:
//
//	当前项目的认证以 Authorization: Bearer SM4 token 为主,该凭据浏览器攻击者无法
//	通过跨站请求偷取 (无 cookie 可被自动附加),所以带 Bearer 头的请求视为 CSRF-safe,
//	直接放行。这一行为让启用 CSRF 不会破坏现有 API 客户端,仍能在切换到 Cookie 认证时
//	立即生效。
//
// originCheck (cs.SafeOrigins 非空):
//
//	额外校验 Origin 头必须与列表中某项精确匹配;留空则不附加 origin 检查
//	(仅依赖 double-submit + SameSite=Strict)。
//
// 错误模式:
//
//	任何被拒绝的请求 403 + JSON,logger 记录 path/ip/method 便于审计。
func CSRF(safeOrigins []string, logger *zap.Logger) gin.HandlerFunc {
	originCheckEnabled := len(safeOrigins) > 0
	allowed := make(map[string]struct{}, len(safeOrigins))
	for _, o := range safeOrigins {
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		// 1) 安全方法直接放行 (无副作用)。同时按需下发 cookie 以让客户端后续
		//    写请求具备 token 基础。这是 Double-Submit Cookie 模式的标准 bootstrap。
		if csrfSafeMethods[c.Request.Method] {
			if cookie, _ := c.Cookie(csrfCookieName); cookie == "" {
				issueCSRFCookie(c)
			}
			c.Next()
			return
		}

		// 2) Origin 校验 (opt-in,适用于所有写请求,作为防御深度的横向检查)。
		//    即使请求带 Bearer 头 (CSRF 本来对其不构成威胁),也从 origin 维度
		//    拒绝来自不可信域名的写请求——例如预防未来引入 cookie 后的横向 reuse。
		if originCheckEnabled && !originMatches(c.GetHeader("Origin"), allowed) {
			if logger != nil {
				logger.Warn("csrf origin rejected",
					zap.String("path", c.Request.URL.Path),
					zap.String("origin", c.GetHeader("Origin")),
					zap.String("ip", c.ClientIP()),
				)
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "CSRF_ORIGIN_REJECTED",
				"message": "origin 不在白名单内",
			})
			return
		}

		// 3) 带 Bearer 头的写请求视为 CSRF-safe (依上注释)。这是 "保持现有 API
		//    客户端可用" 与 "未来 Cookie 认证自动覆盖" 的过渡策略。
		if hasBearerToken(c) {
			c.Next()
			return
		}

		// 4) 写入方法:要求 cookie + header 双提交。
		cookie, err := c.Cookie(csrfCookieName)
		if err != nil || cookie == "" {
			// 非 safe 方法上 cookie 缺失一律拒绝 (不下发 cookie,因为写请求不接受回环)。
			// 客户端应先访问任一安全端点 (如 GET /) 让服务端种下 cookie。
			if logger != nil {
				logger.Warn("csrf cookie missing on unsafe method",
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
					zap.String("method", c.Request.Method),
				)
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "CSRF_COOKIE_MISSING",
				"message": "缺少 CSRF cookie，请先访问任一安全端点获取并随请求回填 X-CSRF-Token",
			})
			return
		}

		header := c.GetHeader(csrfHeaderName)
		if header == "" || subtle.ConstantTimeCompare([]byte(cookie), []byte(header)) != 1 {
			if logger != nil {
				logger.Warn("csrf token mismatch",
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
					zap.String("method", c.Request.Method),
				)
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "CSRF_TOKEN_MISMATCH",
				"message": "CSRF token 不匹配或缺失",
			})
			return
		}

		c.Next()
	}
}

// CSRFIssueCookie 在非 CSRF 路由上需要下发 cookie 时的 helper,例如登录成功后。
// 调用方通过 c.SetCookie 直接设置也可,这里提供一致封装便于测试。
func CSRFIssueCookie(c *gin.Context, secure bool) {
	issueCSRFCookieSecure(c, secure)
}

func issueCSRFCookie(c *gin.Context) {
	issueCSRFCookieSecure(c, false)
}

// issueCSRFCookieSecure 生成 32 字节 base64 token,设置为 _csrf cookie。
func issueCSRFCookieSecure(c *gin.Context, secure bool) {
	tok, err := generateCSRFToken()
	if err != nil {
		// crypto/rand 在现代 OS 不会失败;失败时发出占位 token 避免 panic。
		tok = "fallback-fixed-token"
	}
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(csrfCookieName, tok, 3600*8, "/", "", secure, false)
}

func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hasBearerToken 判断 Authorization 头是否为非空 Bearer。
func hasBearerToken(c *gin.Context) bool {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return false
	}
	// 仅识别 "Bearer xxx" 形式;其它 scheme 不豁免。
	const prefix = "Bearer "
	if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return false
	}
	return strings.TrimSpace(auth[len(prefix):]) != ""
}

// originMatches 精确匹配 Origin 是否在白名单。
func originMatches(origin string, allowed map[string]struct{}) bool {
	if origin == "" || len(allowed) == 0 {
		return false
	}
	_, ok := allowed[origin]
	return ok
}
