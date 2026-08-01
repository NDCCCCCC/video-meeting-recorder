package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 通知服务（STYLE-009：去掉包名冗余，外部引用统一为 notification.Service）。
// HISTORY: 旧名 NotificationService 作为别名保留若干版本，现已全量切换为 Service，别名移除。
type Service struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config

	// BUG-005/PR-D: 后台 worker 用 ctx (processQueue / handleMessage / sweeper)。
	// 由 New 派生子 ctx,Stop() 取消,使 SQL 能被级联取消。
	bgCtx    context.Context
	bgCancel context.CancelFunc

	// 消息队列
	queue  chan *models.NotificationMessage
	stopCh chan struct{}
}

// SendNotificationRequest 发送通知请求
type SendNotificationRequest struct {
	UserID      uint                         `json:"user_id"`
	Type        models.NotificationType      `json:"type"`
	Title       string                       `json:"title,omitempty"`
	Content     string                       `json:"content,omitempty"`
	Data        map[string]interface{}       `json:"data,omitempty"`
	Channels    []models.NotificationChannel `json:"channels"`
	RelatedID   *uint                        `json:"related_id,omitempty"`
	RelatedType string                       `json:"related_type,omitempty"`
	RelatedURL  string                       `json:"related_url,omitempty"`
}

// QueryRequest 查询请求
type QueryRequest struct {
	Type     string `form:"type"`
	IsRead   *bool  `form:"is_read"`
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	Items      []*models.NotificationMessage `json:"items"`
	Total      int64                         `json:"total"`
	Page       int                           `json:"page"`
	PageSize   int                           `json:"page_size"`
	TotalPages int                           `json:"total_pages"`
}

// UnreadCountResponse 未读数量响应
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// New 创建通知服务（STYLE-009：导出构造器统一为 package.New）。
func New(db *gorm.DB, logger *zap.Logger, config *config.Config) *Service {
	bgCtx, bgCancel := context.WithCancel(context.Background())
	service := &Service{
		db:       db,
		logger:   logger,
		config:   config,
		bgCtx:    bgCtx,
		bgCancel: bgCancel,
		queue:    make(chan *models.NotificationMessage, 1000),
		// PERF-014: 无缓冲 channel 仅用作 stop 信号 (close-only)
		// — 关闭后 processQueue 在 select 中收到零值并退出。
		stopCh: make(chan struct{}),
	}

	// BUG-005/PR-D: 启动处理goroutine 并把 bgCtx 传入,SQL 可被 ctx 取消
	go service.processQueue(bgCtx)

	return service
}

// Stop 停止服务
func (s *Service) Stop() {
	// 先关闭队列，停止接收新消息
	close(s.queue)
	// 再通知所有 worker 停止
	close(s.stopCh)
	// BUG-005/PR-D: 取消后台 ctx,让进行中 SQL 也能级联取消
	if s.bgCancel != nil {
		s.bgCancel()
	}
}

// bgCtxOrBackground 返回当前服务持有的后台 ctx（若已 Stop 则降级为 Background()）。
func (s *Service) bgCtxOrBackground() context.Context {
	if s.bgCtx != nil {
		return s.bgCtx
	}
	return context.Background()
}

