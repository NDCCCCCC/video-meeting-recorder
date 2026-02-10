package video_recording

import (
	"context"
	"fmt"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/huawei"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HuaweiConferenceConnector 华为会议连接器
// 负责华为终端与会议的连接管理
type HuaweiConferenceConnector struct {
	db      *gorm.DB
	manager *huawei.Manager
	logger  *zap.Logger
}

// NewHuaweiConferenceConnector 创建华为会议连接器
func NewHuaweiConferenceConnector(
	db *gorm.DB,
	manager *huawei.Manager,
	logger *zap.Logger,
) *HuaweiConferenceConnector {
	return &HuaweiConferenceConnector{
		db:      db,
		manager: manager,
		logger:  logger,
	}
}

// huaweiDBAdapter 实现 huawei.DBInterface 接口
type huaweiDBAdapter struct {
	db *gorm.DB
}

func (a *huaweiDBAdapter) GetHuaweiConfig(configID uint) (*huawei.HuaweiConfigDB, error) {
	var config models.HuaweiConfig
	if err := a.db.First(&config, configID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("华为配置不存在: ID=%d", configID)
		}
		return nil, err
	}

	return &huawei.HuaweiConfigDB{
		ID:             config.ID,
		Server:         config.Server,
		Port:           config.Port,
		Username:       config.Username,
		Password:       config.Password,
		TerminalNumber: config.TerminalNumber,
	}, nil
}

// ConnectToConference 连接到华为会议
// 执行流程：1. 加载配置 2. 锁定终端 3. 呼叫会议 4. 等待连接确认
func (c *HuaweiConferenceConnector) ConnectToConference(ctx context.Context, task *models.VideoRecordingTask) error {
	c.logger.Info("开始连接华为会议",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.Uint("huawei_config_id", task.HuaweiConfigID),
		zap.String("conference_number", task.ConferenceNumber),
	)

	// 1. 加载华为配置
	config, err := c.getHuaweiConfig(task.HuaweiConfigID)
	if err != nil {
		return fmt.Errorf("加载华为配置失败: %w", err)
	}

	// 2. 验证配置
	if err := config.Validate(); err != nil {
		return fmt.Errorf("华为配置验证失败: %w", err)
	}

	// 3. 锁定终端
	if err := c.lockTerminal(ctx, task.ID, config); err != nil {
		return fmt.Errorf("锁定终端失败: %w", err)
	}

	// 4. 呼叫会议
	callReq := &huawei.CallConferenceRequest{
		ConferenceNumber: task.ConferenceNumber,
		TerminalNumber:   config.TerminalNumber,
		Password:         "", // 如果会议有密码，可以从任务配置中获取
		Subject:          fmt.Sprintf("录制任务: %s", task.Name),
	}

	if err := c.manager.SafeCallConference(ctx, task.HuaweiConfigID, callReq); err != nil {
		c.unlockTerminal(config.ID) // 释放锁
		return fmt.Errorf("呼叫会议失败: %w", err)
	}

	// 5. 等待连接确认
	waitTimeout := 30 * time.Second
	if err := c.manager.WaitForConnection(ctx, task.HuaweiConfigID, task.ConferenceNumber, config.TerminalNumber, waitTimeout); err != nil {
		c.unlockTerminal(config.ID)
		return fmt.Errorf("等待连接失败: %w", err)
	}

	c.logger.Info("连接华为会议成功",
		zap.Uint("task_id", task.ID),
		zap.String("conference_number", task.ConferenceNumber),
		zap.String("terminal_number", config.TerminalNumber),
	)

	return nil
}

// DisconnectFromConference 断开华为会议连接
func (c *HuaweiConferenceConnector) DisconnectFromConference(ctx context.Context, task *models.VideoRecordingTask) error {
	c.logger.Info("断开华为会议",
		zap.Uint("task_id", task.ID),
		zap.String("conference_number", task.ConferenceNumber),
	)

	// 1. 加载华为配置
	config, err := c.getHuaweiConfig(task.HuaweiConfigID)
	if err != nil {
		return fmt.Errorf("加载华为配置失败: %w", err)
	}

	// 2. 挂断会议
	hangupReq := &huawei.HangupConferenceRequest{
		TerminalNumber: config.TerminalNumber,
	}

	if err := c.manager.HangupConference(ctx, task.HuaweiConfigID, hangupReq); err != nil {
		c.logger.Warn("挂断会议失败", zap.Error(err))
		// 继续执行解锁操作
	}

	// 3. 解锁终端
	if err := c.unlockTerminal(config.ID); err != nil {
		c.logger.Warn("解锁终端失败", zap.Error(err))
	}

	c.logger.Info("断开华为会议成功",
		zap.Uint("task_id", task.ID),
		zap.String("terminal_number", config.TerminalNumber),
	)

	return nil
}

// GetRTSPStreams 获取会议的RTSP流地址
func (c *HuaweiConferenceConnector) GetRTSPStreams(ctx context.Context, task *models.VideoRecordingTask) ([]huawei.RTSPStream, error) {
	c.logger.Info("获取RTSP流",
		zap.Uint("task_id", task.ID),
		zap.String("conference_number", task.ConferenceNumber),
	)

	streams, err := c.manager.GetRTSPStreams(ctx, task.HuaweiConfigID, task.ConferenceNumber)
	if err != nil {
		return nil, fmt.Errorf("获取RTSP流失败: %w", err)
	}

	c.logger.Info("获取RTSP流成功",
		zap.Uint("task_id", task.ID),
		zap.Int("stream_count", len(streams)),
	)

	for i, stream := range streams {
		c.logger.Debug("RTSP流详情",
			zap.Int("index", i),
			zap.String("type", stream.Type),
			zap.String("url", stream.URL),
		)
	}

	return streams, nil
}

