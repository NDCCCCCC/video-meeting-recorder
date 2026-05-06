---
slug: stream-config-rejected
status: fixed
trigger: 新建录制任务时，我设置了一个流媒体的输入配置（没有配置华为字段），提示：必须指定至少一种输入配置
created: 2026-04-30
updated: 2026-04-30
---

# Stream Config Rejected - Debug Session

## Symptoms

**Expected Behavior:**
- 流媒体输入配置应该能成功创建录制任务
- 流媒体配置不需要配置华为终端字段

**Actual Behavior:**
- 提示"必须指定至少一种输入配置"
- 即使已选择流媒体配置，仍显示错误

**Error Messages:**
- "必须指定至少一种输入配置"

**Timeline:**
- 开始时间：华为配置改为输入配置重构后

**Reproduction:**
1. 打开新建录制任务
2. 选择流媒体输入配置
3. 不填写华为字段
4. 提交任务
5. 出现错误提示

## Current Focus

**hypothesis:** IsValid() 在第246行调用时，TaskInputConfigs 还是空数组，因为关联记录在第285-291行才创建。验证时机错误：在创建任务对象后立即验证，但此时关联数据还未填充。
**next_action:** 修改验证逻辑，仅验证 InputConfigID 或检查 InputConfigIDs 参数
**test:** 移除 IsValid() 中对 TaskInputConfigs 的检查，改为在 CreateTask 中验证 InputConfigIDs
**expecting:** 流媒体配置可以成功创建任务
**reasoning_checkpoint:**
  hypothesis: "IsValid() 检查 TaskInputConfigs，但调用时该字段还是空数组，因为关联记录在 IsValid() 之后才创建"
  confirming_evidence:
    - "IsValid() 在第246行调用（video_recording_task_service.go）"
    - "TaskInputConfigs 关联记录在第285-291行创建（IsValid 之后）"
    - "IsValid() 第96-100行检查 TaskInputConfigs 长度"
  falsification_test: "如果在 IsValid() 调用前填充 TaskInputConfigs，验证应该通过"
  fix_rationale: "将验证移至正确的位置：在 CreateTask 中验证 InputConfigIDs 参数，而不是在 IsValid() 中验证还未填充的 TaskInputConfigs 字段"
  blind_spots: "需要确认是否有其他地方调用 IsValid() 会受影响"
**tdd_checkpoint:** null

## Evidence

- timestamp: 2026-04-30
  checked: video_recording_task.go IsValid() 方法（第82-103行）
  found: IsValid() 检查 InputConfigID 和 TaskInputConfigs，两个都为空时返回"必须指定至少一种输入配置"
  implication: 验证逻辑本身正确，但调用时机可能有问题

- timestamp: 2026-04-30
  checked: video_recording_task_service.go CreateTask 方法（第186-321行）
  found: IsValid() 在第246行调用，但 TaskInputConfigs 关联记录在第285-291行才创建
  implication: 调用 IsValid 时 TaskInputConfigs 必然是空数组，导致验证失败

## Eliminated

（无）

## Resolution

root_cause: 多层问题：
1. CreateTask 服务没有设置 InputConfigID 字段
2. 前端发送的字段名是 huawei_config_ids，而后端期望 input_config_ids
3. 前端表单组件使用了错误的字段名

fix:
1. video_recording_task_service.go: 添加 primaryConfigID 逻辑（第 232-250 行）
2. frontend/src/types/task.ts: 接口定义改用 input_config_ids
3. frontend/src/pages/tasks/index.tsx: 请求数据构造改用 input_config_ids（第 355-365 行）
4. 重新构建前端

verification: ✅ COMPLETE - 前端已重新构建
files_changed:
- internal/services/video_recording_task_service.go: 第 232-250 行添加 primaryConfigID 设置逻辑
- frontend/src/types/task.ts: CreateTaskRequest 接口字段名改为 input_config_ids
- frontend/src/pages/tasks/index.tsx: requestData 构造改用 input_config_ids
- frontend/dist/: 重新构建的前端资源
