package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInputConfig_Validate_HuaweiAuto_WithHuaweiEnabled 验证华为自动类型且启用华为控制时的必填字段
func TestInputConfig_Validate_HuaweiAuto_WithHuaweiEnabled(t *testing.T) {
	config := &InputConfig{
		Name:           "测试配置",
		ConfigType:     ConfigTypeHuaweiAuto,
		HuaweiEnabled:  true,
		Server:         "192.168.1.100",
		Username:       "admin",
		TerminalNumber: "12345678",
	}

	err := config.Validate()
	assert.NoError(t, err, "有效的华为自动配置应该通过验证")
}

// TestInputConfig_Validate_HuaweiAuto_WithoutHuawei 验证华为自动类型但未启用华为控制时必须选择USB或流媒体
func TestInputConfig_Validate_HuaweiAuto_WithoutHuawei(t *testing.T) {
	t.Run("有USB设备应该通过", func(t *testing.T) {
		config := &InputConfig{
			Name:           "测试配置",
			ConfigType:     ConfigTypeHuaweiAuto,
			HuaweiEnabled:  false,
			USBCameraDevice: "video0",
		}

		err := config.Validate()
		assert.NoError(t, err, "华为控制关闭且有USB设备应该通过验证")
	})

	t.Run("有流媒体应该通过", func(t *testing.T) {
		config := &InputConfig{
			Name:       "测试配置",
			ConfigType: ConfigTypeHuaweiAuto,
			HuaweiEnabled: false,
			StreamURL:  "rtmp://example.com/live/stream",
		}

		err := config.Validate()
		assert.NoError(t, err, "华为控制关闭且有流媒体应该通过验证")
	})

	t.Run("无录制源应该失败", func(t *testing.T) {
		config := &InputConfig{
			Name:          "测试配置",
			ConfigType:    ConfigTypeHuaweiAuto,
			HuaweiEnabled: false,
		}

		err := config.Validate()
		assert.Error(t, err, "华为控制关闭且无录制源应该验证失败")
		assert.Contains(t, err.Error(), "必须选择USB或流媒体录制源")
	})
}

// TestInputConfig_Validate_USB_RequiresCameraDevice 验证USB类型必须指定摄像头设备
func TestInputConfig_Validate_USB_RequiresCameraDevice(t *testing.T) {
	t.Run("有摄像头设备应该通过", func(t *testing.T) {
		config := &InputConfig{
			Name:           "测试配置",
			ConfigType:     ConfigTypeUSB,
			USBCameraDevice: "video0",
		}

		err := config.Validate()
		assert.NoError(t, err, "USB配置有摄像头设备应该通过验证")
	})

	t.Run("无摄像头设备应该失败", func(t *testing.T) {
		config := &InputConfig{
			Name:       "测试配置",
			ConfigType: ConfigTypeUSB,
		}

		err := config.Validate()
		assert.Error(t, err, "USB配置无摄像头设备应该验证失败")
		assert.Contains(t, err.Error(), "必须指定摄像头设备")
	})
}

// TestInputConfig_Validate_Stream_RequiresURL 验证流媒体类型必须指定流地址
func TestInputConfig_Validate_Stream_RequiresURL(t *testing.T) {
	t.Run("有流地址应该通过", func(t *testing.T) {
		config := &InputConfig{
			Name:       "测试配置",
			ConfigType: ConfigTypeStream,
			StreamURL:  "rtmp://example.com/live/stream",
		}

		err := config.Validate()
		assert.NoError(t, err, "流媒体配置有流地址应该通过验证")
	})

	t.Run("无流地址应该失败", func(t *testing.T) {
		config := &InputConfig{
			Name:       "测试配置",
			ConfigType: ConfigTypeStream,
		}

		err := config.Validate()
		assert.Error(t, err, "流媒体配置无流地址应该验证失败")
		assert.Contains(t, err.Error(), "必须指定流地址")
	})
}

// TestInputConfig_Validate_InvalidConfigType 验证无效的配置类型
func TestInputConfig_Validate_InvalidConfigType(t *testing.T) {
	config := &InputConfig{
		Name:       "测试配置",
		ConfigType: "invalid_type",
	}

	err := config.Validate()
	assert.Error(t, err, "无效的配置类型应该验证失败")
	assert.Contains(t, err.Error(), "无效的配置类型")
}

// TestInputConfig_Lock_Unlock 测试锁定和解锁机制
func TestInputConfig_Lock_Unlock(t *testing.T) {
	config := &InputConfig{
		Name:       "测试配置",
		ConfigType: ConfigTypeUSB,
		IsLocked:   false,
	}

	t.Run("锁定配置", func(t *testing.T) {
		taskID := uint(1)
		err := config.Lock(taskID)
		assert.NoError(t, err, "锁定应该成功")
		assert.True(t, config.IsLocked, "配置应该被锁定")
		assert.Equal(t, &taskID, config.LockedBy, "锁定者应该被设置")
		assert.NotNil(t, config.LockedAt, "锁定时间应该被设置")
	})

	t.Run("解锁配置", func(t *testing.T) {
		err := config.Unlock()
		assert.NoError(t, err, "解锁应该成功")
		assert.False(t, config.IsLocked, "配置应该被解锁")
		assert.Nil(t, config.LockedBy, "锁定者应该被清除")
		assert.Nil(t, config.LockedAt, "锁定时间应该被清除")
	})

	t.Run("防止双重锁定", func(t *testing.T) {
		taskID1 := uint(1)
		taskID2 := uint(2)

		// 第一次锁定
		err := config.Lock(taskID1)
		assert.NoError(t, err, "第一次锁定应该成功")

		// 尝试用不同任务ID第二次锁定
		err = config.Lock(taskID2)
		assert.Error(t, err, "不同任务ID的第二次锁定应该失败")
		assert.Contains(t, err.Error(), "已被其他任务锁定")

		// 同一任务ID可以重复锁定
		err = config.Lock(taskID1)
		assert.NoError(t, err, "同一任务ID的重复锁定应该成功")
	})
}

// TestInputConfig_IsCameraBound_IsAudioBound 测试设备绑定状态检查
func TestInputConfig_IsCameraBound_IsAudioBound(t *testing.T) {
	config := &InputConfig{
		CameraBindingStatus: DeviceStatusBound,
		AudioBindingStatus:  DeviceStatusUnbound,
	}

	assert.True(t, config.IsCameraBound(), "摄像头应该显示已绑定")
	assert.False(t, config.IsAudioBound(), "音频应该显示未绑定")

	config.AudioBindingStatus = DeviceStatusBound
	assert.True(t, config.IsAudioBound(), "音频应该显示已绑定")
}
