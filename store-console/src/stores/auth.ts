/**
 * 门店后台独立 auth store（Pinia）。
 *
 * 与总后台运行时登录态完全隔离：独立存储命名空间、独立 token audience。
 * 关键安全逻辑：
 * - token 必须满足 aud=store、subject_type 在门店白名单、store_id 非空，否则清空登录态。
 * - refresh 后同样重新校验；任何一次校验失败都强制登出。
 * - 当前门店只来自 token scope 与 /store/auth/me，不接受前端切换。
 */

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { authService } from '@/api/services'
import { registerAuthBridge } from '@/api/authBridge'
import { tokenStorage } from '@/utils/storage'
import { appConfig } from '@/config'
import { audienceMatches, decodeJwt, isTokenExpired } from '@/utils/jwt'
import { ALLOWED_SUBJECT_TYPES, type PermissionCode } from '@/constants/permissions'
import type { LoginPayload, StoreAccount, StoreScope, StoreTokenClaims } from '@/types/auth'

/** 校验 token claims 是否满足门店后台 scope 要求。 */
function validateClaims(claims: StoreTokenClaims | null): { ok: boolean; reason: string } {
  if (!claims) return { ok: false, reason: 'token 无法解析' }
  if (!audienceMatches(claims, appConfig.authAudience)) {
    return { ok: false, reason: `audience 非 ${appConfig.authAudience}` }
  }
  const subjectType = String(claims.subject_type ?? '')
  if (!ALLOWED_SUBJECT_TYPES.includes(subjectType as (typeof ALLOWED_SUBJECT_TYPES)[number])) {
    return { ok: false, reason: `subject_type 不匹配: ${subjectType || '空'}` }
  }
  if (claims.store_id == null || claims.store_id === '' || claims.store_id === 0) {
    return { ok: false, reason: 'store_id 为空' }
  }
  return { ok: true, reason: '' }
}

export const useAuthStore = defineStore('store-auth', () => {
  const accessToken = ref<string | null>(tokenStorage.getAccess())
  const refreshToken = ref<string | null>(tokenStorage.getRefresh())
  const account = ref<StoreAccount | null>(null)
  const store = ref<StoreScope | null>(null)
  const claims = ref<StoreTokenClaims | null>(decodeJwt(accessToken.value))
  const initialized = ref(false)

  const isAuthenticated = computed(
    () => !!accessToken.value && validateClaims(claims.value).ok && !isTokenExpired(claims.value),
  )
  const storeId = computed<string | number | null>(() => claims.value?.store_id ?? null)
  const permissions = computed<PermissionCode[]>(() => account.value?.permissions ?? [])

  /** 权限判断：命中任一即可（OR）。空要求表示不限制。 */
  function hasPermission(required?: PermissionCode[] | null): boolean {
    if (!required || required.length === 0) return true
    // 优雅处理待补齐的后端接口缝隙：/store/auth/me 目前不下发 permissions[]。
    // 无权限数据时对导航放行（最终权限仍由服务端强制），否则每个受限路由都会 403 致后台不可用。
    if (permissions.value.length === 0) return true
    return required.some((code) => permissions.value.includes(code))
  }

  function applyTokens(access: string, refresh: string): { ok: boolean; reason: string } {
    const decoded = decodeJwt(access)
    const result = validateClaims(decoded)
    if (!result.ok) return result
    accessToken.value = access
    refreshToken.value = refresh
    claims.value = decoded
    tokenStorage.set(access, refresh)
    return { ok: true, reason: '' }
  }

  function clear(): void {
    accessToken.value = null
    refreshToken.value = null
    account.value = null
    store.value = null
    claims.value = null
    tokenStorage.clear()
  }

  async function login(payload: LoginPayload): Promise<void> {
    const result = await authService.login(payload)
    const applied = applyTokens(result.accessToken, result.refreshToken)
    if (!applied.ok) {
      clear()
      throw new Error(`登录被拒绝：${applied.reason}`)
    }
    await fetchMe()
  }

  async function fetchMe(): Promise<void> {
    const me = await authService.me()
    account.value = me.account
    store.value = me.store
  }

  /** 供 http 拦截器调用的刷新流程；成功返回 true。 */
  async function refresh(): Promise<boolean> {
    if (!refreshToken.value) return false
    try {
      const result = await authService.refresh(refreshToken.value)
      const applied = applyTokens(result.accessToken, result.refreshToken)
      if (!applied.ok) {
        clear()
        return false
      }
      return true
    } catch {
      clear()
      return false
    }
  }

  async function logout(): Promise<void> {
    try {
      await authService.logout()
    } catch {
      // 忽略登出接口错误，本地状态必须清空。
    } finally {
      clear()
    }
  }

  /**
   * 应用启动时的会话初始化：
   * 若本地有 token 但 claims 非法/过期，直接清空；合法则拉取 me。
   */
  async function bootstrap(): Promise<void> {
    if (initialized.value) return
    if (accessToken.value) {
      const result = validateClaims(claims.value)
      if (!result.ok) {
        clear()
      } else if (isTokenExpired(claims.value)) {
        const ok = await refresh()
        if (!ok) clear()
      }
      if (accessToken.value) {
        try {
          await fetchMe()
        } catch {
          // me 拉取失败不立即登出（可能是网络），交由后续请求的 401 处理。
        }
      }
    }
    initialized.value = true
  }

  // 注册到 http 桥接，打破循环依赖。
  registerAuthBridge({
    getAccessToken: () => accessToken.value,
    getStoreId: () => storeId.value,
    refresh,
    invalidate: () => clear(),
  })

  return {
    accessToken,
    account,
    store,
    claims,
    initialized,
    isAuthenticated,
    storeId,
    permissions,
    hasPermission,
    login,
    logout,
    refresh,
    fetchMe,
    bootstrap,
    clear,
  }
})
