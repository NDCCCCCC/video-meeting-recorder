package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareExactAllowlistAndDefaultDeny(t *testing.T) {
	for _, tc := range []struct {
		name    string
		allowed []string
		origin  string
		want    string
	}{
		{"default deny", nil, "https://evil.example", ""},
		{"exact match", []string{"https://app.example"}, "https://app.example", "https://app.example"},
		{"substring rejected", []string{"https://app.example"}, "https://app.example.evil", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(corsMiddleware(tc.allowed))
			r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tc.want {
				t.Fatalf("origin header = %q, want %q", got, tc.want)
			}
		})
	}
}
