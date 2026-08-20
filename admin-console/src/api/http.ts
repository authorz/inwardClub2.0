/**
 * 总后台统一 API client（唯一 axios 实例）。
 *
 * 强制业务边界与工程化规则：
 *  - 所有请求必须经过此 client；页面/服务不得散落 axios/fetch。
 *  - 只允许调用 /admin/* 与公共资产接口；命中 /store/*、/mini/* 直接抛错（越界防护）。
 *  - 统一注入 Authorization、X-Request-ID；高风险写操作注入 Idempotency-Key。
 *  - 统一处理 401 refresh、403 权限不足、409 状态冲突、422 表单错误、网络错误。
 *  - 统一解包 { data, meta } 响应信封。
 *
 * 说明：本 client 不直接 import auth store（避免循环依赖）。
 * 通过 attachAuthProvider() 注入 token 读取 / refresh / 登出回调，由 auth store 在初始化时装配。
 */
import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios'
import { ALLOWED_PATH_PREFIXES, FORBIDDEN_PATH_PREFIXES } from '@/constants/api-paths'
import { newIdempotencyKey, newRequestId } from '@/utils/id'
import type { ApiError, ApiSuccess, ListResult, NormalizedError, PageMeta } from './types'

/** auth 侧注入的能力，避免 http <-> store 循环依赖 */
export interface AuthProvider {
  getAccessToken: () => string | null
  refresh: () => Promise<boolean>
  onUnauthorized: () => void
}

let authProvider: AuthProvider | null = null
export function attachAuthProvider(provider: AuthProvider): void {
  authProvider = provider
}

/** 请求级扩展配置 */
export interface RequestOptions extends AxiosRequestConfig {
  /** 是否为高风险写操作，需注入 Idempotency-Key（退款/调账/发券/发布/上下架/库存/支付配置） */
  idempotent?: boolean
  /** 关闭默认错误提示，由调用方自行处理 */
  silent?: boolean
}

function assertPathAllowed(url: string): void {
  // 去掉可能的绝对地址，仅看 path 部分
  const path = url.startsWith('http') ? new URL(url).pathname.replace(/^\/api\/v2/, '') : url
  if (FORBIDDEN_PATH_PREFIXES.some((p) => path.startsWith(p))) {
    throw new Error(`[admin-console] 越界调用被拒绝：总后台禁止访问 ${path}`)
  }
  const allowed = ALLOWED_PATH_PREFIXES.some((p) => path.startsWith(p))
  if (!allowed) {
    throw new Error(`[admin-console] 非白名单路径被拒绝：${path}`)
  }
}

const instance: AxiosInstance = axios.create({
  // 生产默认走同源 /api/v2，由 Nginx 反向代理到实际 API 服务。
  baseURL: import.meta.env.VITE_API_BASE_URL?.trim() || '/api/v2',
  timeout: 20_000,
  headers: {
    'Content-Type': 'application/json',
    'X-Admin-App': import.meta.env.VITE_APP_ID,
  },
})

instance.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  assertPathAllowed(config.url ?? '')

  config.headers.set('X-Request-ID', newRequestId())

  const token = authProvider?.getAccessToken()
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`)
  }

  if ((config as RequestOptions).idempotent) {
    config.headers.set('Idempotency-Key', newIdempotencyKey())
  }
  return config
})

/** 归一化 axios 错误为业务错误结构 */
function normalizeError(error: AxiosError<ApiError>): NormalizedError {
  const status = error.response?.status ?? 0
  const payload = error.response?.data
  const apiErr = payload?.error
  let fieldErrors: Record<string, string[]> | undefined
  if (status === 422 && apiErr?.details && typeof apiErr.details === 'object') {
    fieldErrors = apiErr.details as Record<string, string[]>
  }
  return {
    status,
    code: apiErr?.code ?? (status === 0 ? 'NETWORK_ERROR' : `HTTP_${status}`),
    message: apiErr?.message ?? defaultMessageForStatus(status),
    fieldErrors,
    details: apiErr?.details,
  }
}

function defaultMessageForStatus(status: number): string {
  switch (status) {
    case 0:
      return '网络异常，请稍后重试'
    case 401:
      return '登录已过期，请重新登录'
    case 403:
      return '权限不足，无法执行该操作'
    case 409:
      return '资源状态已变化，请刷新后重试'
    case 422:
      return '表单校验未通过'
    case 500:
      return '服务端异常，请稍后重试'
    default:
      return '请求失败'
  }
}

// 401 刷新：并发请求只触发一次 refresh，其余排队等待。
let refreshing: Promise<boolean> | null = null

instance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    const original = error.config as (InternalAxiosRequestConfig & { _retried?: boolean }) | undefined
    const status = error.response?.status

    if (status === 401 && original && !original._retried && authProvider) {
      original._retried = true
      if (!refreshing) {
        refreshing = authProvider.refresh().finally(() => {
          refreshing = null
        })
      }
      const ok = await refreshing
      if (ok) {
        const token = authProvider.getAccessToken()
        if (token) original.headers.set('Authorization', `Bearer ${token}`)
        return instance(original)
      }
      authProvider.onUnauthorized()
    }

    return Promise.reject(normalizeError(error))
  },
)

/** 单体资源请求，返回解包后的 data */
async function requestData<T>(config: RequestOptions): Promise<T> {
  const res = await instance.request<ApiSuccess<T>>(config)
  return res.data.data
}

/** 列表请求，返回 items + meta */
async function requestList<T>(config: RequestOptions): Promise<ListResult<T>> {
  const res = await instance.request<ApiSuccess<T[]>>(config)
  const meta: PageMeta = res.data.meta ?? {
    page: 1,
    pageSize: res.data.data?.length ?? 0,
    total: res.data.data?.length ?? 0,
  }
  return { items: res.data.data ?? [], meta }
}

export const http = {
  raw: instance,
  requestData,
  requestList,
  get: <T>(url: string, params?: Record<string, unknown>, opts?: RequestOptions) =>
    requestData<T>({ ...opts, method: 'GET', url, params }),
  getList: <T>(url: string, params?: Record<string, unknown>, opts?: RequestOptions) =>
    requestList<T>({ ...opts, method: 'GET', url, params }),
  post: <T>(url: string, data?: unknown, opts?: RequestOptions) =>
    requestData<T>({ ...opts, method: 'POST', url, data }),
  patch: <T>(url: string, data?: unknown, opts?: RequestOptions) =>
    requestData<T>({ ...opts, method: 'PATCH', url, data }),
  put: <T>(url: string, data?: unknown, opts?: RequestOptions) =>
    requestData<T>({ ...opts, method: 'PUT', url, data }),
  delete: <T>(url: string, opts?: RequestOptions) =>
    requestData<T>({ ...opts, method: 'DELETE', url }),
}
