<script setup lang="ts">
import { computed } from 'vue'
import type { DataTableColumns } from 'naive-ui'
import { orderService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { PAYMENT_STATUS, PAY_CHANNEL, toOptions } from '@/constants/enums'
import { DataTable, PageHeader, StatusFilterBar } from '@/components/common'
import { dateColumn, moneyColumn, phoneColumn, statusColumn, textColumn } from '@/utils/columns'
import type { StoreOrder } from '@/types/models'

const list = useAsyncList<StoreOrder>((params) => orderService.list({ ...params, orderType: 'offline_collection' }), {
  initialFilters: { paymentStatus: '', keyword: '' },
})
const columns = computed<DataTableColumns<StoreOrder>>(() => [
  textColumn<StoreOrder>('订单号', (row) => row.orderNo),
  textColumn<StoreOrder>('会员', (row) => row.memberNickname),
  phoneColumn<StoreOrder>('手机号', (row) => row.memberPhoneMasked),
  moneyColumn<StoreOrder>('金额', (row) => row.amountCent, { width: 110 }),
  statusColumn<StoreOrder>('支付状态', PAYMENT_STATUS, (row) => row.paymentStatus, { width: 110 }),
  statusColumn<StoreOrder>('渠道', PAY_CHANNEL, (row) => row.payChannel, { width: 90 }),
  dateColumn<StoreOrder>('创建时间', (row) => row.createdAt, { width: 150 }),
])
</script>

<template>
  <div>
    <PageHeader
      title="收款记录"
      description="当前门店通过线下聚合收款产生的订单记录"
    />
    <StatusFilterBar
      :status-options="toOptions(PAYMENT_STATUS)"
      :status="list.filters.paymentStatus as string"
      :keyword="list.filters.keyword as string"
      :loading="list.loading.value"
      search-placeholder="搜索订单号 / 会员 / 手机号"
      @update:status="list.filters.paymentStatus = $event ?? ''"
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
      empty-text="暂无收款记录"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
  </div>
</template>
