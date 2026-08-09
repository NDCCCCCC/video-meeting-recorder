// Package security 提供跨切面的安全工具，当前包含路径包容校验（path containment）。
//
// 解决 CodeQL「Uncontrolled data used in path expression」类告警：用户可控字符串
// 直接 filepath.Join 到 base 后传给文件系统 API，可被 "../" 穿越或绝对路径覆盖。
// SafeJoin 在拼接后做清洗 + 包容校验，调用方使用其返回值作为 sink，污点在 guard 终止。
package security

import (
	"fmt"
	"path/filepath"
	"strings"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// SafeJoin 把不可信的 untrusted 拼接到 base 下，返回【已清洗 + 已校验】的路径。
// 若解析结果逃出 base（含 ".." 穿越与绝对路径覆盖），返回包裹 ErrInvalidInput 的错误。
//
// 调用方必须使用本函数的返回值（而非自行 filepath.Join 的结果）作为文件系统 sink，
// 这样 CodeQL 的路径注入污点在包容 guard 处终止。
func SafeJoin(base, untrusted string) (string, error) {
	cleaned := filepath.Clean(filepath.Join(base, untrusted))
	baseCleaned := filepath.Clean(base)

	rel, err := filepath.Rel(baseCleaned, cleaned)
	if err != nil {
		return "", fmt.Errorf("%w: path escapes base", apperrors.ErrInvalidInput)
	}
	// rel == "." 表示结果恰为 base 本身（允许）；rel == ".." 或以 "../" 开头表示逃逸。
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: path escapes base", apperrors.ErrInvalidInput)
	}
	return cleaned, nil
}
