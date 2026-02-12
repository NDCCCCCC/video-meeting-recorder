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
	Server    ServerConfig    `mapstructure:"server" json:"server" yaml:"server"`
	Database  DatabaseConfig  `mapstructure:"database" json:"database" yaml:"database"`
	Auth      AuthConfig      `mapstructure:"auth" json:"auth" yaml:"auth"`
	Logging   LoggingConfig   `mapstructure:"logging" json:"logging" yaml:"logging"`
	Storage   StorageConfig   `mapstructure:"storage" json:"storage" yaml:"storage"`
	Huawei    HuaweiConfig    `mapstructure:"huawei" json:"huawei" yaml:"huawei"`
	RTSP      RTSPConfig      `mapstructure:"rtsp" json:"rtsp" yaml:"rtsp"`
	FFmpeg    FFmpegConfig    `mapstructure:"ffmpeg" json:"ffmpeg" yaml:"ffmpeg"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host        string `mapstructure:"host" json:"host" yaml:"host"`
	Port        int    `mapstructure:"port" json:"port" yaml:"port"`
	Environment string `mapstructure:"environment" json:"environment" yaml:"environment"`
	ReadTimeout int    `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout int   `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`
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
	ConnMaxLifetime   int    `mapstructure:"conn_max_lifetime" json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret" json:"jwt_secret" yaml:"jwt_secret"`
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
	HLSPath        string `mapstructure:"hls_path" json:"hls_path" yaml:"hls_path"`        // HLS预览文件存储路径
	TempPath       string `mapstructure:"temp_path" json:"temp_path" yaml:"temp_path"`
	MaxDiskUsage   int    `mapstructure:"max_disk_usage" json:"max_disk_usage" yaml:"max_disk_usage"`

	// 文件存储配置
	Local          LocalStorageConfig `mapstructure:"local" json:"local" yaml:"local"`
	MaxFileSize    int64              `mapstructure:"max_file_size" json:"max_file_size" yaml:"max_file_size"`
	AllowedExtensions []string        `mapstructure:"allowed_extensions" json:"allowed_extensions" yaml:"allowed_extensions"`
}

// LocalStorageConfig 本地存储配置
type LocalStorageConfig struct {
	BasePath string `mapstructure:"base_path" json:"base_path" yaml:"base_path"`
	BaseURL  string `mapstructure:"base_url" json:"base_url" yaml:"base_url"`
}

// HuaweiConfig 华为会议系统配置
type HuaweiConfig struct {
	ConferenceServer string        `mapstructure:"conference_server" json:"conference_server" yaml:"conference_server"`
	ConferencePort   int           `mapstructure:"conference_port" json:"conference_port" yaml:"conference_port"`
	Username         string        `mapstructure:"username" json:"username" yaml:"username"`
	Password         string        `mapstructure:"password" json:"password" yaml:"password"`
	HTTPS            bool          `mapstructure:"https" json:"https" yaml:"https"`
	InsecureSkipVerify bool         `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	APITimeout       time.Duration `mapstructure:"api_timeout" json:"api_timeout" yaml:"api_timeout"`
	SessionTimeout   time.Duration `mapstructure:"session_timeout" json:"session_timeout" yaml:"session_timeout"`
	KeepAliveInterval time.Duration `mapstructure:"keep_alive_interval" json:"keep_alive_interval" yaml:"keep_alive_interval"`
	MinTLSVersion    string        `mapstructure:"min_tls_version" json:"min_tls_version" yaml:"min_tls_version"`
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
	Path              string        `mapstructure:"path" json:"path" yaml:"path"`
	MaxProcesses      int           `mapstructure:"max_processes" json:"max_processes" yaml:"max_processes"`
	Timeout           time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	DefaultCodec      string        `mapstructure:"default_codec" json:"default_codec" yaml:"default_codec"`
	DefaultFormat     string        `mapstructure:"default_format" json:"default_format" yaml:"default_format"`
	DefaultVideoBitrate string      `mapstructure:"default_video_bitrate" json:"default_video_bitrate" yaml:"default_video_bitrate"`
	DefaultAudioBitrate string      `mapstructure:"default_audio_bitrate" json:"default_audio_bitrate" yaml:"default_audio_bitrate"`
	// DShow 设备配置
	DShowBufferSize   int    `mapstructure:"dshow_buffer_size" json:"dshow_buffer_size" yaml:"dshow_buffer_size"`   // 实时缓冲区大小（字节）
	DShowThreadQueueSize int  `mapstructure:"dshow_thread_queue_size" json:"dshow_thread_queue_size" yaml:"dshow_thread_queue_size"` // 线程队列大小
	// HLS 配置
	HLSSegmentDuration int    `mapstructure:"hls_segment_duration" json:"hls_segment_duration" yaml:"hls_segment_duration"` // HLS 分片时长（秒）
	HLSListSize       int    `mapstructure:"hls_list_size" json:"hls_list_size" yaml:"hls_list_size"`                 // HLS 播放列表保留分片数
	// 录制监控配置
	MaxRecordingDuration time.Duration `mapstructure:"max_recording_duration" json:"max_recording_duration" yaml:"max_recording_duration"` // 最长录制时长
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

	// 设置配置文件
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	// 环境变量支持
	v.AutomaticEnv()
	v.SetEnvPrefix("RECORD")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
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

	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = "change-me-in-production"
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
	// HLS Token 默认值：使用与 JWT 相同的密钥，5分钟有效期
	if cfg.Auth.HLSTokenSecret == "" {
		cfg.Auth.HLSTokenSecret = cfg.Auth.JWTSecret
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

	// 将相对路径转换为绝对路径，避免工作目录问题
	if cfg.Storage.RecordingsPath == "" {
		cfg.Storage.RecordingsPath = "./data/recordings"
	}
	if !filepath.IsAbs(cfg.Storage.RecordingsPath) {
		absPath, err := filepath.Abs(cfg.Storage.RecordingsPath)
		if err == nil {
			cfg.Storage.RecordingsPath = absPath
		}
	}

	if cfg.Storage.HLSPath == "" {
		cfg.Storage.HLSPath = "./data/hls"
	}
	if !filepath.IsAbs(cfg.Storage.HLSPath) {
		absPath, err := filepath.Abs(cfg.Storage.HLSPath)
		if err == nil {
			cfg.Storage.HLSPath = absPath
		}
	}

	if cfg.Storage.TempPath == "" {
		cfg.Storage.TempPath = "./data/temp"
	}
	if !filepath.IsAbs(cfg.Storage.TempPath) {
		absPath, err := filepath.Abs(cfg.Storage.TempPath)
		if err == nil {
			cfg.Storage.TempPath = absPath
		}
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
		cfg.FFmpeg.Path = "ffmpeg"
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
