/**
 * 认证与门店 scope 相关类型。
 */

import type { PermissionCode, StoreRole } from '@/constants/permissions'

/** 门店后台 JWT 载荷（解码后用于前端 scope 校验；最终判断以服务端为准）。 */
export interface StoreTokenClaims {
  subject_type?: string
  role?: string
  aud?: string | string[]
  store_id?: number | string | null
  token_version?: number
  exp?: number
  iat?: number
  [key: string]: unknown
}

/** 当前登录门店账号信息（来自 /store/auth/me）。 */
export interface StoreAccount {
  id: number | string
  name: string
  username?: string
  role: StoreRole
  subjectType: string
  storeId?: number | string
  status?: string
  phone?: string
  avatarUrl?: string
  permissions: PermissionCode[]
}

/** 当前门店信息（只读展示，不参与请求参数）。name 可能未知（见 AccountProfile）。 */
export interface StoreScope {
  id: number | string
  name?: string
  status?: string
  logoUrl?: string
}

/**
 * /store/auth/me 的服务端裸返回体（AccountProfile）。
 * 注意：不含 permissions[]，也不含 {account,store} 包裹（见 auth 服务归一）。
 */
export interface AccountProfile {
  id: number
  username: string
  displayName: string
  role: StoreRole
  storeId?: number
  status: string
}

/** /store/auth/me 归一化后的返回。 */
export interface StoreMe {
  account: StoreAccount
  store: StoreScope
}

/** 登录接口返回。 */
export interface LoginResult {
  accessToken: string
  refreshToken: string
  expiresIn?: number
}

/** 登录表单。 */
export interface LoginPayload {
  username: string
  password: string
}
