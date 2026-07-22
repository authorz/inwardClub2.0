/**
 * 统一响应契约类型（依据实现规格 5.1）。
 * 成功：{ data, meta? }
 * 失败：{ error: { code, message, details? } }
 */

export interface PageMeta {
  page: number
  pageSize: number
  total: number
}

export interface ApiSuccess<T> {
  data: T
  meta?: PageMeta
}

export interface ApiError {
  error: {
    code: string
    message: string
    details?: unknown
  }
}

/** 列表结果：数据 + 分页 meta */
export interface ListResult<T> {
  items: T[]
  meta: PageMeta
}

/** 分页 + 筛选查询参数基类 */
export interface ListQuery {
  page?: number
  pageSize?: number
  keyword?: string
  [key: string]: unknown
}

/** 后台详情通用审计字段（依据规格 8.7.7） */
export interface AuditableEntity {
  id: string
  scopeType?: string
  storeId?: string | null
  status?: string
  createdAt?: string
  updatedAt?: string
  createdBy?: string
  updatedBy?: string
  /** 可被门店覆盖的资源附加字段 */
  sourceId?: string
  isInherited?: boolean
  overrideFields?: string[]
}

/** 归一化后的业务错误，供 UI 分类处理 */
export interface NormalizedError {
  status: number
  code: string
  message: string
  /** 422 表单字段错误映射：field -> messages */
  fieldErrors?: Record<string, string[]>
  details?: unknown
}