// SendNotification 发送通知
func (s *Service) SendNotification(ctx context.Context, req *SendNotificationRequest) error {
	// 1. 获取用户通知配置
	setting, err := s.getUserSetting(req.UserID)
	if err != nil {
		return err
	}

	// 2. 检查免打扰时段
	if s.isQuietHours(setting) {
		// 记录日志并跳过（实际项目中可以延迟发送）
		s.logger.Info("在免打扰时段，跳过发送",
			zap.Uint("user_id", req.UserID),
		)
	}

	// 3. 检查频率限制
	if !s.checkRateLimit(ctx, req.UserID, setting) {
		return fmt.Errorf("超过频率限制: %w", apperrors.ErrInsufficientQuota)
	}

	// 4. 检查通知类型是否启用
	if !s.isTypeEnabled(setting, req.Type) {
		return fmt.Errorf("通知类型已禁用: %w", apperrors.ErrServiceUnavailable)
	}

	// 5. 构建消息
	message := &models.NotificationMessage{
		UserID:        req.UserID,
		Type:          req.Type,
		Title:         req.Title,
		Content:       req.Content,
		RelatedID:     req.RelatedID,
		RelatedType:   req.RelatedType,
		RelatedURL:    req.RelatedURL,
		IsRead:        false,
		ChannelStatus: "{}",
		CreatedAt:     time.Now(),
	}

	if req.Data != nil {
		dataJSON, _ := json.Marshal(req.Data)
		message.Data = string(dataJSON)
	}

	// 6. 保存到数据库（BUG-005/PR-D: 传播 ctx 以使客户端断开时 SQL 也能级联取消）
	if err := s.db.WithContext(ctx).Create(message).Error; err != nil {
		return fmt.Errorf("保存消息失败: %w: %w", apperrors.ErrInternal, err)
	}

	// 7. 放入发送队列（异步处理）
	select {
	case s.queue <- message:
	default:
		s.logger.Warn("通知队列已满")
	}

	return nil
}

// processQueue 处理发送队列 - 使用 worker pool 模式
func (s *Service) processQueue(bgCtx context.Context) {
	const numWorkers = 5
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case msg, ok := <-s.queue:
					if !ok {
						return
					}
					s.handleMessage(msg)
				case <-s.stopCh:
					return
				case <-bgCtx.Done():
					return
				}
			}
		}()
	}

	// 等待所有 worker 完成
	wg.Wait()
}

// handleMessage 处理消息
func (s *Service) handleMessage(msg *models.NotificationMessage) {
	channelStatus := msg.GetChannelStatusMap()

	// 站内消息始终已发送
	channelStatus["system"] = models.ChannelStatusSent

	// 这里可以添加其他渠道的发送逻辑
	// 例如：邮件、短信、钉钉等

	// 更新状态（BUG-005/PR-D: 走 bgCtx —— processQueue 派生的 worker 上下文）
	bgCtx, cancel := context.WithTimeout(s.bgCtxOrBackground(), 10*time.Second)
	defer cancel()
	msg.SetChannelStatusMap(channelStatus)
	s.db.WithContext(bgCtx).Model(msg).Update("channel_status", msg.ChannelStatus)

	s.logger.Info("消息处理完成",
		zap.Uint("message_id", msg.ID),
		zap.Uint("user_id", msg.UserID),
		zap.String("title", msg.Title),
	)
}

// Query 查询通知列表
func (s *Service) Query(ctx context.Context, userID uint, req *QueryRequest) (*QueryResponse, error) {
	query := s.db.WithContext(ctx).Model(&models.NotificationMessage{}).
		Where("user_id = ?", userID)

	// 筛选条件
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}
	if req.IsRead != nil {
		query = query.Where("is_read = ?", *req.IsRead)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	var messages []*models.NotificationMessage
	err := query.Order("created_at DESC").
		Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &QueryResponse{
		Items:      messages,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// MarkAsRead 标记为已读
func (s *Service) MarkAsRead(ctx context.Context, messageID, userID uint) error {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&models.NotificationMessage{}).
		Where("id = ? AND user_id = ?", messageID, userID).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("消息不存在: %w", apperrors.ErrNotFound)
	}

	return nil
}

// MarkAllAsRead 全部标记为已读
func (s *Service) MarkAllAsRead(ctx context.Context, userID uint) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&models.NotificationMessage{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		}).Error
}

// GetUnreadCount 获取未读数量
func (s *Service) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.NotificationMessage{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// GetUserSetting 获取用户通知配置
func (s *Service) GetUserSetting(ctx context.Context, userID uint) (*models.UserNotificationSetting, error) {
	var setting models.UserNotificationSetting
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回默认配置
			return s.getDefaultSetting(userID), nil
		}
		return nil, err
	}
	return &setting, nil
}

