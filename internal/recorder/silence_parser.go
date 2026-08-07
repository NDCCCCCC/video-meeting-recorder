package recorder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SilenceEventKind 标识一行 silencedetect 解析结果的语义类型。
type SilenceEventKind int

const (
	// SilenceEventNone 行不是有效的 silencedetect 事件（或非目标行）。
	SilenceEventNone SilenceEventKind = iota
	// SilenceEventStart 检测到新的静音段起点。
	SilenceEventStart
	// SilenceEventEnd 检测到静音段结束。
	SilenceEventEnd
)

// SilenceEvent 表示一行 ffmpeg silencedetect stderr 解析结果。
//
//   - Start    : 静音段起点时间戳（Start 事件由 silence_start 字段填充；
//     End 事件由 End - Duration 反推得到，与 ffmpeg 输出端点对齐）
//   - End      : 静音段终点时间戳（仅 End 事件填充）
//   - Duration : 静音持续时长（End 事件 = silence_duration 字段值；
//     duration-only 行同样填充）
//
// 仅当 Kind == SilenceEventNone 且 line 包含 "[silencedetect" 前缀但无法
// 解析时，Parse 才返回非 nil error（供 watcher 累加 failures）。
type SilenceEvent struct {
	Kind     SilenceEventKind
	Start    time.Duration
	End      time.Duration
	Duration time.Duration
}

// SilenceParser 解析 ffmpeg silencedetect stderr 单行。
//
// 行格式示例（FFmpeg 官方文档）：
//
//	[silencedetect @ 0x55a3] silence_start: 12.345
//	[silencedetect @ 0x55a3] silence_end: 45.678 | silence_duration: 33.333
//	[silencedetect @ 0x55a3] silence_duration: 30.000
//
// Parse 是无状态函数；SilenceParser 零值不可用，请通过 NewSilenceParser()
// 预编译正则后再使用。
type SilenceParser struct {
	startRe        *regexp.Regexp
	endRe          *regexp.Regexp
	durationOnlyRe *regexp.Regexp
}

// silencedetectPrefix 是 ffmpeg silencedetect 行在 stderr 中的固定前缀。
// Parse 先用 strings.Contains 过滤非目标行，避免无谓正则匹配。
const silencedetectPrefix = "[silencedetect"

// NewSilenceParser 返回预编译正则后的 *SilenceParser。
//
// 预编译的三条正则：
//   - startRe         匹配 [silencedetect ...] silence_start: X
//   - endRe           匹配 [silencedetect ...] silence_end: X | silence_duration: Y
//   - durationOnlyRe  匹配 [silencedetect ...] silence_duration: Z（无 end 配对）
func NewSilenceParser() *SilenceParser {
	startRe := regexp.MustCompile(
		`\[\s*silencedetect\s+@\s+0x[0-9a-fA-F]+\s*\]\s*silence_start\s*:\s*([0-9]+(?:\.[0-9]+)?)`,
	)
	endRe := regexp.MustCompile(
		`\[\s*silencedetect\s+@\s+0x[0-9a-fA-F]+\s*\]\s*silence_end\s*:\s*([0-9]+(?:\.[0-9]+)?)\s*\|\s*silence_duration\s*:\s*([0-9]+(?:\.[0-9]+)?)`,
	)
	durationOnlyRe := regexp.MustCompile(
		`\[\s*silencedetect\s+@\s+0x[0-9a-fA-F]+\s*\]\s*silence_duration\s*:\s*([0-9]+(?:\.[0-9]+)?)`,
	)
	return &SilenceParser{
		startRe:        startRe,
		endRe:          endRe,
		durationOnlyRe: durationOnlyRe,
	}
}

// Parse 解析一行 ffmpeg silencedetect stderr 输出。
//
// 返回规则：
//   - 不含 "[silencedetect" 子串 → 返回 zero event, nil（非目标行，静默丢弃）
//   - startRe 命中 → SilenceEvent{Kind: Start, Start: parseFloatToDuration(matches[1])}
//   - endRe 命中   → SilenceEvent{Kind: End, End, Duration, Start = End - Duration}
//   - durationOnlyRe 命中 → SilenceEvent{Kind: None, Duration: parseFloatToDuration(matches[1])}
//     （end-less duration 行不算失败，供 watcher 仅做计数 / snapshot）
//   - 仅含 "[silencedetect" 但三条正则都不命中 → 返回 zero event,
//     fmt.Errorf("malformed silencedetect line: <line>")，供 watcher 累加 failures
func (p *SilenceParser) Parse(line string) (SilenceEvent, error) {
	if !strings.Contains(line, silencedetectPrefix) {
		return SilenceEvent{}, nil
	}

	if m := p.startRe.FindStringSubmatch(line); m != nil {
		start, err := parseFloatToDuration(m[1])
		if err != nil {
			return SilenceEvent{}, fmt.Errorf("silence_start parse: %w", err)
		}
		return SilenceEvent{Kind: SilenceEventStart, Start: start}, nil
	}

	if m := p.endRe.FindStringSubmatch(line); m != nil {
		end, err := parseFloatToDuration(m[1])
		if err != nil {
			return SilenceEvent{}, fmt.Errorf("silence_end parse: %w", err)
		}
		dur, err := parseFloatToDuration(m[2])
		if err != nil {
			return SilenceEvent{}, fmt.Errorf("silence_duration parse: %w", err)
		}
		return SilenceEvent{
			Kind:     SilenceEventEnd,
			End:      end,
			Duration: dur,
			Start:    end - dur,
		}, nil
	}

	if m := p.durationOnlyRe.FindStringSubmatch(line); m != nil {
		dur, err := parseFloatToDuration(m[1])
		if err != nil {
			return SilenceEvent{}, fmt.Errorf("silence_duration parse: %w", err)
		}
		return SilenceEvent{Kind: SilenceEventNone, Duration: dur}, nil
	}

	return SilenceEvent{}, fmt.Errorf("malformed silencedetect line: %s", line)
}

// parseFloatToDuration 将 "12.345" 形式的秒数解析为 time.Duration。
// 用 strconv.ParseFloat 保证小数精度，不受 locale 影响。
func parseFloatToDuration(s string) (time.Duration, error) {
	seconds, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
