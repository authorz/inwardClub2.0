/**
 * 统一的 API 响应契约类型。
 *
 * 服务端固定返回 `{ data, meta }` 或 `{ error: { code, message, details } }`。
 * 所有服务/组件只依赖这些类型，不各自定义响应结构。
 */

export interface ApiMeta {
  page?: number
  pageSize?: number
  total?: number
  [key: string]: unknown
}

/** 成功响应包体。 */
export interface ApiEnvelope<T> {
  data: T
  meta?: ApiMeta
}

/** 错误响应包体。 */
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}

/** 列表分页请求参数（门店后台永远不含 storeId，范围来自 token scope）。 */
export interface PageQuery {
  page?: number
  pageSize?: number
  [key: string]: unknown
}

/** 列表返回结果：行数据 + 分页元信息。 */
export interface PagedResult<T> {
  rows: T[]
  page: number
  pageSize: number
  total: number
}

/** 表单字段级错误（422 映射）。 */
export type FieldErrors = Record<string, string>
