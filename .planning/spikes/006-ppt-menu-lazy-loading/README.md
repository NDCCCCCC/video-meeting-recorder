# Spike 006: PPT 预览菜单最佳实践研究

**状态:** completed
**创建时间:** 2026-04-30
**类型:** 方案对比研究

---

## 背景

文件管理页面每 5 秒自动刷新文件列表。之前的批量 PPT 检查功能依赖 `files` 状态：
- 每次 `files` 更新 → 触发批量 PPT 检查
- 每 5 秒调用一次批量 API
- 导致频繁的 token 刷新，引发自动登出

当前状态：批量 PPT 检查已移除，用户无法从文件管理页面访问 PPT 预览。

---

## 方案对比

### 方案 A: 懒加载（点击菜单时检查）

**原理:** 只在用户点击"更多"菜单时才检查 PPT 状态

**优点:**
- ✅ 实现简单
- ✅ 不依赖 `files` 状态
- ✅ 按需检查，节省资源

**缺点:**
- ⚠️ 每次打开菜单都会检查（即使不点击PPT项）
- ⚠️ 首次显示菜单可能有延迟

**实现复杂度:** 低

---

### 方案 B: 后端缓存 + 嵌入响应（推荐）

**原理:** 在文件列表 API 响应中直接包含 PPT 状态

**具体实现:**
1. 修改 `VideoFile` 结构体，添加 `has_ppt` 字段
2. 在获取文件列表时，使用 SQL JOIN 或批量查询获取 PPT 状态
3. 前端直接从 `files` 数据中读取，无需额外请求

**优点:**
- ✅ **零额外请求** - PPT 状态随文件列表一起返回
- ✅ 用户体验最佳 - 无需等待
- ✅ 代码更简洁 - 前端不需要额外的 API 调用
- ✅ 后端可优化 - 使用 SQL JOIN 一次查询完成

**缺点:**
- ⚠️ 文件列表 API 响应稍大（但可忽略）

**实现复杂度:** 中等

---

### 方案 C: WebSocket 推送

**原理:** 后端在 PPT 生成完成时主动推送给前端

**优点:**
- ✅ 实时性最高

**缺点:**
- ❌ 复杂度高 - 需要 WebSocket 连接管理
- ❌ 资源消耗 - 维持长连接
- ❌ 过度设计 - 对于这个场景不必要

**实现复杂度:** 高

---

## 推荐方案

### 🏆 方案 B: 后端嵌入 PPT 状态

**理由:**
1. **最符合 RESTful 设计原则** - 相关资源一起返回
2. **零额外网络请求** - 最优性能
3. **用户体验最佳** - 无延迟，无需额外等待
4. **实现难度适中** - 后端修改简单，前端改动最小

---

## 实施步骤

### 后端修改

**1. 修改 VideoFile 模型** (`internal/models/video_file.go`)

```go
type VideoFile struct {
    // ... 现有字段
    
    // PPT 状态（不存储到数据库，仅用于 API 响应）
    HasPpt bool `gorm:"-" json:"has_ppt,omitempty"`
}
```

**2. 修改文件列表查询** (`internal/handlers/file_handler.go`)

```go
// 获取文件列表后，批量查询 PPT 状态
videoIDs := extractVideoIDs(files)
pptStatus := batchCheckPptStatus(videoIDs) // 使用 EXISTS 查询

// 填充 HasPpt 字段
for i := range files {
    files[i].HasPpt = pptStatus[files[i].ID]
}
```

**3. 优化 SQL 查询**

```sql
-- 使用 LEFT JOIN 和 EXISTS 一次性查询
SELECT v.*, 
       EXISTS(SELECT 1 FROM ppt_files p WHERE p.source_video_file_id = v.id) as has_ppt
FROM video_files v
WHERE ...;
```

### 前端修改

**无需额外 API 调用**，直接使用 `file.has_ppt`：

```tsx
// 预览PPT
if (record.has_ppt) {
  moreMenuItems.push({
    key: 'preview-ppt',
    icon: <FilePptOutlined />,
    label: '预览PPT',
    onClick: () => navigate(`/results/${record.id}`),
  })
}
```

---

## 验收标准

- [x] 方案对比完成
- [x] 推荐方案确定
- [ ] 后端修改完成
- [ ] 前端修改完成
- [ ] 功能测试通过
- [ ] 无自动登出问题

---

## 下一步

使用 `/gsd-quick` 实施方案 B（后端嵌入 PPT 状态）
