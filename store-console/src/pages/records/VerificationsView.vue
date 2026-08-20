<script setup lang="ts">
/**
 * 活动核销记录：仅展示本店活动票核销历史。
 */
import { computed } from 'vue'
import { type DataTableColumns } from 'naive-ui'
import { verificationService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { dateColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, StatusFilterBar } from '@/components/common'
import type { VerificationRecord } from '@/types/models'

const list = useAsyncList<VerificationRecord>((params) => verificationService.records(params), {
  initialFilters: { kind: 'ticket', keyword: '' },
})

const columns = computed<DataTableColumns<VerificationRecord>>(() => [
  textColumn<VerificationRecord>('核销码', (r) => r.code),
  textColumn<VerificationRecord>('活动', (r) => r.activityTitle),
  textColumn<VerificationRecord>('会员', (r) => r.memberName),
  textColumn<VerificationRecord>('操作员', (r) => r.verifiedBy, { width: 100 }),
  dateColumn<VerificationRecord>('核销时间', (r) => r.verifiedAt ?? r.createdAt, { width: 150 }),
])
</script>

<template>
  <div>
    <PageHeader
      title="核销记录"
      description="本店活动票核销历史"
    />

    <StatusFilterBar
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索核销码 / 活动 / 会员"
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
      empty-text="暂无核销记录"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
  </div>
</template>
