# Phase 12: Windows AD域控认证 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-28
**Phase:** 12 - Windows AD域控认证
**Areas discussed:** 认证模式默认值, 安全提示UX, AD用户管理, 错误处理

---

## 认证模式默认值

| Option | Description | Selected |
|--------|-------------|----------|
| 默认local模式 | 最安全的选择，不与AD交互。管理员需要主动切换到AD/hybrid模式。 | ✓ |
| 默认hybrid模式 | 向后兼容，同时支持本地和AD用户。需要先配置AD才能工作。 | |
| 默认ad模式 | 完全AD集中管理，需要完整配置AD。适合纯AD环境。 | |

**User's choice:** 默认local模式

---

| Option | Description | Selected |
|--------|-------------|----------|
| 配置向导式 | 进入系统设置页面，找到认证配置，填写AD信息，保存并切换。 | |
| 简单表单式 | 直接在配置页面填写AD服务器信息，提供测试连接按钮。 | ✓ |
| 模板填充式 | 提供预设模板，只需填写AD服务器地址和凭据即可。 | |

**User's choice:** 简单表单式

---

## 安全提示UX

| Option | Description | Selected |
|--------|-------------|----------|
| 顶部警告条 | 页面顶部显示橙色警告条，包含风险说明和确认复选框。 | |
| 字段内联提示 | 配置字段旁显示警告图标(⚠)，点击查看风险说明。 | ✓ |
| 保存确认弹窗 | 保存时弹出确认对话框，说明风险并提供返回选项。 | |

**User's choice:** 字段内联提示

---

| Option | Description | Selected |
|--------|-------------|----------|
| 需要显式确认 | 显示警告信息，勾选复选框确认了解风险，然后才能保存。 | |
| 被动记录即可 | 显示警告信息，保存即可，在审计日志中记录警告已展示。 | ✓ |

**User's choice:** 被动记录即可

---

## AD用户管理

**用户澄清的关键设计决策:**
- 账号不区分来源，所有账号都需要设置本地密码
- 设置为AD域控登录时本地密码无效，必须使用AD域控账号密码登录
- 域控认证失败（账号不存在）直接返回提示，不进行本地认证
- 设置为本地认证时，使用本地账号密码进行认证

---

## 认证模式重新设计

| Option | Description | Selected |
|--------|-------------|----------|
| 只需要local/ad两种 | 仅支持local和ad两种模式，系统级配置，切换后所有用户使用该模式。 | ✓ |
| 保留hybrid但重新定义 | 保留hybrid模式，按用户的澄清：AD模式不降级，hybrid模式先本地后AD。 | |

**User's choice:** 只需要local/ad两种

**Notes:** 用户明确不需要hybrid模式，简化了认证逻辑设计

---

| Option | Description | Selected |
|--------|-------------|----------|
| 存储但不显示 | 数据库中存储auth_source字段，但UI不区分显示，统一管理。 | |
| 完全透明 | UI也不需要显示用户来源，用户就是用户。 | ✓ |

**User's choice:** 完全透明

**Notes:** 不需要auth_source字段，简化了数据库设计

---

## 错误处理

| Option | Description | Selected |
|--------|-------------|----------|
| 友好提示+详细日志 | 显示友好提示：无法连接到AD服务器，请检查网络和配置。记录详细日志。 | ✓ |
| 技术性错误信息 | 显示技术性错误信息（LDAP错误码），方便管理员排查问题。 | |
| 分级显示 | 分级显示：管理员看详细错误，普通用户看友好提示。 | |

**User's choice:** 友好提示+详细日志

---

| Option | Description | Selected |
|--------|-------------|----------|
| 手动测试按钮 | 在配置页面提供测试按钮，调用AD连通性测试API，显示结果。 | |
| 自动验证+阻止保存 | 配置变更时自动验证，验证失败阻止保存并显示原因。 | ✓ |
| 模式切换时验证 | 切换到AD模式时验证，验证失败阻止模式切换。 | |

**User's choice:** 自动验证+阻止保存

---

## Claude's Discretion

无 - 用户对所有关键决策都提供了明确的选择

---

## Deferred Ideas

- AD组→角色映射
- 定期自动同步AD用户状态
- AD用户单独管理界面
- 支持多个AD服务器配置
- AD用户密码修改功能

---

## Design Evolution

**与Spike验证的主要差异:**
1. **移除hybrid模式** - Spike建议保留local/ad/hybrid三种模式，用户决定只用local/ad
2. **移除auth_source字段** - Spike建议记录用户来源，用户决定完全透明管理
3. **简化认证逻辑** - AD模式不降级，比hybrid模式的"先本地后AD"更简洁

**优势:**
- 更简洁的系统架构
- 更清晰的认证流程
- 更简单的用户管理

**权衡:**
- 失去了hybrid模式的灵活性（同时支持本地和AD用户）
- 需要管理员明确选择认证模式，不能自动降级
