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

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/common"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/frontend"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/handlers"
	huaweiapi "github.com/NDCCCCCC/video-meeting-recorder/internal/huawei"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/migrations"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/recorder"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/scheduler"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/notification"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/storage"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/video_recording"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite" // 纯Go SQLite驱动
)

// MinimalApp 应用程序结构
type MinimalApp struct {
	config       *config.Config
	logger       *zap.Logger
	db           *gorm.DB
	httpServer   *http.Server
	router       *gin.Engine
	tokenService *auth.SM4TokenService
	handlers     *Handlers
	services     map[string]common.Service
	wg           sync.WaitGroup
	// 调度器和协调器
	scheduler            *scheduler.VideoSimpleScheduler
	coordinator          *recorder.SimpleRecordingCoordinator
	huaweiManager        *huaweiapi.Manager
	huaweiConnector      *video_recording.HuaweiConferenceConnector
	videoTaskService     *services.VideoRecordingTaskService
	videoFileService     *services.VideoFileService
	conversionService    services.ConversionService
	splittingService     *services.SplittingService
	snapshotService      *services.SnapshotService
	transcriptionService *services.TranscriptionService
	rateLimiter          *services.RateLimiter
	slideCacheService    *services.SlideCacheService
	pptMergeService      *services.PPTMergeService
	pptEditorService     *services.PPTEditorService
	frameCaptureService  *services.FrameCaptureService
	depsManager          *services.PythonDepsManager
}

// Handlers 处理器集合
type Handlers struct {
	Auth          *handlers.AuthHandler
	User          *handlers.UserHandler
	Role          *handlers.RoleHandler
	Admin         *handlers.AdminHandler
	VideoTask     *handlers.VideoRecordingTaskHandler
	HuaweiConfig  *handlers.HuaweiConfigHandler
	VideoFile     *handlers.VideoFileHandler
	File          *handlers.FileHandler
	Audit         *handlers.AuditHandler
	Notification  *handlers.NotificationHandler
	System        *handlers.SystemHandler
	APIKey        *handlers.APIKeyHandler
	Split         *handlers.SplitHandler
	Transcription *handlers.TranscriptionHandler
	PPT           *handlers.PPThandler
	Dashboard     *handlers.DashboardHandler
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

	// 检查Python依赖
	if err := a.checkPythonDependencies(); err != nil {
		a.logger.Warn("Python依赖检查失败，PPT功能可能不可用", zap.Error(err))
		// 不阻止启动，仅记录警告
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

	// 临时禁用外键约束，避免重建表时的约束冲突
	a.db.Exec("PRAGMA foreign_keys = OFF")
	defer a.db.Exec("PRAGMA foreign_keys = ON")

	// 使用 GORM AutoMigrate 自动管理所有表结构
	// User 模型的 RoleID 字段已删除，与数据库 schema 一致
	err := a.db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.APIKey{},
		&models.APIKeyUsageLog{},
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
		&models.TranscriptionTask{},
		&models.TranscriptionText{},
		&models.SystemSetting{},
	)

	if err != nil {
		return fmt.Errorf("AutoMigrate failed: %w", err)
	}

	a.logger.Info("数据库迁移完成")
	return nil
}

