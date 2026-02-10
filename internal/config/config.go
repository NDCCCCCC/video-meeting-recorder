package config

import (
	"fmt"
	"os"
	"path/filepath"
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
	Path            string        `mapstructure:"path" json:"path" yaml:"path"`
	MaxProcesses    int           `mapstructure:"max_processes" json:"max_processes" yaml:"max_processes"`
	Timeout         time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	DefaultCodec    string        `mapstructure:"default_codec" json:"default_codec" yaml:"default_codec"`
	DefaultFormat   string        `mapstructure:"default_format" json:"default_format" yaml:"default_format"`
	DefaultVideoBitrate string    `mapstructure:"default_video_bitrate" json:"default_video_bitrate" yaml:"default_video_bitrate"`
	DefaultAudioBitrate string    `mapstructure:"default_audio_bitrate" json:"default_audio_bitrate" yaml:"default_audio_bitrate"`
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

	if cfg.Storage.RecordingsPath == "" {
		cfg.Storage.RecordingsPath = "./data/recordings"
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
}

// ensureDirectories 确保目录存在
func ensureDirectories(cfg *Config) error {
	dirs := []string{
		filepath.Dir(cfg.Database.Path),
		cfg.Storage.RecordingsPath,
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
