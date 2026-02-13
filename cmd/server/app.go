package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/auth"
	"github.com/cpic/record_v2/internal/common"
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/frontend"
	"github.com/cpic/record_v2/internal/handlers"
	huaweiapi "github.com/cpic/record_v2/internal/huawei"
	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/recorder"
	"github.com/cpic/record_v2/internal/scheduler"
	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/internal/services/audit"
	"github.com/cpic/record_v2/internal/services/notification"
	"github.com/cpic/record_v2/internal/services/storage"
	"github.com/cpic/record_v2/internal/services/video_recording"
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
	// 调度器和协调器
	scheduler            *scheduler.VideoSimpleScheduler
	coordinator          *recorder.SimpleRecordingCoordinator
	huaweiManager        *huaweiapi.Manager
	huaweiConnector      *video_recording.HuaweiConferenceConnector
	videoTaskService     *services.VideoRecordingTaskService
	videoFileService     *services.VideoFileService
	conversionService    services.ConversionService
}

// Handlers 处理器集合
type Handlers struct {
	Auth         *handlers.AuthHandler
	User         *handlers.UserHandler
	Role         *handlers.RoleHandler
	VideoTask    *handlers.VideoRecordingTaskHandler
	HuaweiConfig *handlers.HuaweiConfigHandler
	VideoFile    *handlers.VideoFileHandler
	File         *handlers.FileHandler
	Audit        *handlers.AuditHandler
	Notification *handlers.NotificationHandler
	System        *handlers.SystemHandler
}

// huaweiDBAdapter 实现 huawei.DBInterface 接口
type huaweiDBAdapter struct {
	db *gorm.DB
}

func (a *huaweiDBAdapter) GetHuaweiConfig(configID uint) (*huaweiapi.HuaweiConfigDB, error) {
	var config models.HuaweiConfig
	if err := a.db.First(&config, configID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("华为配置不存在: ID=%d", configID)
		}
		return nil, err
	}

	return &huaweiapi.HuaweiConfigDB{
		ID:             config.ID,
		Server:         config.Server,
		Port:           config.Port,
		Username:       config.Username,
		Password:       config.Password,
		TerminalNumber: config.TerminalNumber,
	}, nil
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
	a.logger.Info("正在初始化应用...")

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

	a.logger.Info("应用初始化成功")
	return nil
}

