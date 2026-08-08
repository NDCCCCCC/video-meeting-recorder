package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/logging"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger, err := logging.New(cfg.Logging, cfg.Server.Environment)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// SEC-001: 生产环境启动时强制校验 SM4/HLS Token 密钥（缺失或 < 32 字符则 logger.Fatal 退出）。
	cfg.ValidateProductionSecrets(logger)

	// Phase 18: 凭据静态加密密钥族校验（缺失 / 配对错 / 版本格式错 → 立即返回 error）。
	if err := cfg.ValidateCredentialSM4Config(); err != nil {
		logger.Fatal("凭据静态加密密钥配置不合法（Phase 18）", zap.Error(err))
	}
	// 启动期一次性检测 hex secret 截断风险（典型 bug：`openssl rand -hex 32` 生成 64-char）。
	cfg.WarnCredentialSM4Truncation(logger)

	// video_playback_token 配置校验：secret ≥ 32 字符、独立密钥族（不与 SM4/HLS 重用）、
	// duration > 0；任何不合规直接 Fatal，进程不进入运行态。
	if err := cfg.ValidateVideoPlaybackTokenConfig(); err != nil {
		logger.Fatal("video_playback_token 配置不合法", zap.Error(err))
	}

	logger.Info("Starting Record V2 Server",
		zap.String("version", "2.0.0"),
		zap.String("environment", cfg.Server.Environment),
	)

	// 创建应用
	app := NewMinimalApp(cfg, logger)

	// 初始化应用
	if err := app.Initialize(); err != nil {
		logger.Fatal("Failed to initialize app", zap.Error(err))
	}

	// 启动应用
	if err := app.Start(); err != nil {
		logger.Fatal("Failed to start app", zap.Error(err))
	}

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 优雅关闭
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.Stop(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
