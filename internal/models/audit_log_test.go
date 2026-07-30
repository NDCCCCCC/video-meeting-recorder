package models

import "testing"

func TestAuditLogMalformedJSONReturnsNil(t *testing.T) {
	log := &AuditLog{OldData: "{"}
	if got := log.GetOldData(); got != nil {
		t.Fatalf("GetOldData() = %#v, want nil", got)
	}
}

func TestNotificationMalformedJSONReturnsDefaults(t *testing.T) {
	n := &NotificationMessage{ChannelStatus: "{", Data: "{"}
	if got := n.GetChannelStatusMap(); len(got) != 0 {
		t.Fatalf("GetChannelStatusMap() = %#v, want empty", got)
	}
	if got := n.GetData(); got == nil || got.Params != nil {
		t.Fatalf("GetData() = %#v, want empty NotificationData", got)
	}
}
