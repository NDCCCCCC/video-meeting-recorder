package services

import (
	"errors"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RoleService 角色服务
type RoleService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewRoleService 创建角色服务
func NewRoleService(db *gorm.DB, logger *zap.Logger) *RoleService {
	return &RoleService{
		db:     db,
		logger: logger,
	}
}

// ListRolesRequest 角色列表请求
type ListRolesRequest struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Keyword  string `form:"keyword"`
}

// ListRolesResponse 角色列表响应
type ListRolesResponse struct {
	Total int64         `json:"total"`
	Items []models.Role `json:"items"`
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=50"`
	Description string `json:"description" binding:"max=200"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Description string `json:"description" binding:"max=200"`
}

// AssignPermissionsRequest 分配权限请求
type AssignPermissionsRequest struct {
	PermissionIDs []uint `json:"permission_ids" binding:"required"`
}

// ListRoles 获取角色列表
func (s *RoleService) ListRoles(req *ListRolesRequest) (*ListRolesResponse, error) {
	var roles []models.Role
	var total int64

	query := s.db.Model(&models.Role{})

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Preload("Permissions").
		Offset(offset).
		Limit(req.PageSize).
		Order("id ASC").
		Find(&roles).Error; err != nil {
		return nil, err
	}

	return &ListRolesResponse{
		Total: total,
		Items: roles,
	}, nil
}

// GetRoleByID 根据ID获取角色
func (s *RoleService) GetRoleByID(id uint) (*models.Role, error) {
	var role models.Role
	if err := s.db.Preload("Permissions").First(&role, id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(req *CreateRoleRequest) (*models.Role, error) {
	// 检查角色名是否存在
	var existing models.Role
	if err := s.db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, errors.New("角色名称已存在")
	}

	// 创建角色
	role := &models.Role{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.db.Create(role).Error; err != nil {
		return nil, err
	}

	return role, nil
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(id uint, req *UpdateRoleRequest) (*models.Role, error) {
	var role models.Role
	if err := s.db.First(&role, id).Error; err != nil {
		return nil, errors.New("角色不存在")
	}

	// 更新描述
	if req.Description != "" {
		role.Description = req.Description
	}

	if err := s.db.Save(&role).Error; err != nil {
		return nil, err
	}

	return &role, nil
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(id uint) error {
	// 不允许删除ID为1-4的默认角色
	if id <= 4 {
		return errors.New("不允许删除系统默认角色")
	}

	// 检查是否有用户使用该角色
	var count int64
	if err := s.db.Model(&models.User{}).Where("role_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该角色正在被使用，无法删除")
	}

	result := s.db.Delete(&models.Role{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("角色不存在")
	}

	return nil
}

// AssignPermissions 分配权限
func (s *RoleService) AssignPermissions(roleID uint, req *AssignPermissionsRequest) error {
	var role models.Role
	if err := s.db.First(&role, roleID).Error; err != nil {
		return errors.New("角色不存在")
	}

	// 验证权限ID是否存在
	var permissions []models.Permission
	if err := s.db.Find(&permissions, req.PermissionIDs).Error; err != nil {
		return err
	}
	if len(permissions) != len(req.PermissionIDs) {
		return errors.New("部分权限不存在")
	}

	// 替换角色权限
	if err := s.db.Model(&role).Association("Permissions").Replace(permissions); err != nil {
		return err
	}

	return nil
}

// GetRolePermissions 获取角色权限列表
func (s *RoleService) GetRolePermissions(roleID uint) ([]models.Permission, error) {
	var role models.Role
	if err := s.db.Preload("Permissions").First(&role, roleID).Error; err != nil {
		return nil, errors.New("角色不存在")
	}

	return role.Permissions, nil
}

// GetAllPermissions 获取所有权限
func (s *RoleService) GetAllPermissions() ([]models.Permission, error) {
	var permissions []models.Permission
	if err := s.db.Order("resource, action").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}
