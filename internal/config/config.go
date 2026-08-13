package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
)

// fatalFunc 允许测试覆盖 logger.Fatal 行为（默认调用 logger.Fatal 触发 os.Exit）。
// 测试中可替换为 panic 以便 recover 捕获。SEC-001 启动校验使用。
var fatalFunc = func(logger *zap.Logger, msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

// Config 应用配置
type Config struct {
	Server        ServerConfig        `mapstructure:"server" json:"server" yaml:"server"`
	Database      DatabaseConfig      `mapstructure:"database" json:"database" yaml:"database"`
	Auth          AuthConfig          `mapstructure:"auth" json:"auth" yaml:"auth"`
	Logging       LoggingConfig       `mapstructure:"logging" json:"logging" yaml:"logging"`
	Storage       StorageConfig       `mapstructure:"storage" json:"storage" yaml:"storage"`
	Huawei        HuaweiConfig        `mapstructure:"huawei" json:"huawei" yaml:"huawei"`
	RTSP          RTSPConfig          `mapstructure:"rtsp" json:"rtsp" yaml:"rtsp"`
	FFmpeg        FFmpegConfig        `mapstructure:"ffmpeg" json:"ffmpeg" yaml:"ffmpeg"`
	OSS           OSSConfig           `mapstructure:"oss" json:"oss" yaml:"oss"`
	Tingwu        TingwuConfig        `mapstructure:"tingwu" json:"tingwu" yaml:"tingwu"`
	Python        PythonConfig        `mapstructure:"python" json:"python" yaml:"python"`
	Admin         AdminConfig         `mapstructure:"admin" json:"admin" yaml:"admin"`
	Transcription TranscriptionConfig `mapstructure:"transcription" json:"transcription" yaml:"transcription"`
	CORS          CORSConfig          `mapstructure:"cors" json:"cors" yaml:"cors"`
	Security      SecurityConfig      `mapstructure:"security" json:"security" yaml:"security"`
	// SmartEnd Phase 23 (CFG-01): 智能录制收尾 14 项配置。
	// 14 个字段定义见 smart_end.go;bool 默认值通过 Load() 中的 Viper SetDefault
	// 在 Unmarshal 前注册,数字字段默认值由 setDefaults 调 applySmartEndDefaults
	// 在 Unmarshal 后补零值。
	SmartEnd SmartEndConfig `mapstructure:"smart_end" json:"smart_end" yaml:"smart_end"`
}

// CORSConfig controls exact cross-origin allowlisting. Empty denies cross-origin requests.
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins" json:"allowed_origins" yaml:"allowed_origins"`
}

