package config

import (
	"fmt"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// SmartEndConfig 智能录制收尾（Smart Recording End）的 14 项配置。
//
// Phase 23 (CFG-01): 强类型化所有 smart_end 阈值/开关；Phase 24/25 watcher 与
// scheduler 直接读 cfg.SmartEnd.*。默认值走 applySmartEndDefaults；显式 YAML false
// 由 Viper SetDefault（true 在先，YAML false 覆盖后）保证不被吞掉（CFG-03/04）。
//
// 字段顺序按读取频率与依赖顺序排列：先全局开关 (Enabled)，再底层信号 (SilenceDB..)
// /沉默检测时长阈值，再文件停滞，再华为轮询参数，再调度步长/上限，再自检失败阈值，
// 最后降级开关。命名遵循 yaml 三 tag 一致约定 (mapstructure / json / yaml)。所有数字
// 类型的零值都被 applySmartEndDefaults 替换；bool 类型用 Viper SetDefault 处理
// (避免 setDefaults 里 "if !bool" 覆盖 operator 显式 false 的陷阱，见 Pitfall 3)。
type SmartEndConfig struct {
	// Enabled 全局开关。false 时系统退回纯 EndTime 行为 (scheduler 不读
	// taskEndedCh,watcher 不启),便于运维临时回退 (CFG-03)。
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`

	// SilenceDB ffmpeg silencedetect 的 noise 阈值 (dB)。范围 [-100, 0)。
	// -30 dB 等价“明显低于人声”的信号,适合会议场景 (PRD §6 推荐)。
	SilenceDB int `mapstructure:"silence_db" json:"silence_db" yaml:"silence_db"`

	// SilenceDurationS 持续静音判定时长 (秒)。必须 > 0。
	SilenceDurationS int `mapstructure:"silence_duration_s" json:"silence_duration_s" yaml:"silence_duration_s"`

	// FileStallS 文件大小停滞的最大间隔 (秒)。必须 > 0。
	FileStallS int `mapstructure:"file_stall_s" json:"file_stall_s" yaml:"file_stall_s"`

	// FileMinGrowthBPS 文件增长速率下限 (bytes/second)。允许 0 (但不为负)。
	// 1024 B/s ~= 略高于比特率 8 kbps 的最低增长,适合会议静音态。
	FileMinGrowthBPS int64 `mapstructure:"file_min_growth_bps" json:"file_min_growth_bps" yaml:"file_min_growth_bps"`

	// HuaweiEnabled 华为会议状态轮询开关。false 时系统降级只用兜底 A + B
	// (silencedetect + 文件停滞),便于 TE40 设备下线/维护时回退 (CFG-04)。
	HuaweiEnabled bool `mapstructure:"huawei_enabled" json:"huawei_enabled" yaml:"huawei_enabled"`

	// HuaweiPollIntervalS 华为 GetConferenceState 轮询间隔 (秒)。必须 > 0。
	HuaweiPollIntervalS int `mapstructure:"huawei_poll_interval_s" json:"huawei_poll_interval_s" yaml:"huawei_poll_interval_s"`

	// HuaweiPersistS 华为空状态持续判定时长 (秒)。必须 > 0。
	HuaweiPersistS int `mapstructure:"huawei_persist_s" json:"huawei_persist_s" yaml:"huawei_persist_s"`

	// HuaweiFailureThreshold 华为 API 连续失败阈值,达到后降级关闭 H 信号。
	// 必须 > 0。
	HuaweiFailureThreshold int `mapstructure:"huawei_failure_threshold" json:"huawei_failure_threshold" yaml:"huawei_failure_threshold"`

	// CheckIntervalS watcher 主循环检查间隔 (秒)。必须 > 0。
	CheckIntervalS int `mapstructure:"check_interval_s" json:"check_interval_s" yaml:"check_interval_s"`

	// ExtendStepMin 单次自动延时步长 (分钟)。必须 > 0。
	ExtendStepMin int `mapstructure:"extend_step_min" json:"extend_step_min" yaml:"extend_step_min"`

	// MaxExtendCount 单任务最大自动延次数。必须 > 0。默认 4 × 30min = 2h 总上限。
	MaxExtendCount int `mapstructure:"max_extend_count" json:"max_extend_count" yaml:"max_extend_count"`

	// StatFailureThreshold 文件 os.Stat 连续失败阈值,视为流死亡并触发结束。
	// 必须 > 0。
	StatFailureThreshold int `mapstructure:"stat_failure_threshold" json:"stat_failure_threshold" yaml:"stat_failure_threshold"`

	// DegradeOnSilenceLoss silencedetect 解析连续失败时是否自动降级 (关闭 A,
	// 只用 B + H)。true 降级 / false 不降级但会一直走无信号的检测路径。
	DegradeOnSilenceLoss bool `mapstructure:"degrade_on_silence_loss" json:"degrade_on_silence_loss" yaml:"degrade_on_silence_loss"`
}

// 错误消息常量 (包私有,Validate() 使用)。集中维护便于审计。
const (
	errSmartEndSilenceDBOutOfRange   = "smart_end.silence_db 必须在 [-100, 0] 范围内"
	errSmartEndSilenceDurationS      = "smart_end.silence_duration_s 必须 > 0"
	errSmartEndFileStallS            = "smart_end.file_stall_s 必须 > 0"
	errSmartEndFileMinGrowthBPS      = "smart_end.file_min_growth_bps 必须 >= 0"
	errSmartEndHuaweiPollIntervalS   = "smart_end.huawei_poll_interval_s 必须 > 0"
	errSmartEndHuaweiPersistS        = "smart_end.huawei_persist_s 必须 > 0"
	errSmartEndHuaweiFailureThresh   = "smart_end.huawei_failure_threshold 必须 > 0"
	errSmartEndCheckIntervalS        = "smart_end.check_interval_s 必须 > 0"
	errSmartEndExtendStepMin         = "smart_end.extend_step_min 必须 > 0"
	errSmartEndMaxExtendCount        = "smart_end.max_extend_count 必须 > 0"
	errSmartEndStatFailureThreshold  = "smart_end.stat_failure_threshold 必须 > 0"
)

// applySmartEndDefaults 把 SmartEnd 的数字字段按零值替换为默认值。
//
// 三个 true-valued bool 字段 (Enabled / HuaweiEnabled / DegradeOnSilenceLoss)
// 走 Viper SetDefault 在 Load() 中 Unmarshal 前注册;这里**不**触碰 bool,避免
// 覆盖 operator 显式 false (CFG-03/04)。YAML 中只要写了 false,就让它一直是 false。
//
// 返回值: 无 (原地修改 *cfg.SmartEnd)。
func applySmartEndDefaults(cfg *Config) {
	if cfg == nil {
		return
	}
	s := &cfg.SmartEnd
	if s.SilenceDB == 0 {
		s.SilenceDB = -30
	}
	if s.SilenceDurationS == 0 {
		s.SilenceDurationS = 30
	}
	if s.FileStallS == 0 {
		s.FileStallS = 120
	}
	// FileMinGrowthBPS 是 int64,零值语义明确;== 0 即“没设置过”。
	if s.FileMinGrowthBPS == 0 {
		s.FileMinGrowthBPS = 1024
	}
	if s.HuaweiPollIntervalS == 0 {
		s.HuaweiPollIntervalS = 30
	}
	if s.HuaweiPersistS == 0 {
		s.HuaweiPersistS = 30
	}
	if s.HuaweiFailureThreshold == 0 {
		s.HuaweiFailureThreshold = 3
	}
	if s.CheckIntervalS == 0 {
		s.CheckIntervalS = 5
	}
	if s.ExtendStepMin == 0 {
		s.ExtendStepMin = 30
	}
	if s.MaxExtendCount == 0 {
		s.MaxExtendCount = 4
	}
	if s.StatFailureThreshold == 0 {
		s.StatFailureThreshold = 3
	}
	// bool 字段:不触碰。
}

// Validate 校验 SmartEndConfig 数值合理性,首个违规处即返回 error,error 包装
// apperrors.ErrInvalidInput 便于调用方用 errors.Is 分类。
//
// 校验顺序 (遇违则返回):
//  1. SilenceDB ∈ [-100, 0]
//  2. 各类时长/阈值必须 > 0 (SilenceDurationS / FileStallS /
//     HuaweiPollIntervalS / HuaweiPersistS / HuaweiFailureThreshold /
//     CheckIntervalS / ExtendStepMin / MaxExtendCount / StatFailureThreshold)
//  3. FileMinGrowthBPS >= 0
//
// 不强制跨字段业务约束 (如 HuaweiPersistS >= HuaweiPollIntervalS) — 更短的 persistence
// 在下一次 sample 时仍可评估,业务可接受。
func (c *SmartEndConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("smart_end config is nil: %w", apperrors.ErrInvalidInput)
	}
	if c.SilenceDB < -100 || c.SilenceDB > 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndSilenceDBOutOfRange, c.SilenceDB, apperrors.ErrInvalidInput)
	}
	if c.SilenceDurationS <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndSilenceDurationS, c.SilenceDurationS, apperrors.ErrInvalidInput)
	}
	if c.FileStallS <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndFileStallS, c.FileStallS, apperrors.ErrInvalidInput)
	}
	if c.FileMinGrowthBPS < 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndFileMinGrowthBPS, c.FileMinGrowthBPS, apperrors.ErrInvalidInput)
	}
	if c.HuaweiPollIntervalS <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndHuaweiPollIntervalS, c.HuaweiPollIntervalS, apperrors.ErrInvalidInput)
	}
	if c.HuaweiPersistS <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndHuaweiPersistS, c.HuaweiPersistS, apperrors.ErrInvalidInput)
	}
	if c.HuaweiFailureThreshold <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndHuaweiFailureThresh, c.HuaweiFailureThreshold, apperrors.ErrInvalidInput)
	}
	if c.CheckIntervalS <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndCheckIntervalS, c.CheckIntervalS, apperrors.ErrInvalidInput)
	}
	if c.ExtendStepMin <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndExtendStepMin, c.ExtendStepMin, apperrors.ErrInvalidInput)
	}
	if c.MaxExtendCount <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndMaxExtendCount, c.MaxExtendCount, apperrors.ErrInvalidInput)
	}
	if c.StatFailureThreshold <= 0 {
		return fmt.Errorf("%s (got %d): %w", errSmartEndStatFailureThreshold, c.StatFailureThreshold, apperrors.ErrInvalidInput)
	}
	return nil
}
