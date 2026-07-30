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
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Config 华为终端API配置
type Config struct {
	Server             string        // 终端设备IP地址
	Port               int           // 终端设备端口（通常是443）
	Username           string        // 登录用户名
	Password           string        // 登录密码
	APITimeout         time.Duration // API超时时间
	SessionTimeout     time.Duration // 会话超时时间
	KeepAliveInterval  time.Duration // 保活间隔（必须小于60秒）
	InsecureSkipVerify bool          // 是否跳过证书验证
	MinTLSVersion      uint16        // 最小TLS版本
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
	Success   int            `json:"success"`
	Data      string         `json:"data,omitempty"`      // 注意：华为API返回的是JSON字符串，不是对象
	Exception ExceptionInfo  `json:"exception,omitempty"` // 某些API使用exception
	Error     ExceptionInfo  `json:"error,omitempty"`     // 某些API使用error
	Cookies   []*http.Cookie `json:"-"`                   // 从响应中提取的Cookies
}

// ExceptionInfo 异常信息
type ExceptionInfo struct {
	ID     int   `json:"id"`
	Code   int   `json:"code,omitempty"`   // 错误码
	Params []int `json:"params,omitempty"` // 错误参数
}

// GetErrorID 获取错误ID（兼容exception和error字段）
func (e *ExceptionInfo) GetErrorID() int {
	if e.ID != 0 {
		return e.ID
	}
	return e.Code
}

// SessionIDResponse 获取会话ID响应（解析data字段后的结构）
type SessionIDResponse struct {
	AcSessionID string `json:"acSessionId"` // 注意：华为API使用驼峰命名
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

// SiteInfoRequest 站点信息（匹配华为TE40 API格式）
type SiteInfoRequest struct {
	UwID      int     `json:"uwID"`      // 站点ID
	SzName    string  `json:"szName"`    // 站点名称
	SzPName   string  `json:"szPName"`   // 父名称
	UcType    int     `json:"ucType"`    // 类型（8=会议号）
	BIsLdap   int     `json:"bIsLdap"`   // 是否LDAP
	UcDevice  int     `json:"ucDevice"`  // 设备类型
	UcOnline  int     `json:"ucOnline"`  // 在线状态
	UwSortPos int     `json:"uwSortPos"` // 排序位置
	StSIP     SIPInfo `json:"stSIP"`     // SIP信息
}

// SIPInfo SIP信息
type SIPInfo struct {
	UcBaudRate int    `json:"ucBaudRate"` // 带宽
	SzAlias    string `json:"szAlias"`    // 别名
	SzIP       string `json:"szIP"`       // IP地址
	SzUri      string `json:"szUri"`      // URI
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
	Status      string       `json:"status"`       // 会议状态
	Name        string       `json:"name"`         // 会议名称
	Number      string       `json:"number"`       // 会议号
	SiteList    []SiteInfo   `json:"site_list"`    // 站点列表
	StartTime   string       `json:"start_time"`   // 开始时间
	EndTime     string       `json:"end_time"`     // 结束时间
	Duration    int          `json:"duration"`     // 持续时间
	IsActive    bool         `json:"is_active"`    // 是否活跃
	IsRecording bool         `json:"is_recording"` // 是否录制
	RTSPStreams []RTSPStream `json:"rtsp_streams"` // RTSP流列表
}

// SiteInfo 站点信息
type SiteInfo struct {
	SiteURI    string `json:"site_uri"`    // 站点URI
	SiteName   string `json:"site_name"`   // 站点名称
	SiteIP     string `json:"site_ip"`     // 站点IP
	SiteStatus int    `json:"site_status"` // 站点状态
	JoinTime   string `json:"join_time"`   // 加入时间
}

// RTSPStream RTSP流信息
type RTSPStream struct {
	Type string `json:"type"` // main, content
	URL  string `json:"url"`
}

// TerminalStatus 终端状态
type TerminalStatus struct {
	TerminalNumber string          `json:"terminal_number"`
	Name           string          `json:"name"`
	Status         string          `json:"status"` // idle, in_call, offline
	IPAddress      string          `json:"ip_address"`
	Version        string          `json:"version"`
	USBDevices     []USBDeviceInfo `json:"usb_devices"`
}

// USBDeviceInfo USB设备信息
type USBDeviceInfo struct {
	Type     string `json:"type"` // camera, audio
	Name     string `json:"name"`
	DeviceID string `json:"device_id"`
	Status   string `json:"status"` // available, busy, error
}

// MailboxState 邮箱状态信息（来自WEB_GetMailboxDataAPI）
type MailboxState struct {
	State struct {
		Sitename  string `json:"sitename"`
		Speaker   int    `json:"speaker"`
		Mic       int    `json:"mic"`
		Gk        int    `json:"gk"`
		Sip       int    `json:"sip"`
		Callstate int    `json:"callstate"` // 通话状态
		Calltype  int    `json:"calltype"`
		Conftype  int    `json:"conftype"`
		IsInConf  int    `json:"isInConf"` // 是否在会议中，1表示在会议中
	} `json:"state"`
}

// HTTPClient HTTP客户端封装
type HTTPClient struct {
	client  *http.Client
	logger  *zap.Logger
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
					MinVersion:         minTLSVersion,    // SEC-003a: 由调用方注入，默认 tls.VersionTLS12
					MaxVersion:         tls.VersionTLS12, // 限制最大版本为 TLS 1.2（华为终端兼容性）
					// SEC-003a: 密码套件——优先 ECDHE 前向保密，保留 RSA-AES 兼容华为老设备；
					// 已剔除基于 3DES 的弱套件（SWEET32 攻击面）。
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
					// 支持更多的曲线以兼容老设备
					CurvePreferences: []tls.CurveID{
						tls.CurveP256,
						tls.CurveP384,
						tls.CurveP521,
					},
				},
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				// 禁用 HTTP/2，强制使用 HTTP/1.1（华为老设备兼容）
				ForceAttemptHTTP2: false,
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
	}
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 合并所有headers
	allHeaders := make(map[string]string)
	for _, h := range headers {
		for k, v := range h {
			allHeaders[k] = v
		}
	}

	// 只有在没有传递任何自定义headers时才添加默认的userType
	// 这是为了匹配华为API的行为：认证请求不发送userType
	if len(allHeaders) == 0 {
		req.Header.Set("userType", "web")
	}

	// 设置自定义headers（包括Cookie）
	for k, v := range allHeaders {
		req.Header.Set(k, v)
	}

	// 调试日志：显示请求详情（不包含敏感数据）
	if c.logger != nil {
		hasCookie := req.Header.Get("Cookie") != ""
		hasUserType := req.Header.Get("userType") != ""
		c.logger.Debug("发送请求",
			zap.String("action_id", actionID),
			zap.Bool("has_body", body != nil),
			zap.Bool("has_cookie", hasCookie),
			zap.Bool("has_userType", hasUserType),
		)
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
			zap.ByteString("response", huaweiSanitizeResponseBody(respBody)),
		)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取Cookies（华为终端使用SessionID cookie进行认证）
	apiResp.Cookies = resp.Cookies()

	return &apiResp, nil
}

