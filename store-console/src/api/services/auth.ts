/**
 * 认证服务：登录、刷新、当前账号、登出。
 * 这些请求本身跳过 401 自动刷新，避免递归。
 */

import { get, post } from '../request'
import { API_PATHS } from '@/constants/apiPaths'
import type { AccountProfile, LoginPayload, LoginResult, StoreMe } from '@/types/auth'

export const authService = {
  async login(payload: LoginPayload): Promise<LoginResult> {
    // 登录响应为 { token, profile } 包裹结构（refresh 直接返回扁平 token），此处解包 token。
    const res = await post<{ token: LoginResult; profile?: unknown }>(API_PATHS.auth.login, payload, {
      skipAuthRefresh: true,
    })
    return res.token
  },

  refresh(refreshToken: string): Promise<LoginResult> {
    return post<LoginResult>(
      API_PATHS.auth.refresh,
      { refreshToken },
      { skipAuthRefresh: true },
    )
  },

  async me(): Promise<StoreMe> {
    // 优雅处理待补齐的后端接口缝隙：GET /store/auth/me 目前返回裸 AccountProfile
    // {id,username,displayName,role,storeId?,status}——既无 permissions[]，也无 {account,store} 包裹。
    // 这里把裸资料归一为内部 {account,store} 形状：name ← displayName，store 仅由 storeId 得到
    // （门店名未知，留空由布局兜底占位）。permissions 服务端尚未下发，置空，由 auth store 的
    // hasPermission 放行导航（见 stores/auth.ts）；不在前端臆造 role→permission 矩阵（未确认业务规则）。
    const profile = await get<AccountProfile>(API_PATHS.auth.me, { skipAuthRefresh: true })
    return {
      account: {
        id: profile.id,
        name: profile.displayName,
        username: profile.username,
        role: profile.role,
        subjectType: profile.role,
        storeId: profile.storeId,
        status: profile.status,
        permissions: [],
      },
      store: { id: profile.storeId ?? '' },
    }
  },

  logout(): Promise<void> {
    return post<void>(API_PATHS.auth.logout, undefined, { skipAuthRefresh: true })
  },
}
