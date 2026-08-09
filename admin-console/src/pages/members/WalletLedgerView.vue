<script setup lang="ts">
/**
 * 会员资产流水。服务端保留稳定的英文枚举，本页负责转换为管理员可理解的
 * 中文业务文案，并补充会员和关联订单上下文。
 */
import { computed, h, onMounted, ref } from 'vue'
import { NAvatar } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { ASSET_TYPE_OPTIONS } from '@/constants/enums'
import type { OptionItem } from '@/constants/enums'
import { readonlyLists, storeService } from '@/api/services'
import type { WalletLedgerEntry } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError } from '@/utils/feedback'

const DIRECTION_OPTIONS: OptionItem[] = [
  { label: '增加', value: 'credit', tone: 'success' },
  { label: '扣减', value: 'debit', tone: 'error' },
]

const SOURCE_OPTIONS: OptionItem[] = [
  { label: '存积分', value: 'point_saving', tone: 'success' },
  { label: '取积分', value: 'point_withdrawal', tone: 'warning' },
  { label: '购买餐品赠送', value: 'food_order', tone: 'success' },
  { label: '签到奖励', value: 'sign_in', tone: 'success' },
  { label: '充值到账', value: 'recharge_order', tone: 'info' },
  { label: '用户首充奖励', value: 'first_recharge_reward', tone: 'success' },
  { label: '满额充值奖励', value: 'high_value_recharge_reward', tone: 'success' },
  { label: '充值成长值', value: 'recharge_growth', tone: 'info' },
  { label: '微信支付成长值', value: 'wechat_payment_growth', tone: 'info' },
  { label: '订单支付', value: 'payment_order', tone: 'warning' },
  { label: '订单退款', value: 'refund_order', tone: 'success' },
  { label: '管理员调账', value: 'admin_adjustment', tone: 'error' },
  { label: '线下收款奖励', value: 'offline_collection', tone: 'info' },
  { label: '预约低消奖励', value: 'low_spend_reward', tone: 'success' },
]

const STATUS_OPTIONS: OptionItem[] = [
  { label: '已完成', value: 'completed', tone: 'success' },
  { label: '待审核', value: 'pending', tone: 'warning' },
  { label: '已通过', value: 'approved', tone: 'success' },
  { label: '已驳回', value: 'rejected', tone: 'error' },
]

const sourceLabels = Object.fromEntries(SOURCE_OPTIONS.map((item) => [item.value, item.label]))
const reasonLabels: Record<string, string> = {
  point_saving: '存积分',
  point_saving_reward: '存积分审核到账',
  point_saving_coin_reward: '存积分金币奖励',
  point_withdrawal: '取积分',
  food_order_reward: '购买餐品赠送积分',
  food_order_cancel_clawback: '取消订单扣回赠送积分',
  food_order_cancel_rollback: '取消订单失败返还积分',
  sign_in: '签到奖励',
  recharge: '充值到账',
  first_recharge_reward: '用户首充获得积分',
  high_value_recharge_reward: '满额充值获得积分',
  recharge_growth: '充值成长值',
  wechat_payment_growth: '微信支付获得成长值',
  order_payment: '订单支付',
  refund: '订单退款返还',
  admin_adjustment: '管理员调账',
  low_spend_reward: '预约低消达标奖励',
}

const storeOptions = ref<OptionItem[]>([])
const fields = computed<FilterField[]>(() => [
  { key: 'id', label: '流水 ID', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'memberNickname', label: '会员昵称', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'memberPhone', label: '会员手机号', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'storeId', label: '门店', type: 'select', options: storeOptions.value, width: 200 },
  { key: 'assetType', label: '资产类型', type: 'select', options: ASSET_TYPE_OPTIONS },
  { key: 'direction', label: '变动方向', type: 'select', options: DIRECTION_OPTIONS },
  { key: 'sourceType', label: '业务来源', type: 'select', options: SOURCE_OPTIONS },
  { key: 'status', label: '处理状态', type: 'select', options: STATUS_OPTIONS },
  { key: 'reason', label: '原因', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'created', label: '变动时间', type: 'daterange' },
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
  textColumn<WalletLedgerEntry>('ID', 'id', { width: 56 }),
  renderColumn<WalletLedgerEntry>(
    '会员信息',
    'member',
    (row) => {
      const nickname = row.memberNickname?.trim() || `会员 ${row.memberId}`
      const phone = row.memberPhone?.trim() || '暂无手机号'
      const fallback = () => nickname.slice(0, 1)
      return h('div', { class: 'ledger-member' }, [
        h(
          NAvatar,
          {
            class: 'ledger-member__avatar',
            size: 30,
            round: true,
            src: row.memberAvatarUrl || undefined,
            objectFit: 'cover',
          },
          row.memberAvatarUrl ? { fallback } : { default: fallback },
        ),
        h('div', { class: 'ledger-member__details' }, [
          h('span', { class: 'ledger-member__nickname', title: nickname }, nickname),
          h('span', { class: 'ledger-member__phone' }, phone),
        ]),
      ])
    },
    160,
  ),
  textColumn<WalletLedgerEntry>('门店', 'storeName', { width: 110 }),
  statusColumn<WalletLedgerEntry>('资产', 'assetType', ASSET_TYPE_OPTIONS, 78),
  statusColumn<WalletLedgerEntry>('方向', 'direction', DIRECTION_OPTIONS, 78),
  textColumn<WalletLedgerEntry>('变动数量', 'amount', { width: 82 }),
  renderColumn<WalletLedgerEntry>(
    '变动后余额',
    'balanceAfter',
    (row) => row.balanceAfter ?? '—',
    90,
  ),
  statusColumn<WalletLedgerEntry>('处理状态', 'status', STATUS_OPTIONS, 88),
  renderColumn<WalletLedgerEntry>(
    '业务来源',
    'sourceType',
    (row) => sourceLabels[row.sourceType ?? ''] ?? '其他业务',
    100,
  ),
  textColumn<WalletLedgerEntry>('关联订单号', 'relatedOrderNo', { width: 140 }),
  renderColumn<WalletLedgerEntry>(
    '原因',
    'reason',
    (row) => {
      const reason = reasonLabels[row.reason ?? ''] ?? row.reason ?? '—'
      return h('span', { class: 'ledger-cell-ellipsis', title: reason }, reason)
    },
    120,
  ),
  dateTimeColumn<WalletLedgerEntry>('变动时间', 'createdAt', 136),
]
</script>

<template>
  <ResourceListView
    title="资产流水"
    description="查看会员积分、金币、余额、成长值以及存取积分申请记录"
    :breadcrumb="['用户管理', '资产流水']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.walletLedger"
    :row-key="(row) => row.recordKey"
  />
</template>

<style>
.ledger-member {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 2px 0;
}

.ledger-member__avatar {
  flex: none;
}

.ledger-member__details {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.ledger-member__nickname {
  overflow: hidden;
  color: var(--ic-color-text);
  font-weight: 600;
  font-size: 13px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ledger-member__phone {
  color: var(--ic-color-text-secondary);
  font-size: 11px;
  line-height: 16px;
  font-variant-numeric: tabular-nums;
}

.ledger-cell-ellipsis {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
