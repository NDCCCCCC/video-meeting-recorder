package services

import (
	"encoding/base64"
	"encoding/json"

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

// ConfigService handles loading/saving system configuration from/to database
type ConfigService struct {
	db     *gorm.DB
	logger *zap.Logger
	cfg    *config.Config // live config reference
}

// NewConfigService creates a new ConfigService
func NewConfigService(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *ConfigService {
	return &ConfigService{
		db:     db,
		logger: logger,
		cfg:    cfg,
	}
}

// LoadAuthConfig loads auth config from database, overriding YAML defaults
func (s *ConfigService) LoadAuthConfig() error {
	// Load mode
	var modeSetting models.SystemSetting
	if err := s.db.Where("`key` = ?", keyAuthMode).First(&modeSetting).Error; err == nil {
		s.cfg.Auth.Mode = modeSetting.Value
		s.logger.Info("Loaded auth mode from database", zap.String("mode", s.cfg.Auth.Mode))
	}

	// Load AD config
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
	var pwdSetting models.SystemSetting
	if err := s.db.Where("`key` = ?", keyAuthADPassword).First(&pwdSetting).Error; err == nil {
		decrypted, err := s.decryptPassword(pwdSetting.Value)
		if err != nil {
			s.logger.Warn("Failed to decrypt AD password from database", zap.Error(err))
			return err
		}
		s.cfg.Auth.AD.Password = decrypted
		s.logger.Info("Loaded AD password from database")
	}

	return nil
}

// SaveAuthConfig saves auth config to database
func (s *ConfigService) SaveAuthConfig(mode string, adConfig *auth.ADAuthConfig) error {
	// Save mode
	modeSetting := models.SystemSetting{Key: keyAuthMode, Value: mode}
	s.db.Where("`key` = ?", keyAuthMode).Assign(&modeSetting).Save(&modeSetting)

	// Save AD config (without password)
	adConfigForDB := *adConfig
	adConfigJSON, _ := json.Marshal(adConfigForDB)
	adSetting := models.SystemSetting{Key: keyAuthAD, Value: string(adConfigJSON)}
	s.db.Where("`key` = ?", keyAuthAD).Assign(&adSetting).Save(&adSetting)

	// Encrypt and save password
	if adConfig.Password != "" {
		encrypted, err := s.encryptPassword(adConfig.Password)
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

// encryptPassword encrypts password using base64 (TODO: upgrade to SM4-GCM)
func (s *ConfigService) encryptPassword(password string) (string, error) {
	// TODO: Implement SM4-GCM encryption for password at rest
	// For now, use base64 encoding (password is still encrypted in transit via SM4-ECB)
	return base64.StdEncoding.EncodeToString([]byte(password)), nil
}

// decryptPassword decrypts password from database storage
func (s *ConfigService) decryptPassword(encrypted string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
