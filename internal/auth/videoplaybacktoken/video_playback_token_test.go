package videoplaybacktoken

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

const testSecret = "0123456789abcdef0123456789abcdef" // 32 chars

func newTestService(t *testing.T, duration time.Duration) *VideoPlaybackToken {
	t.Helper()
	return NewVideoPlaybackToken(testSecret, duration, zap.NewNop())
}

func TestNewVideoPlaybackToken_ShortSecretPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on short secret, got nil")
		}
	}()
	NewVideoPlaybackToken("short", 5*time.Minute, zap.NewNop())
}

func TestNewVideoPlaybackToken_ZeroDuration_DoesNotPanic(t *testing.T) {
	// duration 校验由 cfg.ValidateVideoPlaybackTokenConfig 负责;这里容忍 0/负值
	// 以便测试构造过期 token 场景。Secret 长度才是构造期的硬约束。
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic on zero duration: %v", r)
		}
	}()
	_ = NewVideoPlaybackToken(testSecret, 0, zap.NewNop())
}

func TestGenerate_Verify_Roundtrip(t *testing.T) {
	svc := newTestService(t, 5*time.Minute)
	tok := svc.Generate(42, 7)
	if tok == "" {
		t.Fatal("Generate returned empty token")
	}
	if strings.Count(tok, ".") != 1 {
		t.Fatalf("token should have exactly one '.': %q", tok)
	}

	claims, err := svc.Verify(tok)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims.FileID != 42 {
		t.Errorf("FileID = %d, want 42", claims.FileID)
	}
	if claims.UserID != 7 {
		t.Errorf("UserID = %d, want 7", claims.UserID)
	}
	if claims.ExpiresAt <= claims.IssuedAt {
		t.Errorf("ExpiresAt %d should be > IssuedAt %d", claims.ExpiresAt, claims.IssuedAt)
	}
	if claims.IsExpired() {
		t.Error("freshly issued token should not be expired")
	}
}

func TestVerify_Expired(t *testing.T) {
	svc := newTestService(t, -1*time.Second) // 已过期
	tok := svc.Generate(1, 1)
	claims, err := svc.Verify(tok)
	if err == nil {
		t.Fatalf("expected error for expired token, got claims %+v", claims)
	}
	if !errors.Is(err, apperrors.ErrTokenExpired) {
		t.Errorf("error chain = %v, want ErrTokenExpired", err)
	}
}

func TestVerify_TamperedSignature(t *testing.T) {
	svc := newTestService(t, 5*time.Minute)
	tok := svc.Generate(1, 1)
	parts := strings.Split(tok, ".")
	// 确定性篡改：替换签名【首字符】为另一个合法 base64url 字符。首字符的全部 6 位都是
	// 数据位（构成解码后的首字节），必定改变 HMAC 比对结果。
	// 注意：不能改末字符——HMAC-SHA256=32B→43 个 base64url 字符，第 43 个字符仅高 4 位是
	// 数据、低 2 位是 padding；改末位 'x'(49)→'z'(51) 时高 4 位均为 1100，解码字节不变，
	// 导致 hmac.Equal 通过、Verify 返回 nil。签名含 time.Now() 秒级时间戳，末字符随时间变化，
	// 旧实现因此是时间相关 flaky（约 6% 的秒末字符 ∈ {w,x,y,z} 时失败）。
	sig := parts[1]
	bad := byte('B')
	if sig[0] == 'B' {
		bad = 'C'
	}
	tampered := parts[0] + "." + string(bad) + sig[1:]
	_, err := svc.Verify(tampered)
	if err == nil {
		t.Fatal("expected error for tampered signature, got nil")
	}
	if !errors.Is(err, apperrors.ErrTokenInvalid) {
		t.Errorf("error chain = %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	signer := NewVideoPlaybackToken(testSecret, 5*time.Minute, zap.NewNop())
	tok := signer.Generate(99, 100)

	verifier := NewVideoPlaybackToken("ffffffffffffffffffffffffffffffff", 5*time.Minute, zap.NewNop())
	_, err := verifier.Verify(tok)
	if err == nil {
		t.Fatal("expected error on wrong-secret verification, got nil")
	}
	if !errors.Is(err, apperrors.ErrTokenInvalid) {
		t.Errorf("error chain = %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_MalformedToken(t *testing.T) {
	svc := newTestService(t, 5*time.Minute)
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"single_segment", "abc"},
		{"three_segments", "a.b.c"},
		{"non_base64_sig", "eyJ4IjoidGVzdCJ9.not_base64!@#"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Verify(tc.token)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", tc.token)
			}
			if !errors.Is(err, apperrors.ErrTokenInvalid) {
				t.Errorf("error chain = %v, want ErrTokenInvalid", err)
			}
		})
	}
}

func TestVerify_DifferentFileIDs_DifferentTokens(t *testing.T) {
	svc := newTestService(t, 5*time.Minute)
	a := svc.Generate(1, 7)
	b := svc.Generate(2, 7)
	if a == b {
		t.Fatal("different FileIDs should produce different tokens")
	}
	ca, err := svc.Verify(a)
	if err != nil {
		t.Fatalf("Verify a: %v", err)
	}
	if ca.FileID != 1 {
		t.Errorf("ca.FileID = %d, want 1", ca.FileID)
	}
}

func TestVerify_DifferentUsers_DifferentTokens(t *testing.T) {
	svc := newTestService(t, 5*time.Minute)
	a := svc.Generate(1, 7)
	b := svc.Generate(1, 8)
	if a == b {
		t.Fatal("different UserIDs should produce different tokens")
	}
}

func TestClaims_TimeRemaining(t *testing.T) {
	svc := newTestService(t, 30*time.Second)
	tok := svc.Generate(1, 1)
	claims, err := svc.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got := claims.TimeRemaining(); got <= 0 || got > 30 {
		t.Errorf("TimeRemaining = %d, want 1..30", got)
	}
}

func TestClaims_String(t *testing.T) {
	c := &Claims{FileID: 42, UserID: 7, ExpiresAt: 12345}
	s := c.String()
	for _, want := range []string{"42", "7", "12345"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() = %q, missing %q", s, want)
		}
	}
}

func TestSecretCopy_DoesNotAliasOriginal(t *testing.T) {
	// secretCopy 应在构造时拷贝,外部修改原 secret 字符串不影响后续签名/校验。
	secret := []byte(testSecret)
	svc := NewVideoPlaybackToken(string(secret), 5*time.Minute, zap.NewNop())

	// 签发 token
	tok := svc.Generate(1, 1)

	// 修改原 secret
	copy(secret, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	// 原 token 仍应校验通过(secretCopy 已被隔离)
	if _, err := svc.Verify(tok); err != nil {
		t.Errorf("Verify after mutating original secret = %v, want nil", err)
	}
}
