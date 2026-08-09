<script setup lang="ts">
/**
 * 订单中心（统一订单读模型）。
 * 支持复合模糊搜索、门店筛选和管理员密码复核后的部分/全额退款。
 */
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NAvatar, NButton, NForm, NFormItem, NInput, NInputNumber } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import {
  actionsColumn,
  dateTimeColumn,
  moneyColumn,
  renderColumn,
  statusColumn,
  textColumn,
} from '@/utils/columns'
import {
  ORDER_STATUS_OPTIONS,
  ORDER_TYPE_OPTIONS,
  PAY_CHANNEL_OPTIONS,
} from '@/constants/enums'
import type { OptionItem } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { orderService, storeService } from '@/api/services'
import type { BusinessOrder } from '@/api/models'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import { formatCent } from '@/utils/format'
import { toastError, toastSuccess } from '@/utils/feedback'

const ORDER_PAYMENT_STATUS_OPTIONS: OptionItem[] = [
  { label: '未支付', value: 'unpaid', tone: 'warning' },
  { label: '已支付', value: 'paid', tone: 'success' },
  { label: '已过期', value: 'expired', tone: 'default' },
  { label: '部分退款', value: 'partially_refunded', tone: 'warning' },
  { label: '已退款', value: 'refunded', tone: 'info' },
]

