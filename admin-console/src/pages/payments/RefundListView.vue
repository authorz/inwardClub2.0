<script setup lang="ts">
/**
 * 退款单（只读列表）。
 *
 * 服务端当前仅提供退款单查询（GET /admin/refunds）与「对支付单发起退款」
 * （POST /admin/refunds，body: paymentOrderId/amountCent/reason），
 * 尚无退款审批 / 处理状态机接口，退款创建后即为 pending。
 * 因此本页为只读；发起退款 / 审批待服务端补充后再接入。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { REFUND_STATUS_OPTIONS } from '@/constants/enums'
import { readonlyLists } from '@/api/services'
import type { RefundOrder } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

// 服务端 GET /admin/refunds 支持 status 与 storeId 过滤。
const fields: FilterField[] = [
  { key: 'status', label: '退款状态', type: 'select', options: REFUND_STATUS_OPTIONS },
  { key: 'storeId', label: '门店 ID', type: 'input' },
]

const columns = [
  textColumn<RefundOrder>('退款单号', 'refundOrderNo', { width: 200 }),
  textColumn<RefundOrder>('支付单 ID', 'paymentOrderId', { width: 120 }),
  textColumn<RefundOrder>('门店', 'storeName', { width: 140 }),
  moneyColumn<RefundOrder>('退款金额', 'amountCent'),
  statusColumn<RefundOrder>('退款状态', 'status', REFUND_STATUS_OPTIONS),
  textColumn<RefundOrder>('渠道', 'channel', { width: 100 }),
  textColumn<RefundOrder>('原因', 'reason'),
  dateTimeColumn<RefundOrder>('创建时间', 'createdAt'),
]
</script>

<template>
  <ResourceListView
    title="退款单"
    description="退款单查询（只读）；退款审批 / 处理状态机接口待服务端补充"
    :breadcrumb="['支付与退款', '退款单']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.refundOrders"
  />
</template>
