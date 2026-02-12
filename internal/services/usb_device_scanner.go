package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// USBDeviceInfo USB设备信息
type USBDeviceInfo struct {
	Type     string `json:"type"`     // "camera" | "audio"
	Name     string `json:"name"`     // 设备名称
	DeviceID string `json:"device_id"` // 设备ID (/dev/video0, hw:1,0, video=0)
	Status   string `json:"status"`   // "available" | "in_use" | "error"
	Backend  string `json:"backend"`  // "v4l2" | "alsa" | "dshow" | "wasapi"
}

// USBDeviceScanner USB设备扫描器
type USBDeviceScanner struct {
	logger *zap.Logger
}

// NewUSBDeviceScanner 创建USB设备扫描器
func NewUSBDeviceScanner(logger *zap.Logger) *USBDeviceScanner {
	return &USBDeviceScanner{
		logger: logger,
	}
}

// ScanVideoDevices 扫描USB摄像头设备
func (s *USBDeviceScanner) ScanVideoDevices() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo
	var err error

	if runtime.GOOS == "windows" {
		devices, err = s.scanWindowsVideoDevices()
	} else {
		devices, err = s.scanLinuxVideoDevices()
	}

	if err != nil {
		s.logger.Error("扫描视频设备失败", zap.Error(err))
	}

	s.logger.Info("已扫描视频设备", zap.Int("count", len(devices)))
	return devices, nil
}

// scanWindowsVideoDevices 扫描Windows摄像头设备
func (s *USBDeviceScanner) scanWindowsVideoDevices() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo

	// 使用PowerShell获取视频设备
	// PowerShell命令：获取所有视频设备
	psCommand := `
		Get-PnpDevice -Class Camera | Where-Object { $_.Status -eq 'OK' } | Select-Object FriendlyName, InstanceId | ConvertTo-Json
	`

	cmd := exec.Command("powershell", "-Command", psCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		s.logger.Warn("PowerShell命令失败，尝试使用ffmpeg", zap.Error(err), zap.String("stderr", stderr.String()))
		// 如果PowerShell失败，尝试使用ffmpeg
		return s.scanWindowsVideoDevicesFFmpeg()
	}

	output := stdout.String()
	if strings.TrimSpace(output) == "" || strings.TrimSpace(output) == "" {
		// 空输出，尝试使用ffmpeg
		return s.scanWindowsVideoDevicesFFmpeg()
	}

	// 解析JSON输出
	var result interface{}
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		switch v := result.(type) {
		case []interface{}:
			// 多个设备
			for _, item := range v {
				if device, ok := s.parseWindowsDevice(item); ok {
					devices = append(devices, device)
				}
			}
		case map[string]interface{}:
			// 单个设备
			if device, ok := s.parseWindowsDevice(v); ok {
				devices = append(devices, device)
			}
		}
	}

	// 如果没有找到设备，尝试ffmpeg
	if len(devices) == 0 {
		return s.scanWindowsVideoDevicesFFmpeg()
	}

	return devices, nil
}

// scanWindowsVideoDevicesFFmpeg 使用ffmpeg扫描Windows摄像头
func (s *USBDeviceScanner) scanWindowsVideoDevicesFFmpeg() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo

	// 使用ffmpeg -list_devices true -f dshow -i dummy获取设备列表
	cmd := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// ffmpeg会输出到stderr
	_ = cmd.Run()
	// ffmpeg会返回错误码，但设备列表仍然在stderr中

	output := stderr.String()
	lines := strings.Split(output, "\n")

	inVideoDevices := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "DirectShow video devices") {
			inVideoDevices = true
			continue
		}
		if strings.Contains(line, "DirectShow audio devices") {
			break
		}
		if inVideoDevices && strings.HasPrefix(line, "[dshow") {
			// 解析设备名称，格式类似：[dshow @ 000000] "Integrated Camera"
			if strings.Contains(line, `"`) {
				start := strings.Index(line, `"`)
				if start != -1 {
					end := strings.Index(line[start+1:], `"`)
					if end != -1 {
						deviceName := line[start+1 : start+1+end]
						if deviceName != "" {
							// 生成设备ID
							deviceID := fmt.Sprintf("video=%d", len(devices))
							devices = append(devices, USBDeviceInfo{
								Type:     "camera",
								Name:     deviceName,
								DeviceID: deviceID,
								Status:   "available",
								Backend:  "dshow",
							})
						}
					}
				}
			}
		}
	}

	return devices, nil
}

