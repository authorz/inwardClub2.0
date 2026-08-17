<script setup lang="ts">
/**
 * 支付与退款：支付订单、支付流水、退款单三个视图。
 * 门店范围来自 token scope，不传 storeId。唯一写操作为按支付单发起退款（POST /store/refunds，
 * 带 Idempotency-Key）；流水/退款单为只读，服务端无对应详情端点，详情弹窗直接复用列表行。
 */
import { computed, h, ref } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NInput,
  NModal,
  NSpace,
  NSpin,
  NTabPane,
  NTabs,
  type DataTableColumns,
} from 'naive-ui'
import { orderService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { ORDER_TYPE, PAY_CHANNEL, PAYMENT_STATUS, REFUND_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { actionColumn, dateColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { formatCent, formatDateTime } from '@/utils/format'
import { feedback } from '@/utils/feedback'
import { ApiError } from '@/api/error'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar, StatusTag } from '@/components/common'
import type { PaymentOrder, PaymentTransactionRecord, RefundOrder } from '@/types/models'

const paymentOrders = useAsyncList<PaymentOrder>((params) => orderService.paymentOrders(params), {
  initialFilters: { status: '', keyword: '' },
})
const paymentTransactions = useAsyncList<PaymentTransactionRecord>(
  (params) => orderService.paymentTransactions(params),
  { initialFilters: { status: '', keyword: '' }, immediate: false },
)
const refunds = useAsyncList<RefundOrder>((params) => orderService.refunds(params), {
  initialFilters: { status: '', keyword: '' },
  immediate: false,
})

function onTabChange(name: string | number) {
  if (name === 'transactions' && paymentTransactions.total.value === 0) paymentTransactions.refresh()
  if (name === 'refunds' && refunds.total.value === 0) refunds.refresh()
}

const action = useAsyncAction()
const refundPasswordShow = ref(false)
const refundPassword = ref('')
const refundPaymentOrder = ref<PaymentOrder | null>(null)

// 退款按支付单发起：服务端 POST /store/refunds 以 paymentOrderId 为键，可退金额与状态由服务端审批。
function onRefund(row: PaymentOrder): void {
  refundPaymentOrder.value = row
  refundPassword.value = ''
  refundPasswordShow.value = true
}

function closeRefundPassword(): void {
  if (action.running.value) return
  refundPasswordShow.value = false
  refundPassword.value = ''
  refundPaymentOrder.value = null
}

async function submitRefund(): Promise<void> {
  const row = refundPaymentOrder.value
  if (!row) return
  const password = refundPassword.value
  if (!password.trim()) {
    feedback.message.error('请输入门店管理员登录密码')
    return
  }
  await action.run(
    () =>
      orderService.requestRefund({
        paymentOrderId: row.id,
        amountCent: row.amountCent,
        reason: '门店后台发起退款申请',
        password,
      }),
    {
      successMessage: '退款申请已提交',
      onSuccess: () => {
        refundPasswordShow.value = false
        refundPassword.value = ''
        refundPaymentOrder.value = null
        paymentOrders.refresh()
        if (refunds.total.value > 0) refunds.refresh()
      },
    },
  )
}

const orderDetailShow = ref(false)
const orderDetailLoading = ref(false)
const currentPaymentOrder = ref<PaymentOrder | null>(null)
async function openOrderDetail(row: PaymentOrder) {
  currentPaymentOrder.value = row
  orderDetailShow.value = true
  orderDetailLoading.value = true
  try {
    currentPaymentOrder.value = await orderService.paymentOrderDetail(row.id)
  } catch (err) {
    if (!(err instanceof ApiError)) throw err
  } finally {
    orderDetailLoading.value = false
  }
}

// 服务端无支付流水/退款单详情端点，详情弹窗直接展示列表行。
const transactionDetailShow = ref(false)
const currentTransaction = ref<PaymentTransactionRecord | null>(null)
function openTransactionDetail(row: PaymentTransactionRecord) {
  currentTransaction.value = row
  transactionDetailShow.value = true
}

const refundDetailShow = ref(false)
const currentRefund = ref<RefundOrder | null>(null)
function openRefundDetail(row: RefundOrder) {
  currentRefund.value = row
  refundDetailShow.value = true
}

const paymentOrderColumns = computed<DataTableColumns<PaymentOrder>>(() => [
  textColumn<PaymentOrder>('支付单号', (r) => r.paymentOrderNo),
  textColumn<PaymentOrder>('业务单号', (r) => r.businessOrderNo),
  statusColumn<PaymentOrder>('业务类型', ORDER_TYPE, (r) => r.orderType, { width: 96 }),
  moneyColumn<PaymentOrder>('金额', (r) => r.amountCent, { width: 110 }),
  statusColumn<PaymentOrder>('支付方式', PAY_CHANNEL, (r) => r.payMethod, { width: 90 }),
  statusColumn<PaymentOrder>('支付状态', PAYMENT_STATUS, (r) => r.status, { width: 100 }),
  dateColumn<PaymentOrder>('创建时间', (r) => r.createdAt, { width: 150 }),
  dateColumn<PaymentOrder>('支付时间', (r) => r.paidAt, { width: 150 }),
  actionColumn<PaymentOrder>(
    (row) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(NButton, { type: 'primary', text: true, onClick: () => openOrderDetail(row) }, { default: () => '详情' }),
          h(
            PermissionButton,
            {
              permissions: [PERM.refundRequest],
              type: 'error',
              text: true,
              disabled: row.paymentStatus !== 'paid',
              onClick: () => onRefund(row),
            },
            { default: () => '退款申请' },
          ),
        ],
      }),
    '操作',
    150,
  ),
])

