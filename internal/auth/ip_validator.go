package auth

import (
	"bytes"
	"errors"
	"net"
	"strings"
)

// IPValidator IP地址验证器
type IPValidator struct{}

// ValidateIP 验证单个IP地址
func (v *IPValidator) ValidateIP(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return errors.New("invalid IP address")
	}
	// Reject IPv6 per D-09
	if ip.To4() == nil {
		return errors.New("IPv6 is not supported")
	}
	return nil
}

// ValidateCIDR 验证CIDR范围
func (v *IPValidator) ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return errors.New("invalid CIDR range")
	}
	return nil
}

// ValidateIPRange 验证IP范围 (e.g., "192.168.1.100-192.168.1.200")
func (v *IPValidator) ValidateIPRange(rangeStr string) error {
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return errors.New("invalid IP range format")
	}
	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))
	if startIP == nil || endIP == nil {
		return errors.New("invalid IP addresses in range")
	}
	if startIP.To4() == nil || endIP.To4() == nil {
		return errors.New("IPv6 is not supported")
	}
	// Check that end IP is not before start IP
	if bytes.Compare(startIP.To4(), endIP.To4()) > 0 {
		return errors.New("invalid IP range format")
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
		return false, errors.New("invalid client IP")
	}

	// Reject IPv6 addresses per D-09
	if clientAddr.To4() == nil {
		return false, errors.New("IPv6 is not supported")
	}

	for _, allowed := range allowedList {
		// Single IP
		if strings.Contains(allowed, "/") == false && strings.Contains(allowed, "-") == false {
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
