package huawei

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
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
		return nil, fmt.Errorf("获取华为配置失败: %w", err)
	}

	config := &Config{
		Server: cfg.Server,
		Port:   cfg.Port,
		// SEC-003b deferred (per CONTEXT.md): 华为密码 DB 加密需独立迁移 + 前端/配置联动，
		// 本 phase 仅完成 SEC-003a（TLS 三项 + ctx 透传），密码仍以明文形式从 DB 读取。
		Username:           cfg.Username,
		Password:           cfg.Password,
		APITimeout:         30 * time.Second,
		SessionTimeout:     1800 * time.Second,      // 30分钟会话有效期
		KeepAliveInterval:  30 * time.Second,        // 30秒保活间隔（必须小于60秒）
		InsecureSkipVerify: m.tlsInsecureSkipVerify, // SEC-003a: 默认 false，可配置
		MinTLSVersion:      m.tlsMinVersion,         // SEC-003a: 默认 tls.VersionTLS12，不再硬编码 TLS1.0
	}

	client := NewHuaweiClient(config, m.logger)

	// 初始化并启动保活
	if err := client.InitializeAndStartKeepAlive(ctx); err != nil {
		return nil, fmt.Errorf("华为终端初始化失败: %w", err)
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
			m.logger.Warn("登出华为客户端失败", zap.Uint("config_id", configID), zap.Error(err))
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
		return fmt.Errorf("获取客户端失败: %w", err)
	}

	// 2. 获取终端状态
	status, err := client.GetTerminalStatus(ctx, req.TerminalNumber)
	if err != nil {
		return fmt.Errorf("获取终端状态失败: %w", err)
	}

	// 3. 如果终端正在通话，先挂断
	if status.Status == "in_call" {
		m.logger.Info("终端正在通话，先挂断", zap.String("terminal_number", req.TerminalNumber))
		if err := client.HangupCall(ctx); err != nil {
			m.logger.Warn("挂断残留连接失败，继续尝试呼叫", zap.Error(err))
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
		return fmt.Errorf("呼叫会议失败: %w", err)
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
			return fmt.Errorf("等待连接超时")
		case <-ticker.C:
			client, err := m.GetClient(ctx, configID)
			if err != nil {
				m.logger.Warn("获取客户端失败，继续等待", zap.Error(err))
				continue
			}

			// 使用 IsInConference 检查终端是否在会议中
			inConf, err := client.IsInConference(ctx)
			if err != nil {
				retryCount++
				if retryCount >= maxRetries {
					return fmt.Errorf("获取会议信息失败，已重试%d次: %w", maxRetries, err)
				}
				m.logger.Warn("获取会议信息失败，继续等待",
					zap.Error(err),
					zap.Int("retry", retryCount),
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
