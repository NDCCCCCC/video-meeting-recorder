package huawei

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Config 华为API配置
type Config struct {
	Server            string
	Port             int
	Username         string
	Password         string
	APIBase          string
	APITimeout       time.Duration
	SessionTimeout   time.Duration
	KeepAliveInterval time.Duration
	InsecureSkipVerify bool
	MinTLSVersion    uint16
}

// Session 会话信息
type Session struct {
	ID        string
	ExpiresAt time.Time
	mu        sync.RWMutex
}

// IsExpired 检查会话是否过期
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Now().After(s.ExpiresAt)
}

// GetID 获取会话ID
func (s *Session) GetID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ID
}

// Extend 延长会话有效期
func (s *Session) Extend(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ExpiresAt = time.Now().Add(duration)
}

// APIResponse 统一API响应格式
type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	SessionID string `json:"session_id"`
	ExpiresIn int    `json:"expires_in"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CallConferenceRequest 呼叫会议请求
type CallConferenceRequest struct {
	ConferenceNumber string `json:"conference_number"`
	TerminalNumber   string `json:"terminal_number"`
	Password         string `json:"password"`
	Subject          string `json:"subject"`
}

// CallConferenceResponse 呼叫会议响应
type CallConferenceResponse struct {
	CallID string `json:"call_id"`
	Status string `json:"status"`
}

// HangupConferenceRequest 挂断会议请求
type HangupConferenceRequest struct {
	CallID          string `json:"call_id"`
	TerminalNumber  string `json:"terminal_number"`
}

// ConferenceInfo 会议信息
type ConferenceInfo struct {
	ConferenceNumber string         `json:"conference_number"`
	Subject          string         `json:"subject"`
	Status           string         `json:"status"`
	StartTime        string         `json:"start_time"`
	EndTime          string         `json:"end_time"`
	AttendeesCount   int            `json:"attendees_count"`
	Attendees        []AttendeeInfo `json:"attendees"`
	RTSPStreams      []RTSPStream   `json:"rtsp_streams"`
}

// AttendeeInfo 参会者信息
type AttendeeInfo struct {
	TerminalNumber string `json:"terminal_number"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	JoinTime       string `json:"join_time"`
}

// RTSPStream RTSP流信息
type RTSPStream struct {
	Type string `json:"type"` // main, content
	URL  string `json:"url"`
}

// TerminalStatus 终端状态
type TerminalStatus struct {
	TerminalNumber string        `json:"terminal_number"`
	Name           string        `json:"name"`
	Status         string        `json:"status"` // idle, in_call, offline
	IPAddress      string        `json:"ip_address"`
	Version        string        `json:"version"`
	USBDevices     []USBDeviceInfo `json:"usb_devices"`
}

// USBDeviceInfo USB设备信息
type USBDeviceInfo struct {
	Type     string `json:"type"`     // camera, audio
	Name     string `json:"name"`
	DeviceID string `json:"device_id"`
	Status   string `json:"status"`   // available, busy, error
}

// HTTPClient HTTP客户端封装
type HTTPClient struct {
	client *http.Client
	logger *zap.Logger
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(timeout time.Duration, insecureSkipVerify bool, minTLSVersion uint16, logger *zap.Logger) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecureSkipVerify,
					MinVersion:         minTLSVersion,
				},
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

// SetLogger 设置日志记录器
func (c *HTTPClient) SetLogger(logger *zap.Logger) {
	c.logger = logger
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, url string, body interface{}, headers ...map[string]string) (*APIResponse, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("HTTP请求",
			zap.String("url", url),
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(respBody)),
		)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &apiResp, nil
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, url string, headers ...map[string]string) (*APIResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &apiResp, nil
}

// HuaweiClient 华为API客户端
type HuaweiClient struct {
	config     *Config
	httpClient *HTTPClient
	session    *Session
	logger     *zap.Logger
	mu         sync.RWMutex
	cancelKeepAlive context.CancelFunc
}

// NewHuaweiClient 创建华为API客户端
func NewHuaweiClient(config *Config, logger *zap.Logger) *HuaweiClient {
	return &HuaweiClient{
		config:     config,
		httpClient: NewHTTPClient(config.APITimeout, config.InsecureSkipVerify, config.MinTLSVersion, logger),
		logger:     logger,
	}
}

// Login 登录获取会话
func (c *HuaweiClient) Login(ctx context.Context) error {
	c.logger.Info("华为API登录",
		zap.String("server", c.config.Server),
		zap.String("username", c.config.Username),
	)

	url := fmt.Sprintf("%s/sessions", c.config.APIBase)

	req := &LoginRequest{
		Username: c.config.Username,
		Password: c.config.Password,
	}

	resp, err := c.httpClient.Post(ctx, url, req)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Code != 0 {
		return NewHuaweiError(resp.Code, fmt.Errorf("登录失败: %s", resp.Msg))
	}

	// 解析响应数据
	dataBytes, _ := json.Marshal(resp.Data)
	var loginResp LoginResponse
	if err := json.Unmarshal(dataBytes, &loginResp); err != nil {
		return fmt.Errorf("解析登录响应失败: %w", err)
	}

	c.mu.Lock()
	c.session = &Session{
		ID:        loginResp.SessionID,
		ExpiresAt: time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second),
	}
	c.mu.Unlock()

	c.logger.Info("华为API登录成功", zap.String("session_id", loginResp.SessionID))

	return nil
}

// Logout 登出
func (c *HuaweiClient) Logout(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 停止保活
	if c.cancelKeepAlive != nil {
		c.cancelKeepAlive()
		c.cancelKeepAlive = nil
	}

	c.session = nil
	c.logger.Info("华为API登出")
	return nil
}

