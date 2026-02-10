package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AuditLogService 审计日志服务
type AuditLogService struct {
	db         *gorm.DB
	logger     *zap.Logger
	sanitizer  *Sanitizer
	asyncQueue chan *models.AuditLog
	stopCh     chan struct{}
}

// LogOperationRequest 操作日志请求
type LogOperationRequest struct {
	UserID    uint
	Username  string
	RoleID    uint
	RoleName  string
	Action    string
	Module    string
	Resource  string
	ResourceID *uint

	// 请求上下文
	RequestID string
	TraceID   string
	Method    string
	Path      string

	// 变更数据
	OldData interface{}
	NewData interface{}

	// 执行结果
	Status    string
	ErrorMsg  string
	ErrorCode string

	// 环境信息
	IPAddress string
	UserAgent string
	Duration  int64
}

// QueryRequest 查询请求
type QueryRequest struct {
	// 基础筛选
	Module   string `form:"module"`
	Action   string `form:"action"`
	Status   string `form:"status"`
	UserID   uint   `form:"user_id"`
	Username string `form:"username"`

	// 时间范围
	StartTime time.Time `form:"start_time"`
	EndTime   time.Time `form:"end_time"`

	// 搜索
	Keyword string `form:"keyword"`

	// 分页
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`

	// 排序
	OrderBy string `form:"order_by"`
	Order   string `form:"order" binding:"oneof=asc desc"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	Items      []*models.AuditLog `json:"items"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// StatisticsItem 统计项
type StatisticsItem struct {
	Date  string `json:"date"`
	Count  int    `json:"count"`
	Module string `json:"module"`
	Action string `json:"action"`
}

// StatisticsResponse 统计响应
type StatisticsResponse struct {
	TotalOps    int64             `json:"total_ops"`
	SuccessOps  int64             `json:"success_ops"`
	FailureOps  int64             `json:"failure_ops"`
	TopUsers    []StatisticsItem  `json:"top_users"`
	TopModules  []StatisticsItem  `json:"top_modules"`
	DailyStats  []StatisticsItem  `json:"daily_stats"`
}

// NewAuditLogService 创建审计日志服务
func NewAuditLogService(db *gorm.DB, logger *zap.Logger) *AuditLogService {
	service := &AuditLogService{
		db:         db,
		logger:     logger,
		sanitizer:  NewSanitizer(),
		asyncQueue: make(chan *models.AuditLog, 1000),
		stopCh:     make(chan struct{}),
	}

	// 启动异步处理goroutine
	go service.processQueue()

	return service
}

// Stop 停止服务
func (s *AuditLogService) Stop() {
	close(s.stopCh)
}

// LogOperation 记录操作日志（异步）
func (s *AuditLogService) LogOperation(ctx context.Context, req *LogOperationRequest) error {
	if req == nil {
		return nil
	}

	// 1. 脱敏处理
	oldData := req.OldData
	newData := req.NewData
	if oldData != nil {
		oldData = s.sanitizer.Sanitize(oldData)
	}
	if newData != nil {
		newData = s.sanitizer.Sanitize(newData)
	}

	// 2. 生成差异
	diff := s.generateDiff(oldData, newData)

	// 3. 序列化数据
	oldDataJSON, _ := json.Marshal(oldData)
	newDataJSON, _ := json.Marshal(newData)
	diffJSON, _ := json.Marshal(diff)

	// 4. 创建日志记录
	log := &models.AuditLog{
		UserID:     req.UserID,
		Username:   req.Username,
		RoleID:     req.RoleID,
		RoleName:   req.RoleName,
		Action:     req.Action,
		Module:     req.Module,
		Resource:   req.Resource,
		ResourceID: req.ResourceID,
		RequestID:  req.RequestID,
		TraceID:    req.TraceID,
		Method:     req.Method,
		Path:       req.Path,
		OldData:    string(oldDataJSON),
		NewData:    string(newDataJSON),
		DiffData:   string(diffJSON),
		Status:     req.Status,
		ErrorMsg:   req.ErrorMsg,
		ErrorCode:  req.ErrorCode,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
		Duration:   req.Duration,
		CreatedAt:  time.Now(),
	}

	// 5. 放入异步队列
	select {
	case s.asyncQueue <- log:
		return nil
	default:
		// 队列满，同步写入
		s.logger.Warn("审计日志队列已满，同步写入")
		return s.db.Create(log).Error
	}
}

// processQueue 异步处理队列
func (s *AuditLogService) processQueue() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	batch := make([]*models.AuditLog, 0, 100)

	for {
		select {
		case log, ok := <-s.asyncQueue:
			if !ok {
				// 队列关闭，处理剩余数据
				if len(batch) > 0 {
					s.flushBatch(batch)
				}
				return
			}
			batch = append(batch, log)
			if len(batch) >= 100 {
				s.flushBatch(batch)
				batch = make([]*models.AuditLog, 0, 100)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.flushBatch(batch)
				batch = make([]*models.AuditLog, 0, 100)
			}
		case <-s.stopCh:
			if len(batch) > 0 {
				s.flushBatch(batch)
			}
			return
		}
	}
}

// flushBatch 批量写入
func (s *AuditLogService) flushBatch(batch []*models.AuditLog) {
	if err := s.db.CreateInBatches(batch, 100).Error; err != nil {
		s.logger.Error("批量写入审计日志失败",
			zap.Int("count", len(batch)),
			zap.Error(err),
		)
	}
}

// generateDiff 生成数据差异
func (s *AuditLogService) generateDiff(oldData, newData interface{}) interface{} {
	// 简单实现：返回包含新旧数据的结构
	// 实际项目中可以使用 github.com/r3labs/diff 或类似库
	return map[string]interface{}{
		"old": oldData,
		"new": newData,
	}
}

// Query 查询审计日志（带数据范围权限）
func (s *AuditLogService) Query(ctx context.Context, req *QueryRequest, userID uint, dataScope string) (*QueryResponse, error) {
	query := s.db.Model(&models.AuditLog{})

	// 应用数据范围权限（个人只能看自己的日志）
	if dataScope == "self" {
		query = query.Where("user_id = ?", userID)
	}

	// 应用筛选条件
	if req.Module != "" {
		query = query.Where("module = ?", req.Module)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Username != "" {
		// 转义 LIKE 特殊字符，防止 SQL 注入
		escapedUsername := strings.ReplaceAll(req.Username, "%", "\\%")
		escapedUsername = strings.ReplaceAll(escapedUsername, "_", "\\_")
		query = query.Where("username LIKE ?", "%"+escapedUsername+"%")
	}
	if !req.StartTime.IsZero() {
		query = query.Where("created_at >= ?", req.StartTime)
	}
	if !req.EndTime.IsZero() {
		query = query.Where("created_at <= ?", req.EndTime)
	}
	if req.Keyword != "" {
		// 转义关键词中的 LIKE 特殊字符
		escapedKeyword := strings.ReplaceAll(req.Keyword, "%", "\\%")
		escapedKeyword = strings.ReplaceAll(escapedKeyword, "_", "\\_")
		query = query.Where("resource LIKE ? OR error_msg LIKE ?",
			"%"+escapedKeyword+"%", "%"+escapedKeyword+"%")
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序 - 使用白名单验证防止注入
	orderBy := req.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	// OrderBy 白名单
	allowedOrderBy := map[string]bool{
		"created_at": true, "username": true, "module": true,
		"action": true, "status": true, "duration": true,
	}
	if !allowedOrderBy[orderBy] {
		orderBy = "created_at"
	}

	order := req.Order
	if order == "" {
		order = "desc"
	}
	// Order 方向白名单
	allowedOrder := map[string]bool{
		"asc": true, "desc": true, "ASC": true, "DESC": true,
	}
	if !allowedOrder[order] {
		order = "desc"
	}

	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// 分页
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	var logs []*models.AuditLog
	err := query.Offset((req.Page - 1) * req.PageSize).
		Limit(req.PageSize).
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	// 计算总页数
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &QueryResponse{
		Items:      logs,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByID 获取单条日志详情
func (s *AuditLogService) GetByID(ctx context.Context, id uint, userID uint, dataScope string) (*models.AuditLog, error) {
	var log models.AuditLog
	err := s.db.First(&log, id).Error
	if err != nil {
		return nil, err
	}

	// 检查数据范围权限
	if dataScope == "self" && log.UserID != userID {
		return nil, gorm.ErrRecordNotFound
	}

	return &log, nil
}

// GetStatistics 获取操作统计
func (s *AuditLogService) GetStatistics(ctx context.Context, days int, userID uint, dataScope string) (*StatisticsResponse, error) {
	if days < 1 {
		days = 7
	}

	startDate := time.Now().AddDate(0, 0, -days)
	query := s.db.Model(&models.AuditLog{}).Where("created_at >= ?", startDate)

	// 应用数据范围权限
	if dataScope == "self" {
		query = query.Where("user_id = ?", userID)
	}

	// 总操作数
	var totalOps int64
	query.Count(&totalOps)

	// 成功/失败统计
	var successOps, failureOps int64
	s.db.Model(&models.AuditLog{}).
		Where("created_at >= ? AND status = ?", startDate, models.StatusSuccess).
		Count(&successOps)

	s.db.Model(&models.AuditLog{}).
		Where("created_at >= ? AND status = ?", startDate, models.StatusFailure).
		Count(&failureOps)

	// 按模块统计
	var moduleStats []struct {
		Module string
		Count  int64
	}
	s.db.Model(&models.AuditLog{}).
		Select("module, count(*) as count").
		Where("created_at >= ?", startDate).
		Group("module").
		Order("count DESC").
		Limit(5).
		Scan(&moduleStats)

	topModules := make([]StatisticsItem, len(moduleStats))
	for i, stat := range moduleStats {
		topModules[i] = StatisticsItem{
			Module: stat.Module,
			Count:  int(stat.Count),
		}
	}

	// 按用户统计
	var userStats []struct {
		Username string
		Count    int64
	}
	userQuery := s.db.Model(&models.AuditLog{}).
		Select("username, count(*) as count").
		Where("created_at >= ?", startDate).
		Group("username").
		Order("count DESC").
		Limit(5)

	if dataScope == "self" {
		userQuery = userQuery.Where("user_id = ?", userID)
	}
	userQuery.Scan(&userStats)

	topUsers := make([]StatisticsItem, len(userStats))
	for i, stat := range userStats {
		topUsers[i] = StatisticsItem{
			Date:  stat.Username,
			Count: int(stat.Count),
		}
	}

	// 每日统计
	var dailyStats []struct {
		Date  string
		Count int64
	}
	s.db.Model(&models.AuditLog{}).
		Select("date(created_at) as date, count(*) as count").
		Where("created_at >= ?", startDate).
		Group("date(created_at)").
		Order("date").
		Scan(&dailyStats)

	dailyResult := make([]StatisticsItem, len(dailyStats))
	for i, stat := range dailyStats {
		dailyResult[i] = StatisticsItem{
			Date:  stat.Date,
			Count: int(stat.Count),
		}
	}

	return &StatisticsResponse{
		TotalOps:   totalOps,
		SuccessOps: successOps,
		FailureOps: failureOps,
		TopUsers:   topUsers,
		TopModules: topModules,
		DailyStats: dailyResult,
	}, nil
}

// CleanupOldLogs 清理旧日志
func (s *AuditLogService) CleanupOldLogs(keepDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -keepDays)

	result := s.db.Where("created_at < ?", cutoffDate).Delete(&models.AuditLog{})
	if result.Error != nil {
		return result.Error
	}

	s.logger.Info("清理审计日志",
		zap.Int("rows", int(result.RowsAffected)),
		zap.Time("cutoff", cutoffDate),
	)

	return nil
}
