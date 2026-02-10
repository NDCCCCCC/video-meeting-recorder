package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cpic/record_v2/internal/huawei"
	"go.uber.org/zap"
)

func main() {
	// 设置环境变量启用curl
	os.Setenv("HUAWEI_USE_CURL", "true")

	logger, _ := zap.NewDevelopment()

	config := &huawei.Config{
		Server:            "10.62.10.3",
		Port:              443,
		Username:          "api",
		Password:          "Hubei@1992",
		APITimeout:        30 * time.Second,
		SessionTimeout:    3600 * time.Second,
		KeepAliveInterval: 300 * time.Second,
		InsecureSkipVerify: true,
		MinTLSVersion:     0x0301, // TLS 1.0
	}

	client := huawei.NewHuaweiClient(config, logger)

	fmt.Println("=== 华为终端认证测试（使用curl）===")

	ctx := context.Background()

	// 步骤1: 获取会话ID
	fmt.Println("步骤1: 获取会话ID...")
	if err := client.GetSessionID(ctx); err != nil {
		fmt.Printf("  失败: %v\n", err)
		return
	}
	fmt.Println("  成功")

	// 步骤2: 认证
	fmt.Println("步骤2: 用户认证...")
	if err := client.Authenticate(ctx); err != nil {
		fmt.Printf("  失败: %v\n", err)
		return
	}
	fmt.Println("  成功")

	// 步骤3: 替换会话ID
	fmt.Println("步骤3: 替换会话ID...")
	if err := client.ChangeSessionID(ctx); err != nil {
		fmt.Printf("  失败: %v\n", err)
		return
	}
	fmt.Println("  成功")

	fmt.Println("\n=== 测试完成 ===")
}
