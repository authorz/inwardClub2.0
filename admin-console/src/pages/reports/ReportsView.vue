<script setup lang="ts">
/**
 * 报表。
 * 按报表分项（总览 / 收款 / 商品 / 活动 / 券 / 核销记录 / 会员 / 预约 / 门店）分标签；
 * 总览为汇总指标卡片，其余分项为分页列表，均对接 /admin/reports/*。
 */
import { onMounted, ref } from 'vue'
import { NEmpty, NGrid, NGridItem, NTabPane, NTabs } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import ResourceListView from '@/components/ResourceListView.vue'
import { moneyColumn, textColumn } from '@/utils/columns'
import { formatCent } from '@/utils/format'
import { reportService } from '@/api/services'
import type {
  ActivityReportRow,
  CatalogItemReportRow,
  CouponReportRow,
  MemberReportRow,
  RecordReportRow,
  ReportOverview,
  ReservationReportRow,
  RevenueReportRow,
  StoreReportRow,
} from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import type { NormalizedError } from '@/api/types'

const tab = ref('overview')
const reportTabs = [
  { key: 'overview', label: '总览' },
  { key: 'revenue', label: '收款' },
  { key: 'catalog', label: '商品' },
  { key: 'activities', label: '活动' },
  { key: 'coupons', label: '券' },
  { key: 'records', label: '核销记录' },
  { key: 'members', label: '会员' },
  { key: 'reservations', label: '预约' },
  { key: 'stores', label: '门店' },
]

const rangeFields: FilterField[] = [{ key: 'created', label: '时间范围', type: 'daterange' }]

interface OverviewMetric {
  key: keyof ReportOverview
  label: string
  value: string
}

const overviewMetrics = ref<OverviewMetric[]>([
  { key: 'storeCount', label: '门店总数', value: '—' },
  { key: 'memberCount', label: '会员总数', value: '—' },
  { key: 'orderCount', label: '订单总数', value: '—' },
  { key: 'grossSalesCent', label: '订单流水', value: '—' },
  { key: 'couponsIssued', label: '发放券数', value: '—' },
  { key: 'couponsRedeemed', label: '已核销券数', value: '—' },
])
const overviewErrored = ref(false)

onMounted(async () => {
  try {
    const data = await reportService.overview()
    if (data) {
      overviewMetrics.value = overviewMetrics.value.map((m) => ({
        ...m,
        value: m.key === 'grossSalesCent' ? formatCent(data[m.key]) : String(data[m.key] ?? '—'),
      }))
    }
  } catch (e) {
    overviewErrored.value = (e as NormalizedError).status !== 401
  }
})

const revenueColumns = [
  textColumn<RevenueReportRow>('日期', 'date', { width: 140 }),
  textColumn<RevenueReportRow>('订单数', 'orderCount', { width: 120 }),
  moneyColumn<RevenueReportRow>('流水', 'grossCent'),
]

const catalogColumns = [
  textColumn<CatalogItemReportRow>('商品', 'itemName', { width: 220 }),
  textColumn<CatalogItemReportRow>('销量', 'soldQty', { width: 120 }),
  moneyColumn<CatalogItemReportRow>('流水', 'grossCent'),
]

const activityColumns = [
  textColumn<ActivityReportRow>('活动', 'activityName', { width: 220 }),
  textColumn<ActivityReportRow>('订单数', 'orderCount', { width: 120 }),
  textColumn<ActivityReportRow>('票数', 'ticketCount', { width: 120 }),
]

const couponColumns = [
  textColumn<CouponReportRow>('券模板', 'name', { width: 220 }),
  textColumn<CouponReportRow>('发放数', 'issued', { width: 120 }),
  textColumn<CouponReportRow>('核销数', 'redeemed', { width: 120 }),
]

const recordColumns = [
  textColumn<RecordReportRow>('类型', 'kind', { width: 140 }),
  textColumn<RecordReportRow>('核销时间', 'createdAt', { width: 180 }),
]

