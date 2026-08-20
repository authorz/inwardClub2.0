/**
 * 路由守卫：认证守卫 + 权限守卫。
 *
 * - 未认证访问受保护路由 -> 跳登录页（带 redirect）。
 * - 已认证访问登录页 -> 跳工作台。
 * - 认证但无页面所需权限码 -> 跳 403。
 * 最终权限仍由服务端强制，前端守卫仅用于收敛入口体验。
 */

import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import type { PermissionCode, StoreRole } from '@/constants/permissions'

export function installRouterGuards(router: Router): void {
  router.beforeEach(async (to) => {
    const auth = useAuthStore()

    // 首次进入时初始化会话（校验/刷新本地 token）。
    if (!auth.initialized) {
      await auth.bootstrap()
    }

    const isPublic = to.meta.public === true
    const requiresAuth = to.meta.requiresAuth === true

    if (isPublic) {
      // 已登录用户访问登录页 -> 回工作台。
      if (to.name === 'login' && auth.isAuthenticated) {
        return { name: 'dashboard' }
      }
      return true
    }

    if (requiresAuth && !auth.isAuthenticated) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }

    const required = (to.meta.permissions as PermissionCode[] | undefined) ?? []
    if (required.length > 0 && !auth.hasPermission(required)) {
      return { name: 'forbidden' }
    }

    const allowedRoles = (to.meta.roles as StoreRole[] | undefined) ?? []
    const currentRole = auth.account?.role ?? auth.claims?.role
    if (allowedRoles.length > 0 && !allowedRoles.includes(currentRole as StoreRole)) {
      return { name: 'forbidden' }
    }

    return true
  })

  router.afterEach((to) => {
    const title = (to.meta.title as string | undefined) ?? '门店后台'
    document.title = `${title} · InwardClub 门店后台`
  })
}
