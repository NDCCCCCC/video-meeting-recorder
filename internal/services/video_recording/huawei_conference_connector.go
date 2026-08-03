package video_recording

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/huawei"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
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

// ConnectToConference 连接到华为会议
// 执行流程：1. 加载配置 2. 锁定终端 3. 呼叫会议 4. 等待连接确认
func (c *HuaweiConferenceConnector) ConnectToConference(ctx context.Context, task *models.VideoRecordingTask) error {
	// 获取配置ID
	configID, err := getConfigID(task)
	if err != nil {
		return err
	}

	c.logger.Info("开始连接华为会议",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.Uint("huawei_config_id", configID),
		zap.String("conference_number", task.ConferenceNumber),
	)

	// 1. 加载华为配置
	config, err := c.getHuaweiConfig(ctx, configID)
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

	if err := c.manager.SafeCallConference(ctx, configID, callReq); err != nil {
		c.unlockTerminal(ctx, config.ID) // 释放锁
		return fmt.Errorf("呼叫会议失败: %w", err)
	}

	// 5. 等待连接确认
	waitTimeout := 30 * time.Second
	if err := c.manager.WaitForConnection(ctx, configID, task.ConferenceNumber, config.TerminalNumber, waitTimeout); err != nil {
		c.unlockTerminal(ctx, config.ID)
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
	// 获取配置ID
	configID, err := getConfigID(task)
	if err != nil {
		return err
	}

	c.logger.Info("断开华为会议",
		zap.Uint("task_id", task.ID),
		zap.String("conference_number", task.ConferenceNumber),
	)

	// 1. 加载华为配置
	config, err := c.getHuaweiConfig(ctx, configID)
	if err != nil {
		return fmt.Errorf("加载华为配置失败: %w", err)
	}

	// 2. 挂断会议
	hangupReq := &huawei.HangupConferenceRequest{
		TerminalNumber: config.TerminalNumber,
	}

	if err := c.manager.HangupConference(ctx, configID, hangupReq); err != nil {
		c.logger.Warn("挂断会议失败", zap.Error(err))
		// 继续执行解锁操作
	}

	// 3. 解锁终端
	if err := c.unlockTerminal(ctx, config.ID); err != nil {
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
	// 获取配置ID
	configID, err := getConfigID(task)
	if err != nil {
		return nil, err
	}

	c.logger.Info("获取RTSP流",
		zap.Uint("task_id", task.ID),
		zap.String("conference_number", task.ConferenceNumber),
	)

	streams, err := c.manager.GetRTSPStreams(ctx, configID, task.ConferenceNumber)
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
	configID, err := getConfigID(task)
	if err != nil {
		return nil, err
	}
	return c.manager.GetConferenceInfo(ctx, configID, task.ConferenceNumber)
}

// GetTerminalStatus 获取终端状态
func (c *HuaweiConferenceConnector) GetTerminalStatus(ctx context.Context, configID uint) (*huawei.TerminalStatus, error) {
	config, err := c.getHuaweiConfig(ctx, configID)
	if err != nil {
		return nil, err
	}
	return c.manager.GetTerminalStatus(ctx, configID, config.TerminalNumber)
}

// IsTerminalAvailable 检查终端是否可用
func (c *HuaweiConferenceConnector) IsTerminalAvailable(ctx context.Context, configID uint) (bool, error) {
	config, err := c.getHuaweiConfig(ctx, configID)
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
func (c *HuaweiConferenceConnector) LockTerminal(ctx context.Context, taskID uint, config *models.InputConfig) error {
	return c.lockTerminal(ctx, taskID, config)
}

// lockTerminal 锁定终端
func (c *HuaweiConferenceConnector) lockTerminal(ctx context.Context, taskID uint, config *models.InputConfig) error {
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
	if err := c.db.WithContext(ctx).Save(config).Error; err != nil {
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
func (c *HuaweiConferenceConnector) unlockTerminal(ctx context.Context, configID uint) error {
	var config models.InputConfig
	if err := c.db.WithContext(ctx).First(&config, configID).Error; err != nil {
		return fmt.Errorf("配置不存在: %w", err)
	}

	if err := config.Unlock(); err != nil {
		return err
	}

	if err := c.db.WithContext(ctx).Save(&config).Error; err != nil {
		return fmt.Errorf("保存解锁状态失败: %w", err)
	}

	c.logger.Info("终端解锁成功",
		zap.Uint("config_id", configID),
		zap.String("terminal_number", config.TerminalNumber),
	)

	return nil
}

// UnlockTerminalByTaskID 通过任务ID解锁终端
// 此方法用于任务取消时强制解锁，即使任务对象无法完整获取
func (c *HuaweiConferenceConnector) UnlockTerminalByTaskID(ctx context.Context, taskID uint) error {
	// 通过任务ID查找华为配置
	var config models.InputConfig
	if err := c.db.WithContext(ctx).Where("locked_by = ?", taskID).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.logger.Info("未找到被此任务锁定的终端",
				zap.Uint("task_id", taskID),
			)
			return nil // 没有被锁定的终端，不算错误
		}
		return fmt.Errorf("查找被锁定的终端失败: %w", err)
	}

	// 确认确实是此任务锁定的
	if !config.IsLocked || config.LockedBy == nil || *config.LockedBy != taskID {
		c.logger.Info("终端未被此任务锁定",
			zap.Uint("task_id", taskID),
			zap.Uint("config_id", config.ID),
		)
		return nil
	}

	// 解锁终端
	if err := config.Unlock(); err != nil {
		return fmt.Errorf("解锁配置失败: %w", err)
	}

	if err := c.db.WithContext(ctx).Save(&config).Error; err != nil {
		return fmt.Errorf("保存解锁状态失败: %w", err)
	}

	c.logger.Info("通过任务ID解锁终端成功",
		zap.Uint("task_id", taskID),
		zap.Uint("config_id", config.ID),
		zap.String("terminal_number", config.TerminalNumber),
	)

	return nil
}

// ClearStaleTerminalLocks 清理过期的终端锁
// 服务异常退出可能导致终端锁没有释放，启动时检查并清理
func (c *HuaweiConferenceConnector) ClearStaleTerminalLocks(ctx context.Context) error {
	c.logger.Info("开始清理过期的终端锁")

	// 查询所有被锁定的华为配置（PERF-002: 加 Limit(1000) 防止全表锁扫描）
	var lockedConfigs []models.InputConfig
	if err := c.db.WithContext(ctx).Where("is_locked = ?", true).Limit(1000).Find(&lockedConfigs).Error; err != nil {
		return fmt.Errorf("查询被锁定的华为配置失败: %w", err)
	}

	cleanedCount := 0
	for _, config := range lockedConfigs {
		if config.LockedBy == nil {
			// 没有锁定者ID，直接解锁
			c.logger.Info("清理无锁定者ID的终端锁",
				zap.Uint("config_id", config.ID),
				zap.String("terminal_number", config.TerminalNumber),
			)
			if err := c.db.WithContext(ctx).Model(&config).Updates(map[string]interface{}{
				"is_locked": false,
				"locked_by": nil,
				"locked_at": nil,
			}).Error; err == nil {
				cleanedCount++
			}
			continue
		}

		// 检查锁定任务的状态
		var task models.VideoRecordingTask
		err := c.db.WithContext(ctx).First(&task, *config.LockedBy).Error
		if err != nil {
			// 任务不存在，解锁
			c.logger.Info("清理不存在任务的终端锁",
				zap.Uint("config_id", config.ID),
				zap.Uint("locked_by_task_id", *config.LockedBy),
			)
			if err := c.db.WithContext(ctx).Model(&config).Updates(map[string]interface{}{
				"is_locked": false,
				"locked_by": nil,
				"locked_at": nil,
			}).Error; err == nil {
				cleanedCount++
			}
			continue
		}

		// 任务存在但状态不是 recording、connecting 或 converting，解锁
		// 注意：converting 是转换中状态，此时终端应该已经释放
		// 如果 converting 状态的终端仍被锁定，说明任务卡住了，需要清理
		if task.Status != models.VideoStatusRecording && task.Status != models.VideoStatusConnecting {
			c.logger.Info("清理已完成/取消/转换中任务的终端锁",
				zap.Uint("config_id", config.ID),
				zap.Uint("locked_by_task_id", *config.LockedBy),
				zap.String("task_status", string(task.Status)),
			)
			if err := c.db.WithContext(ctx).Model(&config).Updates(map[string]interface{}{
				"is_locked": false,
				"locked_by": nil,
				"locked_at": nil,
			}).Error; err == nil {
				cleanedCount++
			}
		}
	}

	c.logger.Info("终端锁清理完成",
		zap.Int("total_locked", len(lockedConfigs)),
		zap.Int("cleaned_configs", cleanedCount),
	)

	return nil
}

// getHuaweiConfig 获取华为配置
func (c *HuaweiConferenceConnector) getHuaweiConfig(ctx context.Context, configID uint) (*models.InputConfig, error) {
	var config models.InputConfig
	if err := c.db.WithContext(ctx).First(&config, configID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("输入配置不存在: ID=%d", configID)
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
		zap.Int("sites_count", len(info.SiteList)),
	)

	return nil
}

// GetActiveTerminals 获取所有可用的终端
func (c *HuaweiConferenceConnector) GetActiveTerminals(ctx context.Context) ([]models.InputConfig, error) {
	// PERF-002: 加 Limit(1000);实际生产中可用终端不会超过此数。
	// BUG-005: ctx 透传到 ORM (audit item)。
	var configs []models.InputConfig
	if err := c.db.WithContext(ctx).Where("is_active = ?", true).Limit(1000).Find(&configs).Error; err != nil {
		return nil, err
	}

	// 过滤掉已锁定的配置
	var available []models.InputConfig
	for _, config := range configs {
		if !config.IsLocked {
			available = append(available, config)
		}
	}

	return available, nil
}

// getConfigID 从任务获取华为配置ID
// 从 TaskInputConfigs 中查找 ConfigType 为 huawei_auto 的配置
func getConfigID(task *models.VideoRecordingTask) (uint, error) {
	for _, tc := range task.TaskInputConfigs {
		if tc.ConfigType == models.ConfigTypeHuaweiAuto {
			return tc.InputConfigID, nil
		}
	}
	return 0, fmt.Errorf("华为配置ID未设置")
}
