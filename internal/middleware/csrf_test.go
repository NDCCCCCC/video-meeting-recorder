package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSRFMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name             string
		method           string
		setCookie        bool
		cookieValue      string
		headerValue      string
		bearer           string
		origin           string
		safeOrigins      []string
		wantStatus       int
		wantCookieSet    bool
		wantSubstringSub string // body should contain this (or "" to skip)
	}{
		{
			name:          "GET 不需要 token 且首次访问会下发 cookie",
			method:        http.MethodGet,
			wantStatus:    http.StatusOK,
			wantCookieSet: true,
		},
		{
			name:        "GET 当 cookie 已存在时不重复下发",
			method:      http.MethodGet,
			setCookie:   true,
			cookieValue: "existing-token",
			wantStatus:  http.StatusOK,
		},
		{
			name:          "OPTIONS 不需要 token 且首次访问会下发 cookie",
			method:        http.MethodOptions,
			wantStatus:    http.StatusOK,
			wantCookieSet: true,
		},
		{
			name:       "POST 无 cookie 且无 Bearer 应被拒绝且不下发 cookie",
			method:     http.MethodPost,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "POST 带 Bearer 头免 CSRF 验证",
			method:     http.MethodPost,
			bearer:     "sm4-xxx",
			wantStatus: http.StatusOK,
		},
		{
			name:        "POST cookie 与 header 匹配 -> 通过",
			method:      http.MethodPost,
			setCookie:   true,
			cookieValue: "matching-token",
			headerValue: "matching-token",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "POST cookie 与 header 不匹配 -> 403",
			method:      http.MethodPost,
			setCookie:   true,
			cookieValue: "cookie-token",
			headerValue: "different-header",
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "POST 仅 header 无 cookie -> 403",
			method:      http.MethodPost,
			headerValue: "some-token",
			wantStatus:  http.StatusForbidden,
		},
		{
			name:             "POST Origin 不在白名单 -> 403",
			method:           http.MethodPost,
			bearer:           "ignored-bearer",
			origin:           "https://evil.example",
			safeOrigins:      []string{"https://app.example"},
			wantStatus:       http.StatusForbidden,
			wantSubstringSub: "CSRF_ORIGIN_REJECTED",
		},
		{
			name:        "POST Origin 在白名单 -> 通过 (Bearer 兜底)",
			method:      http.MethodPost,
			bearer:      "any-bearer",
			origin:      "https://app.example",
			safeOrigins: []string{"https://app.example"},
			wantStatus:  http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(CSRF(tc.safeOrigins, nil, false))
			r.Handle(tc.method, "/x", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(tc.method, "/x", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tc.bearer)
			}
			if tc.headerValue != "" {
				req.Header.Set(csrfHeaderName, tc.headerValue)
			}
			if tc.setCookie {
				req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: tc.cookieValue})
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
			gotCookie := false
			for _, c := range w.Result().Cookies() {
				if c.Name == csrfCookieName && c.Value != "" {
					gotCookie = true
					break
				}
			}
			if gotCookie != tc.wantCookieSet {
				t.Fatalf("cookie issued = %v, want %v", gotCookie, tc.wantCookieSet)
			}
			if tc.wantSubstringSub != "" && !strings.Contains(w.Body.String(), tc.wantSubstringSub) {
				t.Fatalf("body %q must contain %q", w.Body.String(), tc.wantSubstringSub)
			}
		})
	}
}

func TestIssueCSRFTokenRandomness(t *testing.T) {
	a, err := generateCSRFToken()
	if err != nil || a == "" {
		t.Fatalf("generateCSRFToken() failed: err=%v tok=%q", err, a)
	}
	b, err := generateCSRFToken()
	if err != nil || b == "" || a == b {
		t.Fatalf("tokens must be random & non-empty: a=%q b=%q", a, b)
	}
}

func TestOriginMatchesRejectsEmpty(t *testing.T) {
	if originMatches("", map[string]struct{}{"https://app.example": {}}) {
		t.Fatal("empty origin should not match")
	}
	if originMatches("https://evil.example", nil) {
		t.Fatal("nil allowed map must not match")
	}
	if !originMatches("https://app.example", map[string]struct{}{"https://app.example": {}}) {
		t.Fatal("exact match should pass")
	}
}
