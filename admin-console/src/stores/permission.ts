/**
 * 权限 store（独立于门店后台权限上下文）。
 *
 * 权限真值来自 /admin/auth/me 返回的 permissions；
 * super_admin 默认拥有全部权限。前端权限只做显隐，服务端强制最终授权。
 */
import { defineStore } from 'pinia'
import { computed } from 'vue'
import { useAuthStore } from './auth'
import { ADMIN_ROLE } from '@/constants/roles'
import type { PermissionCode } from '@/constants/permissions'

export const usePermissionStore = defineStore('permission', () => {
  const auth = useAuthStore()

  const permissions = computed<string[]>(() => auth.user?.permissions ?? [])
  const isSuperAdmin = computed(() => auth.user?.role === ADMIN_ROLE.SUPER_ADMIN)

  function has(code: PermissionCode | undefined): boolean {
    if (!code) return true
    if (isSuperAdmin.value) return true
    return permissions.value.includes(code)
  }

  /** 需要满足全部权限码 */
  function hasAll(codes: PermissionCode[]): boolean {
    return codes.every((c) => has(c))
  }

  /** 满足任一权限码 */
  function hasAny(codes: PermissionCode[]): boolean {
    if (codes.length === 0) return true
    return codes.some((c) => has(c))
  }

  return { permissions, isSuperAdmin, has, hasAll, hasAny }
})