// runCustomMigrations 执行自定义迁移（SQL迁移）
func (a *MinimalApp) runCustomMigrations() error {
	a.logger.Info("正在执行自定义数据库迁移...")

	// 获取注册的迁移
	migrations := migrations.GetRegisteredMigrations()

	// 执行每个迁移的 Up 方法
	for _, m := range migrations {
		migrationName := ""
		if mi, ok := m.(interface{ Name() string }); ok {
			migrationName = mi.Name()
		}

		a.logger.Info("执行迁移", zap.String("migration", migrationName))

		if mu, ok := m.(interface{ Up(*gorm.DB) error }); ok {
			if err := mu.Up(a.db); err != nil {
				a.logger.Error("迁移失败",
					zap.String("migration", migrationName),
					zap.Error(err),
				)
				return fmt.Errorf("migration %s failed: %w", migrationName, err)
			}
			a.logger.Info("迁移成功", zap.String("migration", migrationName))
		}
	}

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
		{Name: models.RoleSharedViewer, Description: "共享查看者"}, // D-04
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
			IsActive: true,
		}
		if err := admin.SetPassword("admin123"); err != nil {
			return fmt.Errorf("failed to set admin password: %w", err)
		}
		if err := a.db.Create(admin).Error; err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}

		// 使用 Association API 分配角色（多对多关系）
		if err := a.db.Model(admin).Association("Roles").Append(&adminRole); err != nil {
			return fmt.Errorf("failed to assign admin role: %w", err)
		}

		a.logger.Info("Created default admin user",
			zap.String("username", "admin"),
			zap.String("note", "请及时修改默认密码"),
		)
	} else if err == nil {
		// 用户已存在，确保角色关联正确
		if err := a.db.Model(&existingUser).Association("Roles").Append(&adminRole); err != nil {
			return fmt.Errorf("failed to ensure admin role: %w", err)
		}
		a.logger.Info("Ensured admin user has admin role", zap.Uint("userId", existingUser.ID))
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
		models.ResourceTaskCreate: "创建录制任务",
		models.ResourceTaskEdit:   "编辑录制任务",
		models.ResourceTaskDelete: "删除录制任务",
		models.ResourceTaskStart:  "启动录制任务",
		models.ResourceTaskStop:   "停止录制任务",
		models.ResourceFileView:   "查看视频文件",
		models.ResourceFileDelete: "删除视频文件",
		models.ResourceFileScan:      "扫描视频文件",
		models.ResourcePPTView:      "查看PPT文件",
		models.ResourcePPTDelete:    "删除PPT文件",
		models.ResourcePPTEdit:      "编辑PPT文件",
		models.ResourcePPTDownload:  "下载PPT文件",
		models.ResourceUserView:     "查看用户",
		models.ResourceUserCreate: "创建用户",
		models.ResourceUserEdit:   "编辑用户",
		models.ResourceUserDelete: "删除用户",
		models.ResourceRoleView:   "查看角色",
		models.ResourceRoleCreate: "创建角色",
		models.ResourceRoleEdit:   "编辑角色",
		models.ResourceRoleDelete: "删除角色",
		models.ResourceConfigView: "查看华为配置",
		models.ResourceConfigEdit: "编辑华为配置",
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

	// 为 shared_viewer 角色分配只读权限 (仅数据可见性)
	// shared_viewer 只控制是否能看到所有数据，操作权限由其他角色决定
	var sharedViewerRole models.Role
	if err := a.db.Where("name = ?", models.RoleSharedViewer).First(&sharedViewerRole).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			a.logger.Warn("shared_viewer role not found, skipping permission assignment")
		} else {
			return fmt.Errorf("failed to find shared_viewer role: %w", err)
		}
	} else {
		// 只分配查看权限，不分配操作权限
		viewOnlyPermissions := []string{
			models.ResourceDashboardView,
			models.ResourceTaskView,
			models.ResourceFileView,
			models.ResourceAuditView,
		}

		// 获取这些权限对象
		var permissions []models.Permission
		if err := a.db.Where("resource || ':' || action IN ?", viewOnlyPermissions).Find(&permissions).Error; err != nil {
			return fmt.Errorf("failed to find view-only permissions: %w", err)
		}

		// 检查是否需要关联权限
		var existingSharedViewer models.Role
		if err := a.db.Preload("Permissions").Where("name = ?", models.RoleSharedViewer).First(&existingSharedViewer).Error; err == nil {
			// 如果权限数量不匹配，则重新分配
			if len(existingSharedViewer.Permissions) != len(permissions) {
				if err := a.db.Model(&existingSharedViewer).Association("Permissions").Replace(&permissions); err != nil {
					return fmt.Errorf("failed to assign permissions to shared_viewer role: %w", err)
				}
				a.logger.Info("Assigned view-only permissions to shared_viewer role (data visibility only)", zap.Int("count", len(permissions)))
			}
		}
	}

	return nil
}

