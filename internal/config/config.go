package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server" json:"server" yaml:"server"`
	Database DatabaseConfig `mapstructure:"database" json:"database" yaml:"database"`
	Auth     AuthConfig     `mapstructure:"auth" json:"auth" yaml:"auth"`
	Logging  LoggingConfig  `mapstructure:"logging" json:"logging" yaml:"logging"`
	Storage  StorageConfig  `mapstructure:"storage" json:"storage" yaml:"storage"`
	Huawei   HuaweiConfig   `mapstructure:"huawei" json:"huawei" yaml:"huawei"`
	RTSP     RTSPConfig     `mapstructure:"rtsp" json:"rtsp" yaml:"rtsp"`
	FFmpeg   FFmpegConfig   `mapstructure:"ffmpeg" json:"ffmpeg" yaml:"ffmpeg"`
	OSS      OSSConfig      `mapstructure:"oss" json:"oss" yaml:"oss"`
	Tingwu   TingwuConfig   `mapstructure:"tingwu" json:"tingwu" yaml:"tingwu"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string `mapstructure:"host" json:"host" yaml:"host"`
	Port         int    `mapstructure:"port" json:"port" yaml:"port"`
	Environment  string `mapstructure:"environment" json:"environment" yaml:"environment"`
	ReadTimeout  int    `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`
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
	UploadTimeout   int    `mapstructure:"upload_timeout" json:"upload_timeout" yaml:"upload_timeout"`       // seconds
	PresignedURLTTL int    `mapstructure:"presigned_url_ttl" json:"presigned_url_ttl" yaml:"presigned_url_ttl"` // seconds, default 86400
}

// TingwuConfig 阿里通义听悟配置
type TingwuConfig struct {
	AppKey    string `mapstructure:"app_key" json:"app_key" yaml:"app_key"`
	AppSecret string `mapstructure:"app_secret" json:"-" yaml:"app_secret"`
	BaseURL   string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
	Enabled   bool   `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	APITimeout int   `mapstructure:"api_timeout" json:"api_timeout" yaml:"api_timeout"` // seconds
}

// expandEnvWithDefault 展开环境变量，支持 ${VAR:default} 格式
func expandEnvWithDefault(s string) string {
	// 匹配 ${VAR:default} 或 ${VAR} 格式
	re := regexp.MustCompile(`\$\{([^:}]+)(?::([^}]*))?\}`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
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

// expandConfig 递归展开配置中的所有字符串值
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
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err := createDefaultConfigFile(); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			// 重新读取配置文件
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config after creation: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// 获取原始配置
	var rawCfg map[string]interface{}
	if err := v.Unmarshal(&rawCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw config: %w", err)
	}

	// 展开环境变量
	expandedCfg := expandConfig(rawCfg)

	// 将展开后的配置转换回 viper
	v = viper.New()
	v.SetConfigType("yaml")
	if err := v.MergeConfigMap(expandedCfg.(map[string]interface{})); err != nil {
		return nil, fmt.Errorf("failed to merge expanded config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 设置默认值
	setDefaults(&cfg)

	// 创建必要的目录
	if err := ensureDirectories(&cfg); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults(cfg *Config) {
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
		cfg.Database.MaxOpenConns = 1
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 1
	}

	if cfg.Auth.SM4Secret == "" {
		cfg.Auth.SM4Secret = "change-me-in-production"
	}
	if cfg.Auth.AccessTokenDuration == 0 {
		cfg.Auth.AccessTokenDuration = 2 * time.Hour
	}
	if cfg.Auth.RefreshTokenDuration == 0 {
		cfg.Auth.RefreshTokenDuration = 7 * 24 * time.Hour
	}
	if cfg.Auth.MaxSessionDuration == 0 {
		cfg.Auth.MaxSessionDuration = 30 * 24 * time.Hour
	}
	// HLS Token 默认值：使用与SM4相同的密钥，5分钟有效期
	if cfg.Auth.HLSTokenSecret == "" {
		cfg.Auth.HLSTokenSecret = cfg.Auth.SM4Secret
	}
	if cfg.Auth.HLSTokenDuration == 0 {
		cfg.Auth.HLSTokenDuration = 5 * time.Minute
	}

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
		cfg.Huawei.MinTLSVersion = "1.0"
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
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
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
  sm4_secret: "change-me-in-production-please-set-a-secure-random-key"
  access_token_duration: "2h"
  refresh_token_duration: "168h"  # 7 days
  max_session_duration: "720h"   # 30 days
  hls_token_secret: "change-me-in-production"
  hls_token_duration: "5m"

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
  min_tls_version: "1.0"

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
`

	// 写入配置文件
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
