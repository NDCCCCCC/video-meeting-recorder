package huawei

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// fatalFunc 允许测试覆盖 logger.Fatal 行为（默认触发 os.Exit）；测试可替换为 panic。
var fatalFunc = func(logger *zap.Logger, msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

// Manager 华为终端客户端管理器
// 负责管理多个华为终端API客户端的生命周期
type Manager struct {
	clients map[uint]*HuaweiClient // configID -> client
	mu      sync.RWMutex
	logger  *zap.Logger
	db      DBInterface
	// SEC-003a: 全局 TLS 策略（由 app 层通过 SetTLSPolicy 注入）。
	// 默认 InsecureSkipVerify=false、MinTLSVersion=tls.VersionTLS12。
	tlsInsecureSkipVerify bool
	tlsMinVersion         uint16
	// SEC-003a: 私有 CA 信任锚（由 SetCABundle 注入；nil → 系统 CA bundle）。
	// 必须在 createClient 时按指针快照，避免运行时被 SetCABundle 改写导致部分客户端
	// 拿不到新 pool。已缓存的客户端不在 SetCABundle 范围内重新初始化——这与 SetTLSPolicy
	// 同源语义，需要重启或下次 createClient 才可见新锚。
	caCertPool *x509.CertPool
	// REDO (huawei-tls-after-ca-fix) 修复：hostname 校验所需的显式 ServerName（dNSName
	// 路径），由 SetTLSServerName 注入。空字符串保留 Go 默认（用 dial 地址做 hostname
	// 校验），与 SetTLSPolicy / SetCABundle 同源语义。运维可通过
	// huawei.tls_server_name / HUAWEI_TLS_SERVER_NAME 配置（推荐值 "vct.tp.huawei.com"）。
	tlsServerName string
	// SEC-013: 出站 URL 白名单 + 环境标识（开发环境绕过）
	outboundURLAllowlist []string
	environment          string
}

// DBInterface 数据库接口（用于解耦）
type DBInterface interface {
	GetHuaweiConfig(configID uint) (*HuaweiConfigDB, error)
}

// HuaweiConfigDB 数据库中的华为配置
type HuaweiConfigDB struct {
	ID             uint
	Server         string
	Port           int
	Username       string
	Password       string
	TerminalNumber string
}

// NewManager 创建华为终端客户端管理器
func NewManager(logger *zap.Logger, db DBInterface) *Manager {
	return &Manager{
		clients:       make(map[uint]*HuaweiClient),
		logger:        logger,
		db:            db,
		tlsMinVersion: tls.VersionTLS12, // SEC-003a: 默认强制 TLS 1.2 最低
		// tlsInsecureSkipVerify 默认 false（零值）
	}
}

// SetTLSPolicy 设置华为客户端全局 TLS 策略（SEC-003a）。
// insecureSkipVerify 默认 false；生产环境若为 true 则 logger.Fatal（defense-in-depth）。
// minTLSVersion 为 0 时归一化为 tls.VersionTLS12。
func (m *Manager) SetTLSPolicy(insecureSkipVerify bool, minTLSVersion uint16, isProduction bool) {
	if minTLSVersion == 0 {
		minTLSVersion = tls.VersionTLS12
	}
	if isProduction && insecureSkipVerify {
		fatalFunc(m.logger, "生产环境不允许 HUAWEI_INSECURE_SKIP_VERIFY=true，进程终止（SEC-003a）")
		return
	}
	m.tlsInsecureSkipVerify = insecureSkipVerify
	m.tlsMinVersion = minTLSVersion
}

// SetTLSServerName 设置华为客户端全局 ServerName（REDO）。
// non-empty 时把值透传到所有后续 createClient 的 tls.Config.ServerName，让 Go x509
// verifier 用 dNSName 匹配 cert SAN；空字符串保留 Go 默认行为（用 dial 地址做 hostname
// 校验）。与 SetTLSPolicy / SetCABundle 同源——只影响下次 createClient，已缓存的客户端
// 不会被改写。
//
// 入参归一化：调用方传值前自动 strings.TrimSpace，前后空白被剥离；trim 后空字符串视作
// "清空"（opt-out，与 ca_bundle_file 同语义）。所有写入都在 m.mu.Lock 内完成，
// createClient 会在 RLock 内同步快照，避免与读取侧发生数据竞争。
//
// 推荐配置：当 10.62.10.3 的 server cert SAN 仅含 DNS（如 *.tp.huawei.com）且客户端
// 必须 IP-only dial 时，运维应在 huawei.tls_server_name 填入匹配的字面名（如
// "vct.tp.huawei.com"），这是 REDO 修复的入口。
func (m *Manager) SetTLSServerName(name string) {
	trimmed := strings.TrimSpace(name)
	m.mu.Lock()
	m.tlsServerName = trimmed
	m.mu.Unlock()
}

// SetOutboundURLAllowlist 注入 SEC-013 出站 URL 白名单与运行环境。
// env=="development" 时 allowlist 为空也允许所有出站。
func (m *Manager) SetOutboundURLAllowlist(allowlist []string, environment string) {
	m.outboundURLAllowlist = allowlist
	m.environment = environment
}

// SetCABundle 加载华为终端私有 CA 信任锚（SEC-003a-01/02）。
//
// 语义：
//   - path 为空字符串或仅包含空白字符 → 原子把 m.caCertPool 设为 nil，
//     返回 nil（系统 CA bundle 兜底行为，完整证书校验仍启用）。
//   - path 非空 → 读取 PEM 文件、解析每一个 CERTIFICATE block、为
//     每一个 cert 都过 x509.ParseCertificate、再全部 add 到一个全新
//     x509.NewCertPool；全部通过后才在 m.mu 保护下发布 m.caCertPool，
//     期间任何错误都返回 *不* 修改已发布 pool 的 wrapped error。错误
//     文本必须包含 path（运维定位），且通过 %w 保留底层 read/parse 原因。
//
// 拒绝条件（任一触发即整体失败、无部分可信 pool 残留）：
//   - os.ReadFile 失败（缺失/无权限等）
//   - 文件不包含任何 CERTIFICATE block
//   - 任何 PEM 块 type != "CERTIFICATE"
//   - 任何 x509.ParseCertificate 解析失败
//   - 解析后仍有非空白 trailing bytes（防止"看起来通过但实际多块"漏检）
//
// 安全不变量：从不把 PEM 字节或私钥写入日志；仅路径与解析数量。
//
// REDO (huawei-tls-after-ca-fix) 修正：先前 hotfix 在此处加了 "pool 内无 self-signed root
// → Warn" 的防御性日志，但主会话用真实 Go probe 验证：CA chain 实际 OK（pool 含 leaf +
// 已知正确的 chain-2 自签根时，错误就变成 hostname/SAN 校验），并不存在 "trust anchor"
// 缺失问题。先前的 Warn 基于错误前提，已撤回——SetCABundle 只负责原子发布 pool，不再做
// "pool 拓扑扫描"。hostname 修复走 SetTLSServerName + huawei.tls_server_name。
func (m *Manager) SetCABundle(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		m.mu.Lock()
		m.caCertPool = nil
		m.mu.Unlock()
		return nil
	}

	data, err := os.ReadFile(trimmed)
	if err != nil {
		return fmt.Errorf("读取华为 TLS CA bundle 失败: %s: %w", trimmed, err)
	}

	pool := x509.NewCertPool()
	rest := data
	certCount := 0
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return fmt.Errorf("解析华为 TLS CA bundle 失败: %s: 不支持的 PEM 块类型 %q（仅接受 CERTIFICATE）", trimmed, block.Type)
		}
		cert, parseErr := x509.ParseCertificate(block.Bytes)
		if parseErr != nil {
			return fmt.Errorf("解析华为 TLS CA bundle 失败: %s: x509.ParseCertificate 错误: %w", trimmed, parseErr)
		}
		pool.AddCert(cert)
		certCount++
	}
	if certCount == 0 {
		return fmt.Errorf("解析华为 TLS CA bundle 失败: %s: 未找到任何 CERTIFICATE 块", trimmed)
	}
	if strings.TrimSpace(string(rest)) != "" {
		return fmt.Errorf("解析华为 TLS CA bundle 失败: %s: 文件末尾残留非空白字节（请清理 PEM 文件末尾的额外数据）", trimmed)
	}

	// 全部校验通过 → 在锁内一次性发布；中途失败不会到达这里。
	m.mu.Lock()
	m.caCertPool = pool
	m.mu.Unlock()

	m.logger.Info("华为 TLS CA bundle 加载成功",
		zap.String("path", trimmed),
		zap.Int("cert_count", certCount),
	)
	return nil
}

