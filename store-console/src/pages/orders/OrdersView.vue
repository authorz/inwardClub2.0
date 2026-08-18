<script setup lang="ts">
/**
 * 本店订单：统一订单读模型，支持类型/支付状态/关键字筛选。
 * 门店范围来自 token scope，不传 storeId。
 */
import { computed, h } from 'vue'
import { NAvatar, NButton, NInput, NSelect, NSpace, type DataTableColumns } from 'naive-ui'
import { orderService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { ORDER_TYPE, PAYMENT_STATUS, PAY_CHANNEL, toOptions } from '@/constants/enums'
import { dateColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader } from '@/components/common'
import type { StoreOrder } from '@/types/models'

// 退款申请统一在「支付与退款」页按支付单发起（服务端退款以 paymentOrderId 为键，
// 本店订单读模型不含该字段），本页为只读订单总览。
const list = useAsyncList<StoreOrder>((params) => orderService.list(params), {
  initialFilters: { orderType: '', paymentStatus: '', keyword: '' },
})

const columns = computed<DataTableColumns<StoreOrder>>(() => [
  textColumn<StoreOrder>('ID', (r) => r.id, { width: 72 }),
  {
    title: '用户信息',
    key: 'member',
    width: 210,
    render: (row) => {
      const nickname = row.memberNickname?.trim() || '未关联会员'
      const phone = row.memberPhone?.trim() || row.memberPhoneMasked || '暂无手机号'
      const fallback = () => nickname.slice(0, 1)
      return h('div', { class: 'order-member' }, [
        h(
          NAvatar,
          {
            class: 'order-member__avatar',
            round: true,
            size: 36,
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
  },
  textColumn<StoreOrder>('订单号', (r) => r.orderNo, { width: 190 }),
  statusColumn<StoreOrder>('类型', ORDER_TYPE, (r) => r.orderType, { width: 96 }),
  moneyColumn<StoreOrder>('金额', (r) => r.amountCent, { width: 120 }),
  statusColumn<StoreOrder>('支付状态', PAYMENT_STATUS, (r) => r.paymentStatus, { width: 110 }),
  statusColumn<StoreOrder>('渠道', PAY_CHANNEL, (r) => r.payChannel, { width: 90 }),
  dateColumn<StoreOrder>('下单时间', (r) => r.createdAt, { width: 150 }),
])

const typeOptions = toOptions(ORDER_TYPE).map(({ label, value }) => ({ label, value }))
const paymentStatusOptions = toOptions(PAYMENT_STATUS).map(({ label, value }) => ({ label, value }))
</script>

<template>
  <section class="orders">
    <PageHeader
      title="本店订单"
      description="本店全部业务订单统一视图"
    />

    <div class="order-filters">
      <div class="order-filters__fields">
        <label class="order-filter">
          <span>订单类型</span>
          <NSelect
            :value="(list.filters.orderType as string) || null"
            :options="typeOptions"
            placeholder="全部"
            clearable
            @update:value="list.filters.orderType = $event ?? ''"
          />
        </label>
        <label class="order-filter">
          <span>支付状态</span>
          <NSelect
            :value="(list.filters.paymentStatus as string) || null"
            :options="paymentStatusOptions"
            placeholder="全部"
            clearable
            @update:value="list.filters.paymentStatus = $event ?? ''"
          />
        </label>
        <label class="order-filter order-filter--search">
          <span>订单 / 用户</span>
          <NInput
            :value="(list.filters.keyword as string) ?? ''"
            placeholder="搜索订单号 / 会员 / 手机号"
            clearable
            @update:value="list.filters.keyword = $event"
            @keyup.enter="list.applyFilters({})"
          />
        </label>
      </div>
      <NSpace class="order-filters__actions">
        <NButton
          type="primary"
          size="small"
          :loading="list.loading.value"
          @click="list.applyFilters({})"
        >
          查询
        </NButton>
        <NButton
          size="small"
          @click="list.reset()"
        >
          重置
        </NButton>
      </NSpace>
    </div>

    <div class="order-table">
      <DataTable
        :columns="columns"
        :data="list.rows.value"
        :loading="list.loading.value"
        :page="list.page.value"
        :page-size="list.pageSize.value"
        :total="list.total.value"
        :scroll-x="1040"
        empty-text="暂无订单"
        @update:page="list.setPage"
        @update:page-size="list.setPageSize"
      />
    </div>
  </section>
</template>

<style scoped>
.orders {
  max-width: 1400px;
}

.order-filters {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ic-space-4);
  align-items: flex-end;
  justify-content: space-between;
  padding: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
  border: var(--ic-divider);
  border-radius: var(--ic-radius-md);
  background: var(--ic-color-surface);
}

.order-filters__fields {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  gap: var(--ic-space-4);
}

.order-filter {
  width: 160px;
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-1);
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.order-filter--search {
  width: 260px;
}

.order-filters__actions {
  flex-shrink: 0;
}

:deep(.order-member) {
  display: flex;
  gap: var(--ic-space-2);
  align-items: center;
  min-width: 0;
  padding: 2px 0;
}

:deep(.order-member__avatar) {
  flex: none;
}

:deep(.order-member__details) {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

:deep(.order-member__nickname) {
  overflow: hidden;
  color: var(--ic-color-text);
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:deep(.order-member__phone) {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}

.order-table {
  padding: var(--ic-space-2);
  border: var(--ic-divider);
  border-radius: var(--ic-radius-md);
  background: var(--ic-color-surface);
}

@media (max-width: 720px) {
  .order-filter,
  .order-filter--search {
    width: 100%;
  }

  .order-filters__actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
