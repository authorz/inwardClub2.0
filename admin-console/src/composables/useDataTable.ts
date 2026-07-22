/**
 * 列表页通用组合式函数（公共状态）。
 *
 * 统一封装：分页、筛选、loading、错误提示、空状态判定、刷新。
 * 所有列表页复用此 hook，不再各自维护一套 page/pageSize/loading/filters 逻辑。
 */
import { computed, reactive, ref, type Ref } from 'vue'
import type { ListQuery, ListResult, NormalizedError } from '@/api/types'
import { toastError } from '@/utils/feedback'

export interface UseDataTableOptions<T, Q extends ListQuery> {
  /** 数据获取函数（通常是某个 resource.list） */
  fetcher: (query: Q) => Promise<ListResult<T>>
  /** 初始筛选值 */
  initialFilters?: Partial<Q>
  /** 默认每页条数 */
  defaultPageSize?: number
  /** 是否在创建时立即拉取 */
  immediate?: boolean
}

export function useDataTable<T, Q extends ListQuery = ListQuery>(
  options: UseDataTableOptions<T, Q>,
) {
  const { fetcher, initialFilters = {}, defaultPageSize = 20, immediate = true } = options

  // 显式断言为 Ref<T[]>，避免 UnwrapRef 对泛型 T 的深解包干扰类型
  const rows = ref([]) as Ref<T[]>
  const loading = ref(false)
  const error = ref<NormalizedError | null>(null)

  const filters = reactive({ ...initialFilters }) as Partial<Q>
  const pagination = reactive({
    page: 1,
    pageSize: defaultPageSize,
    itemCount: 0,
    pageCount: 1,
  })

  const isEmpty = computed(() => !loading.value && rows.value.length === 0)

  async function load(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const query = {
        ...filters,
        page: pagination.page,
        pageSize: pagination.pageSize,
      } as Q
      const result = await fetcher(query)
      rows.value = result.items
      pagination.itemCount = result.meta.total
      pagination.pageCount = Math.max(1, Math.ceil(result.meta.total / pagination.pageSize))
      pagination.page = result.meta.page || pagination.page
    } catch (e) {
      const err = e as NormalizedError
      error.value = err
      rows.value = []
      pagination.itemCount = 0
      // 401 已由 http 层处理跳转；其余错误统一提示
      if (err.status !== 401) toastError(err.message || '加载失败')
    } finally {
      loading.value = false
    }
  }

  /** 应用筛选：回到第一页并重新加载 */
  function applyFilters(next?: Partial<Q>): void {
    if (next) Object.assign(filters, next)
    pagination.page = 1
    void load()
  }

  /** 重置筛选到初始值 */
  function resetFilters(): void {
    for (const key of Object.keys(filters)) {
      delete (filters as Record<string, unknown>)[key]
    }
    Object.assign(filters, initialFilters)
    pagination.page = 1
    void load()
  }

  function handlePageChange(page: number): void {
    pagination.page = page
    void load()
  }

  function handlePageSizeChange(size: number): void {
    pagination.pageSize = size
    pagination.page = 1
    void load()
  }

  if (immediate) void load()

  return {
    rows,
    loading,
    error,
    filters,
    pagination,
    isEmpty,
    load,
    reload: load,
    applyFilters,
    resetFilters,
    handlePageChange,
    handlePageSizeChange,
  }
}
