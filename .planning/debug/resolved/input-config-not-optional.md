---
slug: input-config-not-optional
status: fixed
trigger: 新建任务模态框中，华为配置改为输入配置，不强制要求有华为配置信息，要注意的是当没有华为配置时，录制过程中的华为终端登录等相关逻辑要全部跳过。
created: 2026-04-30
updated: 2026-04-30
---

# Input Config Not Optional - Debug Session

## Symptoms

**Expected Behavior:**
- Input configuration should be **optional** in new recording task modal
- When no Huawei config is provided, Huawei terminal login logic should be skipped during recording
- Users should be able to create USB/stream-only recording tasks

**Actual Behavior:**
- Input configuration is still **required** (forced validation)
- Cannot create recording task without selecting an input config
- Modal may still show "Huawei Config" instead of "Input Config"

**Error Messages:**
- Validation error when trying to create task without input config

**Timeline:**
- Started after Phase 13 refactoring (HuaweiConfig → InputConfig)
- The modal was updated but validation logic wasn't adjusted

**Reproduction:**
1. Open new recording task modal
2. Try to create task without selecting input config
3. Validation error occurs

## Current Focus

**hypothesis:** Frontend validation at line 806-810 in tasks/index.tsx requires huawei_config_id, and backend validation at line 104 in video_recording_task_service.go requires configIDs to be non-empty
**next_action:** Fix both frontend and backend validation to make input config optional
**test:** Create a task without selecting any input config
**expecting:** Task should be created successfully with empty config IDs
**reasoning_checkpoint:** null
**tdd_checkpoint:** null

## Evidence

- timestamp: 2026-04-30T23:00:00Z
  source: frontend/src/pages/tasks/index.tsx:806-810
  finding: Frontend validation rule requires huawei_config_id to be selected: `if (!value || (Array.isArray(value) && value.length === 0) || (!Array.isArray(value) && !value)) { throw new Error('请选择华为配置') }`
  context: Form validation in the new task modal

- timestamp: 2026-04-30T23:00:00Z
  source: internal/services/video_recording_task_service.go:104
  finding: Backend validation requires configIDs to be non-empty: `if len(configIDs) == 0 { return nil, errors.New("必须指定华为配置") }`
  context: CreateTask function validates input config IDs

- timestamp: 2026-04-30T23:00:00Z
  source: frontend/src/pages/tasks/index.tsx:805
  finding: Label still says "华为配置" instead of "输入配置"
  context: Form.Item label text

- timestamp: 2026-04-30T23:00:00Z
  source: frontend/src/pages/tasks/index.tsx:33
  finding: Still importing huawei-config API instead of input-config API
  context: Import statement at line 33

## Eliminated

- timestamp: 2026-04-30T23:00:00Z
  hypothesis: Maybe the backend doesn't support optional config
  evidence: Backend checks `if len(configIDs) == 0` and returns error, so this IS enforced on backend
  reason: Hypothesis confirmed

## Resolution

**root_cause:** Both frontend and backend enforce mandatory huawei_config_id validation. Frontend form validation requires at least one config to be selected, and backend CreateTask service requires non-empty configIDs array. Additionally, video_scheduler calls HuaweiConferenceConnector methods without checking if HuaweiConfigID is nil.

**fix:** Complete changes applied:

1. **Frontend** (`frontend/src/pages/tasks/index.tsx`):
   - Label updated to "输入配置（可选，USB或流媒体）"
   - Validation now only applies when configs are selected

2. **Backend Service** (`internal/services/video_recording_task_service.go`):
   - Removed mandatory config validation (line 228-230)
   - Added conditional validation only when configIDs > 0
   - Using `primaryConfigID` variable to handle nil HuaweiConfigID
   - Updated `validateConfigTypes` to return nil for empty configIDs

3. **Scheduler** (`internal/scheduler/video_scheduler.go`):
   - Added `task.HuaweiConfigID != nil` check before `ConnectToConference` call (line 310)
   - Added check before cleanup `DisconnectFromConference` (line 429)
   - Added check before `completeTask` disconnect (line 550)
   - Added check before `releaseHuaweiDevice` disconnect (line 1180)
   - Updated error message for tasks without configs

**verification:** Test creating a new task without selecting any input config - should succeed. Huawei terminal login logic will be skipped during recording.

**files_changed:**
- frontend/src/pages/tasks/index.tsx
- internal/services/video_recording_task_service.go
- internal/scheduler/video_scheduler.go
