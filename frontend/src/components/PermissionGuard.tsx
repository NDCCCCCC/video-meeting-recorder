// 权限控制组件

import { useAuthStore } from '../stores/authStore'
import { hasPermission, hasAnyPermission, hasAllPermissions } from '../utils/permissions'
import type { ReactNode } from 'react'

interface PermissionGuardProps {
  permission?: string
  permissions?: string[]
  mode?: 'any' | 'all'
  fallback?: ReactNode
  children: ReactNode
}

/**
 * 权限控制组件
 * 用于根据用户权限显示/隐藏内容
 *
 * @example
 * // 单个权限检查
 * <PermissionGuard permission="tasks:create">
 *   <Button>新建任务</Button>
 * </PermissionGuard>
 *
 * // 多个权限（任一）
 * <PermissionGuard permissions={['tasks:edit', 'tasks:delete']} mode="any">
 *   <Button>操作</Button>
 * </PermissionGuard>
 *
 * // 多个权限（全部）
 * <PermissionGuard permissions={['tasks:view', 'tasks:edit']} mode="all">
 *   <div>内容</div>
 * </PermissionGuard>
 *
 * // 提供备选内容
 * <PermissionGuard permission="tasks:delete" fallback={<span>无权限</span>}>
 *   <Button>删除</Button>
 * </PermissionGuard>
 */
export function PermissionGuard({
  permission,
  permissions,
  mode = 'any',
  fallback = null,
  children,
}: PermissionGuardProps) {
  const { user } = useAuthStore()

  let hasAccess = false

  if (permission) {
    // 单个权限检查
    hasAccess = hasPermission(user, permission)
  } else if (permissions && permissions.length > 0) {
    // 多个权限检查
    if (mode === 'all') {
      hasAccess = hasAllPermissions(user, permissions)
    } else {
      hasAccess = hasAnyPermission(user, permissions)
    }
  } else {
    // 没有指定权限，默认允许访问
    hasAccess = true
  }

  return <>{hasAccess ? children : fallback}</>
}

/**
 * 菜单权限控制组件
 * 用于根据用户权限控制菜单显示
 */
interface MenuPermissionGuardProps {
  path: string
  fallback?: ReactNode
  children: ReactNode
}

export function MenuPermissionGuard({ path, fallback = null, children }: MenuPermissionGuardProps) {
  const { user } = useAuthStore()

  const hasAccess = (() => {
    const MENU_PERMISSIONS: Record<string, string> = {
      '/tasks': 'tasks:view',
      '/files': 'files:view',
      '/system/users': 'users:view',
      '/system/roles': 'roles:view',
      '/system/input-configs': 'configs:view',
    }

    const required = MENU_PERMISSIONS[path]
    if (!required) return true
    return hasPermission(user, required)
  })()

  return <>{hasAccess ? children : fallback}</>
}
