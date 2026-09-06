<script setup lang="ts" generic="T">
/**
 * 数据表格（公共组件）。
 * 统一封装：分页、loading、空状态、行 key。
 *
 * 类型收口点：对外暴露 `T` 泛型（列/数据/行 key 全部按业务实体类型），
 * NDataTable 内部行类型固定为 Record<string, unknown>（不变型），
 * 因此仅在传入 NDataTable 时做一次断言，页面与 ResourceListView 保持完整类型。
 */
import { computed } from 'vue'
import { NDataTable, NEmpty } from 'naive-ui'
import type {
  DataTableColumns,
  DataTableRowKey,
  DataTableSortState,
  PaginationProps,
} from 'naive-ui'
import type { TableColumnList } from './ui-types'

const props = withDefaults(
  defineProps<{
    columns: TableColumnList<T>
    data: T[]
    loading?: boolean
    rowKey?: (row: T) => string
    checkedRowKeys?: DataTableRowKey[]
    page?: number
    pageSize?: number
    itemCount?: number
    emptyText?: string
    scrollX?: number
  }>(),
  {
    loading: false,
    page: 1,
    pageSize: 20,
    itemCount: 0,
    emptyText: '暂无数据',
  },
)

const emit = defineEmits<{
  'update:page': [page: number]
  'update:pageSize': [size: number]
  'update:sorter': [sorter: DataTableSortState | DataTableSortState[] | null]
  'update:checkedRowKeys': [keys: DataTableRowKey[]]
}>()

type AnyRow = Record<string, unknown>

const tableColumns = computed(() => props.columns as unknown as DataTableColumns<AnyRow>)
const tableData = computed(() => props.data as unknown as AnyRow[])
const tableRowKey = computed(() => {
  const fn = props.rowKey
  return (row: AnyRow): string =>
    fn ? fn(row as unknown as T) : String((row as { id?: unknown }).id ?? '')
})

const pagination = computed<PaginationProps>(() => ({
  page: props.page,
  pageSize: props.pageSize,
  itemCount: props.itemCount,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
  prefix: (info: { itemCount?: number }) => `共 ${info.itemCount ?? 0} 条`,
}))
</script>

<template>
  <div class="data-table">
    <NDataTable
      :columns="tableColumns"
      :data="tableData"
      :loading="loading"
      :row-key="tableRowKey"
      :checked-row-keys="checkedRowKeys"
      :pagination="pagination"
      :scroll-x="scrollX"
      remote
      :bordered="false"
      size="small"
      @update:page="(p) => emit('update:page', p)"
      @update:page-size="(s) => emit('update:pageSize', s)"
      @update:sorter="(sorter) => emit('update:sorter', sorter)"
      @update:checked-row-keys="(keys) => emit('update:checkedRowKeys', keys)"
    >
      <template #empty>
        <NEmpty :description="emptyText" />
      </template>
    </NDataTable>
  </div>
</template>

<style scoped>
.data-table {
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
  padding: var(--ic-space-sm);
}
</style>
