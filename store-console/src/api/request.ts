/**
 * 类型化请求助手：统一解包 `{ data, meta }` 包体，并提供列表分页归一化。
 * 服务层只调用这里的 get/post/patch/del/getPaged，不直接操作 axios。
 */

import http, { type StoreRequestConfig } from './http'
import { getAuthBridge } from './authBridge'
import { ApiError } from './error'
import type { ApiEnvelope, PagedResult, PageQuery } from '@/types/api'

/**
 * 跨店数据兜底：若返回行数据显式带有 storeId，且与当前门店不一致，
 * 视为服务端越权返回，阻断展示并抛错（防御性，最终范围由服务端保证）。
 */
function assertSameStore<T>(rows: T[]): void {
  const storeId = getAuthBridge()?.getStoreId()
  if (storeId == null) return
  for (const row of rows) {
    const rowStoreId = (row as Record<string, unknown>)?.storeId
    if (rowStoreId != null && String(rowStoreId) !== String(storeId)) {
      throw new ApiError({
        status: 0,
        code: 'CROSS_STORE_DATA',
        message: '检测到非本门店数据，已阻断展示',
      })
    }
  }
}

export async function get<T>(url: string, config?: StoreRequestConfig): Promise<T> {
  const res = await http.get<ApiEnvelope<T>>(url, config)
  return res.data.data
}

export async function post<T>(
  url: string,
  body?: unknown,
  config?: StoreRequestConfig,
): Promise<T> {
  const res = await http.post<ApiEnvelope<T>>(url, body, config)
  return res.data.data
}

export async function patch<T>(
  url: string,
  body?: unknown,
  config?: StoreRequestConfig,
): Promise<T> {
  const res = await http.patch<ApiEnvelope<T>>(url, body, config)
  return res.data.data
}

export async function put<T>(url: string, body?: unknown, config?: StoreRequestConfig): Promise<T> {
  const res = await http.put<ApiEnvelope<T>>(url, body, config)
  return res.data.data
}

export async function del<T>(url: string, config?: StoreRequestConfig): Promise<T> {
  const res = await http.delete<ApiEnvelope<T>>(url, config)
  return res.data.data
}

/** 分页列表请求：解包 data + meta，归一化为 PagedResult。 */
export async function getPaged<T>(
  url: string,
  params?: PageQuery,
  config?: StoreRequestConfig,
): Promise<PagedResult<T>> {
  const res = await http.get<ApiEnvelope<T[]>>(url, { ...config, params })
  const rows = res.data.data ?? []
  assertSameStore(rows)
  const meta = res.data.meta ?? {}
  return {
    rows,
    page: meta.page ?? params?.page ?? 1,
    pageSize: meta.pageSize ?? params?.pageSize ?? rows.length,
    total: meta.total ?? rows.length,
  }
}
