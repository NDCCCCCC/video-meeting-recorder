package huawei

// Huawei API 错误码定义

const (
	// 认证错误 (1xxx)
	ErrCodeAuthFailed     = 1001 // 认证失败
	ErrCodeSessionExpired = 1002 // 会话过期
	ErrCodeSessionInvalid = 1003 // 会话无效

	// 会议控制错误 (2xxx)
	ErrCodeConferenceNotFound   = 2001 // 会议不存在
	ErrCodeConferenceFull       = 2002 // 会议已满
	ErrCodeConferencePassword   = 2003 // 会议密码错误
	ErrConferenceCallFailed     = 2004 // 呼叫会议失败
	ErrConferenceHangupFailed   = 2005 // 挂断会议失败
	ErrCodeConferenceInProgress = 2006 // 会议进行中

	// 终端错误 (3xxx)
	ErrCodeTerminalNotFound = 3001 // 终端不存在
	ErrCodeTerminalOffline  = 3002 // 终端离线
	ErrCodeTerminalBusy     = 3003 // 终端忙碌
	ErrCodeTerminalLocked   = 3004 // 终端已锁定
	ErrCodeTerminalInCall   = 3005 // 终端正在通话中

	// 设备错误 (4xxx)
	ErrCodeDeviceNotFound = 4001 // 设备未找到
	ErrCodeDeviceBusy     = 4002 // 设备忙碌
	ErrCodeDeviceError    = 4003 // 设备错误

	// 网络错误 (5xxx)
	ErrCodeNetworkTimeout    = 5001 // 网络超时
	ErrCodeNetworkError      = 5002 // 网络错误
	ErrCodeConnectionRefused = 5003 // 连接被拒绝
)

// ErrorMessages 错误码对应的中文消息
var ErrorMessages = map[int]string{
	ErrCodeAuthFailed:           "认证失败，请检查用户名和密码",
	ErrCodeSessionExpired:       "会话已过期，请重新登录",
	ErrCodeSessionInvalid:       "会话无效",
	ErrCodeConferenceNotFound:   "会议不存在",
	ErrCodeConferenceFull:       "会议已满，无法加入",
	ErrCodeConferencePassword:   "会议密码错误",
	ErrConferenceCallFailed:     "呼叫会议失败",
	ErrConferenceHangupFailed:   "挂断会议失败",
	ErrCodeConferenceInProgress: "会议正在进行中",
	ErrCodeTerminalNotFound:     "终端不存在",
	ErrCodeTerminalOffline:      "终端离线",
	ErrCodeTerminalBusy:         "终端忙碌",
	ErrCodeTerminalLocked:       "终端已被其他任务占用",
	ErrCodeTerminalInCall:       "终端正在通话中",
	ErrCodeDeviceNotFound:       "USB设备未找到",
	ErrCodeDeviceBusy:           "USB设备正在使用中",
	ErrCodeDeviceError:          "USB设备错误",
	ErrCodeNetworkTimeout:       "网络连接超时",
	ErrCodeNetworkError:         "网络连接错误",
	ErrCodeConnectionRefused:    "连接被拒绝，请检查服务器地址",
}

// GetErrorMessage 获取错误消息
func GetErrorMessage(code int) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "未知错误"
}

// HuaweiError 华为API错误
type HuaweiError struct {
	Code    int
	Message string
	Err     error
}

func (e *HuaweiError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *HuaweiError) Unwrap() error {
	return e.Err
}

// NewHuaweiError 创建华为API错误
func NewHuaweiError(code int, err error) *HuaweiError {
	return &HuaweiError{
		Code:    code,
		Message: GetErrorMessage(code),
		Err:     err,
	}
}