// SecurityConfig contains transport security feature switches.
type SecurityConfig struct {
	CSRFEnabled bool `mapstructure:"csrf_enabled" json:"csrf_enabled" yaml:"csrf_enabled"`
	// CSRFSafeOrigins: 用于 Double-Submit Cookie 的额外来源校验。
	// 配置后: Origin 头必须与列表中任一项精确匹配才会接受写请求。
	// 留空 (默认): 仅校验 cookie+header 双提交,允许任意来源 (前端通过 Bearer 头时适用)。
	// 仅当 CSRFEnabled=true 时生效。
	CSRFSafeOrigins         []string `mapstructure:"csrf_safe_origins" json:"csrf_safe_origins" yaml:"csrf_safe_origins"`
	AllowedTokenURLPrefixes []string `mapstructure:"allowed_token_url_prefixes" json:"allowed_token_url_prefixes" yaml:"allowed_token_url_prefixes"`
	// OutboundURLAllowlist guards all outbound HTTP requests (SEC-013: SSRF defense).
	// Empty list means "allow all in non-production; deny all in production".
	// Each entry is a suffix matched against the URL host (e.g. "aliyun.com").
	OutboundURLAllowlist []string `mapstructure:"outbound_url_allowlist" json:"outbound_url_allowlist" yaml:"outbound_url_allowlist"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string `mapstructure:"host" json:"host" yaml:"host"`
	Port         int    `mapstructure:"port" json:"port" yaml:"port"`
	Environment  string `mapstructure:"environment" json:"environment" yaml:"environment"`
	ReadTimeout  int    `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`
	// TLS 配置
	TLSEnabled          bool   `mapstructure:"tls_enabled" json:"tls_enabled" yaml:"tls_enabled"`
	TLSCertFile         string `mapstructure:"tls_cert_file" json:"tls_cert_file" yaml:"tls_cert_file"`
	TLSKeyFile          string `mapstructure:"tls_key_file" json:"tls_key_file" yaml:"tls_key_file"`
	HTTPSPort           int    `mapstructure:"https_port" json:"https_port" yaml:"https_port"`
	RedirectHTTPToHTTPS bool   `mapstructure:"redirect_http_to_https" json:"redirect_http_to_https" yaml:"redirect_http_to_https"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver           string `mapstructure:"driver" json:"driver" yaml:"driver"`
	Path             string `mapstructure:"path" json:"path" yaml:"path"`
	EnableWAL        bool   `mapstructure:"enable_wal" json:"enable_wal" yaml:"enable_wal"`
	EnableForeignKey bool   `mapstructure:"enable_foreign_key" json:"enable_foreign_key" yaml:"enable_foreign_key"`
	JournalMode      string `mapstructure:"journal_mode" json:"journal_mode" yaml:"journal_mode"`
	Synchronous      string `mapstructure:"synchronous" json:"synchronous" yaml:"synchronous"`
	CacheSize        int    `mapstructure:"cache_size" json:"cache_size" yaml:"cache_size"`
	BusyTimeout      int    `mapstructure:"busy_timeout" json:"busy_timeout" yaml:"busy_timeout"`
	MaxOpenConns     int    `mapstructure:"max_open_conns" json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns     int    `mapstructure:"max_idle_conns" json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime  int    `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	SM4Secret            string        `mapstructure:"sm4_secret" json:"sm4_secret" yaml:"sm4_secret"`
	AccessTokenDuration  time.Duration `mapstructure:"access_token_duration" json:"access_token_duration" yaml:"access_token_duration"`
	RefreshTokenDuration time.Duration `mapstructure:"refresh_token_duration" json:"refresh_token_duration" yaml:"refresh_token_duration"`
	MaxSessionDuration   time.Duration `mapstructure:"max_session_duration" json:"max_session_duration" yaml:"max_session_duration"`
	HLSTokenSecret       string        `mapstructure:"hls_token_secret" json:"hls_token_secret" yaml:"hls_token_secret"`
	HLSTokenDuration     time.Duration `mapstructure:"hls_token_duration" json:"hls_token_duration" yaml:"hls_token_duration"`
	// 已录制视频播放专用 HMAC token（独立密钥族，与 SM4Secret / HLSTokenSecret 必须互不相同）。
	// 用于让 <video> 元素在不带 Authorization 头的 HTTP 请求里获得 5min 短效播放凭据；
	// 路径模式 /api/v1/files/playback/:token（path 而非 query string，避免 URL 出现在日志/历史/
	// Referer 等泄露面）。SEC-001/PR-B 后视频下载端点已不在 AllowedTokenURLPrefixes 白名单，
	// 通过这条路径恢复可播放性同时不扩大原 query-token 攻击面。
	VideoPlaybackTokenSecret   string        `mapstructure:"video_playback_token_secret" json:"video_playback_token_secret" yaml:"video_playback_token_secret"`
	VideoPlaybackTokenDuration time.Duration `mapstructure:"video_playback_token_duration" json:"video_playback_token_duration" yaml:"video_playback_token_duration"`
	MaxDecryptFailures         int           `mapstructure:"max_decrypt_failures" json:"max_decrypt_failures" yaml:"max_decrypt_failures"`       // 最大解密失败次数
	DecryptFailureWindow       int           `mapstructure:"decrypt_failure_window" json:"decrypt_failure_window" yaml:"decrypt_failure_window"` // 时间窗口（秒）

	// Authentication mode (local, ad) - per D-01, D-02, D-03
	Mode string `mapstructure:"mode" json:"mode" yaml:"mode"`

	// AD configuration
	AD ADAuthConfig `mapstructure:"ad" json:"ad" yaml:"ad"`

	// Phase 18: 凭据静态加密密钥族（与传输密钥 SM4Secret 严格分离）
	//
	// CredentialSM4Version / CredentialSM4Secret 是**当前生效**的凭据加密版本与密钥，
	// 对应 envelope "SM4:<version>:<base64>" 的 version 段。
	//
	// CredentialSM4PreviousVersion / CredentialSM4PreviousSecret 是**轮换过渡期**保留的旧版本
	// 密钥对（可为空）。当有正在轮换的旧版本密文时，operator 通过这两个字段提供旧密钥，
	// RotateIfNeeded 会自动把旧版本 envelope 改写成当前版本。
	//
	// 启动期校验（ValidateCredentialSM4Config）要求：
	//   - CredentialSM4Version 必须匹配 ^v[1-9][0-9]*$
	//   - CredentialSM4Secret ≥ 32 字符（与 SM4Secret/HLSTokenSecret 一致；生产环境 ≥ 32 即足够，
	//     DeriveSM4Key 取前 16 字节作为 SM4 密钥；但保留 32 字符下限以保持与 HLS token 密钥同强度）
	//   - CredentialSM4PreviousVersion 与 CredentialSM4PreviousSecret 必须同时存在或同时缺失
	//   - CredentialSM4Secret != CredentialSM4PreviousSecret（强制轮换）
	//   - CredentialSM4Version != CredentialSM4PreviousVersion
	CredentialSM4Version         string `mapstructure:"credential_sm4_version" json:"credential_sm4_version" yaml:"credential_sm4_version"`
	CredentialSM4Secret          string `mapstructure:"credential_sm4_secret" json:"credential_sm4_secret" yaml:"credential_sm4_secret"`
	CredentialSM4PreviousVersion string `mapstructure:"credential_sm4_previous_version" json:"credential_sm4_previous_version" yaml:"credential_sm4_previous_version"`
	CredentialSM4PreviousSecret  string `mapstructure:"credential_sm4_previous_secret" json:"credential_sm4_previous_secret" yaml:"credential_sm4_previous_secret"`
}

// ADAuthConfig AD域控配置
type ADAuthConfig struct {
	Server   string `mapstructure:"server" json:"server" yaml:"server"`
	BindDN   string `mapstructure:"bind_dn" json:"bind_dn" yaml:"bind_dn"`
	Password string `mapstructure:"password" json:"-" yaml:"password"`
	BaseDN   string `mapstructure:"base_dn" json:"base_dn" yaml:"base_dn"`
	UseTLS   bool   `mapstructure:"use_tls" json:"use_tls" yaml:"use_tls"`

	// Connection pool settings
	PoolSize int `mapstructure:"pool_size" json:"pool_size" yaml:"pool_size"`

	// Timeout settings
	DialTimeout    int `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`
	RequestTimeout int `mapstructure:"request_timeout" json:"request_timeout" yaml:"request_timeout"`

	// Test mode (for development only)
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`

	// AllowAutoCreate controls whether AD users are automatically created on first login
	// If false, only pre-existing AD users in the system can log in (whitelist mode)
	AllowAutoCreate bool `mapstructure:"allow_auto_create" json:"allow_auto_create" yaml:"allow_auto_create"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `mapstructure:"level" json:"level" yaml:"level"`
	Format     string `mapstructure:"format" json:"format" yaml:"format"`
	Output     string `mapstructure:"output" json:"output" yaml:"output"`
	MaxSize    int    `mapstructure:"max_size" json:"max_size" yaml:"max_size"`
	MaxBackups int    `mapstructure:"max_backups" json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `mapstructure:"max_age" json:"max_age" yaml:"max_age"`
	Compress   bool   `mapstructure:"compress" json:"compress" yaml:"compress"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	RecordingsPath string `mapstructure:"recordings_path" json:"recordings_path" yaml:"recordings_path"`
	HLSPath        string `mapstructure:"hls_path" json:"hls_path" yaml:"hls_path"` // HLS预览文件存储路径
	TempPath       string `mapstructure:"temp_path" json:"temp_path" yaml:"temp_path"`
	MaxDiskUsage   int    `mapstructure:"max_disk_usage" json:"max_disk_usage" yaml:"max_disk_usage"`

	// 文件存储配置
	Local             LocalStorageConfig `mapstructure:"local" json:"local" yaml:"local"`
	MaxFileSize       int64              `mapstructure:"max_file_size" json:"max_file_size" yaml:"max_file_size"`
	AllowedExtensions []string           `mapstructure:"allowed_extensions" json:"allowed_extensions" yaml:"allowed_extensions"`
}

// LocalStorageConfig 本地存储配置
type LocalStorageConfig struct {
	BasePath string `mapstructure:"base_path" json:"base_path" yaml:"base_path"`
	BaseURL  string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
}

// HuaweiConfig 华为会议系统配置
type HuaweiConfig struct {
	ConferenceServer   string        `mapstructure:"conference_server" json:"conference_server" yaml:"conference_server"`
	ConferencePort     int           `mapstructure:"conference_port" json:"conference_port" yaml:"conference_port"`
	Username           string        `mapstructure:"username" json:"username" yaml:"username"`
	Password           string        `mapstructure:"password" json:"-" yaml:"password"`
	HTTPS              bool          `mapstructure:"https" json:"https" yaml:"https"`
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	APITimeout         time.Duration `mapstructure:"api_timeout" json:"api_timeout" yaml:"api_timeout"`
	SessionTimeout     time.Duration `mapstructure:"session_timeout" json:"session_timeout" yaml:"session_timeout"`
	KeepAliveInterval  time.Duration `mapstructure:"keep_alive_interval" json:"keep_alive_interval" yaml:"keep_alive_interval"`
	MinTLSVersion      string        `mapstructure:"min_tls_version" json:"min_tls_version" yaml:"min_tls_version"`
	// SEC-003a: 华为终端私有 CA bundle 路径（推荐 ./certs/huawei-10.62.10.3-ca.pem）。
	// 空字符串 = 使用系统 CA bundle（保留完整证书校验）;
	// 非空 = 由 Manager.SetCABundle 解析全部 CERTIFICATE 块后注入 tls.Config.RootCAs。
	// 显式空字符串必须保留为空（system-CA opt-out），不可在 setDefaults 里覆盖。
	CABundleFile string `mapstructure:"ca_bundle_file" json:"ca_bundle_file" yaml:"ca_bundle_file"`
	// REDO (huawei-tls-after-ca-fix): 华为终端 hostname 校验 ServerName——
	// IP-only dial + cert SAN 仅含 DNS（如 *.tp.huawei.com）时，必须把 ServerName
	// 设为与 dNSName 匹配的字面名（例如 "vct.tp.huawei.com"），让 Go x509 verifier 走
	// dNSName 路径而不是被 "doesn't contain any IP SANs" 拒绝。
	// 空字符串 = Go 用 dial 地址做 hostname 校验（不修改现状）；与 ca_bundle_file
	// 同级别可由 HUAWEI_TLS_SERVER_NAME 环境变量覆盖。
	TLSServerName string `mapstructure:"tls_server_name" json:"tls_server_name" yaml:"tls_server_name"`
}

// RTSPConfig RTSP配置
type RTSPConfig struct {
	Enabled          bool          `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	MaxStreams       int           `mapstructure:"max_streams" json:"max_streams" yaml:"max_streams"`
	ReconnectTimeout time.Duration `mapstructure:"reconnect_timeout" json:"reconnect_timeout" yaml:"reconnect_timeout"`
	BufferSize       int           `mapstructure:"buffer_size" json:"buffer_size" yaml:"buffer_size"`
}

