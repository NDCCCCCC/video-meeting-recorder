package huawei

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Config 华为终端API配置
type Config struct {
	Server            string        // 终端设备IP地址
	Port              int           // 终端设备端口（通常是443）
	Username          string        // 登录用户名
	Password          string        // 登录密码
	APITimeout        time.Duration // API超时时间
	SessionTimeout    time.Duration // 会话超时时间
	KeepAliveInterval time.Duration // 保活间隔（必须小于60秒）
	InsecureSkipVerify bool         // 是否跳过证书验证
	MinTLSVersion     uint16        // 最小TLS版本
}

// Session 会话信息
type Session struct {
	ID        string
	ExpiresAt time.Time
	IsActive  bool
	mu        sync.RWMutex
}

// IsExpired 检查会话是否过期
func (s *Session) IsExpired() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.IsActive || time.Now().After(s.ExpiresAt)
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
	s.IsActive = true
}

// SetActive 设置会话活跃状态
func (s *Session) SetActive(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsActive = active
}

// APIResponse 统一API响应格式（华为终端实际返回格式）
type APIResponse struct {
	Success   int             `json:"success"`
	Data      string          `json:"data,omitempty"`      // 注意：华为API返回的是JSON字符串，不是对象
	Exception ExceptionInfo   `json:"exception,omitempty"`
}

// ExceptionInfo 异常信息
type ExceptionInfo struct {
	ID int `json:"id"`
}

// SessionIDResponse 获取会话ID响应（解析data字段后的结构）
type SessionIDResponse struct {
	AcSessionID string `json:"acSessionId"`  // 注意：华为API使用驼峰命名
	SzTermType  string `json:"szTermType"`
}

// AuthenticateResponse 认证响应（解析data字段后的结构）
type AuthenticateResponse struct {
	AcSessionID string `json:"acSessionId"`
	Result      int    `json:"result"`
}

// CallSiteRequest 呼叫会议请求
type CallSiteRequest struct {
	BIsLdapCall  int             `json:"bIsLdapCall"`  // 是否LDAP呼叫
	BIsVideoCall int             `json:"bIsVideoCall"` // 是否视频呼叫
	UcEnableH239 int             `json:"ucEnableH239"` // 是否启用H.239
	StSiteInfo   SiteInfoRequest `json:"stSiteInfo"`   // 站点信息
}

// SiteInfoRequest 站点信息
type SiteInfoRequest struct {
	SiteName     string `json:"site_name"`     // 站点名称
	SiteURI      string `json:"site_uri"`      // 站点URI（会议号）
	SitePassword string `json:"site_password"` // 站点密码
	SiteIP       string `json:"site_ip"`       // 站点IP
}

// CallSiteResponse 呼叫会议响应
type CallSiteResponse struct {
	Result int    `json:"result"`
	Msg    string `json:"msg"`
}

// HangupCallRequest 挂断呼叫请求
type HangupCallRequest struct {
	StHangupType int `json:"stHangupType"` // 挂断类型
}

// HangupCallResponse 挂断呼叫响应
type HangupCallResponse struct {
	Result int    `json:"result"`
	Msg    string `json:"msg"`
}

// ConferenceInfo 会议信息
type ConferenceInfo struct {
	Status      string      `json:"status"`       // 会议状态
	Name        string      `json:"name"`         // 会议名称
	Number      string      `json:"number"`       // 会议号
	SiteList    []SiteInfo  `json:"site_list"`    // 站点列表
	StartTime   string      `json:"start_time"`   // 开始时间
	EndTime     string      `json:"end_time"`     // 结束时间
	Duration    int         `json:"duration"`     // 持续时间
	IsActive    bool        `json:"is_active"`    // 是否活跃
	IsRecording bool        `json:"is_recording"` // 是否录制
	RTSPStreams []RTSPStream `json:"rtsp_streams"` // RTSP流列表
}

// SiteInfo 站点信息
type SiteInfo struct {
	SiteURI      string `json:"site_uri"`       // 站点URI
	SiteName     string `json:"site_name"`      // 站点名称
	SiteIP       string `json:"site_ip"`        // 站点IP
	SiteStatus   int    `json:"site_status"`    // 站点状态
	JoinTime     string `json:"join_time"`      // 加入时间
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
	baseURL *url.URL
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(server string, port int, timeout time.Duration, insecureSkipVerify bool, minTLSVersion uint16, logger *zap.Logger) *HTTPClient {
	// 构建基础URL
	scheme := "https"
	if port == 80 {
		scheme = "http"
	}

	baseURL := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", server, port),
		Path:   "/action.cgi",
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: insecureSkipVerify,
					MinVersion:         minTLSVersion,
					// 华为终端使用的密码套件
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
				},
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:  logger,
		baseURL: baseURL,
	}
}

