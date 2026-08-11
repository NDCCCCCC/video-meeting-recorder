---
phase: quick
plan: 01
type: execute
status: complete
date: 2026-08-11
---

# Quick Task 260811-cxl - 时间选择器分钟约束 实施总结

## 目标

创建/编辑录制任务时，开始/结束时间选择器：
- 分钟仅 00 / 15 / 30 / 45 四个选项
- 秒不需要选，默认 00

## 实施内容

### Task 1: 修改 TaskFormModal 时间选择器配置 ✅

**修改文件:** `frontend/src/pages/tasks/components/TaskFormModal.tsx`

**关键改动:**

```tsx
// 时间选择器配置：分钟仅 00/15/30/45 可选，秒列隐藏并默认 00
const TASK_TIME_PICKER = {
  format: 'HH:mm',
  minuteStep: 15,
  defaultValue: dayjs('00:00:00', 'HH:mm:ss'),
} as const
```

并应用到两个 DatePicker（开始时间、结束时间）：
```tsx
<DatePicker
  showTime={TASK_TIME_PICKER}
  format="YYYY-MM-DD HH:mm:ss"
  ...
/>
```

**实现要点:**
- `minuteStep: 15` → 分钟列只显示 00 / 15 / 30 / 45
- `showTime.format: 'HH:mm'` → 隐藏秒列（picker UI 不再展示秒）
- `defaultValue: dayjs('00:00:00', 'HH:mm:ss')` → 秒默认 00，picker 交互不暴露秒
- 整体 `format="YYYY-MM-DD HH:mm:ss"` 保留，确保 `values.start_time.toISOString()` 提交的 RFC3339 字符串中秒为 00

## 验证结果

- ✅ `npx tsc -b` TypeScript 编译通过
- ✅ `npx eslint` 无错误
- ✅ `npx prettier --check` 通过
- ✅ git diff 仅 1 个文件、10 增 / 2 删（精确隔离，无无关改动）
- ✅ 提交 `c4341a0`

## 影响范围

- 仅前端 UI 行为变更
- 后端 API 不变（仍接受 RFC3339）
- 已有任务数据兼容（编辑时秒位保留，不强行归零以免静默改写历史数据）