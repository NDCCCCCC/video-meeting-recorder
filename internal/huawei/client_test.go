package huawei

import (
	"strings"
	"testing"
)

func TestHuaweiSanitizeResponseBody(t *testing.T) {
	got := string(huaweiSanitizeResponseBody([]byte(`{"username":"admin","password":"secret123","nested":{"certBase64String":"certificate"},"safe":"ok"}`)))
	for _, secret := range []string{"admin", "secret123", "certificate"} {
		if strings.Contains(got, secret) {
			t.Fatalf("sanitized response leaks %q: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"safe":"ok"`) {
		t.Fatalf("safe field missing: %s", got)
	}
}
