<script setup lang="ts" generic="T">
/**
 * 资源列表视图（公共组件，配置驱动）。
 *
 * 后台大量「列表页」结构相同：页头 + 筛选区 + 工具栏 + 分页表格 + 空状态。
 * 本组件把这套骨架抽象为一个配置驱动组件，页面只需提供：
 *  - title / description / breadcrumb
 *  - filter fields（筛选 schema）
 *  - columns（表格列，统一用 utils/columns 工厂生成的 TableColumnList<T>）
 *  - fetcher（数据获取）
 *  - toolbar actions（带权限的操作按钮）
 *
 * 这样各业务列表页不再复制粘贴筛选/表格/分页代码，只维护自己的差异配置。
 * 复杂详情 / 表单通过默认插槽或页面自身的 FormDrawer 承接。
 *
 * 复用列表状态：useDataTable（分页/筛选/loading/空状态）。
 * 通过 defineExpose 暴露 reload，页面用 ResourceListInstance 类型引用。
 */
import PageHeader from './PageHeader.vue'
import FilterBar from './FilterBar.vue'
import DataTable from './DataTable.vue'
import PermissionButton from './PermissionButton.vue'
import type { FilterField, TableColumnList, ToolbarAction } from './ui-types'
import type { DataTableRowKey, DataTableSortState } from 'naive-ui'
import { useDataTable } from '@/composables/useDataTable'
import type { ListQuery, ListResult } from '@/api/types'

const props = defineProps<{
  title: string
  description?: string
  breadcrumb?: string[]
  fields?: FilterField[]
  columns: TableColumnList<T>
  fetcher: (query: ListQuery) => Promise<ListResult<T>>
  toolbarActions?: ToolbarAction[]
  initialFilters?: Record<string, unknown>
  rowKey?: (row: T) => string
  checkedRowKeys?: DataTableRowKey[]
  emptyText?: string
}>()

const emit = defineEmits<{
  'update:sorter': [sorter: DataTableSortState | DataTableSortState[] | null]
  'update:checkedRowKeys': [keys: DataTableRowKey[]]
}>()

const {
  rows,
  loading,
  filters,
  pagination,
  applyFilters,
  resetFilters,
  handlePageChange,
  handlePageSizeChange,
  reload: reloadData,
} = useDataTable<T>({
  fetcher: props.fetcher,
  initialFilters: props.initialFilters,
})

// FilterBar 返回的是新对象，就地合并回 reactive filters，保持与 useDataTable 内部引用一致
function onFiltersUpdate(next: Record<string, unknown>): void {
  const current = filters as Record<string, unknown>
  for (const key of Object.keys(current)) {
    if (!(key in next)) delete current[key]
  }
  Object.assign(current, next)
}

function clearSelection(): void {
  if (props.checkedRowKeys?.length) emit('update:checkedRowKeys', [])
}

function applyCurrentFilters(): void {
  clearSelection()
  applyFilters()
}

function resetCurrentFilters(): void {
  clearSelection()
  resetFilters()
}

function changePage(page: number): void {
  clearSelection()
  handlePageChange(page)
}

function changePageSize(size: number): void {
  clearSelection()
  handlePageSizeChange(size)
}

function changeSorter(sorter: DataTableSortState | DataTableSortState[] | null): void {
  clearSelection()
  emit('update:sorter', sorter)
}

async function reload(): Promise<void> {
  clearSelection()
  await reloadData()
}

// 供父组件在增删改后手动刷新
defineExpose({ reload })
</script>

<template>
  <section class="resource-list">
    <PageHeader
      :title="title"
      :description="description"
      :breadcrumb="breadcrumb"
    >
      <template #actions>
        <PermissionButton
          v-for="action in toolbarActions ?? []"
          :key="action.key"
          :permission="action.permission"
          :type="action.type ?? 'default'"
          :disabled="action.disabled"
          @click="action.onClick"
        >
          {{ action.label }}
        </PermissionButton>
      </template>
    </PageHeader>

    <FilterBar
      v-if="fields && fields.length"
      :model-value="filters"
      :fields="fields"
      @update:model-value="onFiltersUpdate"
      @search="applyCurrentFilters"
      @reset="resetCurrentFilters"
    />

    <DataTable
      :columns="columns"
      :data="rows"
      :loading="loading"
      :page="pagination.page"
      :page-size="pagination.pageSize"
      :item-count="pagination.itemCount"
      :row-key="rowKey"
      :checked-row-keys="checkedRowKeys"
      :empty-text="emptyText"
      @update:page="changePage"
      @update:page-size="changePageSize"
      @update:sorter="changeSorter"
      @update:checked-row-keys="(keys) => emit('update:checkedRowKeys', keys)"
    />

    <slot :reload="reload" />
  </section>
</template>

<style scoped>
.resource-list {
  max-width: 1400px;
}
</style>
