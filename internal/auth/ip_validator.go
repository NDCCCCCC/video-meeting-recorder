package auth

import (
	"bytes"
	"net"
	"strings"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// IPValidator IP地址验证器
type IPValidator struct{}

// ValidateIP 验证单个IP地址
// Phase 19 D8: 9 散点统一复用 apperrors.ErrInvalidInput (400 BadRequest 语义一致);
// 无新 sentinel，因为所有 IP/CIDR/range 错误都是"用户配置错误"400 类型。
// 当前零生产调用方（仅 ip_validator_test.go 测试）；未来 admin UI 配 IP 白名单时，
// 已 standard 错误路径可被 HandleError 识别。
func (v *IPValidator) ValidateIP(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return apperrors.ErrInvalidInput
	}
	// Reject IPv6 per D-09
	if ip.To4() == nil {
		return apperrors.ErrInvalidInput
	}
	return nil
}

// ValidateCIDR 验证CIDR范围
func (v *IPValidator) ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return apperrors.ErrInvalidInput
	}
	return nil
}

// ValidateIPRange 验证IP范围 (e.g., "192.168.1.100-192.168.1.200")
func (v *IPValidator) ValidateIPRange(rangeStr string) error {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return apperrors.ErrInvalidInput
	}
	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))
	if startIP == nil || endIP == nil {
		return apperrors.ErrInvalidInput
	}
	if startIP.To4() == nil || endIP.To4() == nil {
		return apperrors.ErrInvalidInput
	}
	// Check that end IP is not before start IP
	if bytes.Compare(startIP.To4(), endIP.To4()) > 0 {
		return apperrors.ErrInvalidInput
	}
	return nil
}

// IsIPAllowed 检查客户端IP是否在允许列表中
func (v *IPValidator) IsIPAllowed(clientIP string, allowedList []string) (bool, error) {
	// Empty list means no restrictions - allow all IPs
	if len(allowedList) == 0 {
		return true, nil
	}

	clientAddr := net.ParseIP(clientIP)
	if clientAddr == nil {
		return false, apperrors.ErrInvalidInput
	}

	// Reject IPv6 addresses per D-09
	if clientAddr.To4() == nil {
		return false, apperrors.ErrInvalidInput
	}

	for _, allowed := range allowedList {
		// Single IP
		if !strings.Contains(allowed, "/") && !strings.Contains(allowed, "-") {
			if clientIP == allowed {
				return true, nil
			}
			continue
		}

		// CIDR range
		if strings.Contains(allowed, "/") {
			_, ipNet, err := net.ParseCIDR(allowed)
			if err != nil {
				continue // Skip invalid CIDR
			}
			if ipNet.Contains(clientAddr) {
				return true, nil
			}
			continue
		}

		// IP range
		if strings.Contains(allowed, "-") {
			parts := strings.Split(allowed, "-")
			startIP := net.ParseIP(strings.TrimSpace(parts[0]))
			endIP := net.ParseIP(strings.TrimSpace(parts[1]))
			// Compare IPs byte by byte
			if bytes.Compare(clientAddr, startIP) >= 0 && bytes.Compare(clientAddr, endIP) <= 0 {
				return true, nil
			}
			continue
		}
	}

	return false, nil
}
