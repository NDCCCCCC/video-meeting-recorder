package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services/audit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// APIKeyService API密钥服务
type APIKeyService struct {
	db           *gorm.DB
	logger       *zap.Logger
	auditService *audit.AuditLogService
}

// NewAPIKeyService 创建API密钥服务
func NewAPIKeyService(db *gorm.DB, logger *zap.Logger) *APIKeyService {
	return &APIKeyService{
		db:     db,
		logger: logger,
	}
}

// SetAuditService 设置审计服务
func (s *APIKeyService) SetAuditService(auditService *audit.AuditLogService) {
	s.auditService = auditService
}

// findAPIKeyForUser 查询指定 ID 的 API Key，非管理员只能查自己的
func (s *APIKeyService) findAPIKeyForUser(id, userID uint, isAdmin bool) (*models.APIKey, error) {
	var apiKey models.APIKey
	query := s.db.Where("id = ?", id)
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("API密钥不存在")
		}
		return nil, err
	}
	return &apiKey, nil
}

// logAudit 记录审计日志
func (s *APIKeyService) logAudit(userID uint, resourceID uint, action, newData string) {
	if s.auditService == nil {
		return
	}
	s.auditService.LogOperation(nil, &audit.LogOperationRequest{
		UserID:     userID,
		Module:     "apikey",
		Resource:   "apikey",
		ResourceID: &resourceID,
		Action:     action,
		NewData:    newData,
		Status:     "success",
	})
}

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name         string   `json:"name" binding:"required,min=1,max=100"`
	ExpiresAt    *string  `json:"expires_at"`    // ISO 8601格式，null表示永久
	Scopes       []string `json:"scopes" binding:"required"`
	InheritPerms bool     `json:"inherit_perms"`
	IPWhitelist  []string `json:"ip_whitelist"`
	Description  string   `json:"description"`
}

// UpdateAPIKeyRequest 更新API密钥请求
type UpdateAPIKeyRequest struct {
	Name        *string   `json:"name"`
	IsActive    *bool     `json:"is_active"`
	Scopes      *[]string `json:"scopes"`
	IPWhitelist *[]string `json:"ip_whitelist"`
	Description *string   `json:"description"`
}

// ListAPIKeysRequest 列表查询请求
type ListAPIKeysRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Keyword  string `form:"keyword"`
	IsActive *bool  `form:"is_active"`
}

// ListAPIKeysResponse 列表响应
type ListAPIKeysResponse struct {
	Total int64             `json:"total"`
	Items []models.APIKey   `json:"items"`
}

// CreateAPIKey 创建API密钥
func (s *APIKeyService) CreateAPIKey(userID uint, req *CreateAPIKeyRequest) (*models.APIKey, string, error) {
	// 验证作用域
	for _, scope := range req.Scopes {
		if scope != models.ScopeRead && scope != models.ScopeWrite && scope != models.ScopeAdmin {
			return nil, "", fmt.Errorf("无效的作用域: %s", scope)
		}
	}

	// 解析过期时间
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, "", fmt.Errorf("无效的过期时间格式: %w", err)
		}
		if t.Before(time.Now()) {
			return nil, "", errors.New("过期时间不能早于当前时间")
		}
		expiresAt = &t
	}

	// 创建API密钥
	apiKey := &models.APIKey{
		Name:         req.Name,
		UserID:       userID,
		ExpiresAt:    expiresAt,
		InheritPerms: req.InheritPerms,
		IsActive:     true,
		Description:  req.Description,
	}

	// 设置作用域
	if err := apiKey.SetScopes(req.Scopes); err != nil {
		return nil, "", fmt.Errorf("设置作用域失败: %w", err)
	}

	// 设置IP白名单
	if req.IPWhitelist != nil {
		if err := apiKey.SetIPWhitelist(req.IPWhitelist); err != nil {
			return nil, "", fmt.Errorf("设置IP白名单失败: %w", err)
		}
	}

	// 生成密钥并保存
	if err := s.db.Create(apiKey).Error; err != nil {
		s.logger.Error("创建API密钥失败", zap.Error(err))
		return nil, "", errors.New("创建API密钥失败")
	}

	// 记录审计日志
	s.logAudit(userID, apiKey.ID, "create", fmt.Sprintf("创建API密钥: %s", req.Name))

	s.logger.Info("API密钥已创建",
		zap.Uint("user_id", userID),
		zap.Uint("key_id", apiKey.ID),
		zap.String("name", req.Name),
	)

	return apiKey, apiKey.Key, nil
}

// ListAPIKeys 获取API密钥列表
func (s *APIKeyService) ListAPIKeys(userID uint, isAdmin bool, req *ListAPIKeysRequest) (*ListAPIKeysResponse, error) {
	var apiKeys []models.APIKey
	var total int64

	query := s.db.Model(&models.APIKey{})

	// 非管理员只能查看自己的密钥
	if !isAdmin {
		query = query.Where("user_id = ?", userID)
	}

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+req.Keyword+"%")
	}

	// 状态筛选
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&apiKeys).Error; err != nil {
		return nil, err
	}

	return &ListAPIKeysResponse{
		Total: total,
		Items: apiKeys,
	}, nil
}