const memberColumns = [
  textColumn<MemberReportRow>('会员', 'memberId', { width: 220 }),
  textColumn<MemberReportRow>('积分余额', 'pointsBalance', { width: 140 }),
  textColumn<MemberReportRow>('订单数', 'orderCount', { width: 120 }),
]

const reservationColumns = [
  textColumn<ReservationReportRow>('日期', 'date', { width: 140 }),
  textColumn<ReservationReportRow>('预约数', 'count', { width: 120 }),
]

const storeColumns = [
  textColumn<StoreReportRow>('门店', 'storeName', { width: 220 }),
  textColumn<StoreReportRow>('订单数', 'orderCount', { width: 120 }),
  moneyColumn<StoreReportRow>('流水', 'grossCent'),
]
</script>

<template>
  <section>
    <PageHeader
      title="报表"
      description="总部经营报表分项（按权限隔离）"
      :breadcrumb="['报表']"
    />
    <div class="report-panel">
      <NTabs
        v-model:value="tab"
        type="line"
      >
        <NTabPane
          v-for="t in reportTabs"
          :key="t.key"
          :name="t.key"
          :tab="t.label"
        >
          <template v-if="t.key === 'overview'">
            <NGrid
              :cols="3"
              :x-gap="16"
              :y-gap="16"
              responsive="screen"
            >
              <NGridItem
                v-for="m in overviewMetrics"
                :key="m.key"
              >
                <div class="metric">
                  <div class="metric__value">
                    {{ m.value }}
                  </div>
                  <div class="metric__label">
                    {{ m.label }}
                  </div>
                </div>
              </NGridItem>
            </NGrid>
            <NEmpty
              v-if="overviewErrored"
              description="经营总览接口暂不可用（/admin/reports/overview）"
              class="report-empty"
            />
          </template>

          <ResourceListView
            v-else-if="t.key === 'revenue'"
            title="收款报表"
            :fields="rangeFields"
            :columns="revenueColumns"
            :fetcher="reportService.revenue"
          />
          <ResourceListView
            v-else-if="t.key === 'catalog'"
            title="商品销量报表"
            :fields="rangeFields"
            :columns="catalogColumns"
            :fetcher="reportService.catalogItems"
          />
          <ResourceListView
            v-else-if="t.key === 'activities'"
            title="活动报表"
            :fields="rangeFields"
            :columns="activityColumns"
            :fetcher="reportService.activities"
          />
          <ResourceListView
            v-else-if="t.key === 'coupons'"
            title="券报表"
            :fields="rangeFields"
            :columns="couponColumns"
            :fetcher="reportService.coupons"
          />
          <ResourceListView
            v-else-if="t.key === 'records'"
            title="核销记录"
            :fields="rangeFields"
            :columns="recordColumns"
            :fetcher="reportService.records"
          />
          <ResourceListView
            v-else-if="t.key === 'members'"
            title="会员报表"
            :fields="rangeFields"
            :columns="memberColumns"
            :fetcher="reportService.members"
          />
          <ResourceListView
            v-else-if="t.key === 'reservations'"
            title="预约报表"
            :fields="rangeFields"
            :columns="reservationColumns"
            :fetcher="reportService.reservations"
          />
          <ResourceListView
            v-else-if="t.key === 'stores'"
            title="门店经营报表"
            :fields="rangeFields"
            :columns="storeColumns"
            :fetcher="reportService.stores"
          />
        </NTabPane>
      </NTabs>
    </div>
  </section>
</template>

<style scoped>
.report-panel {
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
  padding: var(--ic-space-md) var(--ic-space-lg);
}
.report-empty {
  padding: var(--ic-space-xl) 0;
}
.metric {
  padding: var(--ic-space-lg);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
}
.metric__value {
  font-size: var(--ic-font-2xl);
  font-weight: 700;
}
.metric__label {
  margin-top: var(--ic-space-xs);
  font-size: var(--ic-font-sm);
  color: var(--ic-color-text-secondary);
}
</style>
