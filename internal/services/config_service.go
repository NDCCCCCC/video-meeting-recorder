package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// Setting keys for auth config persistence
	keyAuthMode              = "auth.mode"
	keyAuthAD                = "auth.ad"
	keyAuthADPassword        = "auth.ad.password"
	keyAuthADAllowAutoCreate = "auth.ad.allow_auto_create"
)

// authADForDB 是 system_settings['auth.ad'] JSON 行专用的 DTO。
//
// 设计动机：auth.ADAuthConfig 本身有一个 json:"password" 字段（保留用于环境变量 / YAML 解析）；
// 而 system_settings['auth.ad'] 的 JSON 行必须**永远不包含** password 字段（Phase 18 起）。
// 本 DTO 显式 json:"-" 屏蔽，避免结构体序列化时把 password 字段混入 DB JSON 行。
//
// 任何 adConfigForDB(password ...string) → 不带 password 的 DTO；保留原 AD 字段全集。
type authADForDB struct {
	Server             string `json:"server"`
	BindDN             string `json:"bind_dn"`
	BaseDN             string `json:"base_dn"`
	UseTLS             bool   `json:"use_tls"`
	PoolSize           int    `json:"pool_size"`
	DialTimeout        int    `json:"dial_timeout"`
	RequestTimeout     int    `json:"request_timeout"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
	AllowAutoCreate    bool   `json:"allow_auto_create"`
}

// toADAuthConfigForDB 把 auth.ADAuthConfig 转为不带 password 的 DTO。
func toADAuthConfigForDB(c *auth.ADAuthConfig) authADForDB {
	return authADForDB{
		Server:             c.Server,
		BindDN:             c.BindDN,
		BaseDN:             c.BaseDN,
		UseTLS:             c.UseTLS,
		PoolSize:           c.PoolSize,
		DialTimeout:        c.DialTimeout,
		RequestTimeout:     c.RequestTimeout,
		InsecureSkipVerify: c.InsecureSkipVerify,
		AllowAutoCreate:    c.AllowAutoCreate,
	}
}

// ConfigService handles loading/saving system configuration from/to database
type ConfigService struct {
	db        *gorm.DB
	logger    *zap.Logger
	cfg       *config.Config       // live config reference
	encryptor *CredentialEncryptor // Phase 18: 凭据静态加密器（可为 nil，向后兼容测试）
}

// NewConfigService creates a new ConfigService
// encryptor 可为 nil（用于早期启动或测试场景）；nil 时 SaveAuthConfig/LoadAuthConfig
// 会显式报错，因为 Phase 18 后不允许凭据以 base64-stub 或明文落库。
func NewConfigService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, encryptor *CredentialEncryptor) *ConfigService {
	return &ConfigService{
		db:        db,
		logger:    logger,
		cfg:       cfg,
		encryptor: encryptor,
	}
}

// LoadAuthConfig loads auth config from database, overriding YAML defaults
func (s *ConfigService) LoadAuthConfig() error {
	if s.encryptor == nil {
		return errors.New("ConfigService.encryptor 未注入；LoadAuthConfig 需要 CredentialEncryptor（Phase 18 强制要求）")
	}

	// Load mode
	var modeSetting models.SystemSetting
	if err := s.db.Where("`key` = ?", keyAuthMode).First(&modeSetting).Error; err == nil {
		s.cfg.Auth.Mode = modeSetting.Value
		s.logger.Info("Loaded auth mode from database", zap.String("mode", s.cfg.Auth.Mode))
	}

	// Load AD config（JSON 不含 password 字段——Phase 18 invariant 已强制保证）
	var adSetting models.SystemSetting
	if err := s.db.Where("`key` = ?", keyAuthAD).First(&adSetting).Error; err == nil {
		var adConfig auth.ADAuthConfig
		if err := json.Unmarshal([]byte(adSetting.Value), &adConfig); err != nil {
			s.logger.Error("Failed to unmarshal AD config from database", zap.Error(err))
			return err
		}
		s.cfg.Auth.AD.Server = adConfig.Server
		s.cfg.Auth.AD.BindDN = adConfig.BindDN
		s.cfg.Auth.AD.BaseDN = adConfig.BaseDN
		s.cfg.Auth.AD.UseTLS = adConfig.UseTLS
		s.cfg.Auth.AD.PoolSize = adConfig.PoolSize
		s.cfg.Auth.AD.DialTimeout = adConfig.DialTimeout
		s.cfg.Auth.AD.RequestTimeout = adConfig.RequestTimeout
		s.cfg.Auth.AD.InsecureSkipVerify = adConfig.InsecureSkipVerify
		s.cfg.Auth.AD.AllowAutoCreate = adConfig.AllowAutoCreate
		s.logger.Info("Loaded AD config from database")
	}

	// Load and decrypt AD password
	// Phase 18: 解密失败 → logger.Fatal 终止启动（不能 warn-and-continue）
	var pwdSetting models.SystemSetting
	if err := s.db.Where("`key` = ?", keyAuthADPassword).First(&pwdSetting).Error; err == nil {
		decrypted, err := s.encryptor.Decrypt(pwdSetting.Value)
		if err != nil {
			// Per Phase 18 spec: decrypt error → logger.Fatal（不是 warn-and-continue）
			s.logger.Fatal("Failed to decrypt AD password from database (Phase 18 fail-closed)",
				zap.Uint("setting_id", pwdSetting.ID),
				zap.Error(err),
			)
			return fmt.Errorf("AD password 解密失败（进程已 Fatal）: %w", err)
		}
		s.cfg.Auth.AD.Password = decrypted
		s.logger.Info("Loaded AD password from database")
	}

	return nil
}

// SaveAuthConfig saves auth config to database.
//
// Phase 18 关键变化：
//   - system_settings['auth.ad'] JSON 行**必须不包含 password 字段** → 使用 authADForDB DTO
//   - system_settings['auth.ad.password'] 独立一行存密文 envelope
func (s *ConfigService) SaveAuthConfig(mode string, adConfig *auth.ADAuthConfig) error {
	if s.encryptor == nil {
		return errors.New("ConfigService.encryptor 未注入；SaveAuthConfig 需要 CredentialEncryptor（Phase 18 强制要求）")
	}

	// Save mode
	modeSetting := models.SystemSetting{Key: keyAuthMode, Value: mode}
	s.db.Where("`key` = ?", keyAuthMode).Assign(&modeSetting).Save(&modeSetting)

	// Save AD config — 用专用 DTO 显式剥离 password 字段
	adDB := toADAuthConfigForDB(adConfig)
	adConfigJSON, err := json.Marshal(adDB)
	if err != nil {
		return fmt.Errorf("序列化 AD config JSON 失败: %w", err)
	}
	adSetting := models.SystemSetting{Key: keyAuthAD, Value: string(adConfigJSON)}
	s.db.Where("`key` = ?", keyAuthAD).Assign(&adSetting).Save(&adSetting)

	// Encrypt and save password (envelope)
	if adConfig.Password != "" {
		encrypted, err := s.encryptor.Encrypt(adConfig.Password)
		if err != nil {
			s.logger.Error("Failed to encrypt AD password for database", zap.Error(err))
			return err
		}
		pwdSetting := models.SystemSetting{Key: keyAuthADPassword, Value: encrypted}
		s.db.Where("`key` = ?", keyAuthADPassword).Assign(&pwdSetting).Save(&pwdSetting)
	}

	s.logger.Info("Saved auth config to database")
	return nil
}