// HuaweiClient 华为终端API客户端
type HuaweiClient struct {
	config          *Config
	httpClient      *HTTPClient
	session         *Session
	logger          *zap.Logger
	mu              sync.RWMutex
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

	// 使用正确的API名称：Web_RequestSessionID（注意：首字母大写，没有API后缀）
	resp, err := c.httpClient.Post(ctx, "Web_RequestSessionID", nil)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	// 华为API返回格式：{"success":1,"data":"{\"acSessionId\":\"sK9SGDKmCSuuXWzOeT0vL8OLPTn9rXX\"}"}
	if resp.Success != 1 {
		errorID := resp.Exception.GetErrorID()
		return NewHuaweiError(errorID, fmt.Errorf("获取会话ID失败: 错误码 %d", errorID))
	}

	// 优先从Cookie中提取SessionID（Postman显示Session ID保存在Cookie中）
	var sessionID string
	for _, cookie := range resp.Cookies {
		if cookie.Name == "SessionID" {
			sessionID = cookie.Value
			c.logger.Debug("从Cookie获取到SessionID", zap.String("session_id", sessionID))
			break
		}
	}

	// 如果Cookie中没有，尝试从data字段解析
	if sessionID == "" && resp.Data != "" {
		var sessionResp SessionIDResponse
		if err := json.Unmarshal([]byte(resp.Data), &sessionResp); err != nil {
			return fmt.Errorf("解析会话ID响应失败: %w, data: %s", err, resp.Data)
		}
		sessionID = sessionResp.AcSessionID
		c.logger.Debug("从data字段获取到SessionID", zap.String("session_id", sessionID))
	}

	if sessionID == "" {
		return fmt.Errorf("未能获取到会话ID")
	}

	c.mu.Lock()
	c.session = &Session{
		ID:        sessionID,
		ExpiresAt: time.Now().Add(c.config.SessionTimeout),
		IsActive:  true,
	}
	c.mu.Unlock()

	c.logger.Debug("获取会话ID成功",
		zap.String("session_id", sessionID),
	)

	return nil
}

