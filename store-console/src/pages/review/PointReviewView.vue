<script setup lang="ts">
/**
 * 积分审核：审核会员存积分申请（通过/驳回）。
 * 审核为高风险写操作，服务端带 Idempotency-Key 与审计。
 */
import { computed, h, ref } from 'vue'
import { type DataTableColumns } from 'naive-ui'
import { pointSavingService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { REVIEW_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, phoneColumn, statusColumn, textColumn } from '@/utils/columns'
import {
  DataTable,
  PageHeader,
  PermissionButton,
  ReviewDialog,
  StatusFilterBar,
} from '@/components/common'
import type { PointSavingRequest } from '@/types/models'

const list = useAsyncList<PointSavingRequest>((params) => pointSavingService.list(params), {
  initialFilters: { status: 'pending', keyword: '' },
})
const action = useAsyncAction()

const dialogShow = ref(false)
const activeRow = ref<PointSavingRequest | null>(null)

function openReview(row: PointSavingRequest) {
  activeRow.value = row
  dialogShow.value = true
}

function submit(decision: 'approved' | 'rejected', reason: string) {
  const row = activeRow.value
  if (!row) return
  void action.run(() => pointSavingService.review(row.id, decision, reason), {
    successMessage: decision === 'approved' ? '已通过' : '已驳回',
    onSuccess: () => {
      dialogShow.value = false
      list.refresh()
    },
  })
}

const columns = computed<DataTableColumns<PointSavingRequest>>(() => [
  textColumn<PointSavingRequest>('会员', (r) => r.memberName),
  phoneColumn<PointSavingRequest>('手机号', (r) => r.phone),
  textColumn<PointSavingRequest>('申请积分', (r) => r.points, { align: 'right' }),
  statusColumn<PointSavingRequest>('状态', REVIEW_STATUS, (r) => r.status, { width: 100 }),
  textColumn<PointSavingRequest>('审核人', (r) => r.reviewedBy),
  dateColumn<PointSavingRequest>('提交时间', (r) => r.submittedAt ?? r.createdAt, { width: 150 }),
  {
    title: '操作',
    key: 'actions',
    width: 100,
    fixed: 'right',
    render: (row: PointSavingRequest) =>
      h(
        PermissionButton,
        {
          permissions: [PERM.pointReview],
          type: 'primary',
          text: true,
          disabled: row.status !== 'pending',
          onClick: () => openReview(row),
        },
        { default: () => '审核' },
      ),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="积分审核"
      description="审核会员存积分申请"
    />

    <StatusFilterBar
      :status-options="toOptions(REVIEW_STATUS)"
      :status="(list.filters.status as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索会员"
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
      empty-text="暂无待审核申请"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <ReviewDialog
      v-model:show="dialogShow"
      title="积分存入审核"
      :loading="action.running.value"
      @approve="submit('approved', $event)"
      @reject="submit('rejected', $event)"
    >
      <div
        v-if="activeRow"
        class="review-info ic-band"
      >
        <div><span class="ic-muted">会员：</span>{{ activeRow.memberName ?? '-' }}</div>
        <div><span class="ic-muted">申请积分：</span>{{ activeRow.points }}</div>
      </div>
    </ReviewDialog>
  </div>
</template>

<style scoped>
.review-info {
  padding: var(--ic-space-3) var(--ic-space-4);
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-1);
  font-size: var(--ic-font-sm);
}
</style>