// ParseMinTLSVersion 将配置中的字符串版本号（"1.2"/"1.3"）或数字（"771"）解析为 tls 常量。
// 空串或无法识别返回 0（由调用方归一化为 tls.VersionTLS12）。SEC-003a/D-03.5。
func ParseMinTLSVersion(s string) uint16 {
	switch s {
	case "1.2", "771":
		return tls.VersionTLS12
	case "1.3", "772":
		return tls.VersionTLS13
	case "1.1", "770":
		return tls.VersionTLS11
	case "1.0", "769":
		// SEC-003a: 显式拒绝 TLS 1.0，归一化为 1.2。
		return tls.VersionTLS12
	default:
		return 0
	}
}

// GetClient 获取或创建指定配置的华为客户端
func (m *Manager) GetClient(ctx context.Context, configID uint) (*HuaweiClient, error) {
	m.mu.RLock()
	client, exists := m.clients[configID]
	m.mu.RUnlock()

	if exists {
		// 检查客户端是否仍然有效
		if client.hasSession() {
			return client, nil
		}
		// 会话过期，需要重新创建（SEC-003a：透传 ctx，不再用 context.Background()）
		m.removeClient(ctx, configID)
	}

	// 创建新客户端
	return m.createClient(ctx, configID)
}

// createClient 创建新的华为客户端
func (m *Manager) createClient(ctx context.Context, configID uint) (*HuaweiClient, error) {
	// 从数据库获取配置
	cfg, err := m.db.GetHuaweiConfig(configID)
	if err != nil {
		return nil, fmt.Errorf("获取华为配置失败: %w: %w", apperrors.ErrInternal, err)
	}

	// SEC-003a: 在锁内快照当前 CA pool，避免 SetCABundle 后改写指针
	// 导致新建客户端拿到不一致状态。已缓存的客户端不重新初始化（与 SetTLSPolicy 语义一致）。
	// REDO: 同步快照 tlsServerName，遵循相同的"启动期一次性注入"并发语义——
	// SetTLSServerName 用 m.mu.Lock 写,createClient 在同一 RLock 窗口内读,
	// 避免后续阶段若引入运行期改写时出现 data race。
	m.mu.RLock()
	caPool := m.caCertPool
	name := m.tlsServerName
	m.mu.RUnlock()

	config := &Config{
		Server: cfg.Server,
		Port:   cfg.Port,
		// SEC-003b/Phase 21: 密码已在 cmd/server/app.go 中的 huaweiDBAdapter 里经过
		// CredentialEncryptor.Decrypt 解密后传入——此层视为**明文边界**。原始 DB 字段
		// 走 Phase 18 的"SM4:<version>:<base64>"信封,不解密直接看是密文。
		Username:           cfg.Username,
		Password:           cfg.Password,
		APITimeout:         30 * time.Second,
		SessionTimeout:     1800 * time.Second,      // 30分钟会话有效期
		KeepAliveInterval:  30 * time.Second,        // 30秒保活间隔（必须小于60秒）
		InsecureSkipVerify: m.tlsInsecureSkipVerify, // SEC-003a: 默认 false，可配置
		MinTLSVersion:      m.tlsMinVersion,         // SEC-003a: 默认 tls.VersionTLS12，不再硬编码 TLS1.0
		caCertPool:         caPool,                  // SEC-003a: 由 SetCABundle 注入；nil → 系统 CA
		tlsServerName:      name,                    // REDO: RLock 内已快照；nil/"" → Go 默认（用 dial 地址）
	}

	client := NewHuaweiClient(config, m.logger)
	// SEC-013: 注入出站 URL 白名单与开发环境标识
	client.httpClient.SetOutboundURLAllowlist(m.outboundURLAllowlist, m.environment)

	// 初始化并启动保活（消费 cfg.Password → 其 err 受凭据污染）。
	// 不 %w 包裹受污染 err，返回哨兵断开敏感日志污点链（CodeQL #24/#25/#28）。
	if err := client.InitializeAndStartKeepAlive(ctx); err != nil {
		return nil, apperrors.ErrHuaweiAuthFailed
	}

	// 缓存客户端
	m.mu.Lock()
	m.clients[configID] = client
	m.mu.Unlock()

	m.logger.Info("创建华为终端客户端成功",
		zap.Uint("config_id", configID),
		zap.String("server", cfg.Server),
		zap.Int("port", cfg.Port),
	)

	return client, nil
}

