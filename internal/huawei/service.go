package huawei

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HuaweiService 华为服务层
type HuaweiService struct {
	client *HuaweiClient
	db     *gorm.DB
	logger *zap.Logger
	config *Config
}

// NewHuaweiService 创建华为服务
func NewHuaweiService(db *gorm.DB, config *Config, logger *zap.Logger) *HuaweiService {
	return &HuaweiService{
		db:     db,
		config: config,
		logger: logger,
	}
}

// Start 启动服务
func (s *HuaweiService) Start(ctx context.Context) error {
	s.client = NewHuaweiClient(s.config, s.logger)

	// 登录
	if err := s.client.EnsureLogin(ctx); err != nil {
		return fmt.Errorf("华为API登录失败: %w", err)
	}

	// 启动自动保活
	s.client.StartKeepAlive(ctx)

	s.logger.Info("华为服务启动成功")
	return nil
}

// Stop 停止服务
func (s *HuaweiService) Stop(ctx context.Context) error {
	if s.client != nil {
		if err := s.client.Logout(ctx); err != nil {
			s.logger.Error("华为API登出失败", zap.Error(err))
		}
	}
	s.logger.Info("华为服务已停止")
	return nil
}

// Login 用户登录
func (s *HuaweiService) Login(ctx context.Context) error {
	return s.client.Login(ctx)
}

// KeepAlive 会话保活
func (s *HuaweiService) KeepAlive(ctx context.Context) error {
	return s.client.KeepAlive(ctx)
}

// SafeCallConference 安全呼叫会议（先挂断再呼叫）
func (s *HuaweiService) SafeCallConference(ctx context.Context, req *CallConferenceRequest) error {
	s.logger.Info("安全呼叫会议",
		zap.String("conference_number", req.ConferenceNumber),
		zap.String("terminal_number", req.TerminalNumber),
	)

	// 1. 确保已登录
	if err := s.client.EnsureLogin(ctx); err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}

	// 2. 获取终端状态
	status, err := s.client.GetTerminalStatus(ctx, req.TerminalNumber)
	if err != nil {
		return fmt.Errorf("获取终端状态失败: %w", err)
	}

	// 3. 如果终端正在通话，先挂断
	if status.Status == "in_call" {
		s.logger.Info("终端正在通话，先挂断", zap.String("terminal_number", req.TerminalNumber))
		hangupReq := &HangupConferenceRequest{
			TerminalNumber: req.TerminalNumber,
		}
		if err := s.client.HangupConference(ctx, hangupReq); err != nil {
			s.logger.Warn("挂断残留连接失败，继续尝试呼叫", zap.Error(err))
		}

		// 等待挂断完成
		time.Sleep(1 * time.Second)
	}

	// 4. 呼叫会议
	resp, err := s.client.CallConference(ctx, req)
	if err != nil {
		return fmt.Errorf("呼叫会议失败: %w", err)
	}

	s.logger.Info("安全呼叫会议成功",
		zap.String("conference_number", req.ConferenceNumber),
		zap.String("terminal_number", req.TerminalNumber),
		zap.String("call_id", resp.CallID),
	)

	return nil
}

// CallConference 呼叫会议
func (s *HuaweiService) CallConference(ctx context.Context, req *CallConferenceRequest) (*CallConferenceResponse, error) {
	if err := s.client.EnsureLogin(ctx); err != nil {
		return nil, err
	}
	return s.client.CallConference(ctx, req)
}

// HangupConference 挂断会议
func (s *HuaweiService) HangupConference(ctx context.Context, req *HangupConferenceRequest) error {
	s.logger.Info("挂断会议", zap.String("terminal_number", req.TerminalNumber))
	return s.client.HangupConference(ctx, req)
}

// GetConferenceInfo 获取会议信息
func (s *HuaweiService) GetConferenceInfo(ctx context.Context, conferenceNumber string) (*ConferenceInfo, error) {
	if err := s.client.EnsureLogin(ctx); err != nil {
		return nil, err
	}
	return s.client.GetConferenceInfo(ctx, conferenceNumber)
}

// GetTerminalStatus 获取终端状态
func (s *HuaweiService) GetTerminalStatus(ctx context.Context, terminalNumber string) (*TerminalStatus, error) {
	if err := s.client.EnsureLogin(ctx); err != nil {
		return nil, err
	}
	return s.client.GetTerminalStatus(ctx, terminalNumber)
}

// WaitForConnection 等待终端连接到会议
func (s *HuaweiService) WaitForConnection(ctx context.Context, conferenceNumber, terminalNumber string, timeout time.Duration) error {
	s.logger.Info("等待终端连接到会议",
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
			info, err := s.GetConferenceInfo(ctx, conferenceNumber)
			if err != nil {
				s.logger.Warn("获取会议信息失败，继续等待", zap.Error(err))
				continue
			}

			// 检查终端是否已加入
			for _, attendee := range info.Attendees {
				if attendee.TerminalNumber == terminalNumber && attendee.Status == "connected" {
					s.logger.Info("终端已连接到会议",
						zap.String("terminal_number", terminalNumber),
						zap.String("conference_number", conferenceNumber),
					)
					return nil
				}
			}

			s.logger.Debug("终端尚未连接，继续等待",
				zap.String("terminal_number", terminalNumber),
				zap.Int("attendees_count", info.AttendeesCount),
			)
		}
	}
}

// GetRTSPStreams 获取会议的RTSP流
func (s *HuaweiService) GetRTSPStreams(ctx context.Context, conferenceNumber string) ([]RTSPStream, error) {
	info, err := s.GetConferenceInfo(ctx, conferenceNumber)
	if err != nil {
		return nil, err
	}

	if len(info.RTSPStreams) == 0 {
		return nil, fmt.Errorf("会议没有可用的RTSP流")
	}

	return info.RTSPStreams, nil
}

// IsTerminalIdle 检查终端是否空闲
func (s *HuaweiService) IsTerminalIdle(ctx context.Context, terminalNumber string) (bool, error) {
	status, err := s.GetTerminalStatus(ctx, terminalNumber)
	if err != nil {
		return false, err
	}
	return status.Status == "idle", nil
}

// ValidateTerminal 验证终端是否可用
func (s *HuaweiService) ValidateTerminal(ctx context.Context, terminalNumber string) error {
	status, err := s.GetTerminalStatus(ctx, terminalNumber)
	if err != nil {
		return err
	}

	switch status.Status {
	case "offline":
		return NewHuaweiError(ErrCodeTerminalOffline, nil)
	case "in_call":
		return NewHuaweiError(ErrCodeTerminalInCall, nil)
	case "idle":
		return nil
	default:
		return fmt.Errorf("终端状态未知: %s", status.Status)
	}
}
