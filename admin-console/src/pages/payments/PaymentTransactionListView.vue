<script setup lang="ts">
/**
 * 支付流水（只读）。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { PAYMENT_STATUS_OPTIONS, PAY_CHANNEL_OPTIONS } from '@/constants/enums'
import { readonlyLists } from '@/api/services'
import type { PaymentTransaction } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  { key: 'keyword', label: '支付单号', type: 'input' },
  { key: 'payMethod', label: '支付方式', type: 'select', options: PAY_CHANNEL_OPTIONS },
  { key: 'status', label: '支付状态', type: 'select', options: PAYMENT_STATUS_OPTIONS },
  { key: 'created', label: '创建时间', type: 'daterange' },
]

const columns = [
  textColumn<PaymentTransaction>('支付单号', 'paymentOrderNo', { width: 200 }),
  textColumn<PaymentTransaction>('业务订单号', 'businessOrderNo', { width: 200 }),
  moneyColumn<PaymentTransaction>('金额', 'amountCent'),
  statusColumn<PaymentTransaction>('支付方式', 'payMethod', PAY_CHANNEL_OPTIONS),
  statusColumn<PaymentTransaction>('状态', 'status', PAYMENT_STATUS_OPTIONS),
  dateTimeColumn<PaymentTransaction>('创建时间', 'createdAt'),
]
</script>

<template>
  <ResourceListView
    title="支付流水"
    description="支付网关流水查询（只读）"
    :breadcrumb="['支付与退款', '支付流水']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.paymentTransactions"
  />
</template>