// getSessionID 获取会话ID
func (c *HuaweiClient) getSessionID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.session == nil {
		return ""
	}
	return c.session.GetID()
}

// hasSession 检查是否有有效会话
func (c *HuaweiClient) hasSession() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.session != nil && !c.session.IsExpired()
}

// KeepAlive 保活会话
func (c *HuaweiClient) KeepAlive(ctx context.Context) error {
	if !c.hasSession() {
		return NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	sessionID := c.getSessionID()
	url := fmt.Sprintf("%s/sessions/%s/keep-alive", c.config.APIBase, sessionID)

	_, err := c.httpClient.Post(ctx, url, nil)
	if err != nil {
		// 保活失败，尝试重新登录
		c.logger.Warn("会话保活失败，尝试重新登录", zap.Error(err))
		return c.Login(ctx)
	}

	// 更新过期时间
	c.mu.Lock()
	if c.session != nil {
		c.session.ExpiresAt = time.Now().Add(c.config.SessionTimeout)
	}
	c.mu.Unlock()

	c.logger.Debug("会话保活成功", zap.String("session_id", sessionID))
	return nil
}

// StartKeepAlive 启动自动保活
func (c *HuaweiClient) StartKeepAlive(ctx context.Context) {
	c.mu.Lock()
	if c.cancelKeepAlive != nil {
		c.mu.Unlock()
		return // 已经在保活中
	}
	keepAliveCtx, cancel := context.WithCancel(ctx)
	c.cancelKeepAlive = cancel
	c.mu.Unlock()

	ticker := time.NewTicker(c.config.KeepAliveInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-keepAliveCtx.Done():
				c.logger.Debug("停止会话保活")
				return
			case <-ticker.C:
				if err := c.KeepAlive(ctx); err != nil {
					c.logger.Error("会话保活失败", zap.Error(err))
				}
			}
		}
	}()

	c.logger.Info("启动会话自动保活", zap.Duration("interval", c.config.KeepAliveInterval))
}

// CallConference 呼叫会议
func (c *HuaweiClient) CallConference(ctx context.Context, req *CallConferenceRequest) (*CallConferenceResponse, error) {
	if !c.hasSession() {
		return nil, NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	url := fmt.Sprintf("%s/confctrl/conferences/call", c.config.APIBase)

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Post(ctx, url, req, headers)
	if err != nil {
		return nil, NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Code != 0 {
		return nil, NewHuaweiError(resp.Code, fmt.Errorf("呼叫会议失败: %s", resp.Msg))
	}

	// 解析响应
	dataBytes, _ := json.Marshal(resp.Data)
	var callResp CallConferenceResponse
	if err := json.Unmarshal(dataBytes, &callResp); err != nil {
		return nil, fmt.Errorf("解析呼叫响应失败: %w", err)
	}

	c.logger.Info("呼叫会议成功",
		zap.String("conference_number", req.ConferenceNumber),
		zap.String("terminal_number", req.TerminalNumber),
		zap.String("call_id", callResp.CallID),
	)

	return &callResp, nil
}

// HangupConference 挂断会议
func (c *HuaweiClient) HangupConference(ctx context.Context, req *HangupConferenceRequest) error {
	if !c.hasSession() {
		return NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	url := fmt.Sprintf("%s/confctrl/conferences/hangup", c.config.APIBase)

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Post(ctx, url, req, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Code != 0 {
		return NewHuaweiError(resp.Code, fmt.Errorf("挂断会议失败: %s", resp.Msg))
	}

	c.logger.Info("挂断会议成功", zap.String("terminal_number", req.TerminalNumber))
	return nil
}

// GetConferenceInfo 获取会议信息
func (c *HuaweiClient) GetConferenceInfo(ctx context.Context, conferenceNumber string) (*ConferenceInfo, error) {
	if !c.hasSession() {
		return nil, NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	url := fmt.Sprintf("%s/confctrl/conferences/%s", c.config.APIBase, conferenceNumber)

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Get(ctx, url, headers)
	if err != nil {
		return nil, NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Code != 0 {
		return nil, NewHuaweiError(resp.Code, fmt.Errorf("获取会议信息失败: %s", resp.Msg))
	}

	// 解析响应
	dataBytes, _ := json.Marshal(resp.Data)
	var info ConferenceInfo
	if err := json.Unmarshal(dataBytes, &info); err != nil {
		return nil, fmt.Errorf("解析会议信息失败: %w", err)
	}

	return &info, nil
}

// GetTerminalStatus 获取终端状态
func (c *HuaweiClient) GetTerminalStatus(ctx context.Context, terminalNumber string) (*TerminalStatus, error) {
	if !c.hasSession() {
		return nil, NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	url := fmt.Sprintf("%s/terminals/%s/status", c.config.APIBase, terminalNumber)

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Get(ctx, url, headers)
	if err != nil {
		return nil, NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Code != 0 {
		return nil, NewHuaweiError(resp.Code, fmt.Errorf("获取终端状态失败: %s", resp.Msg))
	}

	// 解析响应
	dataBytes, _ := json.Marshal(resp.Data)
	var status TerminalStatus
	if err := json.Unmarshal(dataBytes, &status); err != nil {
		return nil, fmt.Errorf("解析终端状态失败: %w", err)
	}

	return &status, nil
}

// EnsureLogin 确保已登录
func (c *HuaweiClient) EnsureLogin(ctx context.Context) error {
	if c.hasSession() {
		return nil
	}
	return c.Login(ctx)
}
