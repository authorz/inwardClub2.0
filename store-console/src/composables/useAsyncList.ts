/**
 * 列表数据编排 composable：集中管理 loading、分页、筛选、空状态和错误。
 *
 * 所有列表页统一使用本 composable + DataTable 组件，禁止每页各写一套
 * 加载/分页/筛选逻辑。fetcher 只需返回 PagedResult。
 */

import { reactive, ref, shallowRef } from 'vue'
import type { PagedResult, PageQuery } from '@/types/api'
import { ApiError } from '@/api/error'
import { feedback } from '@/utils/feedback'

export interface UseAsyncListOptions {
  /** 每页数量。 */
  pageSize?: number
  /** 初始筛选参数。 */
  initialFilters?: Record<string, unknown>
  /** 是否在创建时立即加载。 */
  immediate?: boolean
}

export function useAsyncList<T>(
  fetcher: (params: PageQuery) => Promise<PagedResult<T>>,
  options: UseAsyncListOptions = {},
) {
  const rows = shallowRef<T[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const page = ref(1)
  const pageSize = ref(options.pageSize ?? 20)
  const total = ref(0)
  const filters = reactive<Record<string, unknown>>({ ...(options.initialFilters ?? {}) })

  /** 过滤掉空值，避免向服务端发送空字符串筛选项。 */
  function activeFilters(): Record<string, unknown> {
    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(filters)) {
      if (v !== '' && v != null) out[k] = v
    }
    return out
  }

  async function load(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const result = await fetcher({
        page: page.value,
        pageSize: pageSize.value,
        ...activeFilters(),
      })
      rows.value = result.rows
      total.value = result.total
      page.value = result.page
      pageSize.value = result.pageSize
    } catch (err) {
      const message = err instanceof ApiError ? err.message : '加载失败'
      error.value = message
      rows.value = []
      total.value = 0
      feedback.message.error(message)
    } finally {
      loading.value = false
    }
  }

  function setPage(next: number): void {
    page.value = next
    void load()
  }

  function setPageSize(size: number): void {
    pageSize.value = size
    page.value = 1
    void load()
  }

  /** 应用筛选并回到第一页。 */
  function applyFilters(patch: Record<string, unknown>): void {
    Object.assign(filters, patch)
    page.value = 1
    void load()
  }

  function reset(): void {
    for (const key of Object.keys(filters)) delete filters[key]
    Object.assign(filters, options.initialFilters ?? {})
    page.value = 1
    void load()
  }

  function refresh(): void {
    void load()
  }

  if (options.immediate !== false) {
    void load()
  }

  return {
    rows,
    loading,
    error,
    page,
    pageSize,
    total,
    filters,
    load,
    setPage,
    setPageSize,
    applyFilters,
    reset,
    refresh,
  }
}
