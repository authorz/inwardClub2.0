<script setup lang="ts" generic="T">
/**
 * 统一数据表格：封装 loading、远程分页、空状态。
 * 全站列表页共用；页面只需提供 columns 与 useAsyncList 的状态。
 */
import { computed } from 'vue'
import { NDataTable, type DataTableColumns, type DataTableRowKey, type PaginationProps } from 'naive-ui'
import type { DataTableSortState } from 'naive-ui'
import EmptyState from './EmptyState.vue'

/** naive-ui 表格内部行类型，用于把泛型列/数据安全桥接到组件。 */
type NaiveRow = Record<string, unknown>

const props = withDefaults(
  defineProps<{
    columns: DataTableColumns<T>
    data: T[]
    loading?: boolean
    page?: number
    pageSize?: number
    total?: number
    rowKey?: (row: T) => string | number
    checkedRowKeys?: DataTableRowKey[]
    emptyText?: string
    scrollX?: number
  }>(),
  { loading: false, page: 1, pageSize: 20, total: 0, emptyText: '暂无数据' },
)

const emit = defineEmits<{
  'update:page': [page: number]
  'update:pageSize': [size: number]
  'update:sorter': [sorter: DataTableSortState | DataTableSortState[] | null]
  'update:checkedRowKeys': [keys: DataTableRowKey[]]
}>()

// 把泛型 T 的列/数据/rowKey 桥接为 naive 期望的内部行类型（值不变，仅类型转换）。
const nColumns = computed(() => props.columns as unknown as DataTableColumns<NaiveRow>)
const nData = computed(() => props.data as unknown as NaiveRow[])
const nRowKey = computed<(row: NaiveRow) => string | number>(() => {
  const fn = props.rowKey ?? defaultRowKey
  return fn as unknown as (row: NaiveRow) => string | number
})

const pagination = computed<PaginationProps>(() => ({
  page: props.page,
  pageSize: props.pageSize,
  itemCount: props.total,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  prefix: (info: { itemCount?: number }) => `共 ${info.itemCount ?? 0} 条`,
}))

const defaultRowKey = (row: T): string | number => {
  const id = (row as unknown as Record<string, unknown>).id
  return (id as string | number) ?? JSON.stringify(row)
}
</script>

<template>
  <NDataTable
    :columns="nColumns"
    :data="nData"
    :loading="loading"
    :row-key="nRowKey"
    :checked-row-keys="checkedRowKeys"
    :pagination="total > 0 ? pagination : false"
    remote
    :bordered="false"
    :single-line="true"
    :scroll-x="scrollX"
    size="small"
    @update:page="emit('update:page', $event)"
    @update:page-size="emit('update:pageSize', $event)"
    @update:sorter="emit('update:sorter', $event)"
    @update:checked-row-keys="emit('update:checkedRowKeys', $event)"
  >
    <template #empty>
      <EmptyState :description="emptyText" />
    </template>
  </NDataTable>
</template>
