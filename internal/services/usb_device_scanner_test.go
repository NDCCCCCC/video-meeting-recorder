package services

import (
	"encoding/json"
	"testing"
)

// TestPowerShellVideoDevice_JSONUnmarshal 验证 PERF-009 修复：
// 强类型 PowerShellVideoDevice 能正确反序列化 PowerShell JSON 输出。
func TestPowerShellVideoDevice_JSONUnmarshal(t *testing.T) {
	raw := `[{"FriendlyName":"Integrated Camera","InstanceId":"USB\\VID_1234&PID_5678\\00"},{"FriendlyName":"USB Microphone","InstanceId":"USB\\VID_ABCD&PID_EF00\\01"}]`
	var devices []PowerShellVideoDevice
	if err := json.Unmarshal([]byte(raw), &devices); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("len(devices) = %d, want 2", len(devices))
	}
	if devices[0].FriendlyName != "Integrated Camera" {
		t.Errorf("devices[0].FriendlyName = %q", devices[0].FriendlyName)
	}
	if devices[0].InstanceId != "USB\\VID_1234&PID_5678\\00" {
		t.Errorf("devices[0].InstanceId = %q", devices[0].InstanceId)
	}
}

// TestPowerShellAudioDevice_JSONUnmarshal 验证强类型音频设备结构。
func TestPowerShellAudioDevice_JSONUnmarshal(t *testing.T) {
	raw := `[{"Name":"Microphone (Realtek)","ID":"{0.0.0.00000000}.{00000000-0000-0000-0000-000000000000}"}]`
	var devices []PowerShellAudioDevice
	if err := json.Unmarshal([]byte(raw), &devices); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(devices))
	}
	if devices[0].Name != "Microphone (Realtek)" {
		t.Errorf("Name = %q", devices[0].Name)
	}
}

// TestParseWindowsDevice_TypedStruct 验证 parseWindowsDevice 接收强类型
// 后的行为：FriendlyName 为空应返回 false。
func TestParseWindowsDevice_TypedStruct(t *testing.T) {
	s := NewUSBDeviceScanner(nil)

	// 正常 case
	dev, ok := s.parseWindowsDevice(PowerShellVideoDevice{
		FriendlyName: "Test Camera",
		InstanceId:   "USB\\VID_9999&PID_0000\\00",
	})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if dev.Name != "Test Camera" {
		t.Errorf("Name = %q", dev.Name)
	}
	if dev.Type != "camera" {
		t.Errorf("Type = %q", dev.Type)
	}
	if dev.Backend != "dshow" {
		t.Errorf("Backend = %q", dev.Backend)
	}

	// FriendlyName 空 → 返回 ok=false
	if _, ok := s.parseWindowsDevice(PowerShellVideoDevice{FriendlyName: ""}); ok {
		t.Fatal("expected ok=false when FriendlyName empty")
	}
}

// TestParseWindowsAudioDevice_TypedStruct 验证 parseWindowsAudioDevice 强类型路径。
func TestParseWindowsAudioDevice_TypedStruct(t *testing.T) {
	s := NewUSBDeviceScanner(nil)

	dev, ok := s.parseWindowsAudioDevice(PowerShellAudioDevice{
		Name: "Mic (USB)",
		ID:   "USB\\VID_1111",
	})
	if !ok {
		t.Fatal("expected ok=true")
	}
	if dev.Type != "audio" {
		t.Errorf("Type = %q", dev.Type)
	}
	if dev.Backend != "wasapi" {
		t.Errorf("Backend = %q", dev.Backend)
	}
	if dev.Name != "Mic (USB)" {
		t.Errorf("Name = %q", dev.Name)
	}
}