// GetConferenceInfo 获取会议信息
func (c *HuaweiConferenceConnector) GetConferenceInfo(ctx context.Context, task *models.VideoRecordingTask) (*huawei.ConferenceInfo, error) {
	return c.manager.GetConferenceInfo(ctx, task.HuaweiConfigID, task.ConferenceNumber)
}

// GetTerminalStatus 获取终端状态
func (c *HuaweiConferenceConnector) GetTerminalStatus(ctx context.Context, configID uint) (*huawei.TerminalStatus, error) {
	config, err := c.getHuaweiConfig(configID)
	if err != nil {
		return nil, err
	}
	return c.manager.GetTerminalStatus(ctx, configID, config.TerminalNumber)
}

// IsTerminalAvailable 检查终端是否可用
func (c *HuaweiConferenceConnector) IsTerminalAvailable(ctx context.Context, configID uint) (bool, error) {
	config, err := c.getHuaweiConfig(configID)
	if err != nil {
		return false, err
	}

	// 检查是否被锁定
	if config.IsLocked {
		return false, fmt.Errorf("终端已被任务 %d 锁定", *config.LockedBy)
	}

	// 检查终端状态
	idle, err := c.manager.IsTerminalIdle(ctx, configID, config.TerminalNumber)
	if err != nil {
		return false, err
	}

	return idle, nil
}

// LockTerminal 锁定终端（供外部调用）
func (c *HuaweiConferenceConnector) LockTerminal(ctx context.Context, taskID uint, config *models.HuaweiConfig) error {
	return c.lockTerminal(ctx, taskID, config)
}

// lockTerminal 锁定终端
func (c *HuaweiConferenceConnector) lockTerminal(ctx context.Context, taskID uint, config *models.HuaweiConfig) error {
	// 检查是否已被锁定
	if config.IsLocked {
		// 检查是否被同一任务锁定
		if config.LockedBy != nil && *config.LockedBy == taskID {
			c.logger.Debug("终端已被本任务锁定",
				zap.Uint("config_id", config.ID),
				zap.Uint("task_id", taskID),
			)
			return nil
		}
		return fmt.Errorf("终端已被任务 %d 锁定", *config.LockedBy)
	}

	// 获取终端状态
	status, err := c.manager.GetTerminalStatus(ctx, config.ID, config.TerminalNumber)
	if err != nil {
		return fmt.Errorf("获取终端状态失败: %w", err)
	}

	// 检查终端是否空闲
	if status.Status != "idle" {
		return fmt.Errorf("终端状态不是空闲: %s", status.Status)
	}

	// 锁定配置
	if err := config.Lock(taskID); err != nil {
		return fmt.Errorf("锁定配置失败: %w", err)
	}

	// 保存到数据库
	if err := c.db.Save(config).Error; err != nil {
		return fmt.Errorf("保存锁定状态失败: %w", err)
	}

	c.logger.Info("终端锁定成功",
		zap.Uint("config_id", config.ID),
		zap.Uint("task_id", taskID),
		zap.String("terminal_number", config.TerminalNumber),
	)

	return nil
}

// unlockTerminal 解锁终端
func (c *HuaweiConferenceConnector) unlockTerminal(configID uint) error {
	var config models.HuaweiConfig
	if err := c.db.First(&config, configID).Error; err != nil {
		return fmt.Errorf("配置不存在: %w", err)
	}

	if err := config.Unlock(); err != nil {
		return err
	}

	if err := c.db.Save(&config).Error; err != nil {
		return fmt.Errorf("保存解锁状态失败: %w", err)
	}

	c.logger.Info("终端解锁成功",
		zap.Uint("config_id", configID),
		zap.String("terminal_number", config.TerminalNumber),
	)

	return nil
}

// getHuaweiConfig 获取华为配置
func (c *HuaweiConferenceConnector) getHuaweiConfig(configID uint) (*models.HuaweiConfig, error) {
	var config models.HuaweiConfig
	if err := c.db.First(&config, configID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("华为配置不存在: ID=%d", configID)
		}
		return nil, err
	}
	return &config, nil
}

// ValidateConference 验证会议是否可连接
func (c *HuaweiConferenceConnector) ValidateConference(ctx context.Context, configID uint, conferenceNumber string) error {
	// 获取会议信息
	info, err := c.manager.GetConferenceInfo(ctx, configID, conferenceNumber)
	if err != nil {
		return err
	}

	// 检查会议状态
	if info.Status == "ended" {
		return fmt.Errorf("会议已结束")
	}

	c.logger.Info("会议验证通过",
		zap.String("conference_number", conferenceNumber),
		zap.String("status", info.Status),
		zap.Int("attendees_count", info.AttendeesCount),
	)

	return nil
}

// GetActiveTerminals 获取所有可用的终端
func (c *HuaweiConferenceConnector) GetActiveTerminals(ctx context.Context) ([]models.HuaweiConfig, error) {
	var configs []models.HuaweiConfig
	if err := c.db.Where("is_active = ?", true).Find(&configs).Error; err != nil {
		return nil, err
	}

	// 过滤掉已锁定的配置
	var available []models.HuaweiConfig
	for _, config := range configs {
		if !config.IsLocked {
			available = append(available, config)
		}
	}

	return available, nil
}
