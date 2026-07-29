// 路由预加载工具 (bundle-preload: 在 hover/focus 时预加载)

// 预加载状态跟踪
const preloadCache = new Set<string>()

// 路由组件映射
const routeImports: Record<string, () => Promise<unknown>> = {
  '/dashboard': () => import('../pages/dashboard'),
  '/tasks': () => import('../pages/tasks'),
  '/files': () => import('../pages/files'),
  '/audit': () => import('../pages/audit'),
  '/system/users': () => import('../pages/system/users'),
  '/system/roles': () => import('../pages/system/roles'),
  '/system/apikeys': () => import('../pages/system/apikeys'),
  '/system/input-configs': () => import('../pages/system/input-configs'),
  '/system/auth-config': () => import('../pages/system/auth-config'),
  '/system/settings': () => import('../pages/system/settings'),
}

/**
 * 预加载指定路由的组件
 * @param path - 路由路径
 */
export function preloadRoute(path: string): void {
  // 检查精确匹配
  if (routeImports[path] && !preloadCache.has(path)) {
    preloadCache.add(path)
    routeImports[path]()
    return
  }

  // 检查前缀匹配（用于 /system/* 路由）
  const matchedKey = Object.keys(routeImports).find((key) => path.startsWith(key))
  if (matchedKey && !preloadCache.has(matchedKey)) {
    preloadCache.add(matchedKey)
    routeImports[matchedKey]()
  }
}

/**
 * 为链接元素添加预加载功能
 * @param element - HTML 元素
 * @param path - 路由路径
 */
export function setupRoutePreload(element: HTMLElement, path: string): () => void {
  const handleMouseEnter = () => preloadRoute(path)
  const handleFocus = () => preloadRoute(path)

  element.addEventListener('mouseenter', handleMouseEnter, { passive: true })
  element.addEventListener('focus', handleFocus, { passive: true })

  // 返回清理函数
  return () => {
    element.removeEventListener('mouseenter', handleMouseEnter)
    element.removeEventListener('focus', handleFocus)
  }
}

/**
 * 预加载多个路由（用于预测性加载）
 * @param paths - 路由路径数组
 */
export function preloadRoutes(paths: string[]): void {
  paths.forEach(preloadRoute)
}
