package models

import "testing"

func TestAPIKeyMalformedIPWhitelistFailsClosed(t *testing.T) {
	key := &APIKey{IPWhitelist: "{"}
	if key.IsIPAllowed("127.0.0.1") {
		t.Fatal("malformed IP whitelist must fail closed")
	}
}

func TestAPIKeyEmptyIPWhitelistAllowsAll(t *testing.T) {
	key := &APIKey{IPWhitelist: "[]"}
	if !key.IsIPAllowed("127.0.0.1") {
		t.Fatal("valid empty IP whitelist should preserve unrestricted behavior")
	}
}
