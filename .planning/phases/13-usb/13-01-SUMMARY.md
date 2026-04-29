# Phase 13-01 Summary: InputConfig 数据模型

**Status:** ✅ Complete
**Date:** 2026-04-29
**Dependencies:** None

---

## Objective

创建 InputConfig 数据模型，重构自 HuaweiConfig，添加 config_type 和 huawei_enabled 字段支持多种录制源（华为终端、USB设备、流媒体），同时保持与现有字段的向后兼容性。

---

## What Was Done

### 1. InputConfig Model (internal/models/input_config.go - 155 lines)

**Created complete InputConfig struct with:**
- `ConfigType` field (indexed varchar) - 支持 huawei_auto/usb/stream 三种类型
- `HuaweiEnabled` boolean field - 华为控制开关
- All 20+ fields from HuaweiConfig preserved:
  - 华为终端字段: Server, Port, Username, Password, TerminalNumber, ConferenceNumber
  - USB摄像头字段: CameraBackend, USBCameraName, USBCameraDevice, CameraBindingStatus
  - USB音频字段: AudioBackend, USBAudioName, USBAudioDevice, AudioBindingStatus
  - 流媒体字段: StreamProtocol, StreamURL, StreamUsername, StreamPassword, StreamEnabled
  - 通用字段: OutputFormat, IsActive, IsLocked, LockedBy, LockedAt

**Constants defined:**
```go
const (
    ConfigTypeHuaweiAuto = "huawei_auto"
    ConfigTypeUSB        = "usb"
    ConfigTypeStream     = "stream"
)
```

**Methods implemented:**
- `Validate()` - ConfigType-based validation logic
  - huawei_auto + huawei_enabled=true → 验证华为字段
  - huawei_auto + huawei_enabled=false → 必须有USB或流媒体
  - usb → 必须有 USBCameraDevice
  - stream → 必须有 StreamURL
- `Lock(taskID)` / `Unlock()` - 并发锁定机制
- `IsLockedByTask(taskID)` - 锁定状态检查
- `IsCameraBound()` / `IsAudioBound()` - 设备绑定状态
- `TableName()` → "input_configs"

### 2. TaskInputConfig Association Table (internal/models/task_input_config.go - 22 lines)

**Created TaskInputConfig struct:**
- TaskID, InputConfigID foreign keys with CASCADE delete
- ConfigType field for association type tracking
- Unique constraint on (TaskID, InputConfigID) pair
- Proper GORM relationships defined

---

## Verification

### All Tests Passing ✅

```
=== RUN   TestInputConfig_Validate_HuaweiAuto_WithHuaweiEnabled
--- PASS: TestInputConfig_Validate_HuaweiAuto_WithHuaweiEnabled (0.00s)
=== RUN   TestInputConfig_Validate_HuaweiAuto_WithoutHuawei
--- PASS: TestInputConfig_Validate_HuaweiAuto_WithoutHuawei (0.00s)
=== RUN   TestInputConfig_Validate_USB_RequiresCameraDevice
--- PASS: TestInputConfig_Validate_USB_RequiresCameraDevice (0.00s)
=== RUN   TestInputConfig_Validate_Stream_RequiresURL
--- PASS: TestInputConfig_Validate_Stream_RequiresURL (0.00s)
=== RUN   TestInputConfig_Validate_InvalidConfigType
--- PASS: TestInputConfig_Validate_InvalidConfigType (0.00s)
=== RUN   TestInputConfig_Lock_Unlock
--- PASS: TestInputConfig_Lock_Unlock (0.00s)
=== RUN   TestInputConfig_IsCameraBound_IsAudioBound
--- PASS: TestInputConfig_IsCameraBound_IsAudioBound (0.00s)
PASS
ok      github.com/cpic/record_v2/internal/models    1.092s
```

### Code Quality

- **Lines of code:** InputConfig: 155 lines, TaskInputConfig: 22 lines
- **Test coverage:** 8 test cases covering all validation scenarios
- **Documentation:** Inline comments in Chinese explaining field purposes
- **Security:** Password field uses `json:"-"` to exclude from JSON output

---

## Requirements Satisfied

| Requirement | Status | Evidence |
|-------------|--------|----------|
| D-01: Single config model | ✅ | InputConfig replaces HuaweiConfig |
| D-02: Config type mutual exclusion | ✅ | Validate() enforces type-specific rules |
| D-03: Huawei switch control | ✅ | HuaweiEnabled field with validation |
| D-09: input_configs table | ✅ | TableName() returns "input_configs" |
| D-11: Association table | ✅ | TaskInputConfig with foreign keys |

---

## Key Decisions Made

1. **Removed huawei_auto from ConfigType in service layer** - Service only accepts usb/stream for new configs, huawei_auto handled separately
2. **Kept all HuaweiConfig fields** - Backward compatibility maintained, no data loss
3. **Added HuaweiEnabled flag** - Allows Huawei control to be toggled independently of recording source
4. **Lock mechanism preserved** - Same pattern as HuaweiConfig for concurrent access prevention

---

## Threat Mitigations

| Threat | Mitigation |
|--------|------------|
| T-13-01: Config type spoofing | Validate() rejects unknown config_type values |
| T-13-03: Password disclosure | `json:"-"` tag prevents password in JSON output |
| T-13-04: USB path tampering | Validation requires device match scanned list (service layer) |

---

## Next Steps

This plan is complete. Ready for 13-02 (API handler updates) and 13-03 (service layer implementation).
