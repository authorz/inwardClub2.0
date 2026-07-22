<script setup lang="ts">
/**
 * 核销记录：票 + 券统一核销记录读模型，仅本店范围。
 */
import { computed } from 'vue'
import { type DataTableColumns } from 'naive-ui'
import { verificationService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { VERIFICATION_KIND, VERIFICATION_STATUS, toOptions } from '@/constants/enums'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, StatusFilterBar } from '@/components/common'
import type { VerificationRecord } from '@/types/models'

const list = useAsyncList<VerificationRecord>((params) => verificationService.records(params), {
  initialFilters: { status: '', kind: '', keyword: '' },
})

const columns = computed<DataTableColumns<VerificationRecord>>(() => [
  statusColumn<VerificationRecord>('类型', VERIFICATION_KIND, (r) => r.kind, { width: 90 }),
  textColumn<VerificationRecord>('核销码', (r) => r.code),
  textColumn<VerificationRecord>('活动', (r) => r.activityTitle),
  textColumn<VerificationRecord>('会员', (r) => r.memberName),
  statusColumn<VerificationRecord>('状态', VERIFICATION_STATUS, (r) => r.status, { width: 100 }),
  textColumn<VerificationRecord>('操作员', (r) => r.verifiedBy, { width: 100 }),
  dateColumn<VerificationRecord>('核销时间', (r) => r.verifiedAt ?? r.createdAt, { width: 150 }),
])
</script>

<template>
  <div>
    <PageHeader
      title="核销记录"
      description="本店活动票与券核销历史"
    />

    <StatusFilterBar
      :status-options="toOptions(VERIFICATION_STATUS)"
      :status="(list.filters.status as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索核销码 / 会员"
      @update:status="list.filters.status = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    >
      <template #filters>
        <span
          v-for="opt in toOptions(VERIFICATION_KIND)"
          :key="opt.value"
          class="kind-chip"
          :class="{ 'kind-chip--active': list.filters.kind === opt.value }"
          @click="list.applyFilters({ kind: list.filters.kind === opt.value ? '' : opt.value })"
        >
          {{ opt.label }}
        </span>
      </template>
    </StatusFilterBar>

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

<style scoped>
.kind-chip {
  padding: 3px 10px;
  border: var(--ic-divider);
  border-radius: 999px;
  font-size: var(--ic-font-xs);
  cursor: pointer;
  color: var(--ic-color-text-secondary);
}
.kind-chip--active {
  background: var(--ic-color-primary);
  color: #fff;
  border-color: var(--ic-color-primary);
}
</style>