// parseWindowsDevice 解析Windows设备信息
func (s *USBDeviceScanner) parseWindowsDevice(item interface{}) (USBDeviceInfo, bool) {
	deviceMap, ok := item.(map[string]interface{})
	if !ok {
		return USBDeviceInfo{}, false
	}

	name := ""
	instanceID := ""

	if friendlyName, ok := deviceMap["FriendlyName"].(string); ok {
		name = friendlyName
	}
	if id, ok := deviceMap["InstanceId"].(string); ok {
		instanceID = id
	}

	if name == "" {
		return USBDeviceInfo{}, false
	}

	return USBDeviceInfo{
		Type:     "camera",
		Name:     name,
		DeviceID: instanceID,
		Status:   "available",
		Backend:  "dshow",
	}, true
}

// scanLinuxVideoDevices 扫描Linux摄像头设备
func (s *USBDeviceScanner) scanLinuxVideoDevices() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo

	// 检查 /dev/video* 设备
	for i := 0; i <= 10; i++ {
		devicePath := fmt.Sprintf("/dev/video%d", i)
		info, err := s.checkVideoDevice(devicePath)
		if err == nil {
			devices = append(devices, *info)
		}
	}

	// 检查通过ID符号链接的设备
	videoByIdPath := "/dev/v4l/by-id/"
	if files, err := filepath.Glob(videoByIdPath + "*"); err == nil {
		for _, file := range files {
			info, err := s.checkVideoDevice(file)
			if err == nil {
				// 避免重复
				exists := false
				for _, d := range devices {
					if d.DeviceID == info.DeviceID {
						exists = true
						break
					}
				}
				if !exists {
					devices = append(devices, *info)
				}
			}
		}
	}

	return devices, nil
}

// checkVideoDevice 检查视频设备
func (s *USBDeviceScanner) checkVideoDevice(devicePath string) (*USBDeviceInfo, error) {
	// 检查设备是否存在
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("device not found")
	}

	// 使用 v4l2-ctl 获取设备信息（如果有）
	name := s.getDeviceName(devicePath)

	// 检查设备是否被占用
	status := "available"
	if s.isDeviceInUse(devicePath) {
		status = "in_use"
	}

	return &USBDeviceInfo{
		Type:     "camera",
		Name:     name,
		DeviceID: devicePath,
		Status:   status,
		Backend:  "v4l2",
	}, nil
}

// ScanAudioDevices 扫描USB音频设备
func (s *USBDeviceScanner) ScanAudioDevices() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo
	var err error

	if runtime.GOOS == "windows" {
		devices, err = s.scanWindowsAudioDevices()
	} else {
		devices, err = s.scanLinuxAudioDevices()
	}

	if err != nil {
		s.logger.Error("扫描音频设备失败", zap.Error(err))
	}

	s.logger.Info("已扫描音频设备", zap.Int("count", len(devices)))
	return devices, nil
}

// scanWindowsAudioDevices 扫描Windows音频设备
func (s *USBDeviceScanner) scanWindowsAudioDevices() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo

	// 使用PowerShell获取音频设备
	psCommand := `
		Get-AudioDevice -List | Where-Object { $_.Type -eq 'Recording' } | Select-Object Name, ID | ConvertTo-Json
	`

	cmd := exec.Command("powershell", "-Command", psCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
	s.logger.Warn("PowerShell Get-AudioDevice失败，尝试使用ffmpeg", zap.Error(err))
		// 如果PowerShell失败，尝试使用ffmpeg
		return s.scanWindowsAudioDevicesFFmpeg()
	}

	output := stdout.String()
	if strings.TrimSpace(output) == "" {
		return s.scanWindowsAudioDevicesFFmpeg()
	}

	// 解析JSON输出
	var result interface{}
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		switch v := result.(type) {
		case []interface{}:
			for _, item := range v {
				if device, ok := s.parseWindowsAudioDevice(item); ok {
					devices = append(devices, device)
				}
			}
		case map[string]interface{}:
			if device, ok := s.parseWindowsAudioDevice(v); ok {
				devices = append(devices, device)
			}
		}
	}

	// 如果没有找到设备，尝试ffmpeg
	if len(devices) == 0 {
		return s.scanWindowsAudioDevicesFFmpeg()
	}

	return devices, nil
}

