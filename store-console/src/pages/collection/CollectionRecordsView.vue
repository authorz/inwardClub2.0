<script setup lang="ts">
/**
 * 本店统一订单列表：功能与总后台订单中心对齐，但门店范围只取 JWT scope。
 * 页面和请求均不提供 storeId，服务端也会忽略客户端伪造的 storeId。
 */
import { computed, h, reactive, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NDatePicker,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NModal,
  NSelect,
  NSpace,
  type DataTableColumns,
} from 'naive-ui'
import { orderService } from '@/api/services'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { useAsyncList } from '@/composables/useAsyncList'
import {
  ORDER_PAYMENT_STATUS,
  ORDER_STATUS,
  ORDER_TYPE,
  PAY_CHANNEL,
  toOptions,
} from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { DataTable, PageHeader, PermissionButton } from '@/components/common'
import { actionColumn, dateColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { centToYuan, formatCent, yuanToCent } from '@/utils/format'
import { feedback } from '@/utils/feedback'
import type { StoreOrder } from '@/types/models'

const list = useAsyncList<StoreOrder>((params) => orderService.list(params), {
  initialFilters: {
    orderType: '',
    paymentStatus: '',
    orderStatus: '',
    payChannel: '',
    memberNickname: '',
    memberPhone: '',
    keyword: '',
    createdFrom: '',
    createdTo: '',
  },
})

const createdRange = ref<[number, number] | null>(null)
const orderTypeOptions = toOptions(ORDER_TYPE).map(({ label, value }) => ({ label, value }))
const paymentStatusOptions = toOptions(ORDER_PAYMENT_STATUS).map(({ label, value }) => ({
  label,
  value,
}))
const orderStatusOptions = toOptions(ORDER_STATUS).map(({ label, value }) => ({ label, value }))
const payChannelOptions = toOptions(PAY_CHANNEL).map(({ label, value }) => ({ label, value }))

function applyFilters(): void {
  const range = createdRange.value
  list.applyFilters({
    createdFrom: range ? new Date(range[0]).toISOString() : '',
    createdTo: range ? new Date(range[1]).toISOString() : '',
  })
}

function resetFilters(): void {
  createdRange.value = null
  list.reset()
}

const refundTarget = ref<StoreOrder | null>(null)
const refundShow = ref(false)
const refundForm = reactive<{ amountYuan: number | null; reason: string }>({
  amountYuan: null,
  reason: '',
})
const refundAction = useAsyncAction()

function canRefund(row: StoreOrder): boolean {
  return (
    row.paymentStatus === 'paid' &&
    Boolean(row.paymentOrderId) &&
    row.orderType !== 'recharge' &&
    !['pending', 'processing', 'succeeded', 'success'].includes(row.refundStatus ?? '')
  )
}

function openRefund(row: StoreOrder): void {
  if (!canRefund(row)) {
    feedback.message.error('只有未申请退款的已支付订单可以退款')
    return
  }
  refundTarget.value = row
  refundForm.amountYuan = null
  refundForm.reason = ''
  refundShow.value = true
}

function closeRefund(): void {
  if (refundAction.running.value) return
  refundShow.value = false
  refundTarget.value = null
  refundForm.amountYuan = null
  refundForm.reason = ''
}

function fillFullRefund(): void {
  refundForm.amountYuan = centToYuan(refundTarget.value?.amountCent)
}

async function submitRefund(): Promise<void> {
  const order = refundTarget.value
  if (!order?.paymentOrderId) return
  const amountCent = yuanToCent(refundForm.amountYuan)
  if (amountCent <= 0) {
    feedback.message.error('请输入有效的退款金额')
    return
  }
  if (amountCent > order.amountCent) {
    feedback.message.error('退款金额不能超过订单实付金额')
    return
  }
  const reason = refundForm.reason.trim()
  if (!reason) {
    feedback.message.error('请输入退款原因')
    return
  }
  await refundAction.run(
    () =>
      orderService.requestRefund({
        paymentOrderId: order.paymentOrderId!,
        amountCent,
        reason,
      }),
    {
      successMessage: '退款申请已提交',
      onSuccess: () => {
        refundShow.value = false
        refundTarget.value = null
        list.refresh()
      },
    },
  )
}

const columns = computed<DataTableColumns<StoreOrder>>(() => [
  textColumn<StoreOrder>('ID', (row) => row.id, { width: 80 }),
  {
    title: '用户信息',
    key: 'member',
    width: 210,
    render: (row) => {
      const nickname = row.memberNickname?.trim() || '未关联会员'
      const phone = row.memberPhoneMasked || row.memberPhone || '暂无手机号'
      return h('div', { class: 'order-member' }, [
        h(
          NAvatar,
          { round: true, size: 36, src: row.memberAvatarUrl || undefined, objectFit: 'cover' },
          () => nickname.slice(0, 1),
        ),
        h('div', { class: 'order-member__details' }, [
          h('span', { class: 'order-member__nickname', title: nickname }, nickname),
          h('span', { class: 'order-member__phone' }, phone),
        ]),
      ])
    },
  },
  textColumn<StoreOrder>('订单号', (row) => row.orderNo, { width: 200 }),
  statusColumn<StoreOrder>('类型', ORDER_TYPE, (row) => row.orderType, { width: 110 }),
  textColumn<StoreOrder>('门店', (row) => row.storeName, { width: 140 }),
  moneyColumn<StoreOrder>('金额', (row) => row.amountCent, { width: 120 }),
  statusColumn<StoreOrder>('支付状态', ORDER_PAYMENT_STATUS, (row) => row.paymentStatus, {
    width: 110,
  }),
  statusColumn<StoreOrder>('订单状态', ORDER_STATUS, (row) => row.orderStatus, { width: 110 }),
  dateColumn<StoreOrder>('创建时间', (row) => row.createdAt, { width: 160 }),
  actionColumn<StoreOrder>(
    (row) =>
      canRefund(row)
        ? h(
            PermissionButton,
            {
              permissions: [PERM.refundRequest],
              type: 'error',
              text: true,
              onClick: () => openRefund(row),
            },
            { default: () => '退款' },
          )
        : h('span', { class: 'ic-muted' }, '—'),
    '操作',
    90,
  ),
])
</script>

<template>
  <section class="collection-records">
    <PageHeader
      title="收款记录"
      description="本店统一订单列表（点餐 / 活动 / 充值 / 券 / 线下聚合收款）"
    />

    <div class="order-filters">
      <label class="order-filter">
        <span>订单类型</span>
        <NSelect
          :value="list.filters.orderType as string"
          :options="orderTypeOptions"
          placeholder="全部"
          clearable
          @update:value="list.filters.orderType = $event ?? ''"
        />
      </label>
      <label class="order-filter">
        <span>支付状态</span>
        <NSelect
          :value="list.filters.paymentStatus as string"
          :options="paymentStatusOptions"
          placeholder="全部"
          clearable
          @update:value="list.filters.paymentStatus = $event ?? ''"
        />
      </label>
      <label class="order-filter">
        <span>订单状态</span>
        <NSelect
          :value="list.filters.orderStatus as string"
          :options="orderStatusOptions"
          placeholder="全部"
          clearable
          @update:value="list.filters.orderStatus = $event ?? ''"
        />
      </label>
      <label class="order-filter">
        <span>支付渠道</span>
        <NSelect
          :value="list.filters.payChannel as string"
          :options="payChannelOptions"
          placeholder="全部"
          clearable
          @update:value="list.filters.payChannel = $event ?? ''"
        />
      </label>
      <label class="order-filter">
        <span>会员昵称</span>
        <NInput
          :value="list.filters.memberNickname as string"
          placeholder="支持模糊搜索"
          clearable
          @update:value="list.filters.memberNickname = $event"
          @keyup.enter="applyFilters"
        />
      </label>
      <label class="order-filter">
        <span>会员手机号</span>
        <NInput
          :value="list.filters.memberPhone as string"
          placeholder="支持模糊搜索"
          clearable
          @update:value="list.filters.memberPhone = $event"
          @keyup.enter="applyFilters"
        />
      </label>
      <label class="order-filter">
        <span>订单号</span>
        <NInput
          :value="list.filters.keyword as string"
          placeholder="支持模糊搜索"
          clearable
          @update:value="list.filters.keyword = $event"
          @keyup.enter="applyFilters"
        />
      </label>
      <label class="order-filter order-filter--date">
        <span>创建时间</span>
        <NDatePicker
          v-model:value="createdRange"
          type="daterange"
          clearable
        />
      </label>
      <NSpace class="order-filters__actions">
        <NButton
          type="primary"
          :loading="list.loading.value"
          @click="applyFilters"
        >
          查询
        </NButton>
        <NButton @click="resetFilters">
          重置
        </NButton>
      </NSpace>
    </div>

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      :scroll-x="1320"
      empty-text="暂无订单"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="refundShow"
      preset="card"
      title="确认退款"
      class="refund-modal"
      :mask-closable="!refundAction.running.value"
      @after-leave="closeRefund"
    >
      <div class="refund-summary">
        <span>订单号：{{ refundTarget?.orderNo ?? '—' }}</span>
        <strong>订单实付：{{ formatCent(refundTarget?.amountCent) }}</strong>
      </div>
      <NForm label-placement="top">
        <NFormItem
          label="退款金额（元）"
          required
        >
          <div class="refund-amount">
            <NInputNumber
              v-model:value="refundForm.amountYuan"
              :min="0.01"
              :max="centToYuan(refundTarget?.amountCent)"
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
            placeholder="请输入退款原因"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton
            :disabled="refundAction.running.value"
            @click="closeRefund"
          >
            取消
          </NButton>
          <NButton
            type="error"
            :loading="refundAction.running.value"
            @click="submitRefund"
          >
            确认退款
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </section>
</template>

<style scoped>
.collection-records {
  max-width: 1480px;
}

.order-filters {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ic-space-4);
  align-items: flex-end;
  padding-bottom: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
}

.order-filter {
  width: 160px;
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-1);
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.order-filter--date {
  width: 260px;
}

.order-filters__actions {
  margin-left: auto;
}

.order-member {
  display: flex;
  gap: var(--ic-space-2);
  align-items: center;
  min-width: 0;
}

.order-member__details {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.order-member__nickname {
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.order-member__phone {
  color: var(--ic-color-text-tertiary);
  font-size: var(--ic-font-xs);
  font-variant-numeric: tabular-nums;
}

:global(.refund-modal) {
  width: min(520px, calc(100vw - 32px));
}

.refund-summary {
  display: flex;
  gap: var(--ic-space-4);
  justify-content: space-between;
  padding-bottom: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
  color: var(--ic-color-text-secondary);
}

.refund-summary strong {
  color: var(--ic-color-danger);
}

.refund-amount {
  width: 100%;
  display: flex;
  gap: var(--ic-space-2);
}

.refund-amount .n-input-number {
  flex: 1;
}

@media (max-width: 720px) {
  .order-filter,
  .order-filter--date {
    width: 100%;
  }

  .order-filters__actions {
    width: 100%;
    margin-left: 0;
    justify-content: flex-end;
  }

  .refund-summary {
    flex-direction: column;
  }
}
</style>
