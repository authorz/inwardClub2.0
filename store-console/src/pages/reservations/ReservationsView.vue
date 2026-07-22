<script setup lang="ts">
/**
 * 预约 / 桌台视图：本店预约列表（含到店确认）与本店桌台一览。
 * 到店确认为状态流转写操作，服务端带 Idempotency-Key。
 */
import { computed, h } from 'vue'
import { NTabPane, NTabs, type DataTableColumns } from 'naive-ui'
import { reservationService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { RESERVATION_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, phoneColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, EmptyState, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { Reservation } from '@/types/models'

const reservations = useAsyncList<Reservation>((params) => reservationService.list(params), {
  initialFilters: { status: '', keyword: '' },
})
const action = useAsyncAction()

function onArrive(row: Reservation) {
  void action.run(() => reservationService.arrive(row.id), {
    confirm: { content: `确认「${row.memberNickname ?? '会员'}」已到店？` },
    successMessage: '已标记到店',
    onSuccess: () => reservations.refresh(),
  })
}

const reservationColumns = computed<DataTableColumns<Reservation>>(() => [
  textColumn<Reservation>('会员', (r) => r.memberNickname),
  phoneColumn<Reservation>('手机号', (r) => r.memberPhoneMasked),
  textColumn<Reservation>('桌台', (r) => r.tableName, { width: 100 }),
  textColumn<Reservation>('人数', (r) => r.partySize, { width: 70, align: 'right' }),
  dateColumn<Reservation>('预约时间', (r) => r.reservedAt, { width: 150 }),
  statusColumn<Reservation>('状态', RESERVATION_STATUS, (r) => r.status, { width: 96 }),
  {
    title: '操作',
    key: 'actions',
    width: 100,
    fixed: 'right',
    render: (row: Reservation) =>
      h(
        PermissionButton,
        {
          permissions: [PERM.reservationWrite],
          type: 'primary',
          text: true,
          disabled: row.status !== 'booked',
          onClick: () => onArrive(row),
        },
        { default: () => '到店' },
      ),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="预约 / 桌台"
      description="本店预约与桌台一览，不跨店"
    />

    <NTabs
      type="line"
      default-value="reservations"
    >
      <NTabPane
        name="reservations"
        tab="预约"
      >
        <StatusFilterBar
          :status-options="toOptions(RESERVATION_STATUS)"
          :status="(reservations.filters.status as string) ?? null"
          :keyword="(reservations.filters.keyword as string) ?? ''"
          :loading="reservations.loading.value"
          search-placeholder="搜索会员 / 桌台"
          @update:status="reservations.filters.status = $event ?? ''"
          @update:keyword="reservations.filters.keyword = $event"
          @apply="reservations.applyFilters({})"
          @reset="reservations.reset()"
        />
        <DataTable
          :columns="reservationColumns"
          :data="reservations.rows.value"
          :loading="reservations.loading.value"
          :page="reservations.page.value"
          :page-size="reservations.pageSize.value"
          :total="reservations.total.value"
          empty-text="暂无预约"
          @update:page="reservations.setPage"
          @update:page-size="reservations.setPageSize"
        />
      </NTabPane>

      <NTabPane
        name="tables"
        tab="桌台"
      >
        <EmptyState description="暂无桌台（桌台管理功能待后端支持）" />
      </NTabPane>
    </NTabs>
  </div>
</template>
