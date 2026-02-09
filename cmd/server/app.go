package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/auth"
	"github.com/cpic/record_v2/internal/common"
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/handlers"
	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
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
	router     *gin.Engine
	jwtService *auth.JWTService
	handlers   *Handlers
	services   map[string]common.Service
	wg         sync.WaitGroup
}

// Handlers 处理器集合
type Handlers struct {
	Auth *handlers.AuthHandler
	User *handlers.UserHandler
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

	// 初始化Gin路由
	if err := a.initRouter(); err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}

	// 初始化处理器
	if err := a.initHandlers(); err != nil {
		return fmt.Errorf("failed to initialize handlers: %w", err)
	}

	// 注册路由
	if err := a.registerRoutes(); err != nil {
		return fmt.Errorf("failed to register routes: %w", err)
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

	// 创建种子数据
	if err := a.seedDatabase(); err != nil {
		a.logger.Warn("Failed to seed database", zap.Error(err))
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

// seedDatabase 创建种子数据
func (a *MinimalApp) seedDatabase() error {
	// 创建默认角色
	roles := []*models.Role{
		{Name: models.RoleAdmin, Description: "系统管理员"},
		{Name: models.RoleOperator, Description: "操作员"},
		{Name: models.RoleViewer, Description: "查看者"},
		{Name: models.RoleAPIClient, Description: "API客户端"},
	}

	for _, role := range roles {
		var existing models.Role
		if err := a.db.Where("name = ?", role.Name).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := a.db.Create(role).Error; err != nil {
				return fmt.Errorf("failed to create role %s: %w", role.Name, err)
			}
			a.logger.Info("Created default role", zap.String("name", role.Name))
		}
	}

	// 创建默认管理员用户
	var adminRole models.Role
	if err := a.db.Where("name = ?", models.RoleAdmin).First(&adminRole).Error; err != nil {
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	var existingUser models.User
	if err := a.db.Where("username = ?", "admin").First(&existingUser).Error; err == gorm.ErrRecordNotFound {
		admin := &models.User{
			Username: "admin",
			Email:    "admin@example.com",
			FullName: "系统管理员",
			RoleID:   adminRole.ID,
			IsActive:  true,
		}
		if err := admin.SetPassword("admin123"); err != nil {
			return fmt.Errorf("failed to set admin password: %w", err)
		}
		if err := a.db.Create(admin).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		a.logger.Info("Created default admin user",
			zap.String("username", "admin"),
			zap.String("password", "admin123"),
		)
	}

	return nil
}

// initRouter 初始化路由
func (a *MinimalApp) initRouter() error {
	// 设置Gin为发布模式
	gin.SetMode(gin.ReleaseMode)

	a.router = gin.New()
	a.router.Use(gin.Recovery())
	a.router.Use(gin.Logger())

	// CORS中间件
	a.router.Use(corsMiddleware())

	return nil
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-API-Key, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// initHandlers 初始化处理器
func (a *MinimalApp) initHandlers() error {
	a.jwtService = auth.NewJWTService(a.config, a.db, a.logger)
	authService := auth.NewService(a.config, a.db, a.logger)
	userService := services.NewUserService(a.db, a.logger)

	a.handlers = &Handlers{
		Auth: handlers.NewAuthHandler(authService, a.logger),
		User: handlers.NewUserHandler(userService, a.logger),
	}

	return nil
}

// registerRoutes 注册路由
func (a *MinimalApp) registerRoutes() error {
	// 健康检查端点（无需认证）
	a.router.GET("/health", a.healthHandler)
	a.router.GET("/api/v1/system/stats", a.statsHandler)

	// 认证路由
	auth := a.router.Group("/api/v1/auth")
	{
		auth.POST("/login", a.handlers.Auth.Login)
		auth.POST("/refresh", a.handlers.Auth.RefreshToken)
		auth.POST("/validate-password", a.handlers.Auth.ValidatePassword)
	}

	// 需要认证的路由
	authenticated := a.router.Group("/api/v1/auth")
	authenticated.Use(middleware.JWTAuth(a.jwtService))
	{
		authenticated.POST("/logout", a.handlers.Auth.Logout)
		authenticated.POST("/logout-all", a.handlers.Auth.LogoutAll)
		authenticated.POST("/change-password", a.handlers.Auth.ChangePassword)
		authenticated.GET("/me", a.handlers.Auth.GetCurrentUser)
	}

	// API路由组
	api := a.router.Group("/api/v1")
	api.Use(middleware.JWTAuth(a.jwtService)) // 全局认证

	// 用户管理
	users := api.Group("/users")
	{
		users.GET("", a.handlers.User.ListUsers)           // 获取用户列表
		users.GET("/profile", a.handlers.User.GetCurrentProfile) // 获取当前用户资料
		users.PUT("/profile", a.handlers.User.UpdateCurrentProfile) // 更新当前用户资料
		users.GET("/:id", a.handlers.User.GetUser)         // 获取用户详情
		users.POST("", a.handlers.User.CreateUser)         // 创建用户
		users.PUT("/:id", a.handlers.User.UpdateUser)      // 更新用户
		users.DELETE("/:id", a.handlers.User.DeleteUser)   // 删除用户
		users.POST("/:id/reset-password", a.handlers.User.ResetPassword) // 重置密码
		users.POST("/:id/toggle-status", a.handlers.User.ToggleUserStatus) // 切换状态
	}

	// 角色管理（待实现）
	_ = api.Group("/roles")
	// 任务管理（待实现）
	_ = api.Group("/tasks")
	// 会议管理（待实现）
	_ = api.Group("/conferences")
	// 文件管理（待实现）
	_ = api.Group("/files")

	return nil
}

// registerServices 注册服务
func (a *MinimalApp) registerServices() error {
	a.logger.Info("Registering services...")

	// TODO: 注册各个服务
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

// healthHandler 健康检查处理器
func (a *MinimalApp) healthHandler(c *gin.Context) {
	response.GinSuccess(c, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "2.0.0",
	})
}

// statsHandler 系统统计处理器
func (a *MinimalApp) statsHandler(c *gin.Context) {
	// TODO: 实现系统统计信息
	response.GinSuccess(c, map[string]interface{}{
		"services": len(a.services),
		"uptime":   time.Since(time.Now()).Seconds(),
	})
}
