/**
 * 路由实例与全局守卫。
 *
 * 守卫职责：
 *  1) 未登录访问受保护路由 -> 跳登录页（带 redirect）。
 *  2) 已登录访问登录页 -> 跳工作台。
 *  3) 登录态存在但无该页面权限 -> 跳工作台并提示（前端显隐，服务端最终强制）。
 *  4) 首次进入先 bootstrap 恢复会话并校验 token audience。
 */
import { createRouter, createWebHistory } from 'vue-router'
import { routes, type AppRouteMeta } from './routes'
import { useAuthStore } from '@/stores/auth'
import { usePermissionStore } from '@/stores/permission'
import { toastError } from '@/utils/feedback'

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  const permission = usePermissionStore()

  if (!auth.initialized) {
    await auth.bootstrap()
  }

  const meta = to.meta as Partial<AppRouteMeta>
  const requiresAuth = meta.requiresAuth !== false

  // 登录页
  if (to.path === '/login') {
    return auth.isAuthenticated ? { path: '/dashboard' } : true
  }

  if (requiresAuth && !auth.isAuthenticated) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  // 权限校验（前端显隐）
  if (requiresAuth && meta.permission && !permission.has(meta.permission)) {
    toastError('无权限访问该页面')
    return { path: '/dashboard' }
  }

  // 更新页签标题
  if (meta.title) document.title = `${meta.title} · InwardClub 总后台`
  return true
})
