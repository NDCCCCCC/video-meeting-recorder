package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/logging"
	"go.uber.org/zap"
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
	defer logger.Sync()

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
