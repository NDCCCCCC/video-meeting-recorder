package huawei

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager 华为客户端管理器
// 负责管理多个华为API客户端的生命周期
type Manager struct {
	clients map[uint]*HuaweiClient // configID -> client
	mu      sync.RWMutex
	logger  *zap.Logger
	db      DBInterface
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

// NewManager 创建华为客户端管理器
func NewManager(logger *zap.Logger, db DBInterface) *Manager {
	return &Manager{
		clients: make(map[uint]*HuaweiClient),
		logger:  logger,
		db:      db,
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
		// 会话过期，需要重新创建
		m.removeClient(configID)
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

	// 构建API配置
	scheme := "https"
	if cfg.Port == 80 || cfg.Port == 443 {
		// 根据端口判断协议
		if cfg.Port == 80 {
			scheme = "http"
		}
	}

	apiBase := fmt.Sprintf("%s://%s:%d/api/v1", scheme, cfg.Server, cfg.Port)

	config := &Config{
		Server:            cfg.Server,
		Port:              cfg.Port,
		Username:          cfg.Username,
		Password:          cfg.Password,
		APIBase:           apiBase,
		APITimeout:        30 * time.Second,
		SessionTimeout:    3600 * time.Second,
		KeepAliveInterval: 300 * time.Second,
		InsecureSkipVerify: true,
		MinTLSVersion:     0x0301, // TLS 1.0
	}

	client := NewHuaweiClient(config, m.logger)

	// 登录
	if err := client.EnsureLogin(ctx); err != nil {
		return nil, fmt.Errorf("华为API登录失败: %w", err)
	}

	// 启动保活
	client.StartKeepAlive(ctx)

	// 缓存客户端
	m.mu.Lock()
	m.clients[configID] = client
	m.mu.Unlock()

	m.logger.Info("创建华为客户端成功",
		zap.Uint("config_id", configID),
		zap.String("server", cfg.Server),
		zap.Int("port", cfg.Port),
	)

	return client, nil
}

// removeClient 移除客户端
func (m *Manager) removeClient(configID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.clients[configID]; exists {
		client.Logout(context.Background())
		delete(m.clients, configID)
	}
}

// Close 关闭管理器，清理所有客户端
func (m *Manager) Close(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for configID, client := range m.clients {
		if err := client.Logout(ctx); err != nil {
			m.logger.Error("关闭华为客户端失败",
				zap.Uint("config_id", configID),
				zap.Error(err),
			)
		}
	}

	m.clients = make(map[uint]*HuaweiClient)
	return nil
}

// CallConference 使用指定配置呼叫会议
func (m *Manager) CallConference(ctx context.Context, configID uint, req *CallConferenceRequest) (*CallConferenceResponse, error) {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return nil, err
	}
	return client.CallConference(ctx, req)
}

// HangupConference 使用指定配置挂断会议
func (m *Manager) HangupConference(ctx context.Context, configID uint, req *HangupConferenceRequest) error {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return err
	}
	return client.HangupConference(ctx, req)
}

// GetConferenceInfo 获取会议信息
func (m *Manager) GetConferenceInfo(ctx context.Context, configID uint, conferenceNumber string) (*ConferenceInfo, error) {
	client, err := m.GetClient(ctx, configID)
	if err != nil {
		return nil, err
	}
	return client.GetConferenceInfo(ctx, conferenceNumber)
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
		zap.String("terminal_number", req.TerminalNumber),
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
		hangupReq := &HangupConferenceRequest{
			TerminalNumber: req.TerminalNumber,
		}
		if err := client.HangupConference(ctx, hangupReq); err != nil {
			m.logger.Warn("挂断残留连接失败，继续尝试呼叫", zap.Error(err))
		}
		time.Sleep(1 * time.Second)
	}

	// 4. 呼叫会议
	resp, err := client.CallConference(ctx, req)
	if err != nil {
		return fmt.Errorf("呼叫会议失败: %w", err)
	}

	m.logger.Info("安全呼叫会议成功",
		zap.Uint("config_id", configID),
		zap.String("conference_number", req.ConferenceNumber),
		zap.String("terminal_number", req.TerminalNumber),
		zap.String("call_id", resp.CallID),
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

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待连接超时")
		case <-ticker.C:
			info, err := m.GetConferenceInfo(ctx, configID, conferenceNumber)
			if err != nil {
				m.logger.Warn("获取会议信息失败，继续等待", zap.Error(err))
				continue
			}

			// 检查终端是否已加入
			for _, attendee := range info.Attendees {
				if attendee.TerminalNumber == terminalNumber && attendee.Status == "connected" {
					m.logger.Info("终端已连接到会议",
						zap.String("terminal_number", terminalNumber),
						zap.String("conference_number", conferenceNumber),
					)
					return nil
				}
			}

			m.logger.Debug("终端尚未连接，继续等待",
				zap.String("terminal_number", terminalNumber),
				zap.Int("attendees_count", info.AttendeesCount),
			)
		}
	}
}

// GetRTSPStreams 获取会议的RTSP流
func (m *Manager) GetRTSPStreams(ctx context.Context, configID uint, conferenceNumber string) ([]RTSPStream, error) {
	info, err := m.GetConferenceInfo(ctx, configID, conferenceNumber)
	if err != nil {
		return nil, err
	}

	if len(info.RTSPStreams) == 0 {
		return nil, fmt.Errorf("会议没有可用的RTSP流")
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
