# Milestones

## v1.0 视频切割与会议转录PPT (Shipped: 2026-05-06)

**Phases completed:** 14 phases, 69 plans, 2 deferred items

**Key accomplishments:**

1. **视频分割功能** - 用户可以在浏览器中点击时间线添加分割标记，FFmpeg 分割视频为多个 MP4 片段
2. **本地/云端转录** - 集成阿里通义听悟，支持本地和云端两种转录模式，自动生成 PPT
3. **PPT 管理和编辑** - PPT 预览、多结果管理、幻灯片合并、重复检测、删除回滚、视频帧捕获插入
4. **AD 域控认证** - 集成 Windows Active Directory 认证，支持白名单模式
5. **USB/流媒体录制** - 重构配置架构，支持 USB 直录和流媒体 (RTMP/RTSP) 录制模式
6. **批量操作** - 文件管理页面支持批量下载（ZIP 打包）和批量转录（任务组模式）
7. **多角色权限** - 用户可以拥有多个角色，权限聚合使用 OR 逻辑
8. **审计日志** - 记录敏感操作供审计查询
9. **UI 增强** - 预览页面缩略图侧边栏、可编辑进度条、PPT 切换下拉、键盘快捷键

**Known deferred items at close:** 2 (UAT 测试延期 - Phase 01-08 UI 交互测试需要人工运行时验证)

---
*See .planning/milestones/v1.0-ROADMAP.md for full milestone details.*
