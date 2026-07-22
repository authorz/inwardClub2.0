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
  emptyText?: string
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
  reload,
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
      @search="applyFilters()"
      @reset="resetFilters()"
    />

    <DataTable
      :columns="columns"
      :data="rows"
      :loading="loading"
      :page="pagination.page"
      :page-size="pagination.pageSize"
      :item-count="pagination.itemCount"
      :row-key="rowKey"
      :empty-text="emptyText"
      @update:page="handlePageChange"
      @update:page-size="handlePageSizeChange"
    />

    <slot :reload="reload" />
  </section>
</template>

<style scoped>
.resource-list {
  max-width: 1400px;
}
</style>
