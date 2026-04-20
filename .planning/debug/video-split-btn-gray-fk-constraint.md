---
status: resolved
trigger: "视频分割页面两个Bug：1. 设置好标记后确认分割按钮灰色无法执行；2. 快照生成FK约束报错"
created: "2026-04-20"
updated: "2026-04-20"
---

# Debug Session: video-split-btn-gray-fk-constraint

## Symptoms

### Bug 1: 确认分割按钮灰色
- **Expected:** 设置分割标记后，确认分割按钮可点击，执行视频分割
- **Actual:** 设置了 1 个标记后，确认分割按钮是灰色（disabled），无法执行
- **Markers set:** 1 个标记点
- **Error messages:** 无报错，按钮就是灰色不可交互
- **Timeline:** 从未成功过，首次使用就遇到
- **Reproduction:** 打开视频分割页面 → 设置分割标记 → 确认分割按钮灰色

### Bug 2: 快照生成FK约束报错
- **Expected:** 快照生成成功
- **Actual:** 生成快照失败: 注册快照文件失败: 创建文件记录失败: 创建文件记录失败: constraint failed: FOREIGN KEY constraint failed (787)
- **Error messages:** constraint failed: FOREIGN KEY constraint failed (787)
- **Timeline:** 从未成功过
- **Reproduction:** 在非视频分割页面场景下触发快照生成
- **Note:** 错误信息出现两次"创建文件记录失败"，暗示可能有多层调用栈或重试

## Current Focus

- **hypothesis:** Bug 1 可能需要 >=2 个标记才能启用按钮（业务逻辑），也可能是按钮状态计算逻辑有误。Bug 2 的 FK 约束错误表明快照文件记录引用了不存在的父记录（可能是 video_file_id 或 recording_id）。
- **next_action:** gather initial evidence — 读取前端分割按钮的 disabled 逻辑和后端快照文件注册代码
- **test:** (not yet defined)
- **expecting:** (not yet defined)

## Evidence

### Bug 1 Evidence
**File:** `frontend/src/pages/split/index.tsx`

Line 448 - Button disabled condition:
```tsx
disabled={markers.length < 2 || splitting}
```

Line 191-194 - Validation logic:
```tsx
if (!id || markers.length < 2) {
  message.warning('请至少添加 2 个标记点')
  return
}
```

Lines 87-120 - Segment preview calculation (correctly handles 1 marker):
```tsx
const segmentPreviews: SegmentPreview[] = useMemo(() => {
  if (markers.length === 0) return []

  const sortedMarkers = [...markers].sort((a, b) => a - b)
  const segments: SegmentPreview[] = []

  // 第一个段落：0 -> markers[0]
  segments.push({
    index: 1,
    start: 0,
    end: sortedMarkers[0],
    duration: sortedMarkers[0],
  })

  // 中间段落：markers[i] -> markers[i+1]
  for (let i = 0; i < sortedMarkers.length - 1; i++) {
    segments.push({
      index: i + 2,
      start: sortedMarkers[i],
      end: sortedMarkers[i + 1],
      duration: sortedMarkers[i + 1] - sortedMarkers[i],
    })
  }

  // 最后一个段落：markers[last] -> duration
  segments.push({
    index: sortedMarkers.length + 1,
    start: sortedMarkers[sortedMarkers.length - 1],
    end: duration,
    duration: duration - sortedMarkers[sortedMarkers.length - 1],
  })

  return segments
}, [markers, duration])
```

**Analysis:** The segment preview logic correctly creates 2 segments from 1 marker (0→marker, marker→end). However, the button disabled condition requires `markers.length >= 2`, which is overly restrictive.

### Bug 2 Evidence
**File:** `internal/services/snapshot_service.go`

Lines 123-135 - Parent file lookup and CreateSegmentFile call:
```go
// 8. Find the parent VideoFile for this task
var parentFile models.VideoFile
parentID := uint(0)
if err := s.db.Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeRecording).First(&parentFile).Error; err == nil {
    parentID = parentFile.ID
}

// 9. Register snapshot file via VideoFileService callback (D-10, D-13)
// Pass seekOffset as SnapshotOffset so it's stored on the VideoFile record
snapshotFile, err := s.videoFileService.CreateSegmentFile(outputMP4, &parentID, models.SourceTypeSnapshot, createdBy, seekOffset)
if err != nil {
    return nil, fmt.Errorf("注册快照文件失败: %w", err)
}
```

**File:** `internal/models/video_file.go`

Lines 22-25 - ParentID foreign key definition:
```go
ParentID       *uint               `gorm:"index" json:"parent_id,omitempty"`
Parent         *VideoFile          `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
```

**Analysis:** When no parent recording file exists for the task:
1. `parentID` remains `uint(0)`
2. `&parentID` is passed to `CreateSegmentFile`
3. GORM interprets this as `parent_id = 0` (not NULL)
4. SQLite FK constraint rejects because no `video_file.id = 0` exists

**Root cause:** Using `parentID := uint(0)` and passing `&parentID` creates a non-null pointer to zero, which GORM translates to `parent_id = 0` instead of `parent_id = NULL`. The foreign key constraint requires either NULL or a valid existing ID.

## Eliminated

### Bug 1 Eliminated Hypotheses
- ❌ **Hypothesis:** Button needs 2+ markers for valid split (business logic)
  - **Evidence:** Segment preview logic correctly generates 2 segments from 1 marker
  - **Conclusion:** The 2-marker requirement was a bug, not intended behavior

### Bug 2 Eliminated Hypotheses
- ❌ **Hypothesis:** Recording task doesn't exist
  - **Evidence:** Code checks task existence earlier (lines 44-47) before parent file lookup
  - **Conclusion:** Task exists, but parent VideoFile may not

## Resolution

### Bug 1: Split Button Disabled - FIXED
**Root Cause:** Button disabled condition `markers.length < 2` was overly restrictive. The segment calculation logic already correctly handles 1 marker creating 2 segments.

**Fix Applied:** Changed validation from `markers.length < 2` to `markers.length < 1` in both:
1. Button disabled condition (line 448)
2. handleSplit validation (line 191)

**Files Modified:**
- `frontend/src/pages/split/index.tsx`

### Bug 2: Snapshot FK Constraint - FIXED
**Root Cause:** When no parent recording file exists, code passed `&parentID` where `parentID = 0`, causing GORM to insert `parent_id = 0` (non-null) instead of `parent_id = NULL`, violating the foreign key constraint.

**Fix Applied:** Changed `parentID` from `uint(0)` with forced pointer to `*uint` (nilable):
```go
var parentFile models.VideoFile
var parentID *uint // Use pointer to allow nil when no parent exists
if err := s.db.Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeRecording).First(&parentFile).Error; err == nil {
    parentID = &parentFile.ID
} else {
    s.logger.Warn("快照未找到父录制文件，将创建无父级的快照记录",
        zap.Uint("task_id", taskID),
        zap.Error(err),
    )
}
```

**Files Modified:**
- `internal/services/snapshot_service.go`

**Result:** Snapshots now correctly create with `parent_id = NULL` when no parent recording file exists, avoiding the FK constraint error.
