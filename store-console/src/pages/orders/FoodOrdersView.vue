<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import {
  NAvatar, NButton, NDatePicker, NDescriptions, NDescriptionsItem, NForm, NFormItem,
  NImage, NInput, NModal, NSelect, NSpace, NTag, type DataTableColumns, type SelectOption,
} from 'naive-ui'
import { orderService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { FOOD_ORDER_STATUS, PAYMENT_STATUS, PAY_CHANNEL, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { formatCent } from '@/utils/format'
import { DataTable, PageHeader, PermissionButton } from '@/components/common'
import type { FoodOrder } from '@/types/models'

const list = useAsyncList<FoodOrder>((params) => orderService.foodOrders(params), {
  initialFilters: {
    paymentStatus: '', payChannel: '', memberNickname: '',
    memberPhone: '', orderNo: '', itemName: '', createdFrom: '', createdTo: '',
  },
})
const createdRange = ref<[number, number] | null>(null)
const action = useAsyncAction()
const detailShow = ref(false)
const current = ref<FoodOrder | null>(null)
const forceShow = ref(false)
const forceTarget = ref<FoodOrder | null>(null)
const forceForm = reactive({ password: '' })

function selectOptions(items: Array<{ label: string; value: string }>): SelectOption[] {
  return items.map(({ label, value }) => ({ label, value }))
}

const foodPaymentStatusOptions = selectOptions(toOptions(PAYMENT_STATUS).filter((item) =>
  ['unpaid', 'paid', 'refunded', 'partially_refunded'].includes(item.value),
))
const foodPayChannelOptions = selectOptions(toOptions(PAY_CHANNEL).filter((item) =>
  ['wechat', 'coin'].includes(item.value),
))

function applyFilters(): void {
  const [from, to] = createdRange.value ?? []
  const end = to == null ? null : new Date(to)
  if (end) end.setHours(23, 59, 59, 999)
  list.applyFilters({
    createdFrom: from == null ? '' : new Date(from).toISOString(),
    createdTo: end?.toISOString() ?? '',
  })
}

function resetFilters(): void {
  createdRange.value = null
  list.reset()
}

function payChannelLabel(channel: FoodOrder['payChannel']): string {
  return channel ? PAY_CHANNEL[channel].label : '—'
}

async function openDetail(row: FoodOrder): Promise<void> {
  current.value = await orderService.foodOrderDetail(row.id)
  detailShow.value = true
}

function cancelOrder(row: FoodOrder): void {
  void action.run(() => orderService.foodAction(row.id, 'cancel'), {
    confirm: {
      content: `确认取消订单 ${row.businessOrderId}？款项将原路退回，已赠送的 ${row.pointsEarned} 积分将同时扣回。`,
      danger: true,
    },
    successMessage: '订单已取消，款项已原路退回',
    onSuccess: () => list.refresh(),
  })
}

function openForceCancel(row: FoodOrder): void {
  forceTarget.value = row
  forceForm.password = ''
  forceShow.value = true
}

function forceCancel(): void {
  const row = forceTarget.value
  if (!row || !forceForm.password) return
  void action.run(
    () => orderService.foodAction(row.id, 'force-cancel', { password: forceForm.password }),
    {
      successMessage: '订单已强制取消，款项已原路退回',
      onSuccess: () => { forceShow.value = false; list.refresh() },
    },
  )
}

const columns = computed<DataTableColumns<FoodOrder>>(() => [
  textColumn<FoodOrder>('ID', (r) => r.id, { width: 60 }),
  {
    title: '用户信息', key: 'member', width: 170,
    render: (row) => h('div', { class: 'member' }, [
      h(NAvatar, { size: 34, round: true, src: row.memberAvatarUrl || undefined }, { default: () => (row.memberNickname || '用户').slice(0, 1) }),
      h('div', { class: 'member__text' }, [
        h('strong', { title: row.memberNickname || '' }, row.memberNickname || '未设置昵称'),
        h('span', row.memberPhone || '未绑定手机号'),
      ]),
    ]),
  },
  textColumn<FoodOrder>('订单号', (r) => r.businessOrderId, { width: 180, ellipsis: { tooltip: true } }),
  {
    title: '餐品', key: 'itemsSummary', width: 200, ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'items-summary', title: row.itemsSummary || '' }, row.itemsSummary || '—'),
  },
  moneyColumn<FoodOrder>('订单金额', (r) => r.amountCent, { width: 90 }),
  moneyColumn<FoodOrder>('支付金额', (r) => r.paidAmountCent, { width: 90 }),
  statusColumn<FoodOrder>('支付状态', PAYMENT_STATUS, (r) => r.paymentStatus, { width: 90 }),
  statusColumn<FoodOrder>('支付方式', PAY_CHANNEL, (r) => r.payChannel, { width: 80 }),
  textColumn<FoodOrder>('赠送积分', (r) => r.pointsEarned, { width: 80, align: 'right' }),
  statusColumn<FoodOrder>('履约状态', FOOD_ORDER_STATUS, (r) => r.status, { width: 90 }),
  dateColumn<FoodOrder>('下单时间', (r) => r.createdAt, { width: 145 }),
  {
    title: '操作', key: 'actions', width: 180, fixed: 'right',
    render: (row) => h(NSpace, { size: 4 }, {
      default: () => [
        h(NButton, { text: true, onClick: () => openDetail(row) }, { default: () => '详情' }),
        row.paymentStatus === 'paid' && row.status !== 'cancelled'
          ? h(PermissionButton, { permissions: [PERM.orderStatusWrite], text: true, type: 'error', onClick: () => cancelOrder(row) }, { default: () => '取消订单' })
          : null,
        row.paymentStatus === 'paid' && row.status !== 'cancelled' && row.pointsEarned > 0
          ? h(PermissionButton, { permissions: [PERM.orderStatusWrite], text: true, type: 'error', onClick: () => openForceCancel(row) }, { default: () => '强制取消' })
          : null,
      ],
    }),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="点餐订单"
      description="查看本店点餐订单、支付与赠送积分；已支付订单默认完成履约"
    />
    <section class="order-filters ic-band">
      <div class="filter-grid">
        <label class="filter-field filter-field--select"><span>支付状态</span><NSelect
          :value="(list.filters.paymentStatus as string) || null"
          :options="foodPaymentStatusOptions"
          clearable
          placeholder="全部"
          @update:value="list.filters.paymentStatus = $event ?? ''"
        /></label>
        <label class="filter-field filter-field--select"><span>支付方式</span><NSelect
          :value="(list.filters.payChannel as string) || null"
          :options="foodPayChannelOptions"
          clearable
          placeholder="全部"
          @update:value="list.filters.payChannel = $event ?? ''"
        /></label>
        <label class="filter-field filter-field--input"><span>会员昵称</span><NInput
          :value="list.filters.memberNickname as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="list.filters.memberNickname = $event"
          @keyup.enter="applyFilters"
        /></label>
        <label class="filter-field filter-field--input"><span>会员手机号</span><NInput
          :value="list.filters.memberPhone as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="list.filters.memberPhone = $event"
          @keyup.enter="applyFilters"
        /></label>
        <label class="filter-field filter-field--input"><span>订单号</span><NInput
          :value="list.filters.orderNo as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="list.filters.orderNo = $event"
          @keyup.enter="applyFilters"
        /></label>
        <label class="filter-field filter-field--input"><span>餐品名称</span><NInput
          :value="list.filters.itemName as string"
          clearable
          placeholder="支持模糊搜索"
          @update:value="list.filters.itemName = $event"
          @keyup.enter="applyFilters"
        /></label>
        <label class="filter-field filter-field--date"><span>下单时间</span><NDatePicker
          v-model:value="createdRange"
          type="daterange"
          clearable
          format="yyyy-MM-dd"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
        /></label>
      </div>
      <NSpace class="filter-actions">
        <NButton
          type="primary"
          size="small"
          :loading="list.loading.value"
          @click="applyFilters"
        >
          查询
        </NButton>
        <NButton
          size="small"
          @click="resetFilters"
        >
          重置
        </NButton>
      </NSpace>
    </section>
    <section class="table-panel ic-band">
      <DataTable
        :columns="columns"
        :data="list.rows.value"
        :loading="list.loading.value"
        :page="list.page.value"
        :page-size="list.pageSize.value"
        :total="list.total.value"
        :scroll-x="1455"
        empty-text="暂无点餐订单"
        @update:page="list.setPage"
        @update:page-size="list.setPageSize"
      />
    </section>

    <NModal
      v-model:show="detailShow"
      preset="card"
      title="点餐订单详情"
      style="width: min(720px, calc(100vw - 32px))"
    >
      <NDescriptions
        v-if="current"
        :column="2"
        bordered
        label-placement="left"
      >
        <NDescriptionsItem label="订单 ID">
          {{ current.id }}
        </NDescriptionsItem>
        <NDescriptionsItem label="订单号">
          {{ current.businessOrderId }}
        </NDescriptionsItem>
        <NDescriptionsItem label="会员">
          {{ current.memberNickname || '未设置昵称' }}
        </NDescriptionsItem>
        <NDescriptionsItem label="手机号">
          {{ current.memberPhone || '未绑定' }}
        </NDescriptionsItem>
        <NDescriptionsItem label="订单金额">
          {{ formatCent(current.amountCent) }}
        </NDescriptionsItem>
        <NDescriptionsItem label="实际支付">
          {{ formatCent(current.paidAmountCent) }}
        </NDescriptionsItem>
        <NDescriptionsItem label="支付方式">
          {{ payChannelLabel(current.payChannel) }}
        </NDescriptionsItem>
        <NDescriptionsItem label="赠送积分">
          {{ current.pointsEarned }}
        </NDescriptionsItem>
      </NDescriptions>
      <div
        v-if="current"
        class="detail-items"
      >
        <div class="detail-items__head">
          <strong>餐品明细</strong><span>共 {{ current.items.length }} 项</span>
        </div>
        <div
          v-for="item in current.items"
          :key="item.id"
          class="detail-item"
        >
          <NImage
            v-if="item.imageUrl"
            class="detail-item__image"
            :src="item.imageUrl"
            :alt="item.name"
            width="56"
            height="56"
            object-fit="cover"
          />
          <div
            v-else
            class="detail-item__placeholder"
          >
            暂无图片
          </div>
          <div class="detail-item__info">
            <strong>{{ item.name }}</strong>
            <span>单价 {{ formatCent(item.unitPriceCent) }} · 赠送 {{ item.pointsReward * item.quantity }} 积分</span>
          </div>
          <span>×{{ item.quantity }}</span><strong>{{ formatCent(item.subtotalCent) }}</strong>
        </div>
      </div>
    </NModal>

    <NModal
      v-model:show="forceShow"
      preset="card"
      title="强制取消订单"
      style="width: min(480px, calc(100vw - 32px))"
    >
      <div class="force-warning">
        用户积分不足时，系统将扣除其当前可用积分并记录未追回积分；款项仍按原支付方式全额退回。此操作必须验证当前登录密码。
      </div>
      <NForm label-placement="top">
        <NFormItem label="订单号">
          <NTag>{{ forceTarget?.businessOrderId }}</NTag>
        </NFormItem>
        <NFormItem
          label="管理员登录密码"
          required
        >
          <NInput
            v-model:value="forceForm.password"
            type="password"
            show-password-on="click"
            placeholder="请输入当前登录密码"
            @keyup.enter="forceCancel"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="forceShow = false">
            取消
          </NButton><NButton
            type="error"
            :disabled="!forceForm.password"
            :loading="action.running.value"
            @click="forceCancel"
          >
            确认强制取消
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.order-filters { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: var(--ic-space-4); margin-bottom: var(--ic-space-5); padding: var(--ic-space-4); }
.filter-grid { display: flex; flex-wrap: wrap; gap: var(--ic-space-4); }
.filter-field { display: flex; min-width: 0; flex-direction: column; gap: var(--ic-space-1); }
.filter-field > span { color: var(--ic-color-text-secondary); font-size: var(--ic-font-xs); }
.filter-field--select { width: 160px; }
.filter-field--input { width: 180px; }
.filter-field--date { width: 260px; }
.filter-actions { flex-shrink: 0; }
.table-panel { padding: var(--ic-space-3); overflow: hidden; }
:deep(.member) { display: flex; align-items: center; gap: 10px; min-width: 0; padding: 2px 0; }
:deep(.member .n-avatar) { flex: none; }
:deep(.member__text) { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
:deep(.member__text strong), :deep(.member__text span), :deep(.items-summary) { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
:deep(.member__text strong) { line-height: 20px; }
:deep(.member__text span) { color: var(--ic-color-text-secondary); font-size: var(--ic-font-xs); line-height: 18px; font-variant-numeric: tabular-nums; }
.detail-items { margin-top: var(--ic-space-5); }
.detail-items__head, .detail-item { display: flex; align-items: center; justify-content: space-between; gap: var(--ic-space-4); }
.detail-items__head { padding-bottom: var(--ic-space-3); border-bottom: var(--ic-divider); }
.detail-items__head span, .detail-item span { color: var(--ic-color-text-secondary); }
.detail-item { padding: var(--ic-space-3) 0; border-bottom: var(--ic-divider); }
.detail-item__image, .detail-item__placeholder { width: 56px; height: 56px; flex: none; border-radius: var(--ic-radius-sm); }
.detail-item__placeholder { display: grid; place-items: center; background: var(--ic-color-surface-muted); color: var(--ic-color-text-tertiary); font-size: 11px; }
.detail-item__info { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 3px; }
.force-warning { margin-bottom: var(--ic-space-4); padding: var(--ic-space-3); border-radius: var(--ic-radius-md); color: var(--ic-color-danger); background: #fff5f5; line-height: 1.6; }
@media (max-width: 720px) {
  .filter-grid, .filter-field, .filter-field--select, .filter-field--input, .filter-field--date { width: 100%; }
}
</style>
