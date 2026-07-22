/**
 * 总后台认证服务（独立于门店后台）。
 * 只调用 /admin/auth/*。
 */
import { http } from '@/api/http'
import { API_PATHS } from '@/constants/api-paths'
import type { AdminUser } from '@/api/models'

export interface LoginPayload {
  username: string
  password: string
}

export interface TokenPair {
  accessToken: string
  refreshToken: string
  user?: AdminUser
}

export const authService = {
  // 登录响应为 { token, profile } 包裹结构（refresh 直接返回扁平 token），此处解包 token。
  login: async (payload: LoginPayload) =>
    (await http.post<{ token: TokenPair; profile?: AdminUser }>(API_PATHS.auth.login, payload)).token,
  refresh: (refreshToken: string) =>
    http.post<TokenPair>(API_PATHS.auth.refresh, { refreshToken }),
  me: () => http.get<AdminUser>(API_PATHS.auth.me),
  logout: () => http.post<void>(API_PATHS.auth.logout),
}
