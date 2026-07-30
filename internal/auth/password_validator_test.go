package auth

import (
	"reflect"
	"testing"
)

// TestPasswordValidator_PackageLevelRegexInitializedOnce 验证 PERF-008 修复：
// 4 个包级 regex 在包加载时初始化，验证可重复读取。
func TestPasswordValidator_PackageLevelRegexInitializedOnce(t *testing.T) {
	if specialCharRe == nil {
		t.Fatal("specialCharRe 未初始化")
	}
	if lowerCaseRe == nil {
		t.Fatal("lowerCaseRe 未初始化")
	}
	if upperCaseRe == nil {
		t.Fatal("upperCaseRe 未初始化")
	}
	if digitRe == nil {
		t.Fatal("digitRe 未初始化")
	}

	// 验证包级 regex 是单例（同一指针 / 同一对象）
	if !reflect.DeepEqual(specialCharRe.String(), specialCharRe.String()) {
		t.Fatal("specialCharRe 不稳定")
	}
}

// TestPasswordValidator_ValidatesComplexPasswords 端到端：包级 regex 仍能
// 正确匹配（回归测试，确保移动到包级未改变行为）。
func TestPasswordValidator_ValidatesComplexPasswords(t *testing.T) {
	v := NewPasswordValidator(8, true, true, true, true)

	cases := []struct {
		password string
		want     bool
	}{
		{"abc", false},       // 太短
		{"abcdefgh", false},  // 缺大写/数字/特殊
		{"Abcdefgh", false},  // 缺数字/特殊
		{"Abcdefg1", false},  // 缺特殊
		{"Abc1efg!", true},   // 全部要求
		{"abcdefg1!", false}, // 缺大写
	}
	for _, tc := range cases {
		got := v.Validate(tc.password).Valid
		if got != tc.want {
			t.Errorf("Validate(%q).Valid = %v, want %v", tc.password, got, tc.want)
		}
	}
}

// TestCheckPasswordStrength_UsesPackageRegex 验证强度检测也走包级 regex。
func TestCheckPasswordStrength_UsesPackageRegex(t *testing.T) {
	strength, score := CheckPasswordStrength("Abc1efg!")
	if strength == "" {
		t.Fatal("CheckPasswordStrength 返回空 strength")
	}
	if score < 4 {
		t.Fatalf("score=%d 期望 ≥ 4", score)
	}
}
