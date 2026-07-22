/**
 * 门店后台专属的本地存储封装。
 *
 * 关键点：所有 key 使用 `icsc:`（inwardclub store console）前缀，
 * 与总后台 (admin-console) 的存储命名空间物理隔离，禁止复用总后台登录态。
 */

const NS = 'icsc:'

const KEYS = {
  accessToken: `${NS}access_token`,
  refreshToken: `${NS}refresh_token`,
} as const

export const tokenStorage = {
  getAccess(): string | null {
    return localStorage.getItem(KEYS.accessToken)
  },
  getRefresh(): string | null {
    return localStorage.getItem(KEYS.refreshToken)
  },
  set(access: string, refresh: string): void {
    localStorage.setItem(KEYS.accessToken, access)
    localStorage.setItem(KEYS.refreshToken, refresh)
  },
  setAccess(access: string): void {
    localStorage.setItem(KEYS.accessToken, access)
  },
  clear(): void {
    localStorage.removeItem(KEYS.accessToken)
    localStorage.removeItem(KEYS.refreshToken)
  },
}