// GetFirstRegisteredClient 返回当前已注册的第一个 HuaweiClient（按 map 迭代顺序
// 非确定；仅用于单设备部署中 Phase 25 SCHED-01 把 *Manager 桥接到 recorder
// HuaweiStateClient 的场景）。bool=false 表示尚无 client 注册 — caller 应视作
// "H 信号不可用"并降级（HuaweiStateClient.GetConferenceState 返回 error →
// ActivityWatcher 走 huaweiConsecFailures 累加路径）。
//
// 注意：仅返回已缓存 client，不触发 createClient。session 过期由调用方通过
// HuaweiClient.hasSession() 自检或调用 GetClient(ctx, configID) 走重建路径。
// 多设备场景请使用 GetClient(ctx, configID) 显式指定 configID，避免跨终端
// 状态错配。
func (m *Manager) GetFirstRegisteredClient() (*HuaweiClient, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, client := range m.clients {
		return client, true
	}
	return nil, false
}

// removeClient 移除客户端（SEC-003a：透传 ctx；PERF-004：Logout 移出锁，仅 map 操作进锁）
func (m *Manager) removeClient(ctx context.Context, configID uint) {
	m.mu.Lock()
	client, exists := m.clients[configID]
	if exists {
		delete(m.clients, configID)
	}
	m.mu.Unlock()

	if exists {
		if err := client.Logout(ctx); err != nil {
			m.logger.Warn("登出华为客户端失败", zap.Uint("config_id", configID), zap.Error(err), response.SentinelField(err))
		}
	}
}

