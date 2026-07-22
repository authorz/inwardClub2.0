<script setup lang="ts">
/**
 * 点餐订单处理：按状态机推进 confirm/prepare/ready/complete/cancel。
 * 可执行动作来自集中定义的 FOOD_ORDER_TRANSITIONS，不在页面里散写流转判断。
 */
import { computed, h } from 'vue'
import { NButton, NSpace, type DataTableColumns } from 'naive-ui'
import { orderService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import {
  FOOD_ORDER_STATUS,
  FOOD_ORDER_TRANSITIONS,
  PAY_CHANNEL,
  PAYMENT_STATUS,
  toOptions,
  type FoodOrderAction,
} from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { FoodOrder } from '@/types/models'

const list = useAsyncList<FoodOrder>((params) => orderService.foodOrders(params), {
  initialFilters: { orderStatus: '', keyword: '' },
})
const action = useAsyncAction()

function runAction(row: FoodOrder, t: FoodOrderAction) {
  void action.run(() => orderService.foodAction(row.id, t.action), {
    confirm: t.danger
      ? { content: `确认${t.label}订单 ${row.businessOrderId}？`, danger: true }
      : undefined,
    successMessage: `${t.label}成功`,
    onSuccess: () => list.refresh(),
  })
}

const columns = computed<DataTableColumns<FoodOrder>>(() => [
  textColumn<FoodOrder>('订单号', (r) => r.businessOrderId),
  textColumn<FoodOrder>('桌台', (r) => r.tableName, { width: 90 }),
  textColumn<FoodOrder>('餐品', (r) => r.itemsSummary),
  moneyColumn<FoodOrder>('金额', (r) => r.amountCent, { width: 110 }),
  statusColumn<FoodOrder>('支付', PAYMENT_STATUS, (r) => r.paymentStatus, { width: 100 }),
  statusColumn<FoodOrder>('渠道', PAY_CHANNEL, (r) => r.payChannel, { width: 84 }),
  statusColumn<FoodOrder>('履约状态', FOOD_ORDER_STATUS, (r) => r.status, { width: 100 }),
  dateColumn<FoodOrder>('下单时间', (r) => r.createdAt, { width: 150 }),
  {
    title: '处理',
    key: 'actions',
    width: 220,
    fixed: 'right',
    render: (row: FoodOrder) => {
      const transitions = FOOD_ORDER_TRANSITIONS[row.status] ?? []
      if (transitions.length === 0) return h('span', { class: 'ic-muted' }, '—')
      return h(
        NSpace,
        { size: 4 },
        {
          default: () =>
            transitions.map((t) =>
              h(
                PermissionButton,
                {
                  permissions: [PERM.orderStatusWrite],
                  type: t.danger ? 'error' : 'primary',
                  text: true,
                  // 履约状态机（POST /store/food-orders/:id/:action）后端尚未实现，暂禁用避免调用不存在的路由。
                  disabled: true,
                  onClick: () => runAction(row, t),
                },
                { default: () => t.label },
              ),
            ),
        },
      )
    },
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="点餐订单处理"
      description="点餐订单一览。履约流程（接单/备餐/取餐/完成）待后端支持，操作按钮暂不可用。"
    >
      <template #actions>
        <NButton
          :loading="list.loading.value"
          @click="list.refresh()"
        >
          刷新
        </NButton>
      </template>
    </PageHeader>

    <StatusFilterBar
      :status-options="toOptions(FOOD_ORDER_STATUS)"
      :status="(list.filters.orderStatus as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索订单号 / 桌台"
      @update:status="list.filters.orderStatus = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    />

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无点餐订单"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
  </div>
</template>
