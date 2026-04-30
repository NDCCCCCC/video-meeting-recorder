---
plan: 14-03
phase: 14-batch-operations
status: pending-verification
completed_at: 2026-04-30T14:50:00+08:00
---

# Plan 14-03: 前端批量操作UI - 执行摘要

## 实施概述

在文件管理页面添加了批量操作UI，包括批量下载按钮和批量转录按钮及配置对话框。

## 完成的任务

### Task 1: 添加批量下载UI

**实现内容：**
- 添加状态变量 `batchDownloading` 用于跟踪下载状态
- 实现 `handleBatchDownload` 函数：
  - 计算选中文件的总大小
  - 当文件总大小 > 1GB 或文件数量 > 100 时显示警告对话框
  - 调用 `videoFileApi.batchDownloadFiles()` 触发下载
  - 下载完成后清空选择
- 在批量操作区域添加「批量下载 (N)」按钮

**关键代码变更：**
```typescript
// frontend/src/pages/files/index.tsx
const [batchDownloading, setBatchDownloading] = useState(false)

const handleBatchDownload = useCallback(async () => {
  // 计算文件总大小
  const selectedFiles = files.filter(f => selectedRowKeys.includes(f.id))
  const totalSize = selectedFiles.reduce((sum, f) => sum + f.file_size, 0)
  const totalSizeGB = totalSize / (1024 * 1024 * 1024)

  // 显示警告（如需要）
  if (totalSizeGB > 1 || selectedRowKeys.length > 100) {
    Modal.confirm({...})
  }

  // 执行下载
  videoFileApi.batchDownloadFiles(selectedRowKeys as number[])
}, [selectedRowKeys, files])
```

### Task 2: 添加批量转录UI（按钮和对话框）

**实现内容：**
- 添加状态变量：
  - `batchTranscribeModalOpen` - 控制对话框显示
  - `batchTranscribing` - 跟踪转录状态
  - `batchSamplingRate` - 采样率设置
  - `batchTranscribeMode` - 转录模式（local/cloud）
- 实现 `handleBatchTranscribeClick` 函数：
  - 检查文件类型（只允许视频文件）
  - 过滤非视频文件并显示警告
  - 打开配置对话框
- 实现 `confirmBatchTranscription` 函数：
  - 构造 `BatchTranscriptionRequest`
  - 调用 `submitBatchTranscription` API
  - 显示成功/失败统计
  - 刷新活跃转录任务列表
- 在批量操作区域添加「批量转录 (N)」按钮
- 添加批量转录配置对话框：
  - 显示选中文件数量
  - 转录模式选择（本地/云端）
  - 采样率选择（仅本地模式）
  - 云端模式说明文字

**关键代码变更：**
```typescript
// frontend/src/pages/files/index.tsx
const [batchTranscribeModalOpen, setBatchTranscribeModalOpen] = useState(false)
const [batchTranscribing, setBatchTranscribing] = useState(false)
const [batchSamplingRate, setBatchSamplingRate] = useState<number>(0.5)
const [batchTranscribeMode, setBatchTranscribeMode] = useState<TranscriptionMode>('local')

const confirmBatchTranscription = useCallback(async () => {
  const request: BatchTranscriptionRequest = {
    video_file_ids: selectedRowKeys as number[],
    sampling_rate: batchTranscribeMode === 'local' ? batchSamplingRate : undefined,
    mode: batchTranscribeMode,
  }
  const response = await submitBatchTranscription(request)
  // 显示成功/失败统计
}, [selectedRowKeys, batchSamplingRate, batchTranscribeMode])
```

## 修改的文件

| 文件 | 变更类型 | 描述 |
|------|----------|------|
| `frontend/src/pages/files/index.tsx` | 修改 | 添加批量操作UI（下载和转录），修复按钮布局 |

## 实施细节补充

**布局修复：**
- 批量操作按钮（批量删除、批量下载、批量转录）现在与其他按钮（搜索、筛选、刷新、扫描导入、上传视频）在同一行显示
- 移除了嵌套的 `<Space>` 组件和 `marginBottom: 16` 样式
- 所有按钮统一使用外层的 `<Space size="middle" wrap>` 容器

## 依赖关系

- 依赖 14-01: 批量下载API (`batchDownloadFiles`)
- 依赖 14-02: 批量转录API (`submitBatchTranscription`, `BatchTranscriptionRequest`)

## 验证清单

### 自动化验证
- [x] `grep -n "批量下载" frontend/src/pages/files/index.tsx` ✓
- [x] `grep -n "handleBatchDownload" frontend/src/pages/files/index.tsx` ✓
- [x] `grep -n "批量转录" frontend/src/pages/files/index.tsx` ✓
- [x] `grep -n "handleBatchTranscribe" frontend/src/pages/files/index.tsx` ✓
- [x] 前端编译通过 ✓

### 人工验证（待执行）
- [ ] 批量操作按钮正确显示
- [ ] 批量下载功能正常
- [ ] 批量转录功能正常
- [ ] 配置对话框交互流畅
- [ ] 用户反馈清晰明确
- [ ] 边界情况处理正确

## 已知问题

无

## 偏差说明

无偏差，按照计划实施。

## 下一步

人工验证批量操作UI功能：
1. 启动前端开发服务器
2. 访问文件管理页面
3. 测试批量下载和批量转录功能
4. 验证边界情况处理
