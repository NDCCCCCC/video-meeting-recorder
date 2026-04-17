// 权限常量
export const PERMISSIONS = {
  // 仪表盘
  DASHBOARD_VIEW: 'dashboard:view',

  // 录制任务
  TASK_VIEW: 'tasks:view',
  TASK_CREATE: 'tasks:create',
  TASK_EDIT: 'tasks:edit',
  TASK_DELETE: 'tasks:delete',
  TASK_START: 'tasks:start',
  TASK_STOP: 'tasks:stop',
  RECORDING_SNAPSHOT: 'recording:snapshot',

  // 视频文件
  FILE_VIEW: 'files:view',
  FILE_DELETE: 'files:delete',
  FILE_SCAN: 'files:scan',
  FILE_SPLIT: 'files:split',

  // 审计日志
  AUDIT_VIEW: 'audit:view',

  // 系统管理
  USER_VIEW: 'users:view',
  USER_CREATE: 'users:create',
  USER_EDIT: 'users:edit',
  USER_DELETE: 'users:delete',

  ROLE_VIEW: 'roles:view',
  ROLE_CREATE: 'roles:create',
  ROLE_EDIT: 'roles:edit',
  ROLE_DELETE: 'roles:delete',

  // API密钥
  APIKEY_VIEW: 'apikey:view',
  APIKEY_CREATE: 'apikey:create',
  APIKEY_EDIT: 'apikey:edit',
  APIKEY_DELETE: 'apikey:delete',

  CONFIG_VIEW: 'configs:view',
  CONFIG_EDIT: 'configs:edit',

  // 系统设置
  SYSTEM_SETTINGS: 'system:settings',
} as const

// 菜单权限映射
export const MENU_PERMISSIONS: Record<string, string> = {
  '/dashboard': PERMISSIONS.DASHBOARD_VIEW,
  '/tasks': PERMISSIONS.TASK_VIEW,
  '/files': PERMISSIONS.FILE_VIEW,
  '/audit': PERMISSIONS.AUDIT_VIEW,
  '/system/users': PERMISSIONS.USER_VIEW,
  '/system/roles': PERMISSIONS.ROLE_VIEW,
  '/system/apikeys': PERMISSIONS.APIKEY_VIEW,
  '/system/huawei-configs': PERMISSIONS.CONFIG_VIEW,
  '/system/settings': PERMISSIONS.SYSTEM_SETTINGS,
}

// 菜单子项权限映射
export const SUBMENU_PERMISSIONS: Record<string, string> = {
  '/system/users': PERMISSIONS.USER_VIEW,
  '/system/roles': PERMISSIONS.ROLE_VIEW,
  '/system/apikeys': PERMISSIONS.APIKEY_VIEW,
  '/system/huawei-configs': PERMISSIONS.CONFIG_VIEW,
  '/system/settings': PERMISSIONS.SYSTEM_SETTINGS,
}

import type { User } from '../types/auth'

/**
 * 检查用户是否有指定权限
 * @param user 用户对象
 * @param permission 权限字符串 (如 'tasks:view')
 * @returns 是否有权限
 */
export function hasPermission(user: User | null, permission: string): boolean {
  if (!user) return false
  // 管理员拥有所有权限
  if (user.is_admin) return true
  return user.permissions?.includes(permission) ?? false
}

/**
 * 检查用户是否有任一权限
 * @param user 用户对象
 * @param permissions 权限数组
 * @returns 是否有任一权限
 */
export function hasAnyPermission(user: User | null, permissions: string[]): boolean {
  if (!user) return false
  if (user.is_admin) return true
  return permissions.some(perm => user.permissions?.includes(perm) ?? false)
}

/**
 * 检查用户是否有所有权限
 * @param user 用户对象
 * @param permissions 权限数组
 * @returns 是否有所有权限
 */
export function hasAllPermissions(user: User | null, permissions: string[]): boolean {
  if (!user) return false
  if (user.is_admin) return true
  return permissions.every(perm => user.permissions?.includes(perm) ?? false)
}

/**
 * 检查用户是否可以访问指定菜单
 * @param user 用户对象
 * @param path 菜单路径
 * @returns 是否可以访问
 */
export function canAccessMenu(user: User | null, path: string): boolean {
  const required = MENU_PERMISSIONS[path]
  if (!required) return true // 没有定义权限要求的菜单默认可访问
  return hasPermission(user, required)
}

/**
 * 检查用户是否可以访问指定子菜单
 * @param user 用户对象
 * @param path 子菜单路径
 * @returns 是否可以访问
 */
export function canAccessSubMenu(user: User | null, path: string): boolean {
  const required = SUBMENU_PERMISSIONS[path]
  if (!required) return true
  return hasPermission(user, required)
}

/**
 * 过滤菜单项，只返回用户有权限访问的菜单
 * @param user 用户对象
 * @param menuItems 菜单项数组
 * @returns 过滤后的菜单项（过滤掉无权限的项）
 */
export function filterMenuByPermission<T extends { key?: string; children?: T[] }>(
  user: User | null,
  menuItems: T[]
): T[] {
  return menuItems.filter((item) => {
    if (!item || (!item.key && !item.children)) return false
    // 处理带子菜单的项目
    if (item.children && Array.isArray(item.children)) {
      const filteredChildren = item.children.filter((child) => {
        if (!child.key) return true
        return canAccessMenu(user, String(child.key))
      })
      // 如果所有子菜单都被过滤掉了，也隐藏父菜单
      if (filteredChildren.length === 0) return false
      item.children = filteredChildren
      return true
    }
    // 处理普通菜单项
    if (!item.key) return true
    return canAccessMenu(user, String(item.key))
  }).filter(item => item !== null)
}

/**
 * 过滤子菜单项，只返回用户有权限访问的子菜单
 * @param user 用户对象
 * @param subMenuItems 子菜单项数组
 * @returns 过滤后的子菜单项
 */
export function filterSubMenuByPermission<T extends { key?: string }>(
  user: User | null,
  subMenuItems: T[]
): T[] {
  return subMenuItems.filter((item) => {
    if (!item || !item.key) return true
    return canAccessSubMenu(user, String(item.key))
  })
}