// FFmpegConfig FFmpeg配置
type FFmpegConfig struct {
	Path                string        `mapstructure:"path" json:"path" yaml:"path"`
	FFProbePath         string        `mapstructure:"ffprobe_path" json:"ffprobe_path" yaml:"ffprobe_path"`
	MaxProcesses        int           `mapstructure:"max_processes" json:"max_processes" yaml:"max_processes"`
	Timeout             time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	DefaultCodec        string        `mapstructure:"default_codec" json:"default_codec" yaml:"default_codec"`
	DefaultFormat       string        `mapstructure:"default_format" json:"default_format" yaml:"default_format"`
	DefaultVideoBitrate string        `mapstructure:"default_video_bitrate" json:"default_video_bitrate" yaml:"default_video_bitrate"`
	DefaultAudioBitrate string        `mapstructure:"default_audio_bitrate" json:"default_audio_bitrate" yaml:"default_audio_bitrate"`
	// HLSRewriteConcurrency 控制 HLS 改写 handler 的并发度（PERF-005/D-03.8 默认 2）。
	HLSRewriteConcurrency int `mapstructure:"hls_rewrite_concurrency" json:"hls_rewrite_concurrency" yaml:"hls_rewrite_concurrency"`
	// 视频编码质量控制
	CRF             int    `mapstructure:"crf" json:"crf" yaml:"crf"`                                           // CRF质量值（0-51，值越小质量越高，推荐23）
	Preset          string `mapstructure:"preset" json:"preset" yaml:"preset"`                                  // 编码速度预设（ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow）
	MaxVideoBitrate string `mapstructure:"max_video_bitrate" json:"max_video_bitrate" yaml:"max_video_bitrate"` // 最大视频码率（配合CRF使用）
	VideoBufSize    string `mapstructure:"video_bufsize" json:"video_bufsize" yaml:"video_bufsize"`             // 视频缓冲区大小（通常是maxrate的2倍）
	// DShow 设备配置
	DShowBufferSize      int `mapstructure:"dshow_buffer_size" json:"dshow_buffer_size" yaml:"dshow_buffer_size"`                   // 实时缓冲区大小（字节）
	DShowThreadQueueSize int `mapstructure:"dshow_thread_queue_size" json:"dshow_thread_queue_size" yaml:"dshow_thread_queue_size"` // 线程队列大小
	// HLS 配置
	HLSSegmentDuration int `mapstructure:"hls_segment_duration" json:"hls_segment_duration" yaml:"hls_segment_duration"` // HLS 分片时长（秒）
	HLSListSize        int `mapstructure:"hls_list_size" json:"hls_list_size" yaml:"hls_list_size"`                      // HLS 播放列表保留分片数
	// 录制监控配置
	MaxRecordingDuration time.Duration `mapstructure:"max_recording_duration" json:"max_recording_duration" yaml:"max_recording_duration"` // 最长录制时长
}

// OSSConfig 阿里云OSS配置
type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	BucketName      string `mapstructure:"bucket_name" json:"bucket_name" yaml:"bucket_name"`
	AccessKeyID     string `mapstructure:"access_key_id" json:"access_key_id" yaml:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret" json:"-" yaml:"access_key_secret"`
	Enabled         bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	UploadTimeout   int    `mapstructure:"upload_timeout" json:"upload_timeout" yaml:"upload_timeout"`          // seconds
	PresignedURLTTL int    `mapstructure:"presigned_url_ttl" json:"presigned_url_ttl" yaml:"presigned_url_ttl"` // seconds, default 86400
}

// TingwuConfig 阿里通义听悟配置
type TingwuConfig struct {
	AppKey     string `mapstructure:"app_key" json:"app_key" yaml:"app_key"`
	AppSecret  string `mapstructure:"app_secret" json:"-" yaml:"app_secret"`
	BaseURL    string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	Enabled    bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	APITimeout int    `mapstructure:"api_timeout" json:"api_timeout" yaml:"api_timeout"` // seconds
}

// PythonConfig Python依赖配置
type PythonConfig struct {
	PreferUV bool `mapstructure:"prefer_uv" json:"prefer_uv" yaml:"prefer_uv"` // 优先使用uv管理Python依赖
}

// AdminConfig 管理后台配置（PERF-005/D-03.8 bounded concurrency 字段）
type AdminConfig struct {
	MigrationConcurrency int `mapstructure:"migration_concurrency" json:"migration_concurrency" yaml:"migration_concurrency"` // 迁移 handler 并发度
}

// TranscriptionConfig 转录配置（PERF-005/D-03.8）
type TranscriptionConfig struct {
	BatchConcurrency int `mapstructure:"batch_concurrency" json:"batch_concurrency" yaml:"batch_concurrency"` // 批量转录 handler 并发度
}

