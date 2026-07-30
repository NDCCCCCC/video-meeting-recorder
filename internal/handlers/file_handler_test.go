package handlers

import "testing"

func TestMaskAccessToken(t *testing.T) {
	if got := maskAccessToken("abcdef1234567890"); got != "***7890" {
		t.Fatalf("maskAccessToken() = %q", got)
	}
	if got := maskAccessToken("abc"); got != "***" {
		t.Fatalf("short token must be fully masked, got %q", got)
	}
}
