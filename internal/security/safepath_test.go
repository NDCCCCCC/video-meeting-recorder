package security

import (
	"errors"
	"path/filepath"
	"testing"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

func TestSafeJoin(t *testing.T) {
	base := filepath.Clean("/var/app/storage")

	tests := []struct {
		name      string
		untrusted string
		wantOk    bool
	}{
		{"normal subpath", "ppts/123/slide_001.jpg", true},
		{"single segment", "file.mp4", true},
		{"empty stays at base", "", true},
		{"dot stays at base", ".", true},
		{"parent escape single", "..", false},
		{"parent escape nested", "sub/../../etc/passwd", false},
		{"parent escape leading", "../secret", false},
		{"sibling via dotdot", "../sibling/file", false},
		// 绝对路径：filepath.Join 会把它拼到 base 下（不覆盖 base），故被包容
		{"absolute path contained", "/etc/passwd", true},
		{"backslash only segment", "a\\b", true}, // 在 base 内的普通字符
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SafeJoin(base, tt.untrusted)
			if tt.wantOk {
				if err != nil {
					t.Fatalf("expected ok, got error: %v", err)
				}
				// 结果必须落在 base 内
				rel, relErr := filepath.Rel(base, got)
				if relErr != nil {
					t.Fatalf("Rel err: %v", relErr)
				}
				if rel == ".." || (len(rel) >= 2 && rel[0:2] == "..") {
					t.Fatalf("result %q escapes base %q (rel=%q)", got, base, rel)
				}
			} else {
				if err == nil {
					t.Fatalf("expected escape error, got %q", got)
				}
				if !errors.Is(err, apperrors.ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput wrap, got: %v", err)
				}
			}
		})
	}
}
