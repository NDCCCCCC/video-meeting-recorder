package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// TestSmartEndConfig_Defaults verifies that 14 typed fields all load their
// expected defaults when no `smart_end:` YAML section is present.
// This combines the Viper SetDefault path (3 true-valued bools) and the
// applySmartEndDefaults path (numeric zero-value replacement).
func TestSmartEndConfig_Defaults(t *testing.T) {
	v := viper.New()
	// Mimic what Load() does for the 3 true-valued bools.
	v.SetDefault("smart_end.enabled", true)
	v.SetDefault("smart_end.huawei_enabled", true)
	v.SetDefault("smart_end.degrade_on_silence_loss", true)

	// Minimal YAML with no smart_end section at all.
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(`
server:
  host: "0.0.0.0"
  port: 8080
`)))

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))

	// applySmartEndDefaults is what setDefaults() does post-Unmarshal.
	applySmartEndDefaults(&cfg)

	// Exact defaults from REQUIREMENTS.md and Phase 23 RESEARCH.md table.
	assert.True(t, cfg.SmartEnd.Enabled, "Enabled should default to true")
	assert.Equal(t, -30, cfg.SmartEnd.SilenceDB, "SilenceDB should default to -30")
	assert.Equal(t, 30, cfg.SmartEnd.SilenceDurationS, "SilenceDurationS should default to 30")
	assert.Equal(t, 120, cfg.SmartEnd.FileStallS, "FileStallS should default to 120")
	assert.Equal(t, int64(1024), cfg.SmartEnd.FileMinGrowthBPS, "FileMinGrowthBPS should default to 1024")
	assert.True(t, cfg.SmartEnd.HuaweiEnabled, "HuaweiEnabled should default to true")
	assert.Equal(t, 30, cfg.SmartEnd.HuaweiPollIntervalS, "HuaweiPollIntervalS should default to 30")
	assert.Equal(t, 30, cfg.SmartEnd.HuaweiPersistS, "HuaweiPersistS should default to 30")
	assert.Equal(t, 3, cfg.SmartEnd.HuaweiFailureThreshold, "HuaweiFailureThreshold should default to 3")
	assert.Equal(t, 5, cfg.SmartEnd.CheckIntervalS, "CheckIntervalS should default to 5")
	assert.Equal(t, 30, cfg.SmartEnd.ExtendStepMin, "ExtendStepMin should default to 30")
	assert.Equal(t, 4, cfg.SmartEnd.MaxExtendCount, "MaxExtendCount should default to 4")
	assert.Equal(t, 6, cfg.SmartEnd.StatFailureThreshold, "StatFailureThreshold should default to 6")
	assert.True(t, cfg.SmartEnd.DegradeOnSilenceLoss, "DegradeOnSilenceLoss should default to true")
}

// TestSmartEndConfig_ExplicitFalsePreserved verifies the Pitfall 3 trap is fixed:
// operators must be able to explicitly disable bool switches (CFG-03 / CFG-04)
// via YAML. The Viper SetDefault(true) MUST NOT override operator's explicit false.
//
// Flow:
//  1. SetDefault the 3 true-valued bools (production pattern)
//  2. ReadConfig with YAML that explicitly sets them to false + a numeric to 99
//  3. Unmarshal: explicit YAML false must win over SetDefault true
//  4. applySmartEndDefaults: must NOT touch the 3 bools (zero-value is false but
//     we DO have explicit false → we should preserve it)
func TestSmartEndConfig_ExplicitFalsePreserved(t *testing.T) {
	v := viper.New()
	v.SetDefault("smart_end.enabled", true)
	v.SetDefault("smart_end.huawei_enabled", true)
	v.SetDefault("smart_end.degrade_on_silence_loss", true)

	v.SetConfigType("yaml")
	// Operator explicitly turns off all 3 switches + overrides SilenceDurationS to 99.
	require.NoError(t, v.ReadConfig(strings.NewReader(`
smart_end:
  enabled: false
  huawei_enabled: false
  degrade_on_silence_loss: false
  silence_duration_s: 99
`)))

	var cfg Config
	require.NoError(t, v.Unmarshal(&cfg))

	applySmartEndDefaults(&cfg)

	assert.False(t, cfg.SmartEnd.Enabled, "explicit false in YAML must survive SetDefault(true)")
	assert.False(t, cfg.SmartEnd.HuaweiEnabled, "explicit false in YAML must survive SetDefault(true)")
	assert.False(t, cfg.SmartEnd.DegradeOnSilenceLoss, "explicit false in YAML must survive SetDefault(true)")
	// Sanity: numeric override survives.
	assert.Equal(t, 99, cfg.SmartEnd.SilenceDurationS, "SilenceDurationS=99 must be preserved (non-zero)")
	// Other zero-valued numerics still get defaults:
	assert.Equal(t, -30, cfg.SmartEnd.SilenceDB, "SilenceDB zero-value should still get default")
	assert.Equal(t, 30, cfg.SmartEnd.HuaweiPollIntervalS, "HuaweiPollIntervalS zero-value should still get default")
}

