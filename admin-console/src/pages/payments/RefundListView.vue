<script setup lang="ts">
/**
 * 最终退款记录列表。内部处理中的退款不属于审批状态，因此本页只展示
 * succeeded / failed，并提供订单、会员、门店和操作时间维度的查询。
 */
import { computed, h, onMounted, ref } from 'vue'
import { NAvatar } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import {
  dateTimeColumn,
  moneyColumn,
  renderColumn,
  statusColumn,
  textColumn,
} from '@/utils/columns'
import { PAY_CHANNEL_OPTIONS, REFUND_STATUS_OPTIONS } from '@/constants/enums'
import { readonlyLists, storeService } from '@/api/services'
import { toastError } from '@/utils/feedback'
import type { RefundOrder } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import type { OptionItem } from '@/constants/enums'

const storeOptions = ref<OptionItem[]>([])
const fields = computed<FilterField[]>(() => [
  { key: 'id', label: 'ID', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'keyword', label: '订单号', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'memberNickname', label: '会员昵称', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'memberPhone', label: '会员手机号', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'storeId', label: '门店', type: 'select', options: storeOptions.value, width: 200 },
  { key: 'status', label: '退款状态', type: 'select', options: REFUND_STATUS_OPTIONS },
  { key: 'operated', label: '操作时间', type: 'daterange' },
])

async function loadStores(): Promise<void> {
  try {
    const result = await storeService.list({ page: 1, pageSize: 100 })
    storeOptions.value = result.items.map((store) => ({
      label: store.name,
      value: String(store.id),
    }))
  } catch (error) {
    toastError((error as { message?: string }).message ?? '加载门店失败')
  }
}

onMounted(loadStores)

const columns = [
  textColumn<RefundOrder>('ID', 'id', { width: 80 }),
  renderColumn<RefundOrder>(
    '会员信息',
    'member',
    (row) => {
      const nickname = row.memberNickname?.trim() || '未关联会员'
      const phone = row.memberPhone?.trim() || '暂无手机号'
      const fallback = () => nickname.slice(0, 1)
      return h('div', { class: 'refund-member' }, [
        h(
          NAvatar,
          {
            class: 'refund-member__avatar',
            size: 36,
            round: true,
            src: row.memberAvatarUrl || undefined,
            objectFit: 'cover',
          },
          row.memberAvatarUrl ? { fallback } : { default: fallback },
        ),
        h('div', { class: 'refund-member__details' }, [
          h('span', { class: 'refund-member__nickname', title: nickname }, nickname),
          h('span', { class: 'refund-member__phone' }, phone),
        ]),
      ])
    },
    210,
  ),
  textColumn<RefundOrder>('门店', 'storeName', { width: 140 }),
  textColumn<RefundOrder>('订单号', 'businessOrderNo', { width: 200 }),
  moneyColumn<RefundOrder>('订单金额', 'orderAmountCent'),
  moneyColumn<RefundOrder>('退款金额', 'amountCent'),
  statusColumn<RefundOrder>('渠道', 'channel', PAY_CHANNEL_OPTIONS),
  textColumn<RefundOrder>('原因', 'reason', { width: 180 }),
  dateTimeColumn<RefundOrder>('订单创建时间', 'orderCreatedAt'),
  dateTimeColumn<RefundOrder>('操作时间', 'operatedAt'),
]
</script>

<template>
  <ResourceListView
    title="退款记录"
    description="查询退款成功或失败的最终处理记录"
    :breadcrumb="['订单管理', '退款记录']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.refundOrders"
  />
</template>

<style>
.refund-member {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 2px 0;
}

.refund-member__avatar {
  flex: none;
}

.refund-member__details {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.refund-member__nickname {
  overflow: hidden;
  color: var(--ic-color-text);
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.refund-member__phone {
  color: var(--ic-color-text-secondary);
  font-size: 12px;
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}
</style>