// UpdateUserSetting 更新用户通知配置
func (s *Service) UpdateUserSetting(ctx context.Context, userID uint, setting *models.UserNotificationSetting) (*models.UserNotificationSetting, *models.UserNotificationSetting, error) {
	// Snapshot pre-update state for audit OldData capture
	var oldSetting models.UserNotificationSetting
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&oldSetting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Use default setting snapshot when no existing row
			defaultSetting := s.getDefaultSetting(userID)
			oldSetting = *defaultSetting
		} else {
			return nil, nil, err
		}
	}

	setting.UserID = userID
	if err := s.db.WithContext(ctx).Save(setting).Error; err != nil {
		return nil, nil, err
	}

	// Reload committed state for NewData
	var newSetting models.UserNotificationSetting
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&newSetting).Error; err != nil {
		return nil, nil, err
	}

	return &oldSetting, &newSetting, nil
}

// getUserSetting 获取用户通知配置（内部方法）
func (s *Service) getUserSetting(userID uint) (*models.UserNotificationSetting, error) {
	var setting models.UserNotificationSetting
	err := s.db.Where("user_id = ?", userID).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.getDefaultSetting(userID), nil
		}
		return nil, err
	}
	return &setting, nil
}

// getDefaultSetting 获取默认配置
func (s *Service) getDefaultSetting(userID uint) *models.UserNotificationSetting {
	return &models.UserNotificationSetting{
		UserID:                  userID,
		EmailEnabled:            true,
		SMSEnabled:              false,
		DingTalkEnabled:         false,
		WeChatEnabled:           false,
		FeiShuEnabled:           false,
		TaskEnabled:             true,
		SystemEnabled:           true,
		WarningEnabled:          true,
		ReminderEnabled:         true,
		ConferenceEnabled:       true,
		EnableQuietHours:        false,
		QuietHoursStart:         "22:00:00",
		QuietHoursEnd:           "08:00:00",
		MaxNotificationsPerHour: 10,
	}
}

// isQuietHours 检查是否在免打扰时段
func (s *Service) isQuietHours(setting *models.UserNotificationSetting) bool {
	if !setting.EnableQuietHours {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04:05")

	// 处理跨天情况
	if setting.QuietHoursStart < setting.QuietHoursEnd {
		return currentTime >= setting.QuietHoursStart && currentTime <= setting.QuietHoursEnd
	} else {
		return currentTime >= setting.QuietHoursStart || currentTime <= setting.QuietHoursEnd
	}
}

// checkRateLimit 检查频率限制
func (s *Service) checkRateLimit(ctx context.Context, userID uint, setting *models.UserNotificationSetting) bool {
	// 统计当前小时已发送数量
	var count int64
	s.db.WithContext(ctx).Model(&models.NotificationMessage{}).
		Where("user_id = ? AND created_at >= ?", userID, time.Now().Truncate(time.Hour)).
		Count(&count)

	return int(count) < setting.MaxNotificationsPerHour
}

// isTypeEnabled 检查通知类型是否启用
func (s *Service) isTypeEnabled(setting *models.UserNotificationSetting, msgType models.NotificationType) bool {
	switch msgType {
	case models.TypeTask:
		return setting.TaskEnabled
	case models.TypeSystem:
		return setting.SystemEnabled
	case models.TypeWarning:
		return setting.WarningEnabled
	case models.TypeReminder:
		return setting.ReminderEnabled
	case models.TypeConference:
		return setting.ConferenceEnabled
	default:
		return true
	}
}

// CleanupOldMessages 清理旧消息
func (s *Service) CleanupOldMessages(keepDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -keepDays)

	// BUG-005/PR-D: 用 bgCtx 走 ctx-aware 路径
	ctx, cancel := context.WithTimeout(s.bgCtxOrBackground(), 30*time.Second)
	defer cancel()
	result := s.db.WithContext(ctx).Where("created_at < ? AND is_read = ?", cutoffDate, true).
		Delete(&models.NotificationMessage{})

	if result.Error != nil {
		return result.Error
	}

	s.logger.Info("清理通知消息",
		zap.Int("rows", int(result.RowsAffected)),
		zap.Time("cutoff", cutoffDate),
	)

	return nil
}