// TestSmartEndConfig_InvalidRejection covers Validate() rejection semantics.
// Every invalid case must:
//   - return non-nil error
//   - errors.Is(err, apperrors.ErrInvalidInput) must be true
//
// Plus one positive case: a fully-defaulted config (within Validate's scope —
// defaults already satisfy all checks except MaxExtendCount/StatFailureThreshold
// themselves, all positive) must return nil.
func TestSmartEndConfig_InvalidRejection(t *testing.T) {
	// Construct a valid baseline we will mutate per sub-test.
	valid := SmartEndConfig{
		Enabled:                true,
		SilenceDB:              -30,
		SilenceDurationS:       30,
		FileStallS:             120,
		FileMinGrowthBPS:       1024,
		HuaweiEnabled:          true,
		HuaweiPollIntervalS:    30,
		HuaweiPersistS:         30,
		HuaweiFailureThreshold: 3,
		CheckIntervalS:         5,
		ExtendStepMin:          30,
		MaxExtendCount:         4,
		StatFailureThreshold:   3,
		DegradeOnSilenceLoss:   true,
	}

	cases := []struct {
		name   string
		mutate func(c *SmartEndConfig)
	}{
		{
			name:   "silence_db too high (>0)",
			mutate: func(c *SmartEndConfig) { c.SilenceDB = 1 },
		},
		{
			name:   "silence_db too low (<-100)",
			mutate: func(c *SmartEndConfig) { c.SilenceDB = -101 },
		},
		{
			name:   "silence_duration_s zero",
			mutate: func(c *SmartEndConfig) { c.SilenceDurationS = 0 },
		},
		{
			name:   "silence_duration_s negative",
			mutate: func(c *SmartEndConfig) { c.SilenceDurationS = -1 },
		},
		{
			name:   "file_stall_s zero",
			mutate: func(c *SmartEndConfig) { c.FileStallS = 0 },
		},
		{
			name:   "file_min_growth_bps negative",
			mutate: func(c *SmartEndConfig) { c.FileMinGrowthBPS = -1 },
		},
		{
			name:   "huawei_poll_interval_s zero",
			mutate: func(c *SmartEndConfig) { c.HuaweiPollIntervalS = 0 },
		},
		{
			name:   "huawei_persist_s zero",
			mutate: func(c *SmartEndConfig) { c.HuaweiPersistS = 0 },
		},
		{
			name:   "huawei_failure_threshold zero",
			mutate: func(c *SmartEndConfig) { c.HuaweiFailureThreshold = 0 },
		},
		{
			name:   "check_interval_s zero",
			mutate: func(c *SmartEndConfig) { c.CheckIntervalS = 0 },
		},
		{
			name:   "extend_step_min zero",
			mutate: func(c *SmartEndConfig) { c.ExtendStepMin = 0 },
		},
		{
			name:   "max_extend_count zero",
			mutate: func(c *SmartEndConfig) { c.MaxExtendCount = 0 },
		},
		{
			name:   "stat_failure_threshold zero",
			mutate: func(c *SmartEndConfig) { c.StatFailureThreshold = 0 },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bad := valid
			tc.mutate(&bad)
			err := bad.Validate()
			assert.Error(t, err, "expected invalid config to be rejected by Validate()")
			if err != nil {
				assert.True(t, errors.Is(err, apperrors.ErrInvalidInput),
					"Validate() error must wrap apperrors.ErrInvalidInput, got: %v", err)
			}
		})
	}

	// Positive case: fully-defaulted (all values within Validate's scope).
	t.Run("valid defaults", func(t *testing.T) {
		err := valid.Validate()
		assert.NoError(t, err, "defaulted SmartEndConfig must pass Validate()")
	})
}