const transactionColumns = computed<DataTableColumns<PaymentTransactionRecord>>(() => [
  textColumn<PaymentTransactionRecord>('支付单号', (r) => r.paymentOrderNo),
  textColumn<PaymentTransactionRecord>('业务单号', (r) => r.businessOrderNo),
  statusColumn<PaymentTransactionRecord>('业务类型', ORDER_TYPE, (r) => r.orderType, { width: 96 }),
  moneyColumn<PaymentTransactionRecord>('金额', (r) => r.amountCent, { width: 110 }),
  statusColumn<PaymentTransactionRecord>('支付方式', PAY_CHANNEL, (r) => r.payMethod, { width: 90 }),
  textColumn<PaymentTransactionRecord>('状态', (r) => r.status, { width: 100 }),
  dateColumn<PaymentTransactionRecord>('时间', (r) => r.createdAt, { width: 150 }),
  actionColumn<PaymentTransactionRecord>(
    (row) => h(NButton, { type: 'primary', text: true, onClick: () => openTransactionDetail(row) }, { default: () => '详情' }),
    '操作',
    90,
  ),
])

const refundColumns = computed<DataTableColumns<RefundOrder>>(() => [
  textColumn<RefundOrder>('退款单号', (r) => r.refundOrderNo),
  textColumn<RefundOrder>('业务单号', (r) => r.businessOrderId),
  moneyColumn<RefundOrder>('退款金额', (r) => r.amountCent, { width: 110 }),
  statusColumn<RefundOrder>('渠道', PAY_CHANNEL, (r) => r.channel, { width: 90 }),
  statusColumn<RefundOrder>('状态', REFUND_STATUS, (r) => r.status, { width: 100 }),
  textColumn<RefundOrder>('原因', (r) => r.reason),
  dateColumn<RefundOrder>('申请时间', (r) => r.createdAt, { width: 150 }),
  actionColumn<RefundOrder>(
    (row) => h(NButton, { type: 'primary', text: true, onClick: () => openRefundDetail(row) }, { default: () => '详情' }),
    '操作',
    90,
  ),
])
</script>

