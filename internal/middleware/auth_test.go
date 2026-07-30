package middleware

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetUserIDTypeSafety(t *testing.T) {
	cases := []struct {
		name string
		set  any
		want uint
		ok   bool
	}{
		{name: "missing", ok: false},
		{name: "wrong type", set: "1", ok: false},
		{name: "valid", set: uint(42), want: 42, ok: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(nil)
			if tc.set != nil {
				c.Set("user_id", tc.set)
			}
			got, ok := GetUserID(c)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("GetUserID() = (%d, %v), want (%d, %v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestContextHelpersRejectWrongTypes(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Set("username", uint(1))
	c.Set("role_id", "1")
	c.Set("role_ids", "1")
	c.Set("is_admin", "true")
	if GetUsername(c) != "" || GetRoleID(c) != 0 || GetRoleIDs(c) != nil || GetIsAdmin(c) {
		t.Fatal("context helpers must return safe defaults for wrong types")
	}
}

func TestAllowedTokenURLUsesExactPrefixes(t *testing.T) {
	prefixes := []string{"/api/v1/files/download/"}
	if !isAllowedTokenURL("/api/v1/files/download/abc", prefixes) {
		t.Fatal("configured download prefix should match")
	}
	for _, path := range []string{"/api/v1/users/list/download/x", "/evil/api/v1/files/download/abc", "/api/v1/files/downloadish/abc"} {
		if isAllowedTokenURL(path, prefixes) {
			t.Fatalf("unexpected token URL match: %s", path)
		}
	}
}
