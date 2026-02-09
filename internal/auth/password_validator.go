package auth

import (
	"fmt"
	"regexp"
	"unicode"
)

// PasswordValidator 密码验证器
type PasswordValidator struct {
	minLength      int
	requireUpper   bool
	requireLower   bool
	requireNumber  bool
	requireSpecial bool
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// NewPasswordValidator 创建密码验证器
func NewPasswordValidator(minLength int, requireUpper, requireLower, requireNumber, requireSpecial bool) *PasswordValidator {
	return &PasswordValidator{
		minLength:      minLength,
		requireUpper:   requireUpper,
		requireLower:   requireLower,
		requireNumber:  requireNumber,
		requireSpecial: requireSpecial,
	}
}

// Validate 验证密码强度
func (v *PasswordValidator) Validate(password string) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]string, 0),
	}

	// 检查长度
	if len(password) < v.minLength {
		result.Valid = false
		result.Errors = append(result.Errors,
			fmt.Sprintf("密码长度不能少于%d位", v.minLength))
	}

	// 检查大写字母
	if v.requireUpper {
		hasUpper := false
		for _, c := range password {
			if unicode.IsUpper(c) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			result.Valid = false
			result.Errors = append(result.Errors, "密码必须包含至少一个大写字母")
		}
	}

	// 检查小写字母
	if v.requireLower {
		hasLower := false
		for _, c := range password {
			if unicode.IsLower(c) {
				hasLower = true
				break
			}
		}
		if !hasLower {
			result.Valid = false
			result.Errors = append(result.Errors, "密码必须包含至少一个小写字母")
		}
	}

	// 检查数字
	if v.requireNumber {
		hasNumber := false
		for _, c := range password {
			if unicode.IsDigit(c) {
				hasNumber = true
				break
			}
		}
		if !hasNumber {
			result.Valid = false
			result.Errors = append(result.Errors, "密码必须包含至少一个数字")
		}
	}

	// 检查特殊字符
	if v.requireSpecial {
		hasSpecial := false
		specialChars := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
		if specialChars.MatchString(password) {
			hasSpecial = true
		}
		if !hasSpecial {
			result.Valid = false
			result.Errors = append(result.Errors, "密码必须包含至少一个特殊字符")
		}
	}

	return result
}

// HashPassword 哈希密码（已在models/user.go中实现）
// 这里提供验证方法

// CheckPasswordStrength 检查密码强度（简化版）
func CheckPasswordStrength(password string) (strength string, score int) {
	score = 0

	// 长度评分
	if len(password) >= 8 {
		score += 1
	}
	if len(password) >= 12 {
		score += 1
	}

	// 复杂度评分
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	types := 0
	if hasLower {
		types++
	}
	if hasUpper {
		types++
	}
	if hasNumber {
		types++
	}
	if hasSpecial {
		types++
	}

	score += types

	// 确定强度
	if score < 3 {
		strength = "weak"
	} else if score < 5 {
		strength = "medium"
	} else {
		strength = "strong"
	}

	return strength, score
}