// SetLogger 设置日志记录器
func (c *HTTPClient) SetLogger(logger *zap.Logger) {
	c.logger = logger
}

// buildURL 构建API URL
func (c *HTTPClient) buildURL(actionID string) string {
	u := *c.baseURL
	query := u.Query()
	query.Set("ActionID", actionID)
	u.RawQuery = query.Encode()
	return u.String()
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, actionID string, body interface{}, headers ...map[string]string) (*APIResponse, error) {
	url := c.buildURL(actionID)

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
		c.logger.Debug("发送请求",
			zap.String("action_id", actionID),
			zap.String("url", url),
			zap.String("body", string(jsonBody)),
		)
	} else {
		c.logger.Debug("发送请求",
			zap.String("action_id", actionID),
			zap.String("url", url),
		)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// 华为API需要userType header
	req.Header.Set("userType", "web")
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}

	// 添加会话ID头（如果存在）
	if len(headers) > 0 {
		for _, h := range headers {
			if sessionID, ok := h["X-Session-Id"]; ok {
				req.Header.Set("X-Session-Id", sessionID)
			}
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
		c.logger.Debug("收到响应",
			zap.String("action_id", actionID),
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

// HuaweiClient 华为终端API客户端
type HuaweiClient struct {
	config         *Config
	httpClient     *HTTPClient
	session        *Session
	logger         *zap.Logger
	mu             sync.RWMutex
	cancelKeepAlive context.CancelFunc
}

// NewHuaweiClient 创建华为终端API客户端
func NewHuaweiClient(config *Config, logger *zap.Logger) *HuaweiClient {
	return &HuaweiClient{
		config:     config,
		httpClient: NewHTTPClient(config.Server, config.Port, config.APITimeout, config.InsecureSkipVerify, config.MinTLSVersion, logger),
		logger:     logger,
	}
}

// InitializeAndStartKeepAlive 初始化并启动会话保活
// 功能：完成登录并启动后台保活机制
func (c *HuaweiClient) InitializeAndStartKeepAlive(ctx context.Context) error {
	c.logger.Info("初始化华为终端客户端",
		zap.String("server", c.config.Server),
		zap.Int("port", c.config.Port),
		zap.String("username", c.config.Username),
	)

	// 1. 获取会话ID
	if err := c.GetSessionID(ctx); err != nil {
		return fmt.Errorf("获取会话ID失败: %w", err)
	}

	// 2. 用户认证
	if err := c.Authenticate(ctx); err != nil {
		return fmt.Errorf("用户认证失败: %w", err)
	}

	// 3. 替换会话ID
	if err := c.ChangeSessionID(ctx); err != nil {
		return fmt.Errorf("替换会话ID失败: %w", err)
	}

	// 4. 启动后台保活机制
	c.StartKeepAlive(ctx)

	c.logger.Info("华为终端客户端初始化成功",
		zap.String("session_id", c.session.GetID()),
	)

	return nil
}

// GetSessionID 获取会话ID
func (c *HuaweiClient) GetSessionID(ctx context.Context) error {
	c.logger.Debug("获取会话ID")

	resp, err := c.httpClient.Post(ctx, "Web_RequestSessionID", nil)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	// 华为API返回格式：{"success":1,"data":"{\"acSessionId\":\"\",\"szTermType\":\"...\"}"}
	if resp.Success != 1 {
		return NewHuaweiError(resp.Exception.ID, fmt.Errorf("获取会话ID失败: 错误码 %d", resp.Exception.ID))
	}

	// 解析data字段（它是一个JSON字符串）
	var sessionResp SessionIDResponse
	if resp.Data != "" {
		if err := json.Unmarshal([]byte(resp.Data), &sessionResp); err != nil {
			return fmt.Errorf("解析会话ID响应失败: %w, data: %s", err, resp.Data)
		}
	}

	c.mu.Lock()
	c.session = &Session{
		ID:        sessionResp.AcSessionID, // 使用正确的字段名
		ExpiresAt: time.Now().Add(c.config.SessionTimeout),
		IsActive:  true,
	}
	c.mu.Unlock()

	c.logger.Debug("获取会话ID成功",
		zap.String("session_id", sessionResp.AcSessionID),
		zap.String("term_type", sessionResp.SzTermType),
	)
	return nil
}

// Authenticate 用户认证
func (c *HuaweiClient) Authenticate(ctx context.Context) error {
	c.logger.Info("华为终端用户认证",
		zap.String("username", c.config.Username),
	)

	// 华为API需要使用大写的User和Password
	reqBody := map[string]string{
		"User":     c.config.Username,
		"Password": c.config.Password,
	}

	// 如果有会话ID，添加到请求中
	headers := make(map[string]string)
	if sessionID := c.getSessionID(); sessionID != "" {
		headers["X-Session-Id"] = sessionID
	}

	resp, err := c.httpClient.Post(ctx, "WEB_RequestCertificateAPI", reqBody, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	// 华为API返回格式：{"success":1,"data":"{\"acSessionId\":\"...\"}"} 或 {"success":0,"exception":{"id":3}}
	if resp.Success != 1 {
		return NewHuaweiError(resp.Exception.ID, fmt.Errorf("认证失败: 错误码 %d", resp.Exception.ID))
	}

	c.logger.Info("用户认证成功")
	return nil
}

// ChangeSessionID 替换会话ID
func (c *HuaweiClient) ChangeSessionID(ctx context.Context) error {
	c.logger.Debug("替换会话ID")

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Post(ctx, "WEB_ChangeSessionID", nil, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Success != 1 {
		return NewHuaweiError(resp.Exception.ID, fmt.Errorf("替换会话ID失败: 错误码 %d", resp.Exception.ID))
	}

	c.logger.Debug("替换会话ID成功")
	return nil
}

// getSessionID 获取会话ID
func (c *HuaweiClient) getSessionID() string {
	if c.session == nil {
		return ""
	}
	return c.session.ID
}

// hasSession 检查是否有有效会话
func (c *HuaweiClient) hasSession() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.session == nil {
		return false
	}
	return !c.session.IsExpired()
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
				if err := c.GetMailboxData(keepAliveCtx); err != nil {
					c.logger.Error("会话保活失败", zap.Error(err))
					// 保活失败，尝试重新初始化
					if err := c.InitializeAndStartKeepAlive(keepAliveCtx); err != nil {
						c.logger.Error("重新初始化失败", zap.Error(err))
					}
				}
			}
		}
	}()

	c.logger.Info("启动会话自动保活", zap.Duration("interval", c.config.KeepAliveInterval))
}

// GetMailboxData 获取邮箱数据用于保活
func (c *HuaweiClient) GetMailboxData(ctx context.Context) error {
	if !c.hasSession() {
		return NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	_, err := c.httpClient.Post(ctx, "WEB_GetMailboxDataAPI", nil, headers)
	if err != nil {
		return err
	}

	// 更新过期时间
	c.mu.Lock()
	if c.session != nil {
		c.session.ExpiresAt = time.Now().Add(c.config.SessionTimeout)
	}
	c.mu.Unlock()

	c.logger.Debug("会话保活成功")
	return nil
}

// CallConference 呼叫会议
func (c *HuaweiClient) CallConference(ctx context.Context, conferenceNumber string) error {
	if !c.hasSession() {
		return NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	c.logger.Info("呼叫会议", zap.String("conference_number", conferenceNumber))

	req := &CallSiteRequest{
		BIsLdapCall:  0,
		BIsVideoCall: 1, // 默认视频呼叫
		UcEnableH239: 0,
		StSiteInfo: SiteInfoRequest{
			SiteURI: conferenceNumber,
			SiteName: "录制会议",
		},
	}

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Post(ctx, "WEB_CallSiteAPI", req, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	// 华为API: success=1表示成功，success=0表示失败
	if resp.Success == 1 {
		c.logger.Info("呼叫会议成功")
		return nil
	}

	// 检查特殊错误码（可能仍然是正常状态）
	if resp.Exception.ID == 100665897 {
		// 呼叫请求已发出，正在等待响应（正常状态）
		c.logger.Info("呼叫请求已发出，正在等待响应")
		return nil
	}

	return NewHuaweiError(resp.Exception.ID, fmt.Errorf("呼叫会议失败: 错误码 %d", resp.Exception.ID))
}

// HangupCall 挂断呼叫
func (c *HuaweiClient) HangupCall(ctx context.Context) error {
	if !c.hasSession() {
		return NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	c.logger.Info("挂断呼叫")

	req := &HangupCallRequest{
		StHangupType: 0,
	}

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Post(ctx, "WEB_HangupCallAPI", req, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	// 华为API: success=1表示成功
	if resp.Success == 1 {
		c.logger.Info("挂断呼叫成功")
		return nil
	}

	// 检查特殊错误码（可能仍是正常状态）
	if resp.Exception.ID == 100666794 || resp.Exception.ID == 100666777 || resp.Exception.ID == 100666767 {
		// 呼叫已正常结束、没有进行中的呼叫、未进入会议（正常状态）
		c.logger.Info("挂断呼叫成功（特殊状态）", zap.Int("code", resp.Exception.ID))
		return nil
	}

	return NewHuaweiError(resp.Exception.ID, fmt.Errorf("挂断呼叫失败: 错误码 %d", resp.Exception.ID))
}

// GetConferenceInfo 获取会议信息
func (c *HuaweiClient) GetConferenceInfo(ctx context.Context) (*ConferenceInfo, error) {
	if !c.hasSession() {
		return nil, NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	headers := map[string]string{
		"X-Session-Id": c.getSessionID(),
	}

	resp, err := c.httpClient.Post(ctx, "WEB_InitSiteListDataAPI", nil, headers)
	if err != nil {
		return nil, NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Success != 1 {
		return nil, NewHuaweiError(resp.Exception.ID, fmt.Errorf("获取会议信息失败: 错误码 %d", resp.Exception.ID))
	}

	// 解析响应 - data字段是JSON字符串，需要先解析
	var info ConferenceInfo
	if resp.Data != "" {
		// 需要添加适当的延迟处理
		time.Sleep(500 * time.Millisecond)
		if err := json.Unmarshal([]byte(resp.Data), &info); err != nil {
			return nil, fmt.Errorf("解析会议信息失败: %w, data: %s", err, resp.Data)
		}
	}

	return &info, nil
}

// GetTerminalStatus 获取终端状态
func (c *HuaweiClient) GetTerminalStatus(ctx context.Context, terminalNumber string) (*TerminalStatus, error) {
	if !c.hasSession() {
		return nil, NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	// 通过会议信息推断终端状态
	info, err := c.GetConferenceInfo(ctx)
	if err != nil {
		return nil, err
	}

	status := &TerminalStatus{
		TerminalNumber: terminalNumber,
		Name:           "华为终端",
		IPAddress:      c.config.Server,
		Status:         "idle",
	}

	// 检查是否在会议中
	for _, site := range info.SiteList {
		if site.SiteURI == terminalNumber {
			if site.SiteStatus == 1 {
				status.Status = "in_call"
			}
			break
		}
	}

	return status, nil
}

// HealthCheck 健康检查
func (c *HuaweiClient) HealthCheck() error {
	if !c.hasSession() {
		return fmt.Errorf("会话未初始化")
	}
	return nil
}

// GetRTSPStreamURL 获取RTSP流地址
func (c *HuaweiClient) GetRTSPStreamURL(conferenceNumber string) (string, error) {
	// RTSP流地址通常由终端设备提供
	// 格式：rtsp://{server}:554/stream
	return fmt.Sprintf("rtsp://%s:554/stream", c.config.Server), nil
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

	if c.session != nil {
		c.session.SetActive(false)
	}

	c.logger.Info("华为终端登出")
	return nil
}

// EnsureLogin 确保已登录（用于兼容）
func (c *HuaweiClient) EnsureLogin(ctx context.Context) error {
	if c.hasSession() {
		return nil
	}
	return c.InitializeAndStartKeepAlive(ctx)
}

// CallConferenceRequest 呼叫会议请求（兼容旧接口）
type CallConferenceRequest struct {
	ConferenceNumber string
	TerminalNumber   string
	Password         string
	Subject          string
}

// CallConferenceResponse 呼叫会议响应（兼容旧接口）
type CallConferenceResponse struct {
	CallID string
	Status string
}

// HangupConferenceRequest 挂断会议请求（兼容旧接口）
type HangupConferenceRequest struct {
	CallID         string
	TerminalNumber string
}
