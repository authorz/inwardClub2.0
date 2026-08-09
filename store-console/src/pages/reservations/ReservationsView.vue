<script setup lang="ts">
import { computed, h } from 'vue'
import { NButton, NInput, NSpace, type DataTableColumns } from 'naive-ui'
import { reservationService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PERM } from '@/constants/permissions'
import { RESERVATION_STATUS } from '@/constants/enums'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, PermissionButton } from '@/components/common'
import type { Reservation } from '@/types/models'

const reservations = useAsyncList<Reservation>((params) => reservationService.list(params), {
  initialFilters: { tableNo: '', seatNo: '', memberNickname: '', memberPhone: '' },
})
const action = useAsyncAction()

function applyFilters(): void {
  reservations.applyFilters({})
}

function resetFilters(): void {
  reservations.reset()
}

function cancelReservation(row: Reservation): void {
  const seat = `${row.tableNo || '未分配桌子'} · ${row.seatNo || '未分配座位'}`
  void action.run(() => reservationService.cancel(row.id), {
    confirm: {
      content: `确认取消 ${row.memberNickname || '该用户'} 的预约？取消后 ${seat} 将立即释放。`,
      danger: true,
    },
    successMessage: '预约已取消，座位已释放',
    onSuccess: () => reservations.refresh(),
  })
}

const reservationColumns = computed<DataTableColumns<Reservation>>(() => [
  textColumn<Reservation>('ID', (row) => row.id, { width: 70 }),
  {
    title: '用户信息',
    key: 'member',
    width: 240,
    render: (row) => h('div', { class: 'member-cell' }, [
      h('div', { class: 'member-avatar' }, [
        h('span', (row.memberNickname || '会').slice(0, 1)),
        row.memberAvatarUrl
          ? h('img', {
            src: row.memberAvatarUrl,
            alt: `${row.memberNickname || '会员'}头像`,
            referrerpolicy: 'no-referrer',
            onError: (event: Event) => (event.currentTarget as HTMLImageElement).remove(),
          })
          : null,
      ]),
      h('div', { class: 'member-cell__text' }, [
        h('span', row.memberNickname || '未设置昵称'),
        h('small', row.memberPhone || '暂无手机号'),
      ]),
    ]),
  },
  textColumn<Reservation>('桌子号', (row) => row.tableNo || row.tableId || '-', { width: 130 }),
  textColumn<Reservation>('座位号', (row) => row.seatNo || row.seatId || '-', { width: 120 }),
  dateColumn<Reservation>('预约创建时间', (row) => row.createdAt, { width: 180 }),
  statusColumn<Reservation>('到店状态', RESERVATION_STATUS, (row) => row.status, { width: 110 }),
  {
    title: '操作', key: 'actions', width: 100, fixed: 'right',
    render: (row) => row.status === 'booked'
      ? h(NSpace, { size: 4 }, {
        default: () => [
          h(PermissionButton, {
            permissions: [PERM.reservationWrite],
            text: true,
            type: 'error',
            onClick: () => cancelReservation(row),
          }, { default: () => '取消预约' }),
        ],
      })
      : h('span', { class: 'reservation-action-empty' }, '—'),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="预约记录"
      description="已预定会员完成低消后自动变为已到店；两种状态均保持占座，取消预定后立即释放"
    />

    <section class="reservation-filters ic-band">
      <div class="filter-grid">
        <label class="filter-field"><span>桌号</span><NInput
          :value="reservations.filters.tableNo as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="reservations.filters.tableNo = $event"
          @keyup.enter="applyFilters"
        /></label>
        <label class="filter-field"><span>座位号</span><NInput
          :value="reservations.filters.seatNo as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="reservations.filters.seatNo = $event"
          @keyup.enter="applyFilters"
        /></label>
        <label class="filter-field"><span>会员昵称</span><NInput
          :value="reservations.filters.memberNickname as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="reservations.filters.memberNickname = $event"
          @keyup.enter="applyFilters"
        /></label>
        <label class="filter-field"><span>会员手机号</span><NInput
          :value="reservations.filters.memberPhone as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="reservations.filters.memberPhone = $event"
          @keyup.enter="applyFilters"
        /></label>
      </div>
      <NSpace class="filter-actions">
        <NButton
          type="primary"
          size="small"
          :loading="reservations.loading.value"
          @click="applyFilters"
        >
          查询
        </NButton>
        <NButton
          size="small"
          @click="resetFilters"
        >
          重置
        </NButton>
      </NSpace>
    </section>

    <section class="reservation-table ic-band">
      <DataTable
        :columns="reservationColumns"
        :data="reservations.rows.value"
        :loading="reservations.loading.value"
        :page="reservations.page.value"
        :page-size="reservations.pageSize.value"
        :total="reservations.total.value"
        :scroll-x="950"
        empty-text="暂无预约"
        @update:page="reservations.setPage"
        @update:page-size="reservations.setPageSize"
      />
    </section>
  </div>
</template>

<style scoped>
:deep(.member-cell) {
  display: flex;
  align-items: center;
  gap: var(--ic-space-3);
  min-width: 0;
}

:deep(.member-cell__text) {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.45;
}

:deep(.member-cell__text span),
:deep(.member-cell__text small) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.member-cell__text span) { font-weight: 600; line-height: 20px; }

:deep(.member-cell__text small) {
  color: var(--ic-color-text-secondary);
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}

:deep(.member-avatar) {
  position: relative;
  display: grid;
  width: 36px;
  height: 36px;
  flex: none;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  color: var(--ic-color-text-secondary);
  background: var(--ic-color-surface-muted);
  font-size: var(--ic-font-sm);
}

:deep(.member-avatar img) {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

:deep(.reservation-action-empty) { color: var(--ic-color-text-tertiary); }

.reservation-filters {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--ic-space-4);
  margin-bottom: var(--ic-space-5);
  padding: var(--ic-space-4);
}

.filter-grid { display: flex; flex-wrap: wrap; gap: var(--ic-space-4); }
.filter-field { display: flex; width: 180px; min-width: 0; flex-direction: column; gap: var(--ic-space-1); }
.filter-field > span { color: var(--ic-color-text-secondary); font-size: var(--ic-font-xs); }
.filter-actions { flex-shrink: 0; }

.reservation-table { padding: var(--ic-space-3); overflow: hidden; }

@media (max-width: 720px) {
  .filter-grid, .filter-field { width: 100%; }
}
</style>
