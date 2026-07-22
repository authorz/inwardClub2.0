<script setup lang="ts">
/**
 * 本店活动：本店自建活动 + 采用全局活动的本店运营（上下架等）。
 * 门店不可修改全局活动模板。
 */
import { computed, h } from 'vue'
import { NButton, NSpace, type DataTableColumns } from 'naive-ui'
import { activityService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PUBLISH_STATUS, SCOPE_TYPE, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { StoreActivity } from '@/types/models'

const list = useAsyncList<StoreActivity>((params) => activityService.list(params), {
  initialFilters: { status: '', keyword: '' },
})
const action = useAsyncAction()

function togglePublish(row: StoreActivity) {
  const publishing = row.status !== 'published'
  void action.run(
    () => (publishing ? activityService.publish(row.id) : activityService.unpublish(row.id)),
    {
      confirm: { content: `确认${publishing ? '上架' : '下架'}活动「${row.title}」？` },
      successMessage: publishing ? '已上架' : '已下架',
      onSuccess: () => list.refresh(),
    },
  )
}

const columns = computed<DataTableColumns<StoreActivity>>(() => [
  textColumn<StoreActivity>('活动', (r) => r.title),
  statusColumn<StoreActivity>('来源', SCOPE_TYPE, (r) => r.scopeType, { width: 96 }),
  dateColumn<StoreActivity>('开始', (r) => r.startAt, { width: 150 }),
  dateColumn<StoreActivity>('结束', (r) => r.endAt, { width: 150 }),
  textColumn<StoreActivity>('已售', (r) => r.soldCount, { width: 80, align: 'right' }),
  textColumn<StoreActivity>('已核销', (r) => r.verifiedCount, { width: 80, align: 'right' }),
  statusColumn<StoreActivity>('状态', PUBLISH_STATUS, (r) => r.status, { width: 96 }),
  {
    title: '操作',
    key: 'actions',
    width: 120,
    fixed: 'right',
    render: (row: StoreActivity) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(
            PermissionButton,
            { permissions: [PERM.activityWrite], text: true, onClick: () => togglePublish(row) },
            { default: () => (row.status === 'published' ? '下架' : '上架') },
          ),
        ],
      }),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="本店活动"
      description="维护本店活动上下架与本店场次/票档"
    >
      <template #actions>
        <NButton
          :loading="list.loading.value"
          @click="list.refresh()"
        >
          刷新
        </NButton>
      </template>
    </PageHeader>

    <StatusFilterBar
      :status-options="toOptions(PUBLISH_STATUS)"
      :status="(list.filters.status as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索活动名"
      @update:status="list.filters.status = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    />

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无活动"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
  </div>
</template>