// Close 关闭管理器，清理所有客户端（PERF-004：批量取客户端后解锁，再逐个 Logout）
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	clients := m.clients
	m.clients = make(map[uint]*HuaweiClient)
	m.mu.Unlock()

	for configID, client := range clients {
		if err := client.Logout(ctx); err != nil {
			m.logger.Error("关闭华为客户端失败",
				zap.Uint("config_id", configID),
				zap.Error(err),
				response.SentinelField(err),
			)
		}
	}

	return nil
}

// CallConference 使用指定配置呼叫会议
func (m *Manager) CallConference(ctx context.Context, configID uint, req *CallConferenceRequest) (*CallConferenceResponse, error) {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return nil, err
	}

	if err := client.CallConference(ctx, req.ConferenceNumber); err != nil {
		return nil, err
	}

	return &CallConferenceResponse{
		CallID: fmt.Sprintf("call_%d_%s", configID, req.ConferenceNumber),
		Status: "calling",
	}, nil
}

// HangupConference 使用指定配置挂断会议
func (m *Manager) HangupConference(ctx context.Context, configID uint, req *HangupConferenceRequest) error {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return err
	}
	return client.HangupCall(ctx)
}

// GetConferenceInfo 获取会议信息
func (m *Manager) GetConferenceInfo(ctx context.Context, configID uint, conferenceNumber string) (*ConferenceInfo, error) {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return nil, err
	}
	return client.GetConferenceInfo(ctx)
}

// GetTerminalStatus 获取终端状态
func (m *Manager) GetTerminalStatus(ctx context.Context, configID uint, terminalNumber string) (*TerminalStatus, error) {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return nil, err
	}
	return client.GetTerminalStatus(ctx, terminalNumber)
}