// initDatabase 初始化数据库
func (a *MinimalApp) initDatabase() error {
	a.logger.Info("正在初始化数据库...",
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
	a.logger.Info("数据库连接成功")

	// 自动迁移
	if err := a.migrateDatabase(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// 创建种子数据
	if err := a.seedDatabase(); err != nil {
		a.logger.Warn("创建种子数据失败", zap.Error(err))
	}

	return nil
}

// migrateDatabase 执行数据库迁移
func (a *MinimalApp) migrateDatabase() error {
	a.logger.Info("正在执行数据库迁移...")

	err := a.db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.APIKey{},
		&models.Session{},
		&models.HuaweiConfig{},
		&models.VideoRecordingTask{},
		&models.VideoFile{},
		&models.PPTFile{},
		&models.UploadedFile{},
		&models.FileShare{},
		&models.UserStorageQuota{},
		&models.AuditLog{},
		&models.NotificationMessage{},
		&models.UserNotificationSetting{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	a.logger.Info("数据库迁移完成")
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
			IsActive: true,
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

	// 创建默认权限
	if err := a.seedPermissions(); err != nil {
		return fmt.Errorf("failed to seed permissions: %w", err)
	}

	return nil
}

// seedPermissions 创建权限数据
func (a *MinimalApp) seedPermissions() error {
	a.logger.Info("正在创建权限数据...")

	// 权限中文描述映射
	permissionDescriptions := map[string]string{
		models.ResourceTaskView:   "查看录制任务",
		models.ResourceTaskCreate:  "创建录制任务",
		models.ResourceTaskEdit:    "编辑录制任务",
		models.ResourceTaskDelete:  "删除录制任务",
		models.ResourceTaskStart:   "启动录制任务",
		models.ResourceTaskStop:    "停止录制任务",
		models.ResourceFileView:    "查看视频文件",
		models.ResourceFileDelete:  "删除视频文件",
		models.ResourceFileScan:    "扫描视频文件",
		models.ResourceUserView:    "查看用户",
		models.ResourceUserCreate:  "创建用户",
		models.ResourceUserEdit:    "编辑用户",
		models.ResourceUserDelete:  "删除用户",
		models.ResourceRoleView:    "查看角色",
		models.ResourceRoleCreate:  "创建角色",
		models.ResourceRoleEdit:    "编辑角色",
		models.ResourceRoleDelete:  "删除角色",
		models.ResourceConfigView:  "查看华为配置",
		models.ResourceConfigEdit:  "编辑华为配置",
	}

	// 遍历所有权限常量
	for _, permCode := range models.AllPermissions {
		// 解析权限代码（格式：resource:action）
		resource := permCode
		action := ""
		for i, r := range permCode {
			if r == ':' {
				resource = permCode[:i]
				action = permCode[i+1:]
				break
			}
		}

		// 检查权限是否已存在
		var existing models.Permission
		if err := a.db.Where("resource = ? AND action = ?", resource, action).First(&existing).Error; err == gorm.ErrRecordNotFound {
			permission := &models.Permission{
				Resource:    resource,
				Action:      action,
				Description: permissionDescriptions[permCode],
			}
			if err := a.db.Create(permission).Error; err != nil {
				return fmt.Errorf("failed to create permission %s: %w", permCode, err)
			}
			a.logger.Info("Created permission", zap.String("permission", permCode))
		} else if existing.Description == "" {
			// 如果权限已存在但没有描述，更新描述
			if err := a.db.Model(&existing).Update("description", permissionDescriptions[permCode]).Error; err != nil {
				a.logger.Warn("Failed to update permission description", zap.String("permission", permCode), zap.Error(err))
			}
		}
	}

	// 为管理员角色分配所有权限
	var adminRole models.Role
	if err := a.db.Where("name = ?", models.RoleAdmin).First(&adminRole).Error; err != nil {
		return fmt.Errorf("failed to find admin role: %w", err)
	}

	// 获取所有权限
	var allPermissions []models.Permission
	if err := a.db.Find(&allPermissions).Error; err != nil {
		return fmt.Errorf("failed to get permissions: %w", err)
	}

	// 检查是否需要关联权限（需要先预加载权限）
	var existingRole models.Role
	if err := a.db.Preload("Permissions").Where("name = ?", models.RoleAdmin).First(&existingRole).Error; err != nil {
		return fmt.Errorf("failed to find admin role with permissions: %w", err)
	}

	// 如果管理员权限数量与总权限不同，则重新分配
	if len(existingRole.Permissions) != len(allPermissions) {
		if err := a.db.Model(&existingRole).Association("Permissions").Replace(&allPermissions); err != nil {
			return fmt.Errorf("failed to assign permissions to admin role: %w", err)
		}
		a.logger.Info("Assigned all permissions to admin role", zap.Int("count", len(allPermissions)))
	}

	a.logger.Info("权限数据创建完成", zap.Int("total", len(allPermissions)))
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
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-API-Key, Authorization, Range")

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
	roleService := services.NewRoleService(a.db, a.logger)
	huaweiConfigService := services.NewHuaweiConfigService(a.db, a.logger)
	a.videoTaskService = services.NewVideoRecordingTaskService(a.db, a.logger)
	// 优先使用配置中的 ffprobe_path，否则从 ffmpeg 路径推导
	ffprobePath := a.config.FFmpeg.FFProbePath
	if ffprobePath == "" {
		// 从 ffmpeg 路径推导 ffprobe 路径
		if a.config.FFmpeg.Path != "" {
			// 将 ffmpeg 路径中的 ffmpeg 替换为 ffprobe
			ffmpegDir := filepath.Dir(a.config.FFmpeg.Path)
			ffprobePath = filepath.Join(ffmpegDir, "ffprobe")
		} else {
			ffprobePath = "./bin/ffprobe"
		}
	}
	a.videoFileService = services.NewVideoFileService(a.db, a.logger, a.config.Storage.RecordingsPath, ffprobePath)
	a.videoFileService.SetHLSPath(a.config.Storage.HLSPath)
	usbScanner := services.NewUSBDeviceScanner(a.logger)
	fileService := storage.NewFileService(a.db, a.logger, a.config)
	fileHandler := handlers.NewFileHandler(fileService)
	fileHandler.SetLogger(a.logger)
	fileHandler.SetJWTService(a.jwtService)

	// 审计日志服务
	auditService := audit.NewAuditLogService(a.db, a.logger)
	auditHandler := handlers.NewAuditHandler(auditService)
	auditHandler.SetLogger(a.logger)

	// 通知服务
	notificationService := notification.NewNotificationService(a.db, a.logger, a.config)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	notificationHandler.SetLogger(a.logger)

	// 转换服务（需要 videoFileService）
	a.conversionService = services.NewFFmpegConversionService(a.db, a.logger, a.config, a.videoFileService)

	// 华为管理器（使用数据库配置动态创建客户端）
	dbAdapter := &huaweiDBAdapter{db: a.db}
	a.huaweiManager = huaweiapi.NewManager(a.logger, dbAdapter)
	a.huaweiConnector = video_recording.NewHuaweiConferenceConnector(a.db, a.huaweiManager, a.logger)

	// 创建handlers
	a.handlers = &Handlers{
		Auth:         handlers.NewAuthHandler(authService, a.logger),
		User:         handlers.NewUserHandler(userService, a.logger),
		Role:         handlers.NewRoleHandler(roleService, a.logger),
		VideoTask:    handlers.NewVideoRecordingTaskHandler(a.videoTaskService, a.logger, a.config),
		HuaweiConfig: handlers.NewHuaweiConfigHandler(huaweiConfigService, a.logger, usbScanner),
		VideoFile:    handlers.NewVideoFileHandler(a.videoFileService, a.logger),
		File:         fileHandler,
		Audit:        auditHandler,
		Notification: notificationHandler,
		System:       handlers.NewSystemHandler(a.db, a.logger, a.config),
	}

	return nil
}

// registerRoutes 注册路由
func (a *MinimalApp) registerRoutes() error {
	// 健康检查端点（无需认证）
	a.router.GET("/health", a.healthHandler)
	a.router.GET("/api/v1/system/stats", a.statsHandler)

	// 调度器调试端点（需要认证）
	a.router.GET("/api/v1/scheduler/debug", a.schedulerDebugHandler)

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

	// 角色管理
	roles := api.Group("/roles")
	{
		roles.GET("", a.handlers.Role.ListRoles)                    // 获取角色列表
		roles.GET("/:id", a.handlers.Role.GetRole)                   // 获取角色详情
		roles.POST("", a.handlers.Role.CreateRole)                  // 创建角色
		roles.PUT("/:id", a.handlers.Role.UpdateRole)               // 更新角色
		roles.DELETE("/:id", a.handlers.Role.DeleteRole)            // 删除角色
		roles.GET("/:id/permissions", a.handlers.Role.GetRolePermissions) // 获取角色权限
		roles.POST("/:id/permissions", a.handlers.Role.AssignPermissions) // 分配权限
	}

	// 权限列表
	api.GET("/permissions", a.handlers.Role.GetAllPermissions)

	// 录制任务管理 (使用 /recordings 路径符合API文档规范)
	recordings := api.Group("/recordings")
	{
		recordings.GET("", a.handlers.VideoTask.ListTasks)                      // 获取任务列表
		recordings.GET("/:id", a.handlers.VideoTask.GetTask)                    // 获取任务详情
		recordings.POST("", a.handlers.VideoTask.CreateTask)                    // 创建任务
		recordings.PUT("/:id", a.handlers.VideoTask.UpdateTask)                 // 更新任务
		recordings.DELETE("/:id", a.handlers.VideoTask.DeleteTask)              // 删除任务
		recordings.DELETE("/batch", a.handlers.VideoTask.BatchDeleteTasks)      // 批量删除任务
		recordings.POST("/:id/start", a.handlers.VideoTask.StartTask)           // 启动任务
		recordings.POST("/:id/stop", a.handlers.VideoTask.StopTask)             // 停止任务
		recordings.POST("/:id/cancel", a.handlers.VideoTask.CancelTask)         // 取消任务
		recordings.POST("/:id/retry", a.handlers.VideoTask.RetryTask)           // 重试任务
		// 转换相关
		recordings.GET("/:id/conversion-status", a.handlers.VideoTask.GetConversionStatus)  // 获取转换状态
		recordings.POST("/:id/conversion-retry", a.handlers.VideoTask.RetryConversion)     // 重试转换
		// HLS 预览相关
		recordings.GET("/:id/preview", a.handlers.VideoTask.GetHLSPreview)      // 获取HLS预览信息
		// 注意：/:id/preview/stream/:file 路由已移至公开路由（无需JWT认证）
	}

	// 华为配置管理
	huaweiConfigs := api.Group("/huawei-configs")
	{
		huaweiConfigs.GET("/scan-devices", a.handlers.HuaweiConfig.ScanUSBDevices)       // 扫描USB设备
		huaweiConfigs.GET("/recommended-device", a.handlers.HuaweiConfig.GetRecommendedDevice) // 获取推荐设备
		huaweiConfigs.GET("", a.handlers.HuaweiConfig.ListConfigs)             // 获取配置列表
		huaweiConfigs.GET("/active", a.handlers.HuaweiConfig.GetActiveConfigs) // 获取可用配置
		huaweiConfigs.GET("/:id", a.handlers.HuaweiConfig.GetConfig)           // 获取配置详情
		huaweiConfigs.POST("", a.handlers.HuaweiConfig.CreateConfig)           // 创建配置
		huaweiConfigs.PUT("/:id", a.handlers.HuaweiConfig.UpdateConfig)        // 更新配置
		huaweiConfigs.DELETE("/:id", a.handlers.HuaweiConfig.DeleteConfig)     // 删除配置
	}

	// 文件存储管理
	storage := api.Group("/storage")
	{
		storage.POST("/upload", a.handlers.File.Upload)               // 上传文件
		storage.GET("/quota", a.handlers.File.GetQuota)                // 获取配额
		storage.GET("", a.handlers.File.List)                          // 获取文件列表
		storage.DELETE("/:id", a.handlers.File.Delete)                 // 删除文件
		storage.POST("/:id/share", a.handlers.File.Share)              // 生成分享链接
	}

	// 视频文件管理
	files := api.Group("/files")
	{
		files.GET("", a.handlers.VideoFile.ListFiles)                    // 获取文件列表
		files.GET("/stats", a.handlers.VideoFile.GetFileStats)            // 获取文件统计
		files.GET("/:id/download", a.handlers.VideoFile.DownloadFile)    // 下载文件（必须在 /:id 之前）
		files.GET("/:id", a.handlers.VideoFile.GetFile)               // 获取文件详情
		files.DELETE("/:id", a.handlers.VideoFile.DeleteFile)            // 删除文件
		files.POST("/scan", a.handlers.VideoFile.ScanFiles)             // 扫描并导入文件
	}

	// 公开文件访问（无需认证）
	a.router.GET("/api/v1/files/download/:token", a.handlers.File.Download)
	a.router.GET("/api/v1/files/share/:token", a.handlers.File.ShareDownload)

	// HLS 预览流文件访问（无需认证，但需要任务权限验证）
	a.router.GET("/api/v1/recordings/:id/preview/stream/:file", a.handlers.VideoTask.ServeHLSStream)

	// 审计日志管理
	auditLog := api.Group("/audit")
	{
		auditLog.GET("/logs", a.handlers.Audit.Query)                    // 查询审计日志
		auditLog.GET("/logs/:id", a.handlers.Audit.GetByID)              // 获取日志详情
		auditLog.GET("/statistics", a.handlers.Audit.Statistics)        // 获取操作统计
	}

	// 通知管理
	notifications := api.Group("/notifications")
	{
		notifications.GET("", a.handlers.Notification.ListNotifications)                   // 获取通知列表
		notifications.GET("/unread-count", a.handlers.Notification.GetUnreadCount)         // 获取未读数量
		notifications.PUT("/:id/read", a.handlers.Notification.MarkAsRead)               // 标记为已读
		notifications.PUT("/read-all", a.handlers.Notification.MarkAllAsRead)             // 全部标记为已读
		notifications.GET("/settings", a.handlers.Notification.GetUserSetting)           // 获取通知配置
		notifications.PUT("/settings", a.handlers.Notification.UpdateUserSetting)        // 更新通知配置
	}

	// 系统管理（需要 admin 权限）
	system := api.Group("/system")
	{
		system.GET("/config", a.handlers.System.GetConfig)                  // 获取系统配置
		system.PUT("/config", a.handlers.System.UpdateConfig)               // 更新系统配置
		system.POST("/clear-files", a.handlers.System.ClearFiles)           // 清空文件数据库
	}

	// 前端静态文件服务 (SPA 路由回退)
	a.registerFrontendRoutes()

	return nil
}

// registerServices 注册服务
func (a *MinimalApp) registerServices() error {
	// 创建录制协调器
	a.coordinator = recorder.NewSimpleRecordingCoordinator(a.logger, a.config)
	a.logger.Info("录制协调器已创建")

	// 创建任务调度器
	// 注意：这里使用一个适配器将VideoRecordingTaskService转换为TaskServiceInterface
	taskServiceAdapter := &taskServiceAdapter{db: a.db, logger: a.logger}
	a.scheduler = scheduler.NewVideoSimpleScheduler(
		taskServiceAdapter,
		a.coordinator,
		a.huaweiConnector,
		a.conversionService,
		a.videoFileService,
		a.logger,
		a.config,
	)
	a.logger.Info("调度器已创建")

	// 启动调度器
	if err := a.scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	a.logger.Info("调度器启动成功")

	// 设置调度器到任务服务
	if a.videoTaskService != nil {
		a.videoTaskService.SetScheduler(a.scheduler)
	}

	// 启动转换服务
	if a.conversionService != nil {
		if err := a.conversionService.Start(); err != nil {
			return fmt.Errorf("failed to start conversion service: %w", err)
		}
		a.logger.Info("转换服务启动成功")
	}

	a.logger.Info("服务注册完成")
	return nil
}

// taskServiceAdapter 任务服务适配器
type taskServiceAdapter struct {
	db     *gorm.DB
	logger *zap.Logger
}

// GetTask 获取任务
func (a *taskServiceAdapter) GetTask(id uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := a.db.Preload("HuaweiConfig").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetPendingTasks 获取待执行任务
func (a *taskServiceAdapter) GetPendingTasks() ([]*models.VideoRecordingTask, error) {
	var tasks []*models.VideoRecordingTask
	if err := a.db.Where("status = ?", models.VideoStatusPending).
		Preload("HuaweiConfig").
		Order("start_time ASC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTaskStatus 更新任务状态
func (a *taskServiceAdapter) UpdateTaskStatus(id uint, status models.VideoRecordingTaskStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	result := a.db.Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("任务不存在")
	}
	return nil
}

// UpdateRecordingPaths 更新录制文件路径
// 注意：这是适配器方法，将 TaskServiceInterface 接口映射到具体实现
func (a *taskServiceAdapter) UpdateRecordingPaths(id uint, mkvPath, hlsPath string) error {
	updates := map[string]interface{}{
		"recording_file":   mkvPath,
		"mkv_file_path":    mkvPath,
		"hls_preview_path": hlsPath,
	}

	result := a.db.Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("任务不存在")
	}
	return nil
}

// GetHuaweiConfig 获取华为配置
func (a *taskServiceAdapter) GetHuaweiConfig(id uint) (*models.HuaweiConfig, error) {
	var config models.HuaweiConfig
	if err := a.db.First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// Start 启动应用
func (a *MinimalApp) Start() error {
	a.logger.Info("正在启动应用...")

	// 华为管理器无需预先启动，客户端按需创建
	a.logger.Info("华为管理器已初始化（客户端按需创建）")

	// 启动所有服务
	for name, service := range a.services {
		a.logger.Info("正在启动服务", zap.String("name", name))
		if err := service.Start(); err != nil {
			return fmt.Errorf("failed to start service %s: %w", name, err)
		}
	}

	// 启动HTTP服务器
	addr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)

	// 检查端口是否可用
	if err := a.checkPort(addr); err != nil {
		a.logger.Warn("主端口被占用，尝试使用备用端口", zap.Error(err))
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

	a.logger.Info("正在启动HTTP服务器", zap.String("address", addr))

	// 在goroutine中启动服务器
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP服务器错误", zap.Error(err))
		}
	}()

	a.logger.Info("应用启动成功",
		zap.String("address", addr),
	)

	return nil
}

// Stop 停止应用
func (a *MinimalApp) Stop(ctx context.Context) error {
	a.logger.Info("正在停止应用...")

	// 停止HTTP服务器
	if a.httpServer != nil {
		a.logger.Info("正在停止HTTP服务器...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("HTTP服务器关闭错误", zap.Error(err))
		}
	}

	// 停止审计日志服务
	if a.handlers.Audit != nil {
		a.logger.Info("正在停止审计日志服务...")
		a.handlers.Audit.Stop()
	}

	// 停止通知服务
	if a.handlers.Notification != nil {
		a.logger.Info("正在停止通知服务...")
		a.handlers.Notification.Stop()
	}

	// 停止华为管理器
	if a.huaweiManager != nil {
		a.logger.Info("正在停止华为管理器...")
		if err := a.huaweiManager.Close(ctx); err != nil {
			a.logger.Error("关闭华为管理器失败", zap.Error(err))
		}
	}

	// 停止转换服务
	if a.conversionService != nil {
		a.logger.Info("正在停止转换服务...")
		a.conversionService.Stop()
	}

	// 停止所有服务
	for name, service := range a.services {
		a.logger.Info("正在停止服务", zap.String("name", name))
		if err := service.Stop(); err != nil {
			a.logger.Error("停止服务失败",
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

	a.logger.Info("应用已停止")
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
	stats := map[string]interface{}{
		"services": len(a.services),
		"uptime":   time.Since(time.Now()).Seconds(),
	}

	// 添加调度器统计信息
	if a.scheduler != nil {
		schedulerStats := a.scheduler.GetStats()
		stats["scheduler"] = schedulerStats

		// 添加已调度的任务列表
		scheduledTasks := a.scheduler.GetScheduledTasks()
		stats["scheduled_tasks"] = scheduledTasks

		// 添加正在执行的任务列表
		executingTasks := a.scheduler.GetExecutingTasks()
		stats["executing_tasks"] = executingTasks
	}

	response.GinSuccess(c, stats)
}

// schedulerDebugHandler 调度器调试处理器（需要认证）
func (a *MinimalApp) schedulerDebugHandler(c *gin.Context) {
	if a.scheduler == nil {
		response.GinError(c, response.CodeInternalError, "调度器未初始化")
		return
	}

	// 获取当前时间
	now := time.Now()
	nowUTC := time.Now().UTC()

	debugInfo := map[string]interface{}{
		"current_time_local": now.Format(time.RFC3339),
		"current_time_utc":   nowUTC.Format(time.RFC3339),
		"timezone":           nowUTC.Location().String(),
	}

	// 获取调度器统计信息
	debugInfo["stats"] = a.scheduler.GetStats()

	// 获取已调度的任务ID
	scheduledTaskIDs := a.scheduler.GetScheduledTasks()
	debugInfo["scheduled_task_ids"] = scheduledTaskIDs

	// 获取正在执行的任务ID
	executingTaskIDs := a.scheduler.GetExecutingTasks()
	debugInfo["executing_task_ids"] = executingTaskIDs

	// 获取数据库中的待执行任务详情
	if a.videoTaskService != nil {
		pendingTasks, err := a.videoTaskService.GetPendingTasks()
		if err == nil {
			taskDetails := make([]map[string]interface{}, 0, len(pendingTasks))
			for _, task := range pendingTasks {
				triggerTime := task.StartTime.Add(-time.Duration(task.PreJoinMinutes) * time.Minute)
				taskDetail := map[string]interface{}{
					"id":                 task.ID,
					"name":               task.Name,
					"status":             task.Status,
					"start_time":         task.StartTime.Format(time.RFC3339),
					"end_time":           task.EndTime.Format(time.RFC3339),
					"trigger_time":       triggerTime.Format(time.RFC3339),
					"pre_join_minutes":   task.PreJoinMinutes,
					"is_scheduled":       a.scheduler.IsTaskScheduled(task.ID),
					"is_executing":       a.scheduler.IsTaskExecuting(task.ID),
					"seconds_until":      int(triggerTime.Sub(nowUTC).Seconds()),
					"is_past_trigger":    nowUTC.After(triggerTime),
					"is_past_end":        nowUTC.After(task.EndTime),
				}
				taskDetails = append(taskDetails, taskDetail)
			}
			debugInfo["pending_tasks"] = taskDetails
			debugInfo["pending_tasks_count"] = len(pendingTasks)
		}
	}

	response.GinSuccess(c, debugInfo)
}

// registerFrontendRoutes 注册前端静态文件服务
func (a *MinimalApp) registerFrontendRoutes() {
	// 检查是否存在前端构建文件
	if !frontend.HasFiles() {
		a.logger.Warn("前端静态文件未找到，跳过前端路由注册。请先运行 'npm run build' 构建前端")
		return
	}

	a.logger.Info("注册前端静态文件服务")

	// 创建静态文件服务器
	staticFS := frontend.FS()

	// 注册根路由处理前端请求
	a.router.GET("/", func(c *gin.Context) {
		serveFile(c, staticFS, "index.html")
	})

	// 注册静态资源路由
	a.router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 如果请求路径是 API 路径或 WebSocket 路径，返回 404
		if len(path) >= 4 && path[:4] == "/api" || len(path) >= 3 && path[:3] == "/ws" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// 去掉前导斜杠
		filePath := path[1:]
		if filePath == "" {
			filePath = "index.html"
		}

		// 尝试打开文件
		file, err := staticFS.Open(filePath)
		if err != nil {
			// 文件不存在，对于 SPA 返回 index.html
			serveFile(c, staticFS, "index.html")
			return
		}
		defer file.Close()

		// 检查是否是目录
		stat, err := file.Stat()
		if err != nil || stat.IsDir() {
			// 是目录或出错，返回 index.html
			serveFile(c, staticFS, "index.html")
			return
		}

		// 文件存在，直接服务
		serveFile(c, staticFS, filePath)
	})

	a.logger.Info("前端静态文件服务已注册")
}

// serveFile 服务文件到 HTTP 响应
func serveFile(c *gin.Context, fs http.FileSystem, name string) {
	file, err := fs.Open(name)
	if err != nil {
		c.String(http.StatusNotFound, "File not found")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		c.String(http.StatusInternalServerError, "Stat error")
		return
	}

	// 设置 Content-Type
	contentType := getContentType(name)
	c.Header("Content-Type", contentType)

	// 设置缓存头
	c.Header("Cache-Control", "public, max-age=3600")

	// 使用 http.ServeContent 服务文件
	http.ServeContent(c.Writer, c.Request, name, stat.ModTime(), file)
}

// getContentType 根据文件名获取 Content-Type
func getContentType(filename string) string {
	ext := filename
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		ext = filename[idx:]
	}

	switch strings.ToLower(ext) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}
