package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/common"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // 纯Go SQLite驱动
)

// MinimalApp 应用程序结构
type MinimalApp struct {
	config     *config.Config
	logger     *zap.Logger
	db         *gorm.DB
	httpServer *http.Server
	services   map[string]common.Service
	router     http.Handler
	wg         sync.WaitGroup
}

// NewMinimalApp 创建应用实例
func NewMinimalApp(cfg *config.Config, logger *zap.Logger) *MinimalApp {
	return &MinimalApp{
		config:   cfg,
		logger:   logger,
		services: make(map[string]common.Service),
	}
}

// Initialize 初始化应用
func (a *MinimalApp) Initialize() error {
	a.logger.Info("Initializing application...")

	// 初始化数据库
	if err := a.initDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// 初始化HTTP路由
	if err := a.initRouter(); err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}

	// 注册服务
	if err := a.registerServices(); err != nil {
		return fmt.Errorf("failed to register services: %w", err)
	}

	a.logger.Info("Application initialized successfully")
	return nil
}

// initDatabase 初始化数据库
func (a *MinimalApp) initDatabase() error {
	a.logger.Info("Initializing database...",
		zap.String("driver", a.config.Database.Driver),
		zap.String("path", a.config.Database.Path),
	)

	// 配置GORM日志
	gormLogger := logger.Default.LogMode(logger.Silent)

	// 配置SQLite连接参数
	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(%s)&_pragma=synchronous(%s)&_pragma=cache_size(%d)",
		a.config.Database.Path,
		a.config.Database.JournalMode,
		a.config.Database.Synchronous,
		a.config.Database.CacheSize*1024, // 转换为字节
	)

	// 打开数据库连接（使用纯Go SQLite驱动）
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数（SQLite单连接限制）
	sqlDB.SetMaxOpenConns(a.config.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(a.config.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(a.config.Database.ConnMaxLifetime) * time.Second)

	// 使用GORM包装数据库连接
	db, err := gorm.Open(sqlite.New(sqlite.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger:  gormLogger,
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return fmt.Errorf("failed to open gorm: %w", err)
	}

	a.db = db
	a.logger.Info("Database connected successfully")

	// 自动迁移
	if err := a.migrateDatabase(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// migrateDatabase 执行数据库迁移
func (a *MinimalApp) migrateDatabase() error {
	a.logger.Info("Running database migrations...")

	err := a.db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.APIKey{},
		&models.Session{},
		&models.HuaweiConfig{},
		&models.VideoRecordingTask{},
		&models.ConferenceRecord{},
		&models.VideoFile{},
		&models.PPTFile{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	a.logger.Info("Database migrations completed")
	return nil
}

// registerServices 注册服务
func (a *MinimalApp) registerServices() error {
	a.logger.Info("Registering services...")

	// TODO: 注册各个服务
	// 例如:
	// authService := auth.NewService(a.db, a.config.Auth, a.logger)
	// a.services["auth"] = authService
	// if err := authService.Initialize(); err != nil {
	//     return err
	// }

	a.logger.Info("Services registered successfully")
	return nil
}

// Start 启动应用
func (a *MinimalApp) Start() error {
	a.logger.Info("Starting application...")

	// 启动所有服务
	for name, service := range a.services {
		a.logger.Info("Starting service", zap.String("name", name))
		if err := service.Start(); err != nil {
			return fmt.Errorf("failed to start service %s: %w", name, err)
		}
	}

	// 启动HTTP服务器
	addr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)

	// 检查端口是否可用
	if err := a.checkPort(addr); err != nil {
		a.logger.Warn("Primary port busy, trying backup port", zap.Error(err))
		// 尝试使用备用端口
		a.config.Server.Port = 8081
		addr = fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)
	}

	a.httpServer = &http.Server{
		Addr:         addr,
		Handler:      a.router,
		ReadTimeout:  time.Duration(a.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.Server.WriteTimeout) * time.Second,
	}

	a.logger.Info("Starting HTTP server", zap.String("address", addr))

	// 在goroutine中启动服务器
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	a.logger.Info("Application started successfully",
		zap.String("address", addr),
	)

	return nil
}

// Stop 停止应用
func (a *MinimalApp) Stop(ctx context.Context) error {
	a.logger.Info("Stopping application...")

	// 停止HTTP服务器
	if a.httpServer != nil {
		a.logger.Info("Stopping HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	}

	// 停止所有服务
	for name, service := range a.services {
		a.logger.Info("Stopping service", zap.String("name", name))
		if err := service.Stop(); err != nil {
			a.logger.Error("Failed to stop service",
				zap.String("name", name),
				zap.Error(err),
			)
		}
	}

	// 关闭数据库连接
	if a.db != nil {
		sqlDB, _ := a.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	// 等待所有goroutine完成
	a.wg.Wait()

	a.logger.Info("Application stopped")
	return nil
}

// checkPort 检查端口是否可用
func (a *MinimalApp) checkPort(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

// initRouter 初始化路由
func (a *MinimalApp) initRouter() error {
	mux := http.NewServeMux()

	// 健康检查端点
	mux.HandleFunc("/health", a.healthHandler)
	mux.HandleFunc("/api/v1/system/stats", a.statsHandler)

	// API路由（待实现）
	// mux.HandleFunc("/api/v1/auth/login", a.handleLogin)
	// ...

	a.router = mux
	return nil
}

// healthHandler 健康检查处理器
func (a *MinimalApp) healthHandler(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "2.0.0",
	})
}

// statsHandler 系统统计处理器
func (a *MinimalApp) statsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: 实现系统统计信息
	response.Success(w, map[string]interface{}{
		"services": len(a.services),
		"uptime":   time.Since(time.Now()).Seconds(),
	})
}