// Authenticate 用户认证
func (c *HuaweiClient) Authenticate(ctx context.Context) error {
	c.logger.Info("华为终端用户认证",
		zap.String("username", c.config.Username),
	)

	// 检查是否有会话ID
	currentSessionID := c.getSessionID()
	c.logger.Debug("当前会话ID", zap.String("session_id", currentSessionID))

	// 华为API需要使用小写的user和password
	// 使用结构体确保字段顺序（user在前，password在后）
	type authRequest struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	reqBody := authRequest{
		User:     c.config.Username,
		Password: c.config.Password,
	}

	// 使用Cookie头部传递SessionID
	headers := make(map[string]string)
	if currentSessionID != "" {
		headers["Cookie"] = fmt.Sprintf("SessionID=%s", currentSessionID)
		c.logger.Debug("设置Cookie头部", zap.String("cookie", headers["Cookie"]))
	} else {
		c.logger.Warn("会话ID为空，可能无法完成认证")
	}

	resp, err := c.httpClient.Post(ctx, "WEB_RequestCertificateAPI", reqBody, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	// 华为API返回格式：{"success":1,"data":""} 或 {"success":0,"error":{"id":100666995}}
	if resp.Success != 1 {
		errorID := resp.Error.GetErrorID()
		if errorID == 0 {
			errorID = resp.Exception.GetErrorID()
		}
		return NewHuaweiError(errorID, fmt.Errorf("认证失败: 错误码 %d", errorID))
	}

	c.logger.Info("用户认证成功")
	return nil
}

// ChangeSessionID 替换会话ID
func (c *HuaweiClient) ChangeSessionID(ctx context.Context) error {
	c.logger.Debug("替换会话ID")

	headers := map[string]string{
		"Cookie": fmt.Sprintf("SessionID=%s", c.getSessionID()),
	}

	resp, err := c.httpClient.Post(ctx, "WEB_ChangeSessionID", nil, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	if resp.Success != 1 {
		errorID := resp.Error.GetErrorID()
		if errorID == 0 {
			errorID = resp.Exception.GetErrorID()
		}
		return NewHuaweiError(errorID, fmt.Errorf("替换会话ID失败: 错误码 %d", errorID))
	}

	// 解析新的会话ID并更新
	if resp.Data != "" {
		var sessionResp SessionIDResponse
		if err := json.Unmarshal([]byte(resp.Data), &sessionResp); err == nil {
			if sessionResp.AcSessionID != "" {
				c.mu.Lock()
				c.session.ID = sessionResp.AcSessionID
				c.mu.Unlock()
				c.logger.Debug("更新会话ID成功", zap.String("new_session_id", sessionResp.AcSessionID))
			}
		}
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
		defer func() {
			if r := recover(); r != nil {
				c.logger.Error("keep-alive goroutine panicked",
					zap.Any("recover", r), zap.Stack("stack"))
			}
		}()
		defer ticker.Stop()
		for {
			select {
			case <-keepAliveCtx.Done():
				c.logger.Debug("停止会话保活")
				return
			case <-ticker.C:
				if _, err := c.GetMailboxData(keepAliveCtx); err != nil {
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

// GetMailboxData 获取邮箱数据用于保活，返回终端状态
func (c *HuaweiClient) GetMailboxData(ctx context.Context) (*MailboxState, error) {
	if !c.hasSession() {
		return nil, NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	headers := map[string]string{
		"Cookie": fmt.Sprintf("SessionID=%s", c.getSessionID()),
	}

	resp, err := c.httpClient.Post(ctx, "WEB_GetMailboxDataAPI", nil, headers)
	if err != nil {
		return nil, err
	}

	// 更新过期时间
	c.mu.Lock()
	if c.session != nil {
		c.session.ExpiresAt = time.Now().Add(c.config.SessionTimeout)
	}
	c.mu.Unlock()

	// 解析响应数据获取终端状态
	var mailboxState MailboxState
	if resp != nil && resp.Data != "" {
		// data字段是一个JSON字符串，需要再次解析
		var dataWrapper struct {
			State json.RawMessage `json:"state"`
		}
		if err := json.Unmarshal([]byte(resp.Data), &dataWrapper); err == nil {
			if err := json.Unmarshal(dataWrapper.State, &mailboxState.State); err == nil {
				c.logger.Debug("会话保活成功",
					zap.Int("isInConf", mailboxState.State.IsInConf),
					zap.Int("callstate", mailboxState.State.Callstate),
				)
				return &mailboxState, nil
			}
		}
	}

	c.logger.Debug("会话保活成功")
	return &mailboxState, nil
}

// IsInConference 检查终端是否在会议中
func (c *HuaweiClient) IsInConference(ctx context.Context) (bool, error) {
	state, err := c.GetMailboxData(ctx)
	if err != nil {
		return false, err
	}
	// isInConf=1 表示在会议中，callstate=2 表示通话中
	return state.State.IsInConf == 1 && state.State.Callstate == 2, nil
}

// CallConference 呼叫会议
func (c *HuaweiClient) CallConference(ctx context.Context, conferenceNumber string) error {
	if !c.hasSession() {
		return NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	c.logger.Info("呼叫会议", zap.String("conference_number", conferenceNumber))

	// 构造呼叫会议请求（匹配Python脚本格式）
	req := &CallSiteRequest{
		BIsLdapCall:  0,
		BIsVideoCall: 0, // 注意：Python脚本使用0（可能表示语音+H.239内容）
		UcEnableH239: 1, // 启用H.239
		StSiteInfo: SiteInfoRequest{
			UwID:      0,
			SzName:    conferenceNumber, // 使用会议号作为名称
			SzPName:   "",
			UcType:    8, // 8表示会议号类型
			BIsLdap:   0,
			UcDevice:  0,
			UcOnline:  0,
			UwSortPos: 0,
			StSIP: SIPInfo{
				UcBaudRate: 1920,
				SzAlias:    "",
				SzIP:       "",
				SzUri:      "",
			},
		},
	}

	headers := map[string]string{
		"Cookie": fmt.Sprintf("SessionID=%s", c.getSessionID()),
	}

	resp, err := c.httpClient.Post(ctx, "WEB_CallSiteAPI", req, headers)
	if err != nil {
		return NewHuaweiError(ErrCodeNetworkError, err)
	}

	// 检查是否成功
	if resp.Success == 1 {
		c.logger.Info("呼叫会议成功")
		return nil
	}

	// 获取错误ID（兼容error和exception字段）
	errorID := resp.Error.GetErrorID()
	if errorID == 0 {
		errorID = resp.Exception.GetErrorID()
	}

	// 检查特殊错误码（可能仍然是正常状态）
	if errorID == 100665897 {
		// 呼叫请求已发出，正在等待响应（正常状态）
		c.logger.Info("呼叫请求已发出，正在等待响应")
		return nil
	}

	return NewHuaweiError(errorID, fmt.Errorf("呼叫会议失败: 错误码 %d", errorID))
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
		"Cookie": fmt.Sprintf("SessionID=%s", c.getSessionID()),
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

	// 华为老设备没有获取会议列表的API
	// 返回默认的空会议信息，表示当前没有会议
	return &ConferenceInfo{
		Status:      "",
		Name:        "",
		Number:      "",
		SiteList:    []SiteInfo{},
		StartTime:   "",
		EndTime:     "",
		Duration:    0,
		IsActive:    false,
		IsRecording: false,
		RTSPStreams: []RTSPStream{},
	}, nil
}

// GetTerminalStatus 获取终端状态
// 注意：华为老设备没有获取终端状态的API，这里返回默认的空闲状态
// 实际的终端状态会在呼叫会议时通过结果来判断
func (c *HuaweiClient) GetTerminalStatus(ctx context.Context, terminalNumber string) (*TerminalStatus, error) {
	if !c.hasSession() {
		return nil, NewHuaweiError(ErrCodeSessionInvalid, nil)
	}

	// 返回默认的空闲状态
	return &TerminalStatus{
		TerminalNumber: terminalNumber,
		Name:           "华为终端",
		IPAddress:      c.config.Server,
		Status:         "idle", // 假设空闲，实际状态会在呼叫时检查
		Version:        "",
		USBDevices:     []USBDeviceInfo{},
	}, nil
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

// huaweiSanitizeResponseBody masks credentials and certificate payloads before logging.
func huaweiSanitizeResponseBody(body []byte) []byte {
	var value interface{}
	if err := json.Unmarshal(body, &value); err != nil {
		return []byte("[unparseable response omitted]")
	}
	var sanitize func(interface{})
	sanitize = func(current interface{}) {
		switch typed := current.(type) {
		case map[string]interface{}:
			for key, child := range typed {
				lower := strings.ToLower(key)
				if lower == "username" || lower == "password" || lower == "certbase64string" || strings.Contains(lower, "certificate") {
					typed[key] = "***"
				} else {
					sanitize(child)
				}
			}
		case []interface{}:
			for _, child := range typed {
				sanitize(child)
			}
		}
	}
	sanitize(value)
	masked, err := json.Marshal(value)
	if err != nil {
		return []byte("[response omitted]")
	}
	return masked
}