// scanWindowsAudioDevicesFFmpeg 使用ffmpeg扫描Windows音频设备
func (s *USBDeviceScanner) scanWindowsAudioDevicesFFmpeg() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo

	// 使用ffmpeg -list_devices true -f dshow -i dummy获取音频设备列表
	cmd := exec.Command("ffmpeg", "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	_ = cmd.Run() // ffmpeg会返回错误码，但设备列表仍然在stderr中

	output := stderr.String()
	lines := strings.Split(output, "\n")

	inAudioDevices := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "DirectShow audio devices") {
			inAudioDevices = true
			continue
		}
		if inAudioDevices && strings.HasPrefix(line, "[dshow") {
			// 解析设备名称
			if strings.Contains(line, `"`) {
				start := strings.Index(line, `"`)
				if start != -1 {
					end := strings.Index(line[start+1:], `"`)
					if end != -1 {
						deviceName := line[start+1 : start+1+end]
						// 过滤掉虚拟音频设备
						if deviceName != "" && !strings.Contains(strings.ToLower(deviceName), "virtual") {
							devices = append(devices, USBDeviceInfo{
								Type:     "audio",
								Name:     deviceName,
								DeviceID: deviceName, // 使用实际设备名称作为ID，dshow需要完整名称
								Status:   "available",
								Backend:  "dshow",
							})
						}
					}
				}
			}
		}
	}

	return devices, nil
}

// parseWindowsAudioDevice 解析Windows音频设备信息
func (s *USBDeviceScanner) parseWindowsAudioDevice(item interface{}) (USBDeviceInfo, bool) {
	deviceMap, ok := item.(map[string]interface{})
	if !ok {
		return USBDeviceInfo{}, false
	}

	name := ""
	deviceID := ""

	if n, ok := deviceMap["Name"].(string); ok {
		name = n
	}
	if id, ok := deviceMap["ID"].(string); ok {
		deviceID = id
	}

	if name == "" {
		return USBDeviceInfo{}, false
	}

	return USBDeviceInfo{
		Type:     "audio",
		Name:     name,
		DeviceID: deviceID,
		Status:   "available",
		Backend:  "wasapi",
	}, true
}

// scanLinuxAudioDevices 扫描Linux音频设备
func (s *USBDeviceScanner) scanLinuxAudioDevices() ([]USBDeviceInfo, error) {
	var devices []USBDeviceInfo

	// 检查 /dev/snd/* 设备
	audioPath := "/dev/snd"
	if files, err := filepath.Glob(audioPath + "/*"); err == nil {
		for _, file := range files {
			info, err := s.checkAudioDevice(file)
			if err == nil {
				devices = append(devices, *info)
			}
		}
	}

	// 检查 hw:* 设备（ALSA硬件设备）
	if files, err := filepath.Glob("/proc/asound/card*/hw*"); err == nil {
		for _, file := range files {
			info, err := s.checkAudioDevice(file)
			if err == nil {
				// 避免重复
				exists := false
				for _, d := range devices {
					if d.DeviceID == info.DeviceID {
						exists = true
						break
					}
				}
				if !exists {
					devices = append(devices, *info)
				}
			}
		}
	}

	return devices, nil
}

// checkAudioDevice 检查音频设备
func (s *USBDeviceScanner) checkAudioDevice(devicePath string) (*USBDeviceInfo, error) {
	// 跳过非设备文件
	info, err := os.Stat(devicePath)
	if err != nil {
		return nil, err
	}

	// 只检查字符设备和设备文件
	if info.Mode()&os.ModeDevice == 0 {
		name := s.getDeviceName(devicePath)

		return &USBDeviceInfo{
			Type:     "audio",
			Name:     name,
			DeviceID: devicePath,
			Status:   "available",
			Backend:  "alsa",
		}, nil
	}

	return nil, fmt.Errorf("not a valid device")
}

