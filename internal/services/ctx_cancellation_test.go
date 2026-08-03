package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// openCtxTestDB 建立一个内存 sqlite 并迁移角色/权限/用户表用于取消传播测试。
// 与 openInMemoryDB 分离以避免污染其精简 schema（仅 InputConfig/SystemSetting/VideoRecordingTask）。
func openCtxTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.User{},
	))
	return db
}

// TestRoleService_GetAllPermissions_PreCancelledCtx 验证 ctx 取消信号级联到 GORM：
// 预先取消的 ctx → 查询立即返回 context.Canceled（PERF-003/BUG-005 取消传播保证）。
func TestRoleService_GetAllPermissions_PreCancelledCtx(t *testing.T) {
	db := openCtxTestDB(t)
	svc := NewRoleService(db, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 预取消

	_, err := svc.GetAllPermissions(ctx)
	require.Error(t, err, "预取消 ctx 应使查询失败")
	assert.True(t, errors.Is(err, context.Canceled),
		"期望 context.Canceled，实际: %v", err)
}

// TestRoleService_ListRoles_PreCancelledCtx 验证带分页/计数的复杂查询也级联取消。
func TestRoleService_ListRoles_PreCancelledCtx(t *testing.T) {
	db := openCtxTestDB(t)
	svc := NewRoleService(db, zap.NewNop())

	// 预置一条数据确保查询不是空集（Count 仍会执行）
	require.NoError(t, db.Create(&models.Role{Name: "admin"}).Error)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.ListRoles(ctx, &ListRolesRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled),
		"期望 context.Canceled，实际: %v", err)
}

// TestUserService_GetUserByID_PreCancelledCtx 验证 UserService 的 ctx 级联。
func TestUserService_GetUserByID_PreCancelledCtx(t *testing.T) {
	db := openCtxTestDB(t)
	svc := NewUserService(db, zap.NewNop(), nil)

	// 先写入一个用户确保查询有目标
	require.NoError(t, db.Create(&models.User{Username: "alice"}).Error)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.GetUserByID(ctx, 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled),
		"期望 context.Canceled，实际: %v", err)
}

// TestUserService_ListUsers_PreCancelledCtx 验证 ListUsers（Model 链）级联取消。
func TestUserService_ListUsers_PreCancelledCtx(t *testing.T) {
	db := openCtxTestDB(t)
	svc := NewUserService(db, zap.NewNop(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.ListUsers(ctx, &ListUsersRequest{Page: 1, PageSize: 10})
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled),
		"期望 context.Canceled，实际: %v", err)
}
