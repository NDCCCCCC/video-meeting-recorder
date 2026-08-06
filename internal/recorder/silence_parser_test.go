package recorder

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSilenceParser 覆盖 24-VALIDATION.md §Required silence fixture matrix
// 的 6 个 fixture 行 + 1 个拼写错行（合计 7 个子测）。
//
// 严格按 fixture 矩阵断言 Kind/Start/End/Duration 字段与 error。
func TestSilenceParser(t *testing.T) {
	parser := NewSilenceParser()

	t.Run("start", func(t *testing.T) {
		// Fixture: silence_start 行 → 仅填充 Start
		ev, err := parser.Parse("[silencedetect @ 0x55a3] silence_start: 12.345")
		require.NoError(t, err)
		require.Equal(t, SilenceEventStart, ev.Kind)
		require.Equal(t, time.Duration(12.345*float64(time.Second)), ev.Start)
		require.Equal(t, time.Duration(0), ev.End)
		require.Equal(t, time.Duration(0), ev.Duration)
	})

	t.Run("end", func(t *testing.T) {
		// Fixture: silence_end 行 → End/Duration 由 ffmpeg 给出，Start 由 End - Duration 反推
		ev, err := parser.Parse("[silencedetect @ 0x55a3] silence_end: 45.678 | silence_duration: 33.333")
		require.NoError(t, err)
		require.Equal(t, SilenceEventEnd, ev.Kind)
		require.Equal(t, time.Duration(45.678*float64(time.Second)), ev.End)
		require.Equal(t, time.Duration(33.333*float64(time.Second)), ev.Duration)
		require.Equal(t, time.Duration(12.345*float64(time.Second)), ev.Start)
	})

	t.Run("duration_only", func(t *testing.T) {
		// Fixture: end-less duration 行 → Kind=None，Duration 填充，不算失败
		ev, err := parser.Parse("[silencedetect @ 0x55a3] silence_duration: 30.000")
		require.NoError(t, err)
		require.Equal(t, SilenceEventNone, ev.Kind)
		require.Equal(t, time.Duration(30.0*float64(time.Second)), ev.Duration)
		require.Equal(t, time.Duration(0), ev.Start)
		require.Equal(t, time.Duration(0), ev.End)
	})

	t.Run("unrelated_line", func(t *testing.T) {
		// Fixture: 非 silencedetect 行（ffmpeg 进度行）→ 静默丢弃
		ev, err := parser.Parse("frame=  100 fps=30 q=28.0 size=    1024kB time=00:00:03.33 bitrate=2517.0kbits/s")
		require.NoError(t, err)
		require.Equal(t, SilenceEventNone, ev.Kind)
		require.Equal(t, time.Duration(0), ev.Start)
		require.Equal(t, time.Duration(0), ev.End)
		require.Equal(t, time.Duration(0), ev.Duration)
	})

	t.Run("malformed", func(t *testing.T) {
		// Fixture: 空内容 [silencedetect 行 → 解析错误，供 watcher 累加 failures
		_, err := parser.Parse("[silencedetect @ 0x55a3]")
		require.Error(t, err)
		require.Contains(t, err.Error(), "malformed silencedetect line")
	})

	t.Run("typo", func(t *testing.T) {
		// Fixture: 拼写错（[siler 而非 [silencedetect）→ 静默丢弃，不算失败
		ev, err := parser.Parse("[siler @ 0x55a3] silence_start: 1.0")
		require.NoError(t, err)
		require.Equal(t, SilenceEventNone, ev.Kind)
		require.Equal(t, time.Duration(0), ev.Start)
		require.Equal(t, time.Duration(0), ev.End)
		require.Equal(t, time.Duration(0), ev.Duration)
	})
}

// TestSilenceParser_DurationThreshold 验证 24-01 PLAN §Test 7 DurationThreshold
// 子测：给定 endRe 匹配行 Duration=33.333s，断言 Parse 暴露 Duration 字段供
// watcher 与 cfg.SilenceDurationS 比对（允许 ±1ms 浮点舍入误差）。
func TestSilenceParser_DurationThreshold(t *testing.T) {
	parser := NewSilenceParser()

	ev, err := parser.Parse("[silencedetect @ 0x55a3] silence_end: 45.678 | silence_duration: 33.333")
	require.NoError(t, err)
	require.Equal(t, SilenceEventEnd, ev.Kind)

	// 33.333s ± 1ms（覆盖 strconv.ParseFloat → time.Duration 转换的浮点舍入）
	const tolerance = time.Millisecond
	diff := ev.Duration - 33333*time.Millisecond
	if diff < 0 {
		diff = -diff
	}
	require.LessOrEqual(t, diff, tolerance,
		"expected Duration ≈ 33.333s, got %v (diff %v)", ev.Duration, diff)
}