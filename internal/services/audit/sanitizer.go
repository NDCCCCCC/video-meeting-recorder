package audit

import (
	"regexp"
	"strings"
)

// Sanitizer 脱敏器
type Sanitizer struct {
	rules []SanitizeRule
}

// SanitizeRule 脱敏规则
type SanitizeRule struct {
	Field   string
	Pattern *regexp.Regexp
	Replace string
}

// NewSanitizer 创建脱敏器
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		rules: []SanitizeRule{
			// 密码字段
			{Field: "password", Pattern: regexp.MustCompile("."), Replace: "******"},
			{Field: "password_hash", Pattern: regexp.MustCompile("."), Replace: "******"},
			{Field: "new_password", Pattern: regexp.MustCompile("."), Replace: "******"},
			{Field: "old_password", Pattern: regexp.MustCompile("."), Replace: "******"},

			// 手机号
			{Field: "phone", Pattern: regexp.MustCompile(`(\d{3})\d{4}(\d{4})`), Replace: "$1****$2"},
			{Field: "mobile", Pattern: regexp.MustCompile(`(\d{3})\d{4}(\d{4})`), Replace: "$1****$2"},
			{Field: "telephone", Pattern: regexp.MustCompile(`(\d{3})\d{4}(\d{4})`), Replace: "$1****$2"},

			// 身份证
			{Field: "id_card", Pattern: regexp.MustCompile(`(\d{6})\d{8}(\d{4})`), Replace: "$1********$2"},
			{Field: "idcard", Pattern: regexp.MustCompile(`(\d{6})\d{8}(\d{4})`), Replace: "$1********$2"},

			// 邮箱
			{Field: "email", Pattern: regexp.MustCompile(`(.{2}).*@(.*)`), Replace: "$1***@$2"},

			// IP地址（可选，根据安全要求）
			{Field: "ip", Pattern: regexp.MustCompile(`(\d+\.\d+)\.\d+\.\d+`), Replace: "$1.*.*"},

			// Token/Secret
			{Field: "token", Pattern: regexp.MustCompile(`(.{8}).*(.{8})`), Replace: "$1...$2"},
			{Field: "secret", Pattern: regexp.MustCompile("."), Replace: "******"},
			{Field: "access_token", Pattern: regexp.MustCompile(`(.{8}).*(.{8})`), Replace: "$1...$2"},
			{Field: "refresh_token", Pattern: regexp.MustCompile(`(.{8}).*(.{8})`), Replace: "$1...$2"},
			{Field: "api_key", Pattern: regexp.MustCompile(`(.{4}).*(.{4})`), Replace: "$1...$2"},
		},
	}
}

// Sanitize 对数据进行脱敏处理
func (s *Sanitizer) Sanitize(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		return s.sanitizeMap(v)
	case string:
		return s.sanitizeString("", v)
	default:
		return data
	}
}

// sanitizeMap 处理map类型数据
func (s *Sanitizer) sanitizeMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		if strValue, ok := value.(string); ok {
			result[key] = s.sanitizeString(key, strValue)
		} else if mapValue, ok := value.(map[string]interface{}); ok {
			result[key] = s.sanitizeMap(mapValue)
		} else if sliceValue, ok := value.([]interface{}); ok {
			result[key] = s.sanitizeSlice(sliceValue)
		} else {
			result[key] = value
		}
	}
	return result
}

// sanitizeSlice 处理slice类型数据
func (s *Sanitizer) sanitizeSlice(data []interface{}) []interface{} {
	result := make([]interface{}, len(data))
	for i, value := range data {
		if mapValue, ok := value.(map[string]interface{}); ok {
			result[i] = s.sanitizeMap(mapValue)
		} else if strValue, ok := value.(string); ok {
			result[i] = s.sanitizeString("", strValue)
		} else {
			result[i] = value
		}
	}
	return result
}

// sanitizeString 处理字符串类型数据
func (s *Sanitizer) sanitizeString(field, value string) string {
	// 检查字段名是否需要脱敏
	lowerField := strings.ToLower(field)
	for _, rule := range s.rules {
		if strings.Contains(lowerField, rule.Field) {
			return rule.Pattern.ReplaceAllString(value, rule.Replace)
		}
	}
	return value
}
