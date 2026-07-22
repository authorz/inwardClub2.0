<script setup lang="ts">
/**
 * 订单中心（统一订单读模型，只读）。
 * 支持按订单类型 / 支付状态 / 订单状态 / 支付渠道 / 会员手机号 / 门店筛选。
 * 后台不允许任意创建/删除已支付订单，因此本页只读。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import {
  ORDER_STATUS_OPTIONS,
  ORDER_TYPE_OPTIONS,
  PAYMENT_STATUS_OPTIONS,
  PAY_CHANNEL_OPTIONS,
} from '@/constants/enums'
import { orderService } from '@/api/services'
import type { BusinessOrder } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  { key: 'orderType', label: '订单类型', type: 'select', options: ORDER_TYPE_OPTIONS },
  { key: 'paymentStatus', label: '支付状态', type: 'select', options: PAYMENT_STATUS_OPTIONS },
  { key: 'orderStatus', label: '订单状态', type: 'select', options: ORDER_STATUS_OPTIONS },
  { key: 'payChannel', label: '支付渠道', type: 'select', options: PAY_CHANNEL_OPTIONS },
  { key: 'memberPhone', label: '会员手机号', type: 'input' },
  { key: 'keyword', label: '订单号', type: 'input' },
  { key: 'created', label: '创建时间', type: 'daterange' },
]

const columns = [
  textColumn<BusinessOrder>('订单号', 'orderNo', { width: 200 }),
  statusColumn<BusinessOrder>('类型', 'orderType', ORDER_TYPE_OPTIONS, 110),
  textColumn<BusinessOrder>('门店', 'storeName', { width: 140 }),
  moneyColumn<BusinessOrder>('金额', 'amountCent'),
  statusColumn<BusinessOrder>('支付状态', 'paymentStatus', PAYMENT_STATUS_OPTIONS),
  statusColumn<BusinessOrder>('订单状态', 'orderStatus', ORDER_STATUS_OPTIONS),
  textColumn<BusinessOrder>('会员', 'memberPhone', { width: 130 }),
  dateTimeColumn<BusinessOrder>('创建时间', 'createdAt'),
]
</script>

<template>
  <ResourceListView
    title="订单中心"
    description="统一订单读模型（点餐 / 活动 / 充值 / 券 / 线下聚合收款）"
    :breadcrumb="['订单中心']"
    :fields="fields"
    :columns="columns"
    :fetcher="orderService.list"
  />
</template>