// checkPythonDependencies 检查Python依赖是否已安装
func (a *MinimalApp) checkPythonDependencies() error {
	a.logger.Info("检查Python依赖...")

	// 创建Python依赖管理器并存储到 app
	a.depsManager = services.NewPythonDepsManager(a.logger, a.config.Python.PreferUV)

	// 检查依赖
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := a.depsManager.CheckDependencies(ctx)
	if err != nil {
		return fmt.Errorf("Python依赖检查失败: %w", err)
	}

	a.logger.Info("Python依赖检查通过",
		zap.String("python_version", info.PythonVersion),
		zap.String("command", info.Command),
		zap.Any("packages", info.Packages))

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
	allowedOrigins := []string{
		"http://localhost:5173", // dev
		"http://localhost:8080", // production
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Header("Access-Control-Allow-Origin", allowed)
				break
			}
		}
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
	a.tokenService = auth.NewSM4TokenService(a.config, a.db, a.logger)
	authService := auth.NewService(a.config, a.db, a.logger)
	roleService := services.NewRoleService(a.db, a.logger)
	huaweiConfigService := services.NewHuaweiConfigService(a.db, a.logger, a.config)
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
	fileHandler := handlers.NewFileHandler(fileService, a.logger)

	// 审计日志服务（必须在 userService 之前创建）
	auditService := audit.NewAuditLogService(a.db, a.logger)
	userService := services.NewUserService(a.db, a.logger, auditService)
	auditHandler := handlers.NewAuditHandler(auditService)
	auditHandler.SetLogger(a.logger)

	// 通知服务
	notificationService := notification.NewNotificationService(a.db, a.logger, a.config)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	notificationHandler.SetLogger(a.logger)

	// API密钥服务
	apikeyService := services.NewAPIKeyService(a.db, a.logger)
	apikeyService.SetAuditService(auditService)
	apikeyHandler := handlers.NewAPIKeyHandler(apikeyService, a.logger)
	apikeyHandler.SetLogger(a.logger)

	// API密钥速率限制器
	a.rateLimiter = services.NewRateLimiter(a.logger)

	// 转换服务（需要 videoFileService）
	a.conversionService = services.NewFFmpegConversionService(a.db, a.logger, a.config, a.videoFileService)

	// 分割服务（需要 videoFileService）
	a.splittingService = services.NewSplittingService(a.db, a.logger, a.config, a.videoFileService)
	a.snapshotService = services.NewSnapshotService(a.db, a.logger, a.config, a.videoFileService)

	// 转录服务（需要 frame extractor, similarity detector, pptx generator）
	frameExtractor := services.NewFrameExtractor(a.config.FFmpeg.Path, a.logger)
	similarityDetector := services.NewSimilarityDetector(a.logger)
	pptxGenerator := services.NewPPTXGenerator(a.logger, a.config.Python.PreferUV)

	// Cloud transcription services (Phase 4)
	ossService, err := services.NewOSSService(&a.config.OSS, a.logger)
	if err != nil {
		a.logger.Warn("OSS服务初始化失败，云端转录不可用", zap.Error(err))
	}
	tingwuClient := services.NewTingwuClient(&a.config.Tingwu, a.logger)

	a.transcriptionService = services.NewTranscriptionService(
		a.db, a.logger, a.config,
		frameExtractor, similarityDetector, pptxGenerator,
		a.videoFileService,
		ossService, tingwuClient,
	)

	// Start OSS cleanup scheduler
	a.transcriptionService.StartOSSCleanupScheduler()

	// Timestamp mapper for video preview synchronization
	timestampMapper := services.NewTimestampMapper(a.db, a.logger)

	// PPT管理服务
	slideExtractor := services.NewSlideExtractor(a.logger, a.config.Python.PreferUV)
	a.slideCacheService = services.NewSlideCacheService(a.db, a.logger, a.config, slideExtractor)
	a.pptMergeService = services.NewPPTMergeService(a.db, a.logger, a.config, a.slideCacheService, a.depsManager)
	pptFileService := services.NewPPTFileService(a.db, a.logger, a.config)

	// Create PPT editor service (reuse existing similarityDetector and pptxGenerator from transcription service)
	a.pptEditorService = services.NewPPTEditorService(a.db, a.logger, a.config, a.slideCacheService, similarityDetector, pptxGenerator, timestampMapper)

	// Create frame capture service for slide capture feature (reuse ffprobe path from earlier)
	a.frameCaptureService = services.NewFrameCaptureService(a.config.FFmpeg.Path, ffprobePath, a.logger)

	// 华为管理器（使用数据库配置动态创建客户端）
	dbAdapter := &huaweiDBAdapter{db: a.db}
	a.huaweiManager = huaweiapi.NewManager(a.logger, dbAdapter)
	a.huaweiConnector = video_recording.NewHuaweiConferenceConnector(a.db, a.huaweiManager, a.logger)

	// 仪表板服务
	dashboardService := services.NewDashboardService(a.db, a.logger)

	// 配置服务 - 用于持久化系统配置到数据库
	configService := services.NewConfigService(a.db, a.logger, a.config)
	// 从数据库加载持久化的认证配置（覆盖 YAML 默认值）
	if err := configService.LoadAuthConfig(); err != nil {
		a.logger.Warn("Failed to load persisted auth config, using YAML defaults", zap.Error(err))
	}

	// 创建handlers
	a.handlers = &Handlers{
		Auth:          handlers.NewAuthHandler(authService, a.logger),
		User:          handlers.NewUserHandler(userService, a.logger),
		Role:          handlers.NewRoleHandler(roleService, a.logger),
		Admin:         handlers.NewAdminHandler(a.config, a.logger, configService, authService),
		VideoTask:     handlers.NewVideoRecordingTaskHandler(a.videoTaskService, a.logger, a.config),
		HuaweiConfig:  handlers.NewHuaweiConfigHandler(huaweiConfigService, a.logger, usbScanner),
		VideoFile:     handlers.NewVideoFileHandler(a.videoFileService, a.logger),
		File:          fileHandler,
		Audit:         auditHandler,
		Notification:  notificationHandler,
		System:        handlers.NewSystemHandler(a.db, a.logger, a.config),
		APIKey:        apikeyHandler,
		Split:         handlers.NewSplitHandler(a.splittingService, a.snapshotService, a.videoFileService, a.logger),
		Transcription: handlers.NewTranscriptionHandler(a.transcriptionService, a.videoFileService, timestampMapper, a.logger),
		PPT:           handlers.NewPPThandler(pptFileService, a.slideCacheService, a.pptMergeService, a.videoFileService, a.pptEditorService, a.frameCaptureService, a.logger),
		Dashboard:     handlers.NewDashboardHandler(dashboardService, a.logger),
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
	authenticated.Use(middleware.SM4Auth(a.tokenService))
	{
		authenticated.POST("/logout", a.handlers.Auth.Logout)
		authenticated.POST("/logout-all", a.handlers.Auth.LogoutAll)
		authenticated.POST("/change-password", a.handlers.Auth.ChangePassword)
		authenticated.GET("/me", a.handlers.Auth.GetCurrentUser)
		authenticated.POST("/ad/test-connection", a.handlers.Auth.TestADConnection)
	}

	// Admin auth configuration routes (admin-only)
	adminGroup := a.router.Group("/api/v1/admin/auth")
	adminGroup.Use(middleware.SM4Auth(a.tokenService), middleware.RequireRole(a.db, "admin"))
	{
		adminGroup.GET("/config", a.handlers.Admin.GetAuthConfig)
		adminGroup.PUT("/config", a.handlers.Admin.UpdateAuthConfig)
		adminGroup.GET("/me", a.handlers.Admin.GetCurrentUser)
		adminGroup.POST("/lookup-ad-user", a.handlers.Admin.LookupADUser)
	}

	// API路由组
	api := a.router.Group("/api/v1")
	api.Use(middleware.MultiAuth(a.db, a.tokenService, a.logger)) // 支持SM4 Token和API Key认证

	// 用户管理
	users := api.Group("/users")
	{
		users.GET("", a.handlers.User.ListUsers)                           // 获取用户列表
		users.GET("/profile", a.handlers.User.GetCurrentProfile)           // 获取当前用户资料
		users.PUT("/profile", a.handlers.User.UpdateCurrentProfile)        // 更新当前用户资料
		users.GET("/:id", a.handlers.User.GetUser)                         // 获取用户详情
		users.POST("", a.handlers.User.CreateUser)                         // 创建用户
		users.PUT("/:id", a.handlers.User.UpdateUser)                      // 更新用户
		users.DELETE("/:id", a.handlers.User.DeleteUser)                   // 删除用户
		users.POST("/:id/reset-password", a.handlers.User.ResetPassword)   // 重置密码
		users.POST("/:id/toggle-status", a.handlers.User.ToggleUserStatus) // 切换状态
	}

	// 角色管理
	roles := api.Group("/roles")
	{
		roles.GET("", a.handlers.Role.ListRoles)                          // 获取角色列表
		roles.GET("/:id", a.handlers.Role.GetRole)                        // 获取角色详情
		roles.POST("", a.handlers.Role.CreateRole)                        // 创建角色
		roles.PUT("/:id", a.handlers.Role.UpdateRole)                     // 更新角色
		roles.DELETE("/:id", a.handlers.Role.DeleteRole)                  // 删除角色
		roles.GET("/:id/permissions", a.handlers.Role.GetRolePermissions) // 获取角色权限
		roles.POST("/:id/permissions", a.handlers.Role.AssignPermissions) // 分配权限
	}

	// 权限列表
	api.GET("/permissions", a.handlers.Role.GetAllPermissions)

	// API密钥管理
	apikeys := api.Group("/apikeys")
	{
		apikeys.GET("", a.handlers.APIKey.ListAPIKeys)                    // 获取API密钥列表
		apikeys.POST("", a.handlers.APIKey.CreateAPIKey)                  // 创建API密钥
		apikeys.GET("/:id", a.handlers.APIKey.GetAPIKey)                  // 获取API密钥详情
		apikeys.PUT("/:id", a.handlers.APIKey.UpdateAPIKey)               // 更新API密钥
		apikeys.DELETE("/:id", a.handlers.APIKey.DeleteAPIKey)            // 删除API密钥
		apikeys.POST("/:id/toggle", a.handlers.APIKey.ToggleAPIKeyStatus) // 切换状态
		apikeys.GET("/:id/logs", a.handlers.APIKey.ListUsageLogs)         // 获取使用日志
		apikeys.GET("/:id/summary", a.handlers.APIKey.GetUsageLogSummary) // 获取使用统计
	}

	// 录制任务管理 (使用 /recordings 路径符合API文档规范)
	recordings := api.Group("/recordings")
	{
		recordings.GET("", a.handlers.VideoTask.ListTasks)                 // 获取任务列表
		recordings.GET("/:id", a.handlers.VideoTask.GetTask)               // 获取任务详情
		recordings.POST("", a.handlers.VideoTask.CreateTask)               // 创建任务
		recordings.POST("/auto", a.handlers.VideoTask.CreateTaskAuto)      // 自动创建任务（固定华为配置）
		recordings.PUT("/:id", a.handlers.VideoTask.UpdateTask)            // 更新任务
		recordings.DELETE("/:id", a.handlers.VideoTask.DeleteTask)         // 删除任务
		recordings.DELETE("/batch", a.handlers.VideoTask.BatchDeleteTasks) // 批量删除任务
		recordings.POST("/:id/start", a.handlers.VideoTask.StartTask)      // 启动任务
		recordings.POST("/:id/stop", a.handlers.VideoTask.StopTask)        // 停止任务
		recordings.POST("/:id/cancel", a.handlers.VideoTask.CancelTask)    // 取消任务
		recordings.POST("/:id/retry", a.handlers.VideoTask.RetryTask)      // 重试任务
		// 转换相关
		recordings.GET("/:id/conversion-status", a.handlers.VideoTask.GetConversionStatus) // 获取转换状态
		recordings.POST("/:id/conversion-retry", a.handlers.VideoTask.RetryConversion)     // 重试转换
		// HLS 预览相关
		recordings.GET("/:id/preview", a.handlers.VideoTask.GetHLSPreview) // 获取HLS预览信息
		// 注意：/:id/preview/stream/:file 路由已移至公开路由（无需Token认证）
	}

	// 任务管理（使用 /tasks 路径）
	tasks := api.Group("/tasks")
	{
		tasks.POST("/clear-stuck", a.handlers.VideoTask.ClearStuckTasks) // 清理卡住的任务
		tasks.POST("/:id/snapshot", a.handlers.Split.GenerateSnapshot)   // 生成录制快照
	}

	// 华为配置管理
	huaweiConfigs := api.Group("/huawei-configs")
	{
		huaweiConfigs.GET("/scan-devices", a.handlers.HuaweiConfig.ScanUSBDevices)             // 扫描USB设备
		huaweiConfigs.GET("/recommended-device", a.handlers.HuaweiConfig.GetRecommendedDevice) // 获取推荐设备
		huaweiConfigs.GET("", a.handlers.HuaweiConfig.ListConfigs)                             // 获取配置列表
		huaweiConfigs.GET("/active", a.handlers.HuaweiConfig.GetActiveConfigs)                 // 获取可用配置
		huaweiConfigs.GET("/:id", a.handlers.HuaweiConfig.GetConfig)                           // 获取配置详情
		huaweiConfigs.POST("", a.handlers.HuaweiConfig.CreateConfig)                           // 创建配置
		huaweiConfigs.POST("/test-stream", a.handlers.HuaweiConfig.TestStream)                 // 测试流媒体连接
		huaweiConfigs.PUT("/:id", a.handlers.HuaweiConfig.UpdateConfig)                        // 更新配置
		huaweiConfigs.DELETE("/:id", a.handlers.HuaweiConfig.DeleteConfig)                     // 删除配置
	}

	// 文件存储管理
	storage := api.Group("/storage")
	{
		storage.POST("/upload", a.handlers.File.Upload)   // 上传文件
		storage.GET("/quota", a.handlers.File.GetQuota)   // 获取配额
		storage.GET("", a.handlers.File.List)             // 获取文件列表
		storage.DELETE("/:id", a.handlers.File.Delete)    // 删除文件
		storage.POST("/:id/share", a.handlers.File.Share) // 生成分享链接
	}

	// 视频文件管理
	files := api.Group("/files")
	{
		files.GET("", a.handlers.VideoFile.ListFiles)                 // 获取文件列表
		files.GET("/stats", a.handlers.VideoFile.GetFileStats)        // 获取文件统计
		files.DELETE("/batch", a.handlers.VideoFile.BatchDeleteFiles) // 批量删除文件（必须在 /:id 之前）
		files.GET("/:id/download", a.handlers.VideoFile.DownloadFile) // 下载文件（必须在 /:id 之前）
		files.GET("/:id", a.handlers.VideoFile.GetFile)               // 获取文件详情
		files.DELETE("/:id", a.handlers.VideoFile.DeleteFile)         // 删除文件
		files.POST("/scan", a.handlers.VideoFile.ScanFiles)           // 扫描并导入文件
	}

	// 公开文件访问（无需认证）
	a.router.GET("/api/v1/files/download/:token", a.handlers.File.Download)
	a.router.GET("/api/v1/files/share/:token", a.handlers.File.ShareDownload)

	// 视频分割和快照
	videos := api.Group("/videos")
	{
		videos.POST("/:id/split", a.handlers.Split.SubmitSplit)                                  // 提交分割任务
		videos.GET("/:id/split-status", a.handlers.Split.GetSplitStatus)                         // 获取分割状态
		videos.GET("/:id/segments", a.handlers.Split.GetSegments)                                // 获取分割段落列表
		videos.POST("/:id/transcribe", a.handlers.Transcription.SubmitTranscription)             // 提交转录任务
		videos.GET("/:id/transcription-status", a.handlers.Transcription.GetTranscriptionStatus) // 获取转录状态
		videos.GET("/:id/transcription-text", a.handlers.Transcription.GetTranscriptionText)     // 获取转录文字内容
		videos.GET("/:id/ppts", a.handlers.PPT.GetPptsByVideo)                                   // 获取视频的所有PPT结果
		videos.POST("/:id/rename", a.handlers.VideoFile.RenameFile)                              // 重命名视频文件
	}

	// 转录任务管理
	transcriptions := api.Group("/transcriptions")
	{
		transcriptions.GET("/active", a.handlers.Transcription.ListActiveTasks)                       // 获取活跃的转录任务列表
	transcriptions.GET("/:videoFileId/timestamps", a.handlers.Transcription.GetTimestampMapHandler) // 获取时间戳映射
	}

	// PPT管理
	ppts := api.Group("/ppts")
	{
		ppts.GET("/:id/slides", a.handlers.PPT.GetSlides)                                  // 获取幻灯片图片列表
// 		ppts.GET("/:id/slides/:resolution/:filename", a.handlers.PPT.ServeSlideImage)      // 服务幻灯片图片
		ppts.POST("/merge", a.handlers.PPT.MergeSlides)                                    // 合并幻灯片
		ppts.GET("/:id/download", a.handlers.PPT.DownloadPPT)                              // 下载PPT文件
		ppts.DELETE("/:id", a.handlers.PPT.DeletePPT)                                      // 删除PPT
		ppts.POST("/:id/rename", a.handlers.PPT.RenamePPT)                                 // 重命名PPT文件
		ppts.GET("/:id/duplicates", a.handlers.PPT.DetectDuplicatesHandler)                // 检测重复幻灯片
		ppts.DELETE("/:id/slides", a.handlers.PPT.DeleteSlidesHandler)                     // 删除指定幻灯片
		ppts.POST("/:id/rollback", a.handlers.PPT.RollbackHandler)                         // 回滚到备份版本
			ppts.POST(":id/reorder", a.handlers.PPT.ReorderSlidesHandler)					// 重排序幻灯片
			ppts.POST(":id/capture", a.handlers.PPT.CaptureFrameHandler)					// 捕获视频帧
			ppts.POST(":id/slides", a.handlers.PPT.InsertSlideHandler)					// 插入幻灯片
		}


	// HLS 预览流文件访问（无需认证，但需要任务权限验证）
	a.router.GET("/api/v1/recordings/:id/preview/stream/:file", a.handlers.VideoTask.ServeHLSStream)

	// PPT幻灯片图片访问（无需认证，handler内部验证权限，用于<img>标签显示）
	a.router.GET("/api/v1/ppts/:id/slides/:resolution/:filename", a.handlers.PPT.ServeSlideImage)

	// 审计日志管理
	auditLog := api.Group("/audit")
	{
		auditLog.GET("/logs", a.handlers.Audit.Query)            // 查询审计日志
		auditLog.GET("/logs/:id", a.handlers.Audit.GetByID)      // 获取日志详情
		auditLog.GET("/statistics", a.handlers.Audit.Statistics) // 获取操作统计
	}

	// 通知管理
	notifications := api.Group("/notifications")
	{
		notifications.GET("", a.handlers.Notification.ListNotifications)           // 获取通知列表
		notifications.GET("/unread-count", a.handlers.Notification.GetUnreadCount) // 获取未读数量
		notifications.PUT("/:id/read", a.handlers.Notification.MarkAsRead)         // 标记为已读
		notifications.PUT("/read-all", a.handlers.Notification.MarkAllAsRead)      // 全部标记为已读
		notifications.GET("/settings", a.handlers.Notification.GetUserSetting)     // 获取通知配置
		notifications.PUT("/settings", a.handlers.Notification.UpdateUserSetting)  // 更新通知配置
	}

	// 仪表板（需要 admin 权限）
	dashboard := api.Group("/dashboard")
	dashboard.Use(middleware.RequirePermission(a.db, "dashboard", "view"))
	{
		dashboard.GET("/stats", a.handlers.Dashboard.GetStats) // 获取仪表板统计数据
	}

	// 系统管理（需要 admin 权限）
	system := api.Group("/system")
	{
		system.GET("/config", a.handlers.System.GetConfig)        // 获取系统配置
		system.PUT("/config", a.handlers.System.UpdateConfig)     // 更新系统配置
		system.POST("/clear-files", a.handlers.System.ClearFiles) // 清空文件数据库
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

	// 启动分割服务
	if a.splittingService != nil {
		if err := a.splittingService.Start(); err != nil {
			return fmt.Errorf("failed to start splitting service: %w", err)
		}
		a.logger.Info("分割服务启动成功")
	}

	// 启动转录服务
	if a.transcriptionService != nil {
		if err := a.transcriptionService.Start(); err != nil {
			return fmt.Errorf("failed to start transcription service: %w", err)
		}
		a.logger.Info("转录服务启动成功")
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

// GetDB 获取数据库连接
func (a *taskServiceAdapter) GetDB() *gorm.DB {
	return a.db
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

	// 启动HTTP/HTTPS服务器
	if a.config.Server.TLSEnabled {
		return a.startHTTPSServer()
	}

	// HTTP 模式
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

	// 停止分割服务
	if a.splittingService != nil {
		a.logger.Info("正在停止分割服务...")
		a.splittingService.Stop()
	}

	// 停止速率限制器
	if a.rateLimiter != nil {
		a.rateLimiter.Shutdown()
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

// startHTTPSServer 启动 HTTPS 服务器
func (a *MinimalApp) startHTTPSServer() error {
	// HTTPS 地址
	httpsAddr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.HTTPSPort)

	// 检查 HTTPS 端口是否可用
	if err := a.checkPort(httpsAddr); err != nil {
		return fmt.Errorf("HTTPS端口被占用: %w", err)
	}

	// 创建 HTTPS 服务器
	a.httpServer = &http.Server{
		Addr:         httpsAddr,
		Handler:      a.router,
		ReadTimeout:  time.Duration(a.config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(a.config.Server.WriteTimeout) * time.Second,
	}

	// 启动 HTTPS 服务器
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.logger.Info("正在启动HTTPS服务器", zap.String("address", httpsAddr))
		if err := a.httpServer.ListenAndServeTLS(a.config.Server.TLSCertFile, a.config.Server.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTPS服务器错误", zap.Error(err))
		}
	}()

	// 如果启用了 HTTP 到 HTTPS 的重定向，同时启动 HTTP 服务器
	if a.config.Server.RedirectHTTPToHTTPS {
		httpAddr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port)
		redirectServer := &http.Server{
			Addr:         httpAddr,
			Handler:      a.createRedirectHandler(),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			a.logger.Info("正在启动HTTP重定向服务器", zap.String("address", httpAddr), zap.String("redirect_to", httpsAddr))
			if err := redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				a.logger.Warn("HTTP重定向服务器错误", zap.Error(err))
			}
		}()
	}

	a.logger.Info("HTTPS应用启动成功",
		zap.String("https_address", httpsAddr),
	)

	return nil
}

// createRedirectHandler 创建 HTTP 到 HTTPS 的重定向处理器
func (a *MinimalApp) createRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 构造 HTTPS URL
		host := r.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			// 移除端口部分
			host = host[:idx]
		}
		target := fmt.Sprintf("https://%s:%d%s", host, a.config.Server.HTTPSPort, r.RequestURI)

		// 永久重定向到 HTTPS
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
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
					"id":               task.ID,
					"name":             task.Name,
					"status":           task.Status,
					"start_time":       task.StartTime.Format(time.RFC3339),
					"end_time":         task.EndTime.Format(time.RFC3339),
					"trigger_time":     triggerTime.Format(time.RFC3339),
					"pre_join_minutes": task.PreJoinMinutes,
					"is_scheduled":     a.scheduler.IsTaskScheduled(task.ID),
					"is_executing":     a.scheduler.IsTaskExecuting(task.ID),
					"seconds_until":    int(triggerTime.Sub(nowUTC).Seconds()),
					"is_past_trigger":  nowUTC.After(triggerTime),
					"is_past_end":      nowUTC.After(task.EndTime),
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
