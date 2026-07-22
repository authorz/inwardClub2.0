/**
 * 总后台独立 auth store（不复用门店后台登录态）。
 *
 * 关键防护（依据 ADMIN_CONSOLE_ARCHITECTURE.md 第 4 节 / 验收 11）：
 *  - 独立命名空间存储 token（storage.ts 已带 VITE_APP_ID 前缀）。
 *  - 登录 / refresh / me 拿到 token 后校验 aud=admin 且 subject_type 属于总后台允许集（super_admin）；
 *    不匹配立即清空登录态并回登录页。
 *  - 向 http client 注入 AuthProvider，统一处理 401 refresh 与登出。
 */
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { attachAuthProvider, http } from '@/api/http'
import { authService, type LoginPayload, type TokenPair } from '@/api/services/auth'
import type { AdminUser } from '@/api/models'
import { EXPECTED_AUDIENCE, isAdminSubjectType } from '@/constants/roles'
import { audienceMatches, decodeJwt, isJwtExpired } from '@/utils/jwt'
import { readStorage, removeStorage, STORAGE_KEYS, writeStorage } from '@/utils/storage'
import { toastError } from '@/utils/feedback'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(readStorage<string>(STORAGE_KEYS.ACCESS_TOKEN))
  const refreshToken = ref<string | null>(readStorage<string>(STORAGE_KEYS.REFRESH_TOKEN))
  const user = ref<AdminUser | null>(null)
  const initialized = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value)

  /** 校验 token 的 audience / subject_type 是否为总后台预期，防止门店后台 token 混用 */
  function tokenAudienceValid(token: string): boolean {
    const payload = decodeJwt(token)
    if (!payload) return false
    if (isJwtExpired(payload)) return false
    if (!audienceMatches(payload.aud, EXPECTED_AUDIENCE)) return false
    if (payload.subject_type && !isAdminSubjectType(payload.subject_type)) return false
    return true
  }

  function persist(): void {
    if (accessToken.value) writeStorage(STORAGE_KEYS.ACCESS_TOKEN, accessToken.value)
    else removeStorage(STORAGE_KEYS.ACCESS_TOKEN)
    if (refreshToken.value) writeStorage(STORAGE_KEYS.REFRESH_TOKEN, refreshToken.value)
    else removeStorage(STORAGE_KEYS.REFRESH_TOKEN)
  }

  function clear(): void {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    persist()
  }

  /** 应用 token pair 前先做 audience 校验；非法则清空并报错 */
  function applyTokenPair(pair: TokenPair): boolean {
    if (!tokenAudienceValid(pair.accessToken)) {
      clear()
      toastError('登录凭证不属于总后台，已清除登录态')
      return false
    }
    accessToken.value = pair.accessToken
    refreshToken.value = pair.refreshToken
    if (pair.user) user.value = pair.user
    persist()
    return true
  }

  async function login(payload: LoginPayload): Promise<boolean> {
    const pair = await authService.login(payload)
    if (!applyTokenPair(pair)) return false
    if (!user.value) await fetchMe()
    return true
  }

  async function fetchMe(): Promise<void> {
    user.value = await authService.me()
    // 二次防护：me 返回的身份也必须是 admin
    if (user.value.audience && user.value.audience !== EXPECTED_AUDIENCE) {
      clear()
      toastError('账号身份不匹配总后台，已清除登录态')
    }
  }

  async function refresh(): Promise<boolean> {
    if (!refreshToken.value) return false
    try {
      const pair = await authService.refresh(refreshToken.value)
      return applyTokenPair(pair)
    } catch {
      clear()
      return false
    }
  }

  async function logout(): Promise<void> {
    try {
      if (accessToken.value) await authService.logout()
    } catch {
      /* 忽略登出接口错误，本地一定清空 */
    } finally {
      clear()
    }
  }

  /** 应用启动时恢复会话：本地 token 非法/过期则清空 */
  async function bootstrap(): Promise<void> {
    if (initialized.value) return
    if (accessToken.value && !tokenAudienceValid(accessToken.value)) {
      // access token 过期，尝试用 refresh 续期
      const ok = await refresh()
      if (!ok) clear()
    }
    if (accessToken.value && !user.value) {
      try {
        await fetchMe()
      } catch {
        clear()
      }
    }
    initialized.value = true
  }

  // 装配 http client 的鉴权回调（避免循环依赖）
  attachAuthProvider({
    getAccessToken: () => accessToken.value,
    refresh,
    onUnauthorized: () => clear(),
  })
  // 让 http 实例可被 store 复用同一 baseURL（保持单实例）
  void http

  return {
    accessToken,
    refreshToken,
    user,
    initialized,
    isAuthenticated,
    login,
    logout,
    refresh,
    fetchMe,
    bootstrap,
    clear,
  }
})
