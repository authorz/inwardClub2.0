<script setup lang="ts">
/**
 * 本店订单：统一订单读模型，支持类型/支付状态/关键字筛选。
 * 门店范围来自 token scope，不传 storeId。
 */
import { computed, h } from 'vue'
import { NAvatar, NButton, type DataTableColumns } from 'naive-ui'
import { orderService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { ORDER_TYPE, PAYMENT_STATUS, PAY_CHANNEL, toOptions } from '@/constants/enums'
import { dateColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, StatusFilterBar } from '@/components/common'
import type { StoreOrder } from '@/types/models'

// 退款申请统一在「支付与退款」页按支付单发起（服务端退款以 paymentOrderId 为键，
// 本店订单读模型不含该字段），本页为只读订单总览。
const list = useAsyncList<StoreOrder>((params) => orderService.list(params), {
  initialFilters: { paymentStatus: '', keyword: '' },
})

const columns = computed<DataTableColumns<StoreOrder>>(() => [
  textColumn<StoreOrder>('ID', (r) => r.id, { width: 72 }),
  {
    title: '用户信息', key: 'member', width: 190,
    render: (row) => h('div', { class: 'member-cell' }, [
      h(NAvatar, { round: true, size: 34, src: row.memberAvatarUrl }, () => (row.memberNickname || '会').slice(0, 1)),
      h('div', { class: 'member-cell__text' }, [
        h('span', row.memberNickname || '未关联会员'),
        h('small', row.memberPhoneMasked || row.memberPhone || '暂无手机号'),
      ]),
    ]),
  },
  textColumn<StoreOrder>('订单号', (r) => r.orderNo, { width: 190 }),
  statusColumn<StoreOrder>('类型', ORDER_TYPE, (r) => r.orderType, { width: 96 }),
  moneyColumn<StoreOrder>('金额', (r) => r.amountCent, { width: 120 }),
  statusColumn<StoreOrder>('支付状态', PAYMENT_STATUS, (r) => r.paymentStatus, { width: 110 }),
  statusColumn<StoreOrder>('渠道', PAY_CHANNEL, (r) => r.payChannel, { width: 90 }),
  dateColumn<StoreOrder>('下单时间', (r) => r.createdAt, { width: 150 }),
])

const typeOptions = toOptions(ORDER_TYPE)
</script>

<template>
  <div>
    <PageHeader
      title="本店订单"
      description="本店全部业务订单统一视图"
    />

    <StatusFilterBar
      :status-options="toOptions(PAYMENT_STATUS)"
      :status="(list.filters.paymentStatus as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索订单号 / 会员 / 手机号"
      @update:status="list.filters.paymentStatus = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    >
      <template #filters>
        <NButton
          v-for="opt in typeOptions"
          :key="opt.value"
          size="small"
          :type="list.filters.orderType === opt.value ? 'primary' : 'default'"
          quaternary
          @click="
            list.applyFilters({ orderType: list.filters.orderType === opt.value ? '' : opt.value })
          "
        >
          {{ opt.label }}
        </NButton>
      </template>
    </StatusFilterBar>

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无订单"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
  </div>
</template>

<style scoped>
.member-cell { display: flex; align-items: center; gap: var(--ic-space-2); min-width: 0; }
.member-cell__text { display: flex; flex-direction: column; min-width: 0; }
.member-cell__text span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.member-cell__text small { color: var(--ic-color-text-tertiary); }
</style>
