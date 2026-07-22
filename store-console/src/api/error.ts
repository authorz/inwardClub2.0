/**
 * 统一 API 错误模型与归一化。
 * 所有服务/页面捕获到的错误都应是 ApiError，字段稳定，便于统一提示与表单映射。
 */

import type { AxiosError } from 'axios'
import type { ApiErrorBody, FieldErrors } from '@/types/api'

export class ApiError extends Error {
  /** HTTP 状态码，0 表示网络/未知错误。 */
  status: number
  /** 业务错误码（服务端 error.code）。 */
  code: string
  /** 422 字段级错误映射。 */
  fieldErrors?: FieldErrors
  /** 原始 details。 */
  details?: Record<string, unknown>

  constructor(params: {
    status: number
    code: string
    message: string
    fieldErrors?: FieldErrors
    details?: Record<string, unknown>
  }) {
    super(params.message)
    this.name = 'ApiError'
    this.status = params.status
    this.code = params.code
    this.fieldErrors = params.fieldErrors
    this.details = params.details
  }
}

/** 把 details 中的字段错误尽量抽成 { field: message } 结构。 */
function extractFieldErrors(details?: Record<string, unknown>): FieldErrors | undefined {
  if (!details) return undefined
  const source = (details.fields ?? details.errors ?? details) as Record<string, unknown>
  const result: FieldErrors = {}
  for (const [key, val] of Object.entries(source)) {
    if (typeof val === 'string') result[key] = val
    else if (Array.isArray(val) && typeof val[0] === 'string') result[key] = val[0]
  }
  return Object.keys(result).length ? result : undefined
}

/** 将 axios 错误归一化为 ApiError。 */
export function normalizeError(err: AxiosError<ApiErrorBody>): ApiError {
  if (!err.response) {
    return new ApiError({
      status: 0,
      code: 'NETWORK_ERROR',
      message: err.code === 'ECONNABORTED' ? '请求超时，请重试' : '网络异常，请检查连接',
    })
  }
  const { status, data } = err.response
  const body = data?.error
  const details = body?.details
  return new ApiError({
    status,
    code: body?.code ?? `HTTP_${status}`,
    message: body?.message ?? defaultMessageForStatus(status),
    fieldErrors: status === 422 ? extractFieldErrors(details) : undefined,
    details,
  })
}

function defaultMessageForStatus(status: number): string {
  switch (status) {
    case 400:
      return '请求参数有误'
    case 401:
      return '登录已失效，请重新登录'
    case 403:
      return '无操作权限'
    case 404:
      return '资源不存在'
    case 409:
      return '操作冲突，请刷新后重试'
    case 422:
      return '表单校验未通过'
    case 429:
      return '操作过于频繁，请稍后再试'
    default:
      return status >= 500 ? '服务异常，请稍后重试' : '请求失败'
  }
}