const listRef = ref<ResourceListInstance | null>(null)
const storeOptions = ref<OptionItem[]>([])
const fields = computed<FilterField[]>(() => [
  { key: 'orderType', label: '订单类型', type: 'select', options: ORDER_TYPE_OPTIONS },
  { key: 'paymentStatus', label: '支付状态', type: 'select', options: ORDER_PAYMENT_STATUS_OPTIONS },
  { key: 'orderStatus', label: '订单状态', type: 'select', options: ORDER_STATUS_OPTIONS },
  { key: 'payChannel', label: '支付渠道', type: 'select', options: PAY_CHANNEL_OPTIONS },
  { key: 'storeId', label: '门店', type: 'select', options: storeOptions.value, width: 200 },
  { key: 'memberNickname', label: '会员昵称', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'memberPhone', label: '会员手机号', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'keyword', label: '订单号', type: 'input', placeholder: '支持模糊搜索' },
  { key: 'created', label: '创建时间', type: 'daterange' },
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

const refundShow = ref(false)
const refundSubmitting = ref(false)
const refundTarget = ref<BusinessOrder | null>(null)
const refundForm = reactive<{ amountYuan: number | null; reason: string; password: string }>({
  amountYuan: null,
  reason: '',
  password: '',
})

function canRefund(row: BusinessOrder): boolean {
  return (
    row.paymentStatus === 'paid' &&
    Boolean(row.paymentOrderId) &&
    row.orderType !== 'recharge' &&
    row.refundStatus !== 'processing' &&
    row.refundStatus !== 'succeeded'
  )
}

function openRefund(row: BusinessOrder): void {
  if (!canRefund(row)) {
    toastError('只有未退款的已支付订单可以退款')
    return
  }
  refundTarget.value = row
  refundForm.amountYuan = null
  refundForm.reason = ''
  refundForm.password = ''
  refundShow.value = true
}

function closeRefund(): void {
  refundShow.value = false
  refundTarget.value = null
  refundForm.amountYuan = null
  refundForm.reason = ''
  refundForm.password = ''
}

function fillFullRefund(): void {
  refundForm.amountYuan = (refundTarget.value?.amountCent ?? 0) / 100
}

async function submitRefund(): Promise<void> {
  const order = refundTarget.value
  if (!order?.paymentOrderId) return
  const amountCent = Math.round((refundForm.amountYuan ?? 0) * 100)
  if (amountCent <= 0) {
    toastError('请输入有效的退款金额')
    return
  }
  if (amountCent > order.amountCent) {
    toastError('退款金额不能超过订单实付金额')
    return
  }
  if (!refundForm.reason.trim()) {
    toastError('请输入退款原因')
    return
  }
  if (!refundForm.password) {
    toastError('请输入管理员登录密码')
    return
  }
  refundSubmitting.value = true
  try {
    await orderService.refund({
      paymentOrderId: order.paymentOrderId,
      amountCent,
      reason: refundForm.reason.trim(),
      password: refundForm.password,
    })
    toastSuccess('退款已提交并处理成功')
    closeRefund()
    await listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '退款失败')
  } finally {
    refundSubmitting.value = false
  }
}

const columns = [
  textColumn<BusinessOrder>('ID', 'id', { width: 80 }),
  renderColumn<BusinessOrder>(
    '用户信息',
    'member',
    (row) => {
      const nickname = row.memberNickname?.trim() || '未关联会员'
      const phone = row.memberPhone?.trim() || '暂无手机号'
      const fallback = () => nickname.slice(0, 1)
      return h('div', { class: 'order-member' }, [
        h(
          NAvatar,
          {
            class: 'order-member__avatar',
            size: 36,
            round: true,
            src: row.memberAvatarUrl || undefined,
            objectFit: 'cover',
          },
          row.memberAvatarUrl ? { fallback } : { default: fallback },
        ),
        h('div', { class: 'order-member__details' }, [
          h('span', { class: 'order-member__nickname', title: nickname }, nickname),
          h('span', { class: 'order-member__phone' }, phone),
        ]),
      ])
    },
    210,
  ),
  textColumn<BusinessOrder>('订单号', 'orderNo', { width: 200 }),
  statusColumn<BusinessOrder>('类型', 'orderType', ORDER_TYPE_OPTIONS, 110),
  textColumn<BusinessOrder>('门店', 'storeName', { width: 140 }),
  moneyColumn<BusinessOrder>('金额', 'amountCent'),
  statusColumn<BusinessOrder>('支付状态', 'paymentStatus', ORDER_PAYMENT_STATUS_OPTIONS),
  statusColumn<BusinessOrder>('订单状态', 'orderStatus', ORDER_STATUS_OPTIONS),
  dateTimeColumn<BusinessOrder>('创建时间', 'createdAt'),
  actionsColumn<BusinessOrder>(
    (row) => {
      if (!canRefund(row)) return null
      return h(
        PermissionButton,
        {
          permission: PERMISSIONS.REFUND_APPROVE,
          type: 'error',
          onClick: () => openRefund(row),
        },
        () => '退款',
      )
    },
    100,
  ),
]
</script>

<template>
  <ResourceListView
    ref="listRef"
    title="订单中心"
    description="统一订单读模型（点餐 / 活动 / 充值 / 券 / 线下聚合收款）"
    :breadcrumb="['订单中心']"
    :fields="fields"
    :columns="columns"
    :fetcher="orderService.list"
  />

  <FormDrawer
    v-model:show="refundShow"
    title="确认退款"
    :submitting="refundSubmitting"
    :width="520"
    high-risk
    cross-store
    submit-text="确认退款"
    @submit="submitRefund"
    @cancel="closeRefund"
  >
    <NForm label-placement="top">
      <div class="refund-summary">
        <span>订单号：{{ refundTarget?.orderNo ?? '—' }}</span>
        <strong>订单实付：{{ formatCent(refundTarget?.amountCent ?? 0) }}</strong>
      </div>
      <NFormItem
        label="退款金额（元）"
        required
      >
        <div class="refund-amount">
          <NInputNumber
            v-model:value="refundForm.amountYuan"
            :min="0.01"
            :max="(refundTarget?.amountCent ?? 0) / 100"
            :precision="2"
            placeholder="请输入退款金额"
          />
          <NButton @click="fillFullRefund">
            全额退款
          </NButton>
        </div>
      </NFormItem>
      <NFormItem
        label="退款原因"
        required
      >
        <NInput
          v-model:value="refundForm.reason"
          type="textarea"
          maxlength="200"
          show-count
          placeholder="请输入退款原因，操作将记录审计"
        />
      </NFormItem>
      <NFormItem
        label="管理员登录密码"
        required
      >
        <NInput
          v-model:value="refundForm.password"
          type="password"
          show-password-on="click"
          autocomplete="current-password"
          placeholder="请输入当前登录密码确认退款"
          @keyup.enter="submitRefund"
        />
      </NFormItem>
    </NForm>
  </FormDrawer>
</template>

<style>
.order-member {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 2px 0;
}

.order-member__avatar {
  flex: none;
}

.order-member__details {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.order-member__nickname {
  overflow: hidden;
  color: var(--ic-color-text);
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.order-member__phone {
  color: var(--ic-color-text-secondary);
  font-size: 12px;
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}

.refund-summary {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  margin: 16px 0 20px;
  padding: 14px 16px;
  border-radius: var(--ic-radius-sm);
  background: var(--ic-color-surface-muted);
  color: var(--ic-color-text-secondary);
}

.refund-summary strong {
  color: var(--ic-color-error);
}

.refund-amount {
  display: flex;
  width: 100%;
  gap: 10px;
}

.refund-amount .n-input-number {
  flex: 1;
}
</style>
