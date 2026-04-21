package services

import (
	"errors"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// UserService 用户服务
type UserService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewUserService 创建用户服务
func NewUserService(db *gorm.DB, logger *zap.Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

// ListRequest 用户列表请求
type ListUsersRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size" binding:"max=100"`
	Keyword  string `form:"keyword"`
	RoleID   uint   `form:"role_id"`
	IsActive *bool  `form:"is_active"`
}

// ListResponse 用户列表响应
type ListUsersResponse struct {
	Total int64         `json:"total"`
	Items []models.User `json:"items"`
}

// CreateRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
	Email    string `json:"email" binding:"omitempty,email"`
	FullName string `json:"full_name" binding:"omitempty,max=100"`
	RoleID   uint   `json:"role_id" binding:"required"`
	IsActive bool   `json:"is_active"`
}

// UpdateRequest 更新用户请求
type UpdateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	FullName string `json:"full_name" binding:"omitempty,max=100"`
	RoleID   uint   `json:"role_id"`
	IsActive *bool  `json:"is_active"`
}

// AssignRolesRequest 分配角色请求
type AssignRolesRequest struct {
	RoleIDs       []uint `json:"role_ids" binding:"required,min=1"`
	CurrentUserID uint   `json:"-"` // 当前执行操作的用户ID
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(req *ListUsersRequest) (*ListUsersResponse, error) {
	var users []models.User
	var total int64

	query := s.db.Model(&models.User{})

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR full_name LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 角色筛选
	if req.RoleID > 0 {
		query = query.Where("role_id = ?", req.RoleID)
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
	if err := query.Preload("Role").
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
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.Preload("Role").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建用户
func (s *UserService) CreateUser(req *CreateUserRequest) (*models.User, error) {
	// 检查用户名是否存在
	var existing models.User
	if err := s.db.Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否存在
	if req.Email != "" {
		if err := s.db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
			return nil, errors.New("邮箱已被使用")
		}
	}

	// 检查角色是否存在
	var role models.Role
	if err := s.db.First(&role, req.RoleID).Error; err != nil {
		return nil, errors.New("角色不存在")
	}

	// 创建用户
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
		RoleID:   req.RoleID,
		IsActive: req.IsActive,
	}

	if err := user.SetPassword(req.Password); err != nil {
		return nil, err
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	// 重新加载用户信息
	s.db.Preload("Role").First(user, user.ID)

	return user, nil
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(id uint, req *UpdateUserRequest) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	// 检查邮箱是否被其他用户使用
	if req.Email != "" && req.Email != user.Email {
		var existing models.User
		if err := s.db.Where("email = ? AND id != ?", req.Email, id).First(&existing).Error; err == nil {
			return nil, errors.New("邮箱已被其他用户使用")
		}
		user.Email = req.Email
	}

	// 更新角色
	if req.RoleID > 0 {
		var role models.Role
		if err := s.db.First(&role, req.RoleID).Error; err != nil {
			return nil, errors.New("角色不存在")
		}
		user.RoleID = req.RoleID
	}

	// 更新其他字段
	if req.FullName != "" {
		user.FullName = req.FullName
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	// 重新加载用户信息
	s.db.Preload("Role").First(&user, user.ID)

	return &user, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id uint) error {
	// 不允许删除ID为1的管理员
	if id == 1 {
		return errors.New("不允许删除系统管理员")
	}

	result := s.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("用户不存在")
	}

	return nil
}

// ResetPassword 重置用户密码
func (s *UserService) ResetPassword(id uint, newPassword string) error {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return errors.New("用户不存在")
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return s.db.Save(&user).Error
}

// ToggleUserStatus 切换用户状态
func (s *UserService) ToggleUserStatus(id uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	// 不允许禁用ID为1的管理员
	if user.ID == 1 && user.IsActive {
		return nil, errors.New("不允许禁用系统管理员")
	}

	user.IsActive = !user.IsActive
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// AssignRoles 分配多个角色给用户
func (s *UserService) AssignRoles(userID uint, req *AssignRolesRequest) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 验证角色ID是否存在
	var roles []models.Role
	if err := s.db.Find(&roles, req.RoleIDs).Error; err != nil {
		return err
	}
	if len(roles) != len(req.RoleIDs) {
		return errors.New("部分角色不存在")
	}

	// D-13: 仅管理员可分配 shared_viewer 角色
	var currentUser models.User
	if req.CurrentUserID > 0 {
		if err := s.db.Preload("Roles").First(&currentUser, req.CurrentUserID).Error; err == nil {
			for _, role := range roles {
				if role.Name == models.RoleSharedViewer && !currentUser.HasRole(models.RoleAdmin) {
					return errors.New("仅管理员可分配'共享查看者'角色")
				}
			}
		}
	}

	// 使用 Clear + Append 方式（参考 RoleService.AssignPermissions）
	if err := s.db.Model(&user).Association("Roles").Clear(); err != nil {
		return err
	}

	if err := s.db.Model(&user).Association("Roles").Append(roles); err != nil {
		return err
	}

	return nil
}
