package config

import (
	"fmt"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// SmartEndConfig 智能录制延长（Smart Extend）的 3 项配置。
//
// Phase 23 (CFG-01): 强类型化所有 smart_end 阈值/开关；Phase 24/25 智能退出
// 撤回后保留 Extend 段（仅控制"按 ffmpeg 进程状态自动延长会议"），不再触发
// 提前结束。默认值走 applySmartEndDefaults；显式 YAML false
// 由 Viper SetDefault（true 在先，YAML false 覆盖后）保证不被吞掉（CFG-03/04）。
//
// 字段顺序按读取频率排列：先全局开关 (Enabled)，再单次延长步长 (ExtendStepMin)，
// 再最大延长次数 (MaxExtendCount)。命名遵循 yaml 三 tag 一致约定
// (mapstructure / json / yaml)。
type SmartEndConfig struct {
	// Enabled 全局开关。false 时系统退回纯 EndTime 行为 (scheduler 不读
	// IsProcessAlive,不调 UpdateTaskExtension),便于运维临时回退 (CFG-03)。
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`

	// ExtendStepMin 单次自动延时步长 (分钟)。必须 > 0。
	ExtendStepMin int `mapstructure:"extend_step_min" json:"extend_step_min" yaml:"extend_step_min"`

	// MaxExtendCount 单任务最大自动延次数。必须 > 0。默认 4 × 30min = 2h 总上限。
	MaxExtendCount int `mapstructure:"max_extend_count" json:"max_extend_count" yaml:"max_extend_count"`
}

// 错误消息常量 (包私有,Validate() 使用)。集中维护便于审计。
const (
	errSmartEndExtendStepMin  = "smart_end.extend_step_min 必须 > 0"
	errSmartEndMaxExtendCount = "smart_end.max_extend_count 必须 > 0"
)

// applySmartEndDefaults 把 SmartEnd 的数字字段按零值替换为默认值。
//
// Enabled 走 Viper SetDefault 在 Load() 中 Unmarshal 前注册;这里**不**触碰 bool,
// 避免覆盖 operator 显式 false (CFG-03/04)。YAML 中只要写了 false,就让它一直是 false。
//
// 返回值: 无 (原地修改 *cfg.SmartEnd)。
func applySmartEndDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	s := &cfg.SmartEnd
	if s.ExtendStepMin == 0 {
		s.ExtendStepMin = 30
	}
	if s.MaxExtendCount == 0 {
		s.MaxExtendCount = 4
	}
	// bool 字段:不触碰。
}

// Validate 校验 SmartEndConfig 数值合理性,首个违规处即返回 error,error 包装
// apperrors.ErrInvalidInput 便于调用方用 errors.Is 分类。
//
// 校验顺序 (遇违则返回):
//  1. ExtendStepMin > 0
//  2. MaxExtendCount > 0
//
// 不强制跨字段业务约束。
func (c *SmartEndConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("smart_end config is nil: %w", apperrors.ErrInvalidInput)
	}
	if c.ExtendStepMin <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndExtendStepMin, c.ExtendStepMin, apperrors.ErrInvalidInput)
	}
	if c.MaxExtendCount <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndMaxExtendCount, c.MaxExtendCount, apperrors.ErrInvalidInput)
	}
	return nil
}