// GetAPIKey 获取API密钥详情
func (s *APIKeyService) GetAPIKey(id uint, userID uint, isAdmin bool) (*models.APIKey, error) {
	return s.findAPIKeyForUser(id, userID, isAdmin)
}

// UpdateAPIKey 更新API密钥
func (s *APIKeyService) UpdateAPIKey(id uint, userID uint, isAdmin bool, req *UpdateAPIKeyRequest) (*models.APIKey, error) {
	apiKey, err := s.findAPIKeyForUser(id, userID, isAdmin)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	oldValues, _ := json.Marshal(apiKey)

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Scopes != nil {
		if err := apiKey.SetScopes(*req.Scopes); err != nil {
			return nil, fmt.Errorf("设置作用域失败: %w", err)
		}
		updates["scopes"] = apiKey.Scopes
	}
	if req.IPWhitelist != nil {
		if err := apiKey.SetIPWhitelist(*req.IPWhitelist); err != nil {
			return nil, fmt.Errorf("设置IP白名单失败: %w", err)
		}
		updates["ip_whitelist"] = apiKey.IPWhitelist
	}

	if len(updates) > 0 {
		if err := s.db.Model(apiKey).Updates(updates).Error; err != nil {
			return nil, err
		}

		newValues, _ := json.Marshal(apiKey)
		s.logAudit(userID, apiKey.ID, "update", string(oldValues)+" -> "+string(newValues))

		s.logger.Info("API密钥已更新",
			zap.Uint("user_id", userID),
			zap.Uint("key_id", apiKey.ID),
		)
	}

	// 重新查询获取更新后的数据
	if err := s.db.First(apiKey, id).Error; err != nil {
		return nil, err
	}

	return apiKey, nil
}

// DeleteAPIKey 删除API密钥
func (s *APIKeyService) DeleteAPIKey(id uint, userID uint, isAdmin bool) error {
	apiKey, err := s.findAPIKeyForUser(id, userID, isAdmin)
	if err != nil {
		return err
	}

	if err := s.db.Delete(apiKey).Error; err != nil {
		return err
	}

	s.logAudit(userID, apiKey.ID, "delete", fmt.Sprintf("删除API密钥: %s", apiKey.Name))

	s.logger.Info("API密钥已删除",
		zap.Uint("user_id", userID),
		zap.Uint("key_id", apiKey.ID),
	)

	return nil
}

// ToggleAPIKeyStatus 切换API密钥状态
func (s *APIKeyService) ToggleAPIKeyStatus(id uint, userID uint, isAdmin bool) (*models.APIKey, error) {
	apiKey, err := s.findAPIKeyForUser(id, userID, isAdmin)
	if err != nil {
		return nil, err
	}

	apiKey.IsActive = !apiKey.IsActive
	if err := s.db.Save(apiKey).Error; err != nil {
		return nil, err
	}

	status := "禁用"
	if apiKey.IsActive {
		status = "启用"
	}
	s.logAudit(userID, apiKey.ID, "toggle", fmt.Sprintf("%sAPI密钥: %s", status, apiKey.Name))

	s.logger.Info("API密钥状态已切换",
		zap.Uint("user_id", userID),
		zap.Uint("key_id", apiKey.ID),
		zap.Bool("is_active", apiKey.IsActive),
	)

	return apiKey, nil
}

// ValidateAPIKey 验证API密钥（供中间件使用）
func (s *APIKeyService) ValidateAPIKey(key string, clientIP string) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := s.db.Preload("User").Preload("User.Role").Where("key = ?", key).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("无效的API密钥")
		}
		return nil, err
	}

	// 检查是否过期
	if apiKey.IsExpired() {
		return nil, errors.New("API密钥已过期")
	}

	// 检查是否启用
	if !apiKey.IsActive {
		return nil, errors.New("API密钥已禁用")
	}

	// 检查用户状态
	if !apiKey.User.IsActive {
		return nil, errors.New("用户已被禁用")
	}

	// 检查IP白名单
	if !apiKey.IsIPAllowed(clientIP) {
		return nil, errors.New("IP地址不在白名单中")
	}

	// 更新最后使用时间
	now := time.Now()
	apiKey.LastUsedAt = &now
	s.db.Save(&apiKey)

	return &apiKey, nil
}

// ListUsageLogsRequest 使用日志查询请求
type ListUsageLogsRequest struct {
	Page         int    `form:"page" binding:"min=1"`
	PageSize     int    `form:"page_size" binding:"min=1,max=100"`
	APIKeyID     *uint  `form:"api_key_id"`     // 可选，筛选特定密钥
	Success      *bool  `form:"success"`        // 可选，筛选成功/失败
	StartTime    string `form:"start_time"`     // 可选，开始时间 ISO 8601
	EndTime      string `form:"end_time"`       // 可选，结束时间 ISO 8601
	Method       string `form:"method"`         // 可选，HTTP 方法
}