// SafeCallConference 安全呼叫会议（先挂断再呼叫）
func (m *Manager) SafeCallConference(ctx context.Context, configID uint, req *CallConferenceRequest) error {
	m.logger.Info("安全呼叫会议",
		zap.Uint("config_id", configID),
		zap.String("conference_number", req.ConferenceNumber),
	)

	// 1. 获取客户端
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return fmt.Errorf("获取客户端失败: %w: %w", apperrors.ErrInternal, err)
	}

	// 2. 获取终端状态
	status, err := client.GetTerminalStatus(ctx, req.TerminalNumber)
	if err != nil {
		return fmt.Errorf("获取终端状态失败: %w: %w", apperrors.ErrInternal, err)
	}

	// 3. 如果终端正在通话，先挂断
	if status.Status == "in_call" {
		m.logger.Info("终端正在通话，先挂断", zap.String("terminal_number", req.TerminalNumber))
		if err := client.HangupCall(ctx); err != nil {
			m.logger.Warn("挂断残留连接失败，继续尝试呼叫", zap.Error(err), response.SentinelField(err))
		}
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	// 4. 呼叫会议
	if err := client.CallConference(ctx, req.ConferenceNumber); err != nil {
		return fmt.Errorf("呼叫会议失败: %w: %w", apperrors.ErrInternal, err)
	}

	m.logger.Info("安全呼叫会议成功",
		zap.Uint("config_id", configID),
		zap.String("conference_number", req.ConferenceNumber),
		zap.String("terminal_number", req.TerminalNumber),
	)

	return nil
}

// WaitForConnection 等待终端连接到会议
func (m *Manager) WaitForConnection(ctx context.Context, configID uint, conferenceNumber, terminalNumber string, timeout time.Duration) error {
	m.logger.Info("等待终端连接到会议",
		zap.Uint("config_id", configID),
		zap.String("conference_number", conferenceNumber),
		zap.String("terminal_number", terminalNumber),
		zap.Duration("timeout", timeout),
	)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	maxRetries := 5
	retryCount := 0

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待连接超时: %w", apperrors.ErrServiceUnavailable)
		case <-ticker.C:
			client, err := m.GetClient(ctx, configID)
			if err != nil {
				// 不记录原始 err（client 持凭据，错误可能受污染）；仅记 SentinelField 分类（CodeQL #24）
				m.logger.Warn("获取客户端失败，继续等待", response.SentinelField(err))
				continue
			}

			// 使用 IsInConference 检查终端是否在会议中
			inConf, err := client.IsInConference(ctx)
			if err != nil {
				retryCount++
				if retryCount >= maxRetries {
					return fmt.Errorf("获取会议信息失败，已重试%d次: %w: %w", maxRetries, apperrors.ErrInternal, err)
				}
				m.logger.Warn("获取会议信息失败，继续等待",
					zap.Error(err),
					zap.Int("retry", retryCount),
					response.SentinelField(err),
				)
				continue
			}
			retryCount = 0 // 重置重试计数

			if inConf {
				m.logger.Info("终端已连接到会议",
					zap.String("terminal_number", terminalNumber),
					zap.String("conference_number", conferenceNumber),
				)
				return nil
			}

			m.logger.Debug("终端尚未连接，继续等待",
				zap.String("terminal_number", terminalNumber),
			)
		}
	}
}

// GetRTSPStreams 获取会议的RTSP流
func (m *Manager) GetRTSPStreams(ctx context.Context, configID uint, conferenceNumber string) ([]RTSPStream, error) {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return nil, err
	}

	info, err := client.GetConferenceInfo(ctx)
	if err != nil {
		return nil, err
	}

	if len(info.RTSPStreams) == 0 {
		// 如果没有返回RTSP流，使用默认格式
		streamURL, _ := client.GetRTSPStreamURL(conferenceNumber)
		return []RTSPStream{
			{Type: "main", URL: streamURL},
		}, nil
	}

	return info.RTSPStreams, nil
}

// IsTerminalIdle 检查终端是否空闲
func (m *Manager) IsTerminalIdle(ctx context.Context, configID uint, terminalNumber string) (bool, error) {
	status, err := m.GetTerminalStatus(ctx, configID, terminalNumber)
	if err != nil {
		return false, err
	}
	return status.Status == "idle", nil
}