// expandEnvWithDefault 展开环境变量，支持 ${VAR:default} 格式
func expandEnvWithDefault(s string) string {
	// 使用包级正则 expandEnvRegex（PERF-008），避免每次调用重新编译
	return expandEnvRegex.ReplaceAllStringFunc(s, func(match string) string {
		parts := expandEnvRegex.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultValue := ""
		if len(parts) >= 3 {
			defaultValue = parts[2]
		}

		// 优先使用环境变量的值
		if envValue := os.Getenv(varName); envValue != "" {
			return envValue
		}

		// 否则使用默认值
		return defaultValue
	})
}

// expandEnvRegex 匹配 \${VAR} 或 \${VAR:default} 格式（PERF-008 提到包级）
var expandEnvRegex = regexp.MustCompile(`\$\{([^:}]+)(?::([^}]*))?\}`)

// ConfigDiffEntry 类型化配置变更条目（PERF-009）。
// 历史 rawCfg 走 map[string]interface{} 会触发装箱；定义专用类型避免反射/类型断言。
type ConfigDiffEntry struct {
	Key      string `json:"key"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// expandConfig 递归展开配置中的所有字符串值（保留 map[string]interface{}
// 以兼容 viper 的 MergeConfigMap 入参类型；调用方拿到展开后的值再转为强类型）。
func expandConfig(cfg interface{}) interface{} {
	switch v := cfg.(type) {
	case map[string]interface{}:
		for key, val := range v {
			v[key] = expandConfig(val)
		}
		return v
	case []interface{}:
		for i, val := range v {
			v[i] = expandConfig(val)
		}
		return v
	case string:
		return expandEnvWithDefault(v)
	default:
		return v
	}
}

// Load 加载配置
func Load() (*Config, error) {
	v := viper.New()

	// 设置配置文件（仅在项目根目录）
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	// 环境变量支持
	v.AutomaticEnv()
	v.SetEnvPrefix("RECORD")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		// 配置文件不存在时，创建默认配置文件
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			if err := createDefaultConfigFile(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w: %w", apperrors.ErrInternal, err)
			}
			// 重新读取配置文件
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config after creation: %w: %w", apperrors.ErrInternal, err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w: %w", apperrors.ErrInternal, err)
		}
	}

	// 获取原始配置
	var rawCfg map[string]interface{}
	if err := v.Unmarshal(&rawCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw config: %w: %w", apperrors.ErrInternal, err)
	}

	// 展开环境变量
	expandedCfg := expandConfig(rawCfg)

	// 将展开后的配置转换回 viper
	v = viper.New()
	v.SetConfigType("yaml")
	if err := v.MergeConfigMap(expandedCfg.(map[string]interface{})); err != nil {
		return nil, fmt.Errorf("failed to merge expanded config: %w: %w", apperrors.ErrInternal, err)
	}

	// Phase 23 (CFG-01) + RESEARCH.md Pitfall 3 修复:smart_end 的 3 个 true-valued
	// bool 字段必须通过 Viper SetDefault **在 Unmarshal 之前** 注册,这样当用户
	// 在 config.yaml 里显式写 `smart_end.enabled: false` 时,SetDefault(true) 不会
	// 把它覆盖为 true (CFG-03/04 退回开关)。YAML 读取顺序天然优先于默认。
	v.SetDefault("smart_end.enabled", true)
	v.SetDefault("smart_end.huawei_enabled", true)
	v.SetDefault("smart_end.degrade_on_silence_loss", true)

	// SEC-001: 显式绑定部署文档中的环境变量名（无 RECORD 前缀），使运维设置的
	// SM4_SECRET/HLS_TOKEN_SECRET 等真正加载到配置结构（绕过 SetEnvPrefix 机制）。
	bindSecretEnv(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w: %w", apperrors.ErrInternal, err)
	}

	// 设置默认值
	setDefaults(&cfg)

	// Phase 23 (CFG-01): SmartEnd 数值/范围校验。fail-closed — 配置错误直接
	// 阻断启动,避免后续 watcher/scheduler 在错的阈值上跑空跑。
	if err := cfg.SmartEnd.Validate(); err != nil {
		return nil, fmt.Errorf("smart_end config validation failed: %w: %w", apperrors.ErrInternal, err)
	}

	if value := os.Getenv("CORS_ALLOWED_ORIGINS"); value != "" {
		cfg.CORS.AllowedOrigins = splitCommaSeparated(value)
	}
	if value := os.Getenv("ALLOWED_TOKEN_URL_PREFIXES"); value != "" {
		cfg.Security.AllowedTokenURLPrefixes = splitCommaSeparated(value)
	}
	// CSRF_SAFE_ORIGINS 提供 comma-separated origin 列表; 空字符串表示不附加 origin 检查。
	if value := os.Getenv("CSRF_SAFE_ORIGINS"); value != "" {
		cfg.Security.CSRFSafeOrigins = splitCommaSeparated(value)
	}

	// 创建必要的目录
	if err := ensureDirectories(&cfg); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w: %w", apperrors.ErrInternal, err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(cfg *Config) { //nolint:gocyclo,cyclop // 大量配置项默认值赋值，复杂度自然高，拆分收益低
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Environment == "" {
		cfg.Server.Environment = "production"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30
	}
	// TLS 默认值
	if cfg.Server.HTTPSPort == 0 {
		cfg.Server.HTTPSPort = 8443
	}
	if !cfg.Server.TLSEnabled && cfg.Server.TLSCertFile != "" {
		// 如果指定了证书文件，自动启用 TLS
		cfg.Server.TLSEnabled = true
	}

	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = "./data/record.db"
	}
	if cfg.Database.JournalMode == "" {
		cfg.Database.JournalMode = "WAL"
	}
	if cfg.Database.Synchronous == "" {
		cfg.Database.Synchronous = "NORMAL"
	}
	if cfg.Database.CacheSize == 0 {
		cfg.Database.CacheSize = 2000
	}
	if cfg.Database.BusyTimeout == 0 {
		cfg.Database.BusyTimeout = 5000
	}
	if cfg.Database.MaxOpenConns == 0 {
		// PERF-015: SQLite 单 writer 限制下默认 1；MySQL/PostgreSQL 部署时
		// 应通过 env DB_MAX_OPEN_CONNS 覆盖为 25。
		cfg.Database.MaxOpenConns = 1
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 1
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		// PERF-015: 默认 30 分钟连接生命周期，避免长连接被服务端回收后报错。
		cfg.Database.ConnMaxLifetime = 1800 // 30 minutes in seconds
	}

	// SEC-001/PR-B: 收窄默认 ALLOWED_TOKEN_URL_PREFIXES,移除过宽的 /api/v1/recordings/ 前缀。
	// 注意:HLS stream 路由 (/api/v1/recordings/:id/preview/stream/:file) 在 a.router 上
	// 直接注册,**不经过 MultiAuth**,因此它有自己的 c.Query("token") 校验,不受此白名单影响。
	// 这里的白名单只控制 MultiAuth 内部对 Authorization 缺失时是否回退到 query token。
	// 收紧到仅允许"明确需要 query token"的端点(下载/分享/PPT 静态文件)。
	if cfg.Security.AllowedTokenURLPrefixes == nil {
		cfg.Security.AllowedTokenURLPrefixes = []string{
			"/api/v1/files/download/",
			"/api/v1/files/share/",
			"/api/v1/ppts/",
		}
	}
	// SEC-013: 出站 URL 白名单默认 nil（空列表 = 生产环境拒绝所有出站请求；
	// 开发环境由 caller 的 development bypass 跳过）。BindEnv 留给调用方。
	if cfg.Security.OutboundURLAllowlist == nil {
		cfg.Security.OutboundURLAllowlist = []string{}
	}

	// SEC-001/D-03.4: 不再保留硬编码 fallback 默认密钥；
	// 缺失时保持空字符串，由 ValidateProductionSecrets 在启动时决定是否 Fatal。
	if cfg.Auth.AccessTokenDuration == 0 {
		cfg.Auth.AccessTokenDuration = 2 * time.Hour
	}
	if cfg.Auth.RefreshTokenDuration == 0 {
		cfg.Auth.RefreshTokenDuration = 7 * 24 * time.Hour
	}
	if cfg.Auth.MaxSessionDuration == 0 {
		cfg.Auth.MaxSessionDuration = 30 * 24 * time.Hour
	}
	// SEC-001/PR-B: 缩短 HLS Token 默认有效期 5min -> 30s。窗口越小则即便 URL/CDN 泄露，
	// token 重放窗口越小。生产建议 ≤ 60s；现阶段每个 .ts 分片仍共用同一 token，需 m3u8
	// 在 30s 内完成播放——若存在慢客户端可调大但请同步审计日志告警。
	if cfg.Auth.HLSTokenDuration == 0 {
		cfg.Auth.HLSTokenDuration = 30 * time.Second
	}
	// video_playback_token：默认 5min，覆盖 30min 视频 preview 的 Range 请求 + 暂停 +
	// 拖拽 + 5min 兜底重试；泄露窗口适中。可由 cfg.video_playback_token_duration 调小/调大。
	if cfg.Auth.VideoPlaybackTokenDuration == 0 {
		cfg.Auth.VideoPlaybackTokenDuration = 5 * time.Minute
	}
	// 解密失败速率限制默认值
	if cfg.Auth.MaxDecryptFailures == 0 {
		cfg.Auth.MaxDecryptFailures = 5 // 最大失败次数
	}
	if cfg.Auth.DecryptFailureWindow == 0 {
		cfg.Auth.DecryptFailureWindow = 300 // 5分钟时间窗口
	}
	// Default to local mode (safest option per D-02)
	if cfg.Auth.Mode == "" {
		cfg.Auth.Mode = "local"
	}
	// Set AD configuration defaults
	if cfg.Auth.AD.PoolSize == 0 {
		cfg.Auth.AD.PoolSize = 10
	}
	if cfg.Auth.AD.DialTimeout == 0 {
		cfg.Auth.AD.DialTimeout = 10 // seconds
	}
	if cfg.Auth.AD.RequestTimeout == 0 {
		cfg.Auth.AD.RequestTimeout = 30 // seconds
	}
	// Never default InsecureSkipVerify to true (security)

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
	if cfg.Logging.Output == "" {
		cfg.Logging.Output = "logs"
	}
	if cfg.Logging.MaxSize == 0 {
		cfg.Logging.MaxSize = 100
	}
	if cfg.Logging.MaxBackups == 0 {
		cfg.Logging.MaxBackups = 10
	}
	if cfg.Logging.MaxAge == 0 {
		cfg.Logging.MaxAge = 30
	}

	// 设置默认值（保持相对路径，避免 Windows 盘符转义问题）
	if cfg.Storage.RecordingsPath == "" {
		cfg.Storage.RecordingsPath = "./data/recordings"
	}
	if cfg.Storage.HLSPath == "" {
		cfg.Storage.HLSPath = "./data/hls"
	}
	if cfg.Storage.TempPath == "" {
		cfg.Storage.TempPath = "./data/temp"
	}
	if cfg.Storage.MaxDiskUsage == 0 {
		cfg.Storage.MaxDiskUsage = 90 // 90%
	}

	// 文件存储默认值
	if cfg.Storage.Local.BasePath == "" {
		cfg.Storage.Local.BasePath = "./data/files"
	}
	if !filepath.IsAbs(cfg.Storage.Local.BasePath) {
		absPath, err := filepath.Abs(cfg.Storage.Local.BasePath)
		if err == nil {
			cfg.Storage.Local.BasePath = absPath
		}
	}
	if cfg.Storage.Local.BaseURL == "" {
		cfg.Storage.Local.BaseURL = fmt.Sprintf("http://%s:%d/files", cfg.Server.Host, cfg.Server.Port)
	}
	if cfg.Storage.MaxFileSize == 0 {
		cfg.Storage.MaxFileSize = 5 * 1024 * 1024 * 1024 // 5GB
	}

	if cfg.FFmpeg.Path == "" {
		cfg.FFmpeg.Path = "./bin/ffmpeg" // 默认使用项目内置的 ffmpeg
	}
	if cfg.FFmpeg.FFProbePath == "" {
		cfg.FFmpeg.FFProbePath = "./bin/ffprobe" // 默认使用项目内置的 ffprobe
	}
	if cfg.FFmpeg.MaxProcesses == 0 {
		cfg.FFmpeg.MaxProcesses = 5
	}
	if cfg.FFmpeg.Timeout == 0 {
		cfg.FFmpeg.Timeout = 5 * time.Minute
	}
	if cfg.FFmpeg.DefaultCodec == "" {
		cfg.FFmpeg.DefaultCodec = "h264"
	}
	if cfg.FFmpeg.DefaultFormat == "" {
		cfg.FFmpeg.DefaultFormat = "mp4"
	}
	// 视频编码质量控制默认值
	if cfg.FFmpeg.CRF == 0 {
		cfg.FFmpeg.CRF = 23 // 推荐值，质量与大小平衡
	}
	if cfg.FFmpeg.Preset == "" {
		cfg.FFmpeg.Preset = "medium" // 编码速度与压缩率平衡
	}
	if cfg.FFmpeg.MaxVideoBitrate == "" {
		cfg.FFmpeg.MaxVideoBitrate = "3M" // 最大码率3Mbps
	}
	if cfg.FFmpeg.VideoBufSize == "" {
		cfg.FFmpeg.VideoBufSize = "6M" // 缓冲区大小（2倍maxrate）
	}
	// DShow 设备默认值
	if cfg.FFmpeg.DShowBufferSize == 0 {
		cfg.FFmpeg.DShowBufferSize = 104857600 // 100MB
	}
	if cfg.FFmpeg.DShowThreadQueueSize == 0 {
		cfg.FFmpeg.DShowThreadQueueSize = 1024
	}
	// HLS 默认值
	if cfg.FFmpeg.HLSSegmentDuration == 0 {
		cfg.FFmpeg.HLSSegmentDuration = 10 // 10秒
	}
	if cfg.FFmpeg.HLSListSize == 0 {
		cfg.FFmpeg.HLSListSize = 5 // 保留5个分片
	}
	// 录制监控默认值
	if cfg.FFmpeg.MaxRecordingDuration == 0 {
		cfg.FFmpeg.MaxRecordingDuration = 24 * time.Hour
	}
	// PERF-005/D-03.8: 三个 handler 的有界并发度默认值。
	if cfg.FFmpeg.HLSRewriteConcurrency == 0 {
		cfg.FFmpeg.HLSRewriteConcurrency = 2 // FFmpeg 较重，默认较低
	}
	if cfg.Admin.MigrationConcurrency == 0 {
		cfg.Admin.MigrationConcurrency = 4
	}
	if cfg.Transcription.BatchConcurrency == 0 {
		cfg.Transcription.BatchConcurrency = 4
	}

	// Huawei config defaults
	if cfg.Huawei.ConferencePort == 0 {
		cfg.Huawei.ConferencePort = 80
	}
	if cfg.Huawei.APITimeout == 0 {
		cfg.Huawei.APITimeout = 30 * time.Second
	}
	if cfg.Huawei.SessionTimeout == 0 {
		cfg.Huawei.SessionTimeout = 3600 * time.Second
	}
	if cfg.Huawei.KeepAliveInterval == 0 {
		cfg.Huawei.KeepAliveInterval = 300 * time.Second
	}
	if cfg.Huawei.MinTLSVersion == "" {
		cfg.Huawei.MinTLSVersion = "1.2"
	}

	// OSS defaults
	if cfg.OSS.UploadTimeout == 0 {
		cfg.OSS.UploadTimeout = 300 // 5 minutes
	}
	if cfg.OSS.PresignedURLTTL == 0 {
		cfg.OSS.PresignedURLTTL = 86400 // 24 hours
	}

	// Tingwu defaults
	if cfg.Tingwu.BaseURL == "" {
		cfg.Tingwu.BaseURL = "https://tingwu.cn-beijing.aliyuncs.com"
	}
	if cfg.Tingwu.APITimeout == 0 {
		cfg.Tingwu.APITimeout = 30
	}

	// Phase 23 (CFG-01): SmartEnd 数字字段默认值 (bool 由 Viper SetDefault 处理)。
	applySmartEndDefaults(cfg)
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ensureDirectories 确保目录存在
func ensureDirectories(cfg *Config) error {
	dirs := []string{
		filepath.Dir(cfg.Database.Path),
		cfg.Storage.RecordingsPath,
		cfg.Storage.HLSPath,
		cfg.Storage.TempPath,
		cfg.Storage.Local.BasePath,
		cfg.Logging.Output,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w: %w", dir, apperrors.ErrInternal, err)
		}
	}

	return nil
}

// createDefaultConfigFile 创建默认配置文件
func createDefaultConfigFile() error {
	// 在项目根目录创建配置文件
	configPath := "./config.yaml"

	// 默认配置内容
	defaultConfig := `# 视频会议录制系统 V2.0 配置文件
# 配置文件不存在时自动生成

server:
  host: "0.0.0.0"
  port: 8080
  environment: "production"
  read_timeout: 30
  write_timeout: 30

database:
  driver: "sqlite"
  path: "./data/record.db"
  journal_mode: "WAL"
  synchronous: "NORMAL"
  cache_size: 2000
  busy_timeout: 5000
  max_open_conns: 1
  max_idle_conns: 1
  conn_max_lifetime: 3600

auth:
  # SEC-001: 密钥必须显式设置（最小 32 字符），推荐通过环境变量 SM4_SECRET / HLS_TOKEN_SECRET 注入。
  # 生产环境缺失或过短将触发 logger.Fatal 终止启动。
  sm4_secret: "${SM4_SECRET:}"
  access_token_duration: "2h"
  refresh_token_duration: "168h"  # 7 days
  max_session_duration: "720h"   # 30 days
  hls_token_secret: "${HLS_TOKEN_SECRET:}"
  # SEC-001/PR-B: 缩短到 30s — 上一默认值 5m 在 CDN/浏览器历史中可被无限重放。
  hls_token_duration: "30s"

logging:
  level: "info"
  format: "json"
  output: "logs"
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true

storage:
  recordings_path: "./data/recordings"
  hls_path: "./data/hls"
  temp_path: "./data/temp"
  max_disk_usage: 90
  local:
    base_path: "./data/files"
    base_url: "http://localhost:8080/files"
  max_file_size: 5368709120  # 5GB
  allowed_extensions: [".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".ppt", ".pptx", ".pdf", ".doc", ".docx"]

huawei:
  conference_server: ""
  conference_port: 80
  username: ""
  password: ""
  https: true
  insecure_skip_verify: false
  api_timeout: "30s"
  session_timeout: "3600s"
  keep_alive_interval: "300s"
  min_tls_version: "1.2"
  # SEC-003a: 华为终端私有 CA bundle 路径（与 certs/*.pem 一并由运维部署）
  ca_bundle_file: "./certs/huawei-10.62.10.3-ca.pem"
  # IP-only dial 且证书只含 DNS SAN 时，填写与 SAN 匹配的名称；空字符串保留 Go 默认校验。
  # 可由 HUAWEI_TLS_SERVER_NAME 环境变量覆盖。
  tls_server_name: ""

ffmpeg:
  path: "./bin/ffmpeg"  # 使用项目内置的 ffmpeg
  ffprobe_path: "./bin/ffprobe"  # 使用项目内置的 ffprobe
  max_processes: 5
  timeout: "5m"
  default_codec: "h264"
  default_format: "mp4"
  default_video_bitrate: "2000"   # 已废弃，使用 crf/max_video_bitrate
  default_audio_bitrate: "128"
  # 视频编码质量控制（CRF模式 - 推荐）
  crf: 23                         # CRF质量值：18-28，值越小质量越高。23为推荐值，可减少30-50%文件大小
  preset: "medium"                # 编码速度预设：ultrafast~veryslow。medium平衡速度与压缩率
  max_video_bitrate: "3M"         # 最大视频码率（配合CRF使用，防止码率过高）
  video_bufsize: "6M"             # 视频缓冲区大小（通常是maxrate的2倍）
  # DShow 设备配置
  dshow_buffer_size: 2097152      # 2MB 实时缓冲区
  dshow_thread_queue_size: 8      # 线程队列大小
  # HLS 配置
  hls_segment_duration: 10        # HLS 分片时长（秒）
  hls_list_size: 5                # HLS 播放列表保留分片数
  # 录制监控配置
  max_recording_duration: "24h"   # 最长录制时长

rtsp:
  enabled: false
  max_streams: 10
  reconnect_timeout: "30s"
  buffer_size: 1048576

# 阿里云OSS配置
oss:
  endpoint: "${ALIYUN_OSS_ENDPOINT:}"
  bucket_name: "${ALIYUN_OSS_BUCKET:}"
  access_key_id: "${ALIYUN_OSS_ACCESS_KEY_ID:}"
  access_key_secret: "${ALIYUN_OSS_ACCESS_KEY_SECRET:}"
  enabled: false
  upload_timeout: 300
  presigned_url_ttl: 86400

# 阿里通义听悟配置
tingwu:
  app_key: "${TYTW_APP_KEY:}"
  app_secret: "${TYTW_APP_SECRET:}"
  base_url: "https://tingwu.cn-beijing.aliyuncs.com"
  enabled: true
  api_timeout: 30

# Python依赖配置
python:
  prefer_uv: true  # 优先使用uv管理Python依赖（需要安装uv）

# Phase 23 (CFG-02) + 23-RESEARCH.md: smart_end 段 — 与 SmartEndConfig struct 14
# 字段一一对应。auto-generated config.yaml 自动含此段,与 bin/config.yaml 部署
# 模板及 REQUIREMENTS.md:58 锁定列表保持同步,smart_end_yaml_test.go 的 RED gate
# 因此在干净 checkout 上可正常验证 root config.yaml(部署期 bin/config.yaml
# 仍由运维手工准备,测试在缺失时安全 skip)。
smart_end:
  enabled: true
  silence_db: -30
  silence_duration_s: 30
  file_stall_s: 120
  file_min_growth_bps: 1024
  huawei_enabled: true
  huawei_poll_interval_s: 30
  huawei_persist_s: 30
  huawei_failure_threshold: 3
  check_interval_s: 5
  extend_step_min: 30
  max_extend_count: 4
  stat_failure_threshold: 6
  degrade_on_silence_loss: true
`

	// 写入配置文件
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w: %w", apperrors.ErrInternal, err)
	}

	return nil
}

// bindSecretEnv 显式将部署文档中的环境变量名（无 RECORD 前缀）绑定到 viper 配置键。
// 背景：项目历史上用 SetEnvPrefix("RECORD")，但 DEPLOYMENT.md/.env.example/运维都写
// SM4_SECRET/HLS_TOKEN_SECRET（无前缀），导致环境变量无法生效（SEC-001）。
// 这里用显式 BindEnv 精确映射，绕过 prefix 机制，使 os.Getenv("SM4_SECRET") 真正加载到配置。
func bindSecretEnv(v *viper.Viper) {
	_ = v.BindEnv("auth.sm4_secret", "SM4_SECRET")
	_ = v.BindEnv("auth.hls_token_secret", "HLS_TOKEN_SECRET")
	// video_playback_token 密钥族环境变量覆盖；与 HLS_TOKEN_SECRET / SM4_SECRET 解耦。
	_ = v.BindEnv("auth.video_playback_token_secret", "VIDEO_PLAYBACK_TOKEN_SECRET")
	_ = v.BindEnv("huawei.insecure_skip_verify", "HUAWEI_INSECURE_SKIP_VERIFY")
	_ = v.BindEnv("huawei.min_tls_version", "HUAWEI_MIN_TLS_VERSION")
	// SEC-003a: 华为终端私有 CA bundle 路径覆盖——运维无需重新编译二进制即可切换。
	_ = v.BindEnv("huawei.ca_bundle_file", "HUAWEI_CA_BUNDLE_FILE")
	// REDO (huawei-tls-after-ca-fix): 华为终端 hostname 校验 ServerName 覆盖。
	_ = v.BindEnv("huawei.tls_server_name", "HUAWEI_TLS_SERVER_NAME")
	_ = v.BindEnv("cors.allowed_origins", "CORS_ALLOWED_ORIGINS")
	_ = v.BindEnv("security.csrf_enabled", "CSRF_ENABLED")
	_ = v.BindEnv("security.csrf_safe_origins", "CSRF_SAFE_ORIGINS")
	_ = v.BindEnv("security.allowed_token_url_prefixes", "ALLOWED_TOKEN_URL_PREFIXES")
	// PERF-005/D-03.8: 三个 handler 的有界并发度可通过环境变量覆盖。
	_ = v.BindEnv("admin.migration_concurrency", "ADMIN_MIGRATION_CONCURRENCY")
	_ = v.BindEnv("transcription.batch_concurrency", "TRANSCRIPTION_BATCH_CONCURRENCY")
	_ = v.BindEnv("ffmpeg.hls_rewrite_concurrency", "FFMPEG_HLS_REWRITE_CONCURRENCY")
	// Phase 18: 凭据静态加密密钥族（与传输密钥 SM4_SECRET 解耦，独立 env 名）
	_ = v.BindEnv("auth.credential_sm4_version", "CREDENTIAL_SM4_VERSION")
	_ = v.BindEnv("auth.credential_sm4_secret", "CREDENTIAL_SM4_SECRET")
	_ = v.BindEnv("auth.credential_sm4_previous_version", "CREDENTIAL_SM4_PREVIOUS_VERSION")
	_ = v.BindEnv("auth.credential_sm4_previous_secret", "CREDENTIAL_SM4_PREVIOUS_SECRET")
}

// ValidateProductionSecrets 在生产环境强制校验 SM4/HLS Token 密钥：
// 必须显式设置、长度 ≥ 32 字符、且互不相同。缺失或不合规时调用 logger.Fatal 终止启动。
// 非生产环境仅打印警告（便于本地/测试运行）。SEC-001/D-03.1/D-03.4。
//
// 附加：无论生产/非生产，均调用 utils.WarnOnKeyTruncation 检测 hex 编码 secret 是否会被
// DeriveSM4Key 静默截断（典型 bug 场景：`openssl rand -hex 32` 生成 64-char secret）。
func (c *Config) ValidateProductionSecrets(logger *zap.Logger) {
	if logger == nil {
		return
	}
	isProd := c.Server.Environment == "production"
	sm4, hls := c.Auth.SM4Secret, c.Auth.HLSTokenSecret

	if isProd {
		if len(sm4) < 32 {
			fatalFunc(logger, "SM4_SECRET 必须显式设置且 ≥ 32 字符（生产环境），进程终止",
				zap.Int("length", len(sm4)))
			return
		}
		if len(hls) < 32 {
			fatalFunc(logger, "HLS_TOKEN_SECRET 必须显式设置且 ≥ 32 字符（生产环境），进程终止",
				zap.Int("length", len(hls)))
			return
		}
		if sm4 == hls {
			fatalFunc(logger, "SM4_SECRET 与 HLS_TOKEN_SECRET 必须互不相同（生产环境），进程终止")
			return
		}
	} else {
		if len(sm4) < 32 {
			logger.Warn("SM4_SECRET 长度不足 32 字符（非生产环境仅警告）", zap.Int("length", len(sm4)))
		}
		if len(hls) < 32 {
			logger.Warn("HLS_TOKEN_SECRET 长度不足 32 字符（非生产环境仅警告）", zap.Int("length", len(hls)))
		}
	}

	// 检测 SM4 传输密钥 hex 编码过长（典型 64-char → 32 bytes 被 DeriveSM4Key 静默截断）。
	// 仅警告，不阻断启动；让运维知道应改为 `openssl rand -hex 16` 输出 32 hex chars。
	// 注意：HLS_TOKEN_SECRET 用 HMAC-SHA256（任意长度都接受，无截断）— 不报警。
	utils.WarnOnKeyTruncation(logger, sm4, "SM4_SECRET", utils.SM4KeyTransport)
}

// WarnCredentialSM4Truncation 检测 Phase 18 凭据密钥族的 hex 截断风险。
// 应在 ValidateCredentialSM4Config 之后调用一次；不阻断启动。
func (c *Config) WarnCredentialSM4Truncation(logger *zap.Logger) {
	if logger == nil {
		return
	}
	utils.WarnOnKeyTruncation(logger, c.Auth.CredentialSM4Secret, "CREDENTIAL_SM4_SECRET", utils.SM4KeyStatic)
	utils.WarnOnKeyTruncation(logger, c.Auth.CredentialSM4PreviousSecret, "CREDENTIAL_SM4_PREVIOUS_SECRET", utils.SM4KeyStatic)
}

// CredentialSM4VersionPattern 是 envelope version 段的合法正则：v1, v2, ...
var CredentialSM4VersionPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

// ValidateCredentialSM4Config 校验凭据静态加密密钥族配置。
//
// 规则（全部失败条件 → 返回 error，调用方决定如何处理；当前 cmd/server/app.go
// 的 Initialize() 把它作为 fail-closed 步骤）：
//   - CredentialSM4Version 必须非空且匹配 ^v[1-9][0-9]*$
//   - CredentialSM4Secret 必须 ≥ 32 字符
//   - CredentialSM4PreviousVersion 与 CredentialSM4PreviousSecret 必须同时存在或同时缺失
//   - 当 PreviousVersion 存在时：必须匹配 ^v[1-9][0-9]*$，且必须 != CurrentVersion
//   - 当 PreviousSecret 存在时：必须 ≥ 32 字符，且必须 != CurrentSecret
//
// 非生产环境下不阻断启动，但返回 error 让 caller 决定日志级别；
// 生产环境下 cmd/server/app.go 直接 Fatal。
func (c *Config) ValidateCredentialSM4Config() error {
	cur := c.Auth.CredentialSM4Version
	sec := c.Auth.CredentialSM4Secret
	prevVer := c.Auth.CredentialSM4PreviousVersion
	prevSec := c.Auth.CredentialSM4PreviousSecret

	if cur == "" {
		return fmt.Errorf("CREDENTIAL_SM4_VERSION 必须显式设置（Phase 18 凭据静态加密要求）")
	}
	if !CredentialSM4VersionPattern.MatchString(cur) {
		return fmt.Errorf("CREDENTIAL_SM4_VERSION 格式非法: %q（必须匹配 ^v[1-9][0-9]*$）", cur)
	}
	if len(sec) < 32 {
		return fmt.Errorf("CREDENTIAL_SM4_SECRET 必须 ≥ 32 字符（当前长度=%d）", len(sec))
	}

	// Previous 必须配对出现
	if (prevVer == "") != (prevSec == "") {
		return fmt.Errorf("CREDENTIAL_SM4_PREVIOUS_VERSION 与 CREDENTIAL_SM4_PREVIOUS_SECRET 必须同时设置或同时缺失")
	}
	// 全部缺失 → 跳过轮换校验（正常启动场景）
	if prevVer == "" && prevSec == "" {
		return nil
	}
	if !CredentialSM4VersionPattern.MatchString(prevVer) {
		return fmt.Errorf("CREDENTIAL_SM4_PREVIOUS_VERSION 格式非法: %q", prevVer)
	}
	if prevVer == cur {
		return fmt.Errorf("CREDENTIAL_SM4_PREVIOUS_VERSION 必须不等于 CREDENTIAL_SM4_VERSION（否则轮换无意义）")
	}
	if len(prevSec) < 32 {
		return fmt.Errorf("CREDENTIAL_SM4_PREVIOUS_SECRET 必须 ≥ 32 字符（当前长度=%d）", len(prevSec))
	}
	if prevSec == sec {
		return fmt.Errorf("CREDENTIAL_SM4_PREVIOUS_SECRET 必须不等于 CREDENTIAL_SM4_SECRET（强制轮换）")
	}
	return nil
}

// ValidateVideoPlaybackTokenConfig 校验 video_playback_token 配置合法性。
// 镜像 ValidateCredentialSM4Config 的 fail-closed 语义：生产环境必须显式设置 ≥ 32
// 字符的密钥、且与 SM4Secret / HLSTokenSecret 必须互不相同（独立密钥族，密钥重用会
// 抹平 HMAC/SM4 之间的安全边界）。duration 必须 > 0（0/负 TTL 无意义且可能误签）。
func (c *Config) ValidateVideoPlaybackTokenConfig() error {
	sec := c.Auth.VideoPlaybackTokenSecret
	dur := c.Auth.VideoPlaybackTokenDuration

	if sec == "" {
		return fmt.Errorf("VIDEO_PLAYBACK_TOKEN_SECRET 必须显式设置")
	}
	if len(sec) < 32 {
		return fmt.Errorf("VIDEO_PLAYBACK_TOKEN_SECRET 必须 ≥ 32 字符（当前长度=%d）", len(sec))
	}
	if sec == c.Auth.SM4Secret {
		return fmt.Errorf("VIDEO_PLAYBACK_TOKEN_SECRET 必须不等于 SM4_SECRET（独立密钥族）")
	}
	if sec == c.Auth.HLSTokenSecret {
		return fmt.Errorf("VIDEO_PLAYBACK_TOKEN_SECRET 必须不等于 HLS_TOKEN_SECRET（独立密钥族）")
	}
	if dur <= 0 {
		return fmt.Errorf("video_playback_token_duration 必须 > 0（当前=%v）", dur)
	}
	return nil
}
