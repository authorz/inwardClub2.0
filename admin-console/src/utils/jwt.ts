/**
 * 轻量 JWT 解析工具（仅解码 payload，不做签名校验）。
 *
 * 用途：登录后在前端校验 token 的 aud / subject_type，
 * 若不匹配总后台预期（audience=admin, subject_type=super_admin）则拒绝并清空登录态。
 * 真正的签名校验与授权由服务端强制，这里只是前端第一道防错配防护。
 */
export interface JwtPayload {
  sub?: string
  subject_type?: string
  role?: string
  aud?: string | string[]
  store_id?: string | number | null
  token_version?: number
  exp?: number
  [key: string]: unknown
}

function base64UrlDecode(input: string): string {
  const padded = input.replace(/-/g, '+').replace(/_/g, '/')
  const pad = padded.length % 4 === 0 ? '' : '='.repeat(4 - (padded.length % 4))
  const decoded = atob(padded + pad)
  // 处理 UTF-8
  try {
    return decodeURIComponent(
      decoded
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join(''),
    )
  } catch {
    return decoded
  }
}

export function decodeJwt(token: string): JwtPayload | null {
  const parts = token.split('.')
  if (parts.length !== 3) return null
  try {
    return JSON.parse(base64UrlDecode(parts[1])) as JwtPayload
  } catch {
    return null
  }
}

/** token 是否已过期（含 30s 时钟偏移余量） */
export function isJwtExpired(payload: JwtPayload | null): boolean {
  if (!payload?.exp) return false
  return Date.now() >= payload.exp * 1000 - 30_000
}

/** audience 声明可能是字符串或数组，统一判定是否包含期望值 */
export function audienceMatches(aud: JwtPayload['aud'], expected: string): boolean {
  if (aud == null) return false
  return Array.isArray(aud) ? aud.includes(expected) : aud === expected
}
