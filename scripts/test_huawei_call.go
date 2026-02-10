package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/huawei"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 从命令行参数获取会议号
	conferenceNumber := "521270003"
	if len(os.Args) > 1 {
		conferenceNumber = os.Args[1]
	}

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

	fmt.Println("=== 华为终端呼叫会议测试 ===")
	fmt.Printf("会议号: %s\n\n", conferenceNumber)

	ctx := context.Background()

	// 步骤1: 初始化（获取会话ID、认证、替换会话ID）
	fmt.Println("步骤1: 初始化华为终端客户端...")
	if err := client.InitializeAndStartKeepAlive(ctx); err != nil {
		fmt.Printf("  失败: %v\n", err)
		return
	}
	fmt.Println("  成功\n")

	// 步骤2: 呼叫会议
	fmt.Println("步骤2: 呼叫会议...")
	if err := client.CallConference(ctx, conferenceNumber); err != nil {
		fmt.Printf("  失败: %v\n", err)
		// 即使失败也尝试挂断
		client.HangupCall(ctx)
		client.Logout(ctx)
		return
	}
	fmt.Println("  成功\n")

	// 等待一段时间让用户观察
	fmt.Println("等待5秒...")
	time.Sleep(5 * time.Second)

	// 步骤3: 挂断呼叫
	fmt.Println("步骤3: 挂断呼叫...")
	if err := client.HangupCall(ctx); err != nil {
		fmt.Printf("  失败: %v\n", err)
	} else {
		fmt.Println("  成功")
	}

	// 步骤4: 登出
	client.Logout(ctx)

	fmt.Println("\n=== 测试完成 ===")
}
