/**
 * 命名空间化的本地存储工具。
 *
 * 总后台使用独立的存储命名空间（含 VITE_APP_ID），确保即使与门店后台同域调试，
 * 登录态也不会互相覆盖或复用。禁止直接使用 localStorage 字符串键。
 */
const NAMESPACE = `${import.meta.env.VITE_APP_ID || 'inwardclub-admin-console'}:`

export function readStorage<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(NAMESPACE + key)
    return raw == null ? null : (JSON.parse(raw) as T)
  } catch {
    return null
  }
}

export function writeStorage<T>(key: string, value: T): void {
  try {
    localStorage.setItem(NAMESPACE + key, JSON.stringify(value))
  } catch {
    /* 忽略写入失败（隐私模式/配额） */
  }
}

export function removeStorage(key: string): void {
  try {
    localStorage.removeItem(NAMESPACE + key)
  } catch {
    /* noop */
  }
}

export const STORAGE_KEYS = {
  ACCESS_TOKEN: 'access_token',
  REFRESH_TOKEN: 'refresh_token',
  SIDER_COLLAPSED: 'sider_collapsed',
} as const
