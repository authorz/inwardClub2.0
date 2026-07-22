<script setup lang="ts">
/**
 * 支付单（只读）。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { PAYMENT_STATUS_OPTIONS, PAY_CHANNEL_OPTIONS } from '@/constants/enums'
import { readonlyLists } from '@/api/services'
import type { PaymentOrder } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

// 服务端 GET /admin/payment-orders 仅支持 status 与 storeId 过滤（见 payment/admin_handler.go）。
const fields: FilterField[] = [
  { key: 'status', label: '支付状态', type: 'select', options: PAYMENT_STATUS_OPTIONS },
  { key: 'storeId', label: '门店 ID', type: 'input' },
]

const columns = [
  textColumn<PaymentOrder>('支付单号', 'paymentOrderNo', { width: 200 }),
  textColumn<PaymentOrder>('业务订单号', 'businessOrderNo', { width: 200 }),
  textColumn<PaymentOrder>('门店', 'storeName', { width: 140 }),
  moneyColumn<PaymentOrder>('金额', 'amountCent'),
  statusColumn<PaymentOrder>('支付方式', 'payMethod', PAY_CHANNEL_OPTIONS),
  statusColumn<PaymentOrder>('支付状态', 'paymentStatus', PAYMENT_STATUS_OPTIONS),
  dateTimeColumn<PaymentOrder>('创建时间', 'createdAt'),
]
</script>

<template>
  <ResourceListView
    title="支付单"
    description="统一支付单查询（只读）"
    :breadcrumb="['支付与退款', '支付单']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.paymentOrders"
  />
</template>