// ScanAllUSBDevices 扫描所有USB设备
func (s *USBDeviceScanner) ScanAllUSBDevices() map[string][]USBDeviceInfo {
	result := make(map[string][]USBDeviceInfo)

	cameras, _ := s.ScanVideoDevices()
	if len(cameras) > 0 {
		result["cameras"] = cameras
	}

	audios, _ := s.ScanAudioDevices()
	if len(audios) > 0 {
		result["audios"] = audios
	}

	return result
}

// getDeviceName 获取设备名称
func (s *USBDeviceScanner) getDeviceName(devicePath string) string {
	// 尝试从 udev 规则获取设备名称
	// 检查 /sys/class/video4linux/* 或 /sys/class/sound/*

	basePath := strings.ReplaceAll(devicePath, "/dev/", "/sys/class/")
	if strings.HasPrefix(devicePath, "/dev/video") {
		// 视频设备
		if name := s.readDeviceName(basePath); name != "" {
			return name
		}
		// 尝试从 /sys/class/video4linux 读取
		if dirs, err := filepath.Glob("/sys/class/video4linux/*"); err == nil {
			for _, dir := range dirs {
				if name := s.readDeviceName(dir); name != "" {
					return name
				}
			}
		}
	} else if strings.HasPrefix(devicePath, "/dev/snd") || strings.Contains(devicePath, "hw:") {
		// 音频设备
		if name := s.readDeviceName(basePath); name != "" {
			return name
		}
	}

	// 返回设备路径作为默认名称
	return filepath.Base(devicePath)
}

// readDeviceName 从设备目录读取名称
func (s *USBDeviceScanner) readDeviceName(devicePath string) string {
	// 尝试读取 name 文件
	nameFile := filepath.Join(devicePath, "name")
	if data, err := os.ReadFile(nameFile); err == nil {
		return strings.TrimSpace(string(data))
	}

	// 尝试从 uevent 读取
	if files, err := filepath.Glob(filepath.Join(devicePath, "device/uevent")); err == nil && len(files) > 0 {
		data, err := os.ReadFile(files[0])
		if err == nil {
			// 解析 uevent 文件中的设备名称
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "ID_PATH=") {
					// 提取设备名称
					parts := strings.Split(line, "=")
					if len(parts) > 1 {
						pathParts := strings.Split(parts[1], "/")
						if len(pathParts) > 0 {
							lastPart := pathParts[len(pathParts)-1]
							// 转换为可读名称
							name := strings.ReplaceAll(lastPart, "-", " ")
							if len(name) > 0 {
								name = strings.ToUpper(string(name[0])) + name[1:]
							}
							return "USB " + name
						}
					}
				}
			}
		}
	}

	return ""
}

// isDeviceInUse 检查设备是否被占用
func (s *USBDeviceScanner) isDeviceInUse(devicePath string) bool {
	// 尝试打开设备检查是否被锁定
	file, err := os.Open(devicePath)
	if err != nil {
		return true
	}
	file.Close()
	return false
}

// GetRecommendedDevice 获取推荐的设备（第一个可用设备）
func (s *USBDeviceScanner) GetRecommendedDevice(deviceType string) *USBDeviceInfo {
	var devices []USBDeviceInfo
	var err error

	if deviceType == "camera" {
		devices, err = s.ScanVideoDevices()
	} else {
		devices, err = s.ScanAudioDevices()
	}

	if err != nil || len(devices) == 0 {
		return nil
	}

	// 返回第一个可用设备
	for _, dev := range devices {
		if dev.Status == "available" {
			return &dev
		}
	}

	// 如果没有可用的，返回第一个
	return &devices[0]
}

// FormatAsJSON 格式化为JSON（用于调试）
func (s *USBDeviceScanner) FormatAsJSON() string {
	devices := s.ScanAllUSBDevices()
	data, _ := json.MarshalIndent(devices, "", "  ")
	return string(data)
}
