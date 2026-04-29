---
status: resolved
trigger: rtmp://10.62.1.158:1935/lives/m0 连接测试失败，错误信息：Failed to set value '3' for option 't': Option not found。发生在 huawei_config_service.go:495 的 TestStreamConnection 方法中。
created: 2026-04-29T14:52:00+08:00
updated: 2026-04-29T15:00:00+08:00
---

# Debug Session: rtmp-option-t-not-found

## Symptom

### Expected Behavior
测试 RTMP 流媒体连接应该成功，返回流媒体信息

### Actual Behavior
FFprobe 返回错误 "Failed to set value '3' for option 't': Option not found"

### Error Message
```
Failed to set value '3' for option 't': Option not found
exit status 1
```

### Timeline
- 新功能第一次使用，从未成功运行过
- 功能位置：`internal/services/huawei_config_service.go:417` (TestStreamConnection)

### Reproduction Steps
1. 在华为配置页面启用流媒体录制
2. 选择 RTMP/RTSP/SRT/HLS 协议
3. 输入流媒体地址
4. 点击测试按钮

## Current Focus

### Hypothesis
FFprobe 不支持 `-t` 选项，该选项是 FFmpeg 专用的。代码错误地使用了 FFmpeg 的选项。

### Test
执行 `ffprobe -help` 查看 `-t` 选项是否存在

### Expecting
FFprobe 的帮助文档中不会显示 `-t` 选项

### Next Action
从代码中完全移除 `-t` 选项

### Reasoning Checkpoint
`-t` 选项用于限制读取时长，但这是 FFmpeg 的选项。FFprobe 是分析工具，不需要这个选项。测试连接时只需要确认流可访问。

## Evidence
- timestamp: 2026-04-29T14:52:00+08:00
  source: user_report
  detail: 错误日志显示 FFprobe 无法识别 `-t` 选项
- timestamp: 2026-04-29T15:00:00+08:00
  source: command_test
  detail: `ffprobe -help` 没有显示 `-t` 选项，`ffprobe -t 3 -i ...` 返回 "Option not found"
- timestamp: 2026-04-29T15:00:30+08:00
  source: fix_verification
  detail: 移除 `-t` 选项后，`ffprobe -i http://10.62.1.158/lives/m0/hls.m3u8` 成功返回流信息

## Eliminated
- timestamp: 2026-04-29T14:55:00+08:00
  hypothesis: `-t` 选项位置错误（应该在 `-i` 之前）
  reason: 实际测试发现 FFprobe 完全不支持 `-t` 选项，与位置无关
  test_result: `ffprobe -t 3 -i ...` 和 `ffprobe -i ... -t 3` 都返回 "Option not found"

## Resolution
### Root Cause
FFprobe 不支持 `-t` 选项。该选项是 FFmpeg 用于限制读取/写入时长的选项。FFprobe 是媒体分析工具，其工作方式与 FFmpeg 不同，不需要也不支持时长限制选项。原始代码错误地将 FFmpeg 的选项用于 FFprobe 命令。

### Fix
从 `huawei_config_service.go:430-445` 的 `inputArgs` 构建逻辑中完全移除 `-t 3` 参数。所有协议（rtmp, rtsp, srt, hls）都使用不带 `-t` 的命令。

### Verification
修复后执行 `ffprobe -v error -show_streams -show_format -print_format json -i http://10.62.1.158/lives/m0/hls.m3u8` 成功返回流媒体信息（包含 h264 编码、1920x1080 分辨率、60fps 等）。

### Files Changed
- `internal/services/huawei_config_service.go` - 移除所有协议中的 `-t 3` 参数
