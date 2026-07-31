package services

import (
	"context"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserService 用户服务
type UserService struct {
	db           *gorm.DB
	logger       *zap.Logger
	auditService *audit.AuditLogService
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, logger *zap.Logger, auditService *audit.AuditLogService) *UserService {
	return &UserService{
		db:           db,
		logger:       logger,
		auditService: auditService,
	}
}

// ListRequest 用户列表请求
type ListUsersRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size" binding:"max=100"`
	Keyword  string `form:"keyword"`
	IsActive *bool  `form:"is_active"`
}

// ListResponse 用户列表响应
type ListUsersResponse struct {
	Total int64         `json:"total"`
	Items []models.User `json:"items"`
}

// CreateRequest 创建用户请求
type CreateUserRequest struct {
	Username   string   `json:"username" binding:"required,min=3,max=50"`
	Password   string   `json:"password" binding:"required,min=8"`
	Email      string   `json:"email" binding:"omitempty,email"`
	FullName   string   `json:"full_name" binding:"omitempty,max=100"`
	RoleIDs    []uint   `json:"role_ids"`
	AllowedIPs []string `json:"allowed_ips"`
	IsActive   bool     `json:"is_active"`
}

// UpdateRequest 更新用户请求
type UpdateUserRequest struct {
	Email      string   `json:"email" binding:"omitempty,email"`
	FullName   string   `json:"full_name" binding:"omitempty,max=100"`
	RoleIDs    []uint   `json:"role_ids"`
	AllowedIPs []string `json:"allowed_ips"`
	IsActive   *bool    `json:"is_active"`
}

// AssignRolesRequest 分配角色请求
type AssignRolesRequest struct {
	RoleIDs       []uint `json:"role_ids" binding:"required,min=1"`
	CurrentUserID uint   `json:"-"` // 当前执行操作的用户ID
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	var users []models.User
	var total int64

	query := s.db.WithContext(ctx).Model(&models.User{})

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR full_name LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 状态筛选
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("Roles").
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return &ListUsersResponse{
		Total: total,
		Items: users,
	}, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}



// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	// 检查用户名是否存在
	var existing models.User
	if err := s.db.WithContext(ctx).Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, apperrors.ErrUsernameExists
	}

	// 检查邮箱是否存在
	if req.Email != "" {
		if err := s.db.WithContext(ctx).Where("email = ?", req.Email).First(&existing).Error; err == nil {
			return nil, apperrors.ErrEmailExists
		}
	}

	// 创建用户
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
		IsActive: req.IsActive,
	}

	if err := user.SetPassword(req.Password); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}

	// 分配角色（如果有）
	if len(req.RoleIDs) > 0 {
		// 验证角色ID是否存在
		var roles []models.Role
		if err := s.db.WithContext(ctx).Find(&roles, req.RoleIDs).Error; err != nil {
			return nil, err
		}
		if len(roles) != len(req.RoleIDs) {
			return nil, apperrors.ErrRoleNotFound
		}

		// 使用 AssignRoles 分配角色
		if err := s.AssignRoles(ctx, user.ID, &AssignRolesRequest{
			RoleIDs:       req.RoleIDs,
			CurrentUserID: 0, // 系统创建
		}); err != nil {
			return nil, err
		}
	}

	// 设置 IP 限制
	if len(req.AllowedIPs) > 0 {
		if err := user.SetAllowedIPs(req.AllowedIPs); err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Save(user).Error; err != nil {
			return nil, err
		}
	}

	// 重新加载用户信息
	s.db.WithContext(ctx).Preload("Roles").First(user, user.ID)

	return user, nil
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(ctx context.Context, id uint, req *UpdateUserRequest, currentUserID uint) (*models.User, *models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, id).Error; err != nil {
		return nil, nil, apperrors.ErrUserNotFound
	}
	oldUser := user

	// 检查邮箱是否被其他用户使用
	if req.Email != "" && req.Email != user.Email {
		var existing models.User
		if err := s.db.WithContext(ctx).Where("email = ? AND id != ?", req.Email, id).First(&existing).Error; err == nil {
			return nil, nil, apperrors.ErrEmailExists
		}
		user.Email = req.Email
	}

	// 更新角色
	if len(req.RoleIDs) > 0 {
		if err := s.UpdateRoles(ctx, id, req.RoleIDs, currentUserID); err != nil {
			return nil, nil, err
		}
	}

	// 更新其他字段
	if req.FullName != "" {
		user.FullName = req.FullName
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	// 更新 IP 限制（总是更新，包括清空）
	if err := user.SetAllowedIPs(req.AllowedIPs); err != nil {
		return nil, nil, err
	}

	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, nil, err
	}

	// 重新加载用户信息
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, user.ID).Error; err != nil {
		return nil, nil, err
	}

	return &oldUser, &user, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id uint) (*models.User, *models.User, error) {
	// 不允许删除ID为1的管理员
	if id == 1 {
		return nil, nil, apperrors.ErrSystemAdminProtected
	}

	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, id).Error; err != nil {
		return nil, nil, apperrors.ErrUserNotFound
	}
	oldUser := user

	result := s.db.WithContext(ctx).Delete(&user)
	if result.Error != nil {
		return nil, nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil, apperrors.ErrUserNotFound
	}

	return &oldUser, nil, nil
}

// ResetPassword 重置用户密码
func (s *UserService) ResetPassword(ctx context.Context, id uint, newPassword string) (map[string]interface{}, map[string]interface{}, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, nil, apperrors.ErrUserNotFound
	}

	// 密码重置快照明确排除 PasswordHash；审计只记录目标用户身份和动作结果。
	oldSnapshot := map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	}

	if err := user.SetPassword(newPassword); err != nil {
		return nil, nil, err
	}
	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, nil, err
	}

	newSnapshot := map[string]interface{}{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"password_reset": true,
	}
	return oldSnapshot, newSnapshot, nil
}

// ToggleUserStatus 切换用户状态
func (s *UserService) ToggleUserStatus(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	// 不允许禁用ID为1的管理员
	if user.ID == 1 && user.IsActive {
		return nil, apperrors.ErrSystemAdminProtected
	}

	user.IsActive = !user.IsActive
	if err := s.db.WithContext(ctx).Save(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// AssignRoles 分配多个角色给用户
func (s *UserService) AssignRoles(ctx context.Context, userID uint, req *AssignRolesRequest) error {
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return apperrors.ErrUserNotFound
	}

	// 验证角色ID是否存在
	var roles []models.Role
	if err := s.db.WithContext(ctx).Find(&roles, req.RoleIDs).Error; err != nil {
		return err
	}
	if len(roles) != len(req.RoleIDs) {
		return apperrors.ErrRoleNotFound
	}

	// 使用 Clear + Append 方式（参考 RoleService.AssignPermissions）
	if err := s.db.WithContext(ctx).Model(&user).Association("Roles").Clear(); err != nil {
		return err
	}

	if err := s.db.WithContext(ctx).Model(&user).Association("Roles").Append(roles); err != nil {
		return err
	}

	return nil
}

// UpdateRoles 更新用户角色（带审计日志）
func (s *UserService) UpdateRoles(ctx context.Context, userID uint, roleIDs []uint, currentUserID uint) error {
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, userID).Error; err != nil {
		return apperrors.ErrUserNotFound
	}

	// 调用 AssignRoles
	req := &AssignRolesRequest{
		RoleIDs:       roleIDs,
		CurrentUserID: currentUserID,
	}

	if err := s.AssignRoles(ctx, userID, req); err != nil {
		return err
	}

	return nil
}