<template>
  <div>
    <PageHeader
      title="支付与退款"
      description="本店支付订单、支付流水与退款单（只读）"
    />

    <NTabs
      type="line"
      default-value="orders"
      @update:value="onTabChange"
    >
      <NTabPane
        name="orders"
        tab="支付订单"
      >
        <StatusFilterBar
          :status-options="toOptions(PAYMENT_STATUS)"
          :status="(paymentOrders.filters.status as string) ?? null"
          :keyword="(paymentOrders.filters.keyword as string) ?? ''"
          :loading="paymentOrders.loading.value"
          search-placeholder="搜索支付单号 / 业务单号"
          @update:status="paymentOrders.filters.status = $event ?? ''"
          @update:keyword="paymentOrders.filters.keyword = $event"
          @apply="paymentOrders.applyFilters({})"
          @reset="paymentOrders.reset()"
        />
        <DataTable
          :columns="paymentOrderColumns"
          :data="paymentOrders.rows.value"
          :loading="paymentOrders.loading.value"
          :page="paymentOrders.page.value"
          :page-size="paymentOrders.pageSize.value"
          :total="paymentOrders.total.value"
          empty-text="暂无支付订单"
          @update:page="paymentOrders.setPage"
          @update:page-size="paymentOrders.setPageSize"
        />
      </NTabPane>

      <NTabPane
        name="transactions"
        tab="支付流水"
      >
        <StatusFilterBar
          :keyword="(paymentTransactions.filters.keyword as string) ?? ''"
          :loading="paymentTransactions.loading.value"
          search-placeholder="搜索支付单号 / 业务单号"
          @update:keyword="paymentTransactions.filters.keyword = $event"
          @apply="paymentTransactions.applyFilters({})"
          @reset="paymentTransactions.reset()"
        />
        <DataTable
          :columns="transactionColumns"
          :data="paymentTransactions.rows.value"
          :loading="paymentTransactions.loading.value"
          :page="paymentTransactions.page.value"
          :page-size="paymentTransactions.pageSize.value"
          :total="paymentTransactions.total.value"
          empty-text="暂无支付流水"
          @update:page="paymentTransactions.setPage"
          @update:page-size="paymentTransactions.setPageSize"
        />
      </NTabPane>

      <NTabPane
        name="refunds"
        tab="退款单"
      >
        <StatusFilterBar
          :status-options="toOptions(REFUND_STATUS)"
          :status="(refunds.filters.status as string) ?? null"
          :keyword="(refunds.filters.keyword as string) ?? ''"
          :loading="refunds.loading.value"
          search-placeholder="搜索退款单号 / 业务单号"
          @update:status="refunds.filters.status = $event ?? ''"
          @update:keyword="refunds.filters.keyword = $event"
          @apply="refunds.applyFilters({})"
          @reset="refunds.reset()"
        />
        <DataTable
          :columns="refundColumns"
          :data="refunds.rows.value"
          :loading="refunds.loading.value"
          :page="refunds.page.value"
          :page-size="refunds.pageSize.value"
          :total="refunds.total.value"
          empty-text="暂无退款单"
          @update:page="refunds.setPage"
          @update:page-size="refunds.setPageSize"
        />
      </NTabPane>
    </NTabs>

    <NModal
      v-model:show="refundPasswordShow"
      preset="card"
      title="验证门店管理员密码"
      class="payment-refund-password-modal"
      :mask-closable="!action.running.value"
      @after-leave="closeRefundPassword"
    >
      <p class="payment-refund-password-note">
        请输入本店门店管理员的登录密码。验证通过后，系统才会提交退款申请。
      </p>
      <div class="payment-refund-password-order">
        <span>{{ refundPaymentOrder?.paymentOrderNo ?? '—' }}</span>
        <strong>{{ formatCent(refundPaymentOrder?.amountCent) }}</strong>
      </div>
      <NForm label-placement="top">
        <NFormItem
          label="门店管理员登录密码"
          required
        >
          <NInput
            v-model:value="refundPassword"
            type="password"
            show-password-on="click"
            autocomplete="current-password"
            placeholder="请输入门店管理员登录密码"
            autofocus
            @keyup.enter="submitRefund"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton
            :disabled="action.running.value"
            @click="closeRefundPassword"
          >
            取消
          </NButton>
          <NButton
            type="error"
            :loading="action.running.value"
            @click="submitRefund"
          >
            验证并退款
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <NModal
      v-model:show="orderDetailShow"
      preset="card"
      title="支付订单详情"
      style="width: 560px"
    >
      <NSpin :show="orderDetailLoading">
        <div class="payment-detail__summary">
          <div>
            <span class="ic-muted">支付单号：</span>{{ currentPaymentOrder?.paymentOrderNo ?? '-' }}
          </div>
          <div>
            <span class="ic-muted">业务单号：</span>{{ currentPaymentOrder?.businessOrderNo ?? '-' }}
          </div>
          <div>
            <span class="ic-muted">业务类型：</span>{{ currentPaymentOrder?.orderType ?? '-' }}
          </div>
          <div>
            <span class="ic-muted">金额：</span>{{ formatCent(currentPaymentOrder?.amountCent) }}
          </div>
          <div>
            <span class="ic-muted">支付方式：</span>{{ currentPaymentOrder?.payMethod ?? '-' }}
          </div>
          <div>
            <span class="ic-muted">支付状态：</span>
            <StatusTag
              :dict="PAYMENT_STATUS"
              :value="currentPaymentOrder?.status"
            />
          </div>
          <div>
            <span class="ic-muted">创建时间：</span>{{ formatDateTime(currentPaymentOrder?.createdAt) }}
          </div>
          <div>
            <span class="ic-muted">支付时间：</span>{{ formatDateTime(currentPaymentOrder?.paidAt) }}
          </div>
        </div>
      </NSpin>
      <template #footer>
        <div class="payment-detail__footer">
          <NButton @click="orderDetailShow = false">
            关闭
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      v-model:show="transactionDetailShow"
      preset="card"
      title="支付流水详情"
      style="width: 560px"
    >
      <div class="payment-detail__summary">
        <div>
          <span class="ic-muted">支付单号：</span>{{ currentTransaction?.paymentOrderNo ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">业务单号：</span>{{ currentTransaction?.businessOrderNo ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">业务类型：</span>{{ currentTransaction?.orderType ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">金额：</span>{{ formatCent(currentTransaction?.amountCent) }}
        </div>
        <div>
          <span class="ic-muted">支付方式：</span>{{ currentTransaction?.payMethod ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">状态：</span>{{ currentTransaction?.status ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">时间：</span>{{ formatDateTime(currentTransaction?.createdAt) }}
        </div>
      </div>
      <template #footer>
        <div class="payment-detail__footer">
          <NButton @click="transactionDetailShow = false">
            关闭
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      v-model:show="refundDetailShow"
      preset="card"
      title="退款单详情"
      style="width: 560px"
    >
      <div class="payment-detail__summary">
        <div>
          <span class="ic-muted">退款单号：</span>{{ currentRefund?.refundOrderNo ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">支付单号：</span>{{ currentRefund?.paymentOrderId ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">业务单号：</span>{{ currentRefund?.businessOrderId ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">退款金额：</span>{{ formatCent(currentRefund?.amountCent) }}
        </div>
        <div>
          <span class="ic-muted">渠道：</span>{{ currentRefund?.channel ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">状态：</span>
          <StatusTag
            :dict="REFUND_STATUS"
            :value="currentRefund?.status"
          />
        </div>
        <div class="payment-detail__full">
          <span class="ic-muted">原因：</span>{{ currentRefund?.reason ?? '-' }}
        </div>
        <div>
          <span class="ic-muted">申请时间：</span>{{ formatDateTime(currentRefund?.createdAt) }}
        </div>
      </div>
      <template #footer>
        <div class="payment-detail__footer">
          <NButton @click="refundDetailShow = false">
            关闭
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
:global(.payment-refund-password-modal) {
  width: min(440px, calc(100vw - 32px));
}

.payment-refund-password-note {
  margin: 0 0 var(--ic-space-4);
  color: var(--ic-color-text-secondary);
  line-height: 1.6;
}

.payment-refund-password-order {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-3);
  padding-bottom: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
  color: var(--ic-color-text-secondary);
  font-variant-numeric: tabular-nums;
}

.payment-refund-password-order strong {
  color: var(--ic-color-danger);
}

.payment-detail__summary {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
.payment-detail__full {
  grid-column: 1 / -1;
}
.payment-detail__footer {
  display: flex;
  justify-content: flex-end;
}
</style>