// UsageLogSummary 使用日志统计
type UsageLogSummary struct {
	TotalRequests    int64 `json:"total_requests"`
	SuccessRequests  int64 `json:"success_requests"`
	FailedRequests   int64 `json:"failed_requests"`
	AvgDuration      int64 `json:"avg_duration"`    // 毫秒
	MaxDuration      int64 `json:"max_duration"`    // 毫秒
	UniqueIPs        int64 `json:"unique_ips"`
	TodayRequests    int64 `json:"today_requests"`
}

// ListUsageLogs 获取 API Key 使用日志
func (s *APIKeyService) ListUsageLogs(userID uint, isAdmin bool, apiKeyID uint, req *ListUsageLogsRequest) ([]models.APIKeyUsageLog, int64, error) {
	var logs []models.APIKeyUsageLog
	var total int64

	query := s.db.Model(&models.APIKeyUsageLog{})

	// 如果指定了 API Key ID，只查询该密钥的日志
	if apiKeyID > 0 {
		// 验证用户是否有权限查看该密钥的日志
		var apiKey models.APIKey
		keyQuery := s.db.Where("id = ?", apiKeyID)
		if !isAdmin {
			keyQuery = keyQuery.Where("user_id = ?", userID)
		}
		if err := keyQuery.First(&apiKey).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, errors.New("API密钥不存在")
			}
			return nil, 0, err
		}
		query = query.Where("api_key_id = ?", apiKeyID)
	} else {
		// 非管理员只能查看自己密钥的日志
		if !isAdmin {
			// 获取用户的所有密钥 ID
			var keyIDs []uint
			if err := s.db.Model(&models.APIKey{}).Where("user_id = ?", userID).Pluck("id", &keyIDs).Error; err != nil {
				return nil, 0, err
			}
			if len(keyIDs) == 0 {
				return []models.APIKeyUsageLog{}, 0, nil
			}
			query = query.Where("api_key_id IN ?", keyIDs)
		}
	}

	// 可选筛选条件
	if req.Success != nil {
		query = query.Where("success = ?", *req.Success)
	}
	if req.Method != "" {
		query = query.Where("method = ?", req.Method)
	}
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetUsageLogSummary 获取 API Key 使用统计
func (s *APIKeyService) GetUsageLogSummary(userID uint, isAdmin bool, apiKeyID uint) (*UsageLogSummary, error) {
	// 先构建筛选条件
	conditions := s.buildUsageLogConditions(userID, isAdmin, apiKeyID)
	if conditions == nil {
		return &UsageLogSummary{}, nil
	}

	summary := &UsageLogSummary{}

	// 总请求数
	conditions.Count(&summary.TotalRequests)

	// 成功请求数
	s.db.Model(&models.APIKeyUsageLog{}).
		Where(conditions).
		Where("success = ?", true).
		Count(&summary.SuccessRequests)

	// 失败请求数
	s.db.Model(&models.APIKeyUsageLog{}).
		Where(conditions).
		Where("success = ?", false).
		Count(&summary.FailedRequests)

	// 平均/最大响应时间
	type StatsResult struct {
		AvgDuration *int64
		MaxDuration *int64
	}
	var stats StatsResult
	s.db.Model(&models.APIKeyUsageLog{}).
		Select("AVG(duration) as avg_duration, MAX(duration) as max_duration").
		Where(conditions).
		Scan(&stats)
	if stats.AvgDuration != nil {
		summary.AvgDuration = *stats.AvgDuration
	}
	if stats.MaxDuration != nil {
		summary.MaxDuration = *stats.MaxDuration
	}

	// 唯一 IP 数量
	s.db.Model(&models.APIKeyUsageLog{}).
		Where(conditions).
		Distinct("client_ip").
		Count(&summary.UniqueIPs)

	// 今日请求数
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Model(&models.APIKeyUsageLog{}).
		Where(conditions).
		Where("created_at >= ?", today).
		Count(&summary.TodayRequests)

	return summary, nil
}

// buildUsageLogConditions 构建使用日志查询的条件
// 返回 nil 表示无需查询（没有匹配的密钥）
func (s *APIKeyService) buildUsageLogConditions(userID uint, isAdmin bool, apiKeyID uint) *gorm.DB {
	query := s.db.Model(&models.APIKeyUsageLog{})

	if apiKeyID > 0 {
		// 验证用户是否有权限查看该密钥
		var apiKey models.APIKey
		keyQuery := s.db.Where("id = ?", apiKeyID)
		if !isAdmin {
			keyQuery = keyQuery.Where("user_id = ?", userID)
		}
		if err := keyQuery.First(&apiKey).Error; err != nil {
			return nil
		}
		query = query.Where("api_key_id = ?", apiKeyID)
	} else {
		// 非管理员只能查看自己密钥的统计
		if !isAdmin {
			var keyIDs []uint
			if err := s.db.Model(&models.APIKey{}).Where("user_id = ?", userID).Pluck("id", &keyIDs).Error; err != nil {
				return nil
			}
			if len(keyIDs) == 0 {
				return nil
			}
			query = query.Where("api_key_id IN ?", keyIDs)
		}
	}

	return query
}
