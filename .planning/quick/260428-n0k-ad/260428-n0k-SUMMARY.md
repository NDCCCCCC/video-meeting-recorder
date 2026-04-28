---
phase: quick
plan: 01
type: execute
status: complete
date: 2026-04-28
---

# AD 配置持久化 - 实施总结

## 目标
持久化 AD 域控制器配置到 SQLite 数据库，使通过管理员 API 保存的设置在服务器重启后保持有效。

## 实施内容

### Task 1: SystemSetting 模型和 ConfigService ✅

**文件创建:**
- `internal/models/system_setting.go` - 键值对存储模型
- `internal/services/config_service.go` - 配置加载/保存服务

**关键特性:**
- SystemSetting 使用简单结构（无 Base 嵌入，无软删除）
- 定义了三个配置键常量：`auth.mode`, `auth.ad`, `auth.ad.password`
- ConfigService 提供启动时加载（LoadAuthConfig）和运行时保存（SaveAuthConfig）
- 密码使用 base64 编码存储（TODO: SM4-GCM 加密）

### Task 2: 集成到 AdminHandler 和启动流程 ✅

**修改文件:**
- `internal/handlers/admin_handler.go` - 添加 ConfigService 字段，UpdateAuthConfig 调用保存
- `cmd/server/app.go` - 创建 ConfigService，调用 LoadAuthConfig，传递给 AdminHandler

**关键流程:**
1. 服务器启动时：创建 ConfigService → LoadAuthConfig() 从数据库加载（覆盖 YAML 默认值）
2. 用户通过 API 修改配置：UpdateAuthConfig → 更新内存 → SaveAuthConfig() 保存到数据库
3. 重启服务器：LoadAuthConfig() 恢复之前保存的配置

## 验证结果

- ✅ `go build ./cmd/server/` 编译通过
- ✅ SystemSetting 添加到 AutoMigrate
- ✅ ConfigService 在 initHandlers 中创建并调用 LoadAuthConfig
- ✅ AdminHandler 接收 ConfigService 并在配置更新时持久化

## 注意事项

1. **密码加密**: 当前使用 base64 编码，代码中有 TODO 注释标记需要升级到 SM4-GCM
2. **降级处理**: 如果 SM4 解密失败（如密钥未就绪），会降级到 base64 并记录警告
3. **内存优先**: 如果 DB 保存失败，不会回滚内存中的配置更新（避免丢失用户输入）

## 后续建议

1. 实现真正的 SM4-GCM 加密用于密码静态存储
2. 考虑为其他系统配置（如 HuaweiConfig）添加类似持久化机制
3. 添加配置变更审计日志
