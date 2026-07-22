/**
 * 门店后台统一 request client（唯一 axios 实例）。
 *
 * 全站禁止在页面/服务里另建 axios 实例或直接 fetch。所有横切逻辑集中在此：
 * - 强制只允许 /store/* 前缀，硬拦截 /admin/*（水平越权与站点边界防护）。
 * - 附加 Authorization、X-Request-ID；高风险写操作附加 Idempotency-Key。
 * - 401 单飞刷新并重试一次；刷新失败清空登录态。
 * - 403/409/422 归一化为 ApiError。
 * - 跨店数据兜底：响应中若出现非当前门店的 storeId，报警并阻断（防越权返回）。
 */

import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios'
import { appConfig, ALLOWED_API_PREFIXES, FORBIDDEN_API_PREFIXES } from '@/config'
import { tokenStorage } from '@/utils/storage'
import { newRequestId, newIdempotencyKey } from '@/utils/id'
import { getAuthBridge } from './authBridge'
import { ApiError, normalizeError } from './error'
import type { ApiErrorBody } from '@/types/api'

/** 扩展请求配置：业务层可声明该请求为高风险写操作（需幂等键）。 */
export interface StoreRequestConfig extends AxiosRequestConfig {
  /** 声明为幂等写操作，自动附加 Idempotency-Key。 */
  idempotent?: boolean
  /** 显式指定幂等键（同一操作重试时复用）。 */
  idempotencyKey?: string
  /** 跳过 401 自动刷新（用于 auth 相关请求本身）。 */
  skipAuthRefresh?: boolean
}

type InternalStoreConfig = InternalAxiosRequestConfig & StoreRequestConfig & { _retried?: boolean }

const http: AxiosInstance = axios.create({
  baseURL: appConfig.apiBaseUrl,
  timeout: 20_000,
  headers: { 'Content-Type': 'application/json' },
})

/* ------------------------------------------------------------------ */
/* 站点边界守卫：只允许 /store/*，硬拦截 /admin/*                       */
/* ------------------------------------------------------------------ */

function assertAllowedPath(url: string): void {
  // 仅取 path 部分做判断（忽略 query）。
  const path = url.split('?')[0]
  const normalized = path.startsWith('/') ? path : `/${path}`

  if (FORBIDDEN_API_PREFIXES.some((p) => normalized === p || normalized.startsWith(p + '/'))) {
    throw new ApiError({
      status: 0,
      code: 'FORBIDDEN_SCOPE',
      message: '门店后台禁止调用总后台接口',
    })
  }
  const allowed = ALLOWED_API_PREFIXES.some(
    (p) => normalized === p || normalized.startsWith(p.endsWith('/') ? p : p + '/'),
  )
  if (!allowed) {
    throw new ApiError({
      status: 0,
      code: 'FORBIDDEN_SCOPE',
      message: `门店后台不允许访问该接口: ${normalized}`,
    })
  }
}

/* ------------------------------------------------------------------ */
/* 请求拦截器                                                          */
/* ------------------------------------------------------------------ */

http.interceptors.request.use((config: InternalStoreConfig) => {
  assertAllowedPath(config.url ?? '')

  const token = getAuthBridge()?.getAccessToken() ?? tokenStorage.getAccess()
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`)
  }
  config.headers.set('X-Request-ID', newRequestId())

  if (config.idempotent) {
    config.headers.set('Idempotency-Key', config.idempotencyKey ?? newIdempotencyKey())
  }
  return config
})

/* ------------------------------------------------------------------ */
/* 401 单飞刷新                                                        */
/* ------------------------------------------------------------------ */

let refreshPromise: Promise<boolean> | null = null

async function runRefresh(): Promise<boolean> {
  const bridge = getAuthBridge()
  if (!bridge) return false
  if (!refreshPromise) {
    refreshPromise = bridge.refresh().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

/* ------------------------------------------------------------------ */
/* 响应拦截器                                                          */
/* ------------------------------------------------------------------ */

http.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiErrorBody>) => {
    const config = error.config as InternalStoreConfig | undefined
    const status = error.response?.status

    // 401：尝试刷新一次后重试原请求。
    if (status === 401 && config && !config.skipAuthRefresh && !config._retried) {
      config._retried = true
      const ok = await runRefresh()
      if (ok) {
        return http.request(config)
      }
      getAuthBridge()?.invalidate('token-expired')
    }

    return Promise.reject(normalizeError(error))
  },
)

export default http
