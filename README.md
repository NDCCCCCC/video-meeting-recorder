# Record V2

视频切割和会议转录PPT系统

## 功能

- 视频多点分割
- 本地/云端转录（阿里通义听悟）
- PPT自动生成和编辑
- Windows AD域控认证
- USB/流媒体录制支持
- 批量操作

## Windows AD域控认证

系统支持Windows Active Directory域控认证，允许企业用户使用AD账号登录。

### 配置要求

- Windows Server 2008+ Active Directory
- LDAP端口389或LDAPS端口636
- AD管理员账号

### 快速配置

1. 使用管理员账号登录
2. 导航到"系统设置 > 域控认证"
3. 选择"AD域控认证"模式
4. 配置AD服务器信息并测试连接
5. 保存配置

### 安全注意事项

- ⚠️ **生产环境必须使用LDAPS端口636**，LDAP端口389存在明文传输风险
- AD管理员密码使用环境变量存储

详细配置说明：[Windows AD域控认证管理员指南](./.planning/phases/12-windows-ad/12-DOCS.md)
