/**
 * 轻量 JWT 解码（仅解析 payload，不做签名校验；签名校验由服务端负责）。
 * 前端解码仅用于本地 scope 防护：audience、store_id、subject_type、过期判断。
 */

import type { StoreTokenClaims } from '@/types/auth'

/** base64url -> JSON 对象；失败返回 null。 */
export function decodeJwt(token: string | null | undefined): StoreTokenClaims | null {
  if (!token) return null
  const parts = token.split('.')
  if (parts.length !== 3) return null
  try {
    const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = payload.padEnd(payload.length + ((4 - (payload.length % 4)) % 4), '=')
    const json = decodeURIComponent(
      atob(padded)
        .split('')
        .map((c) => '%' + c.charCodeAt(0).toString(16).padStart(2, '0'))
        .join(''),
    )
    return JSON.parse(json) as StoreTokenClaims
  } catch {
    return null
  }
}

/** token 是否已过期（带 30s 容差）。无 exp 视为未过期。 */
export function isTokenExpired(claims: StoreTokenClaims | null): boolean {
  if (!claims?.exp) return false
  return Date.now() >= claims.exp * 1000 - 30_000
}

/** audience 可能是字符串或数组，统一判断是否包含目标 audience。 */
export function audienceMatches(claims: StoreTokenClaims | null, expected: string): boolean {
  if (!claims?.aud) return false
  return Array.isArray(claims.aud) ? claims.aud.includes(expected) : claims.aud === expected
}
