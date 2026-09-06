<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { NAlert, NEmpty, NSelect, NSpin, NTabPane, NTabs } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import ResourceListView from '@/components/ResourceListView.vue'
import { moneyColumn, renderColumn, textColumn } from '@/utils/columns'
import { formatCent } from '@/utils/format'
import { reportService, storeService } from '@/api/services'
import type {
  ActivityReportRow,
  CatalogItemReportRow,
  CouponReportRow,
  MemberReportRow,
  RecordReportRow,
  ReportOverview,
  ReportOverviewBreakdown,
  ReportOverviewTrendPoint,
  ReservationReportRow,
  RevenueReportRow,
  StoreReportRow,
} from '@/api/models'
import type { ListQuery, NormalizedError } from '@/api/types'
import type { FilterField } from '@/components/ui-types'

const tab = ref('overview')
const selectedStoreId = ref<string | null>(null)
const storeOptions = ref<Array<{ label: string; value: string }>>([])
const storesLoading = ref(false)
const storesError = ref('')
const overview = ref<ReportOverview | null>(null)
const overviewLoading = ref(false)
const overviewError = ref('')

const reportTabs = [
  { key: 'overview', label: '总览' },
  { key: 'revenue', label: '收款趋势' },
  { key: 'catalog', label: '商品销售' },
  { key: 'activities', label: '活动经营' },
  { key: 'coupons', label: '券效率' },
  { key: 'records', label: '核销记录' },
  { key: 'members', label: '会员消费' },
  { key: 'reservations', label: '预约趋势' },
  { key: 'stores', label: '门店对比' },
]
const reportOptions = reportTabs.map((item) => ({ label: item.label, value: item.key }))

function reportWidth(columns: Array<{ width?: string | number }>): number {
  return columns.reduce((width, column) => width + Number(column.width ?? 120), 0)
}

const rangeFields: FilterField[] = [
  { key: 'created', label: '时间范围', type: 'daterange', width: 280, mobileNative: true },
]

const selectedStoreName = computed(
  () => storeOptions.value.find((item) => item.value === selectedStoreId.value)?.label ?? '全部门店',
)
const reportScopeKey = computed(() => selectedStoreId.value ?? 'all')
const countFormatter = new Intl.NumberFormat('zh-CN')

function formatCount(value: number | null | undefined): string {
  return value == null ? '—' : countFormatter.format(value)
}

function formatCoins(value: number | null | undefined): string {
  return value == null ? '—' : `${countFormatter.format(value)} 金币`
}

function formatAverage(grossCent: number, orderCount: number): string {
  return formatCent(orderCount > 0 ? Math.round(grossCent / orderCount) : 0)
}

function formatRate(numerator: number, denominator: number): string {
  if (denominator <= 0) return '0.0%'
  return `${((numerator / denominator) * 100).toFixed(1)}%`
}

function scopedQuery(query?: Record<string, unknown>): Record<string, unknown> | undefined {
  if (!selectedStoreId.value) return query
  return { ...query, storeId: selectedStoreId.value }
}

async function loadStores(): Promise<void> {
  storesLoading.value = true
  storesError.value = ''
  try {
    const result = await storeService.list({ page: 1, pageSize: 100 })
    storeOptions.value = result.items.map((store) => ({
      label: store.name,
      value: String(store.id),
    }))
  } catch (error) {
    storesError.value = (error as NormalizedError).message || '门店列表加载失败'
  } finally {
    storesLoading.value = false
  }
}

async function loadOverview(): Promise<void> {
  overviewLoading.value = true
  overviewError.value = ''
  try {
    overview.value = await reportService.overview(scopedQuery())
  } catch (error) {
    overviewError.value = (error as NormalizedError).message || '经营总览加载失败'
  } finally {
    overviewLoading.value = false
  }
}

watch(selectedStoreId, loadOverview)
onMounted(() => {
  void Promise.all([loadStores(), loadOverview()])
})

const summaryMetrics = computed(() => {
  const data = overview.value
  return [
    { key: 'gross', label: '累计订单流水', value: data ? formatCent(data.grossSalesCent) : '—' },
    { key: 'orders', label: '累计已支付订单', value: formatCount(data?.orderCount) },
    {
      key: 'average',
      label: '平均订单金额',
      value: data ? formatAverage(data.grossSalesCent, data.orderCount) : '—',
    },
    {
      key: 'members',
      label: selectedStoreId.value ? '累计消费会员' : '平台注册会员',
      value: formatCount(data?.memberCount),
    },
    { key: 'wechat', label: '微信实收', value: data ? formatCent(data.wechatRevenue.total) : '—' },
    { key: 'coins', label: '金币消费', value: formatCoins(data?.coinConsumption.total) },
    {
      key: 'offline',
      label: '线下收款流水',
      value: data ? formatCent(data.offlineCollectionRevenueCent) : '—',
    },
    {
      key: 'couponRate',
      label: '券核销率',
      value: data ? formatRate(data.couponsRedeemed, data.couponsIssued) : '—',
    },
  ]
})

const todayMetrics = computed(() => {
  const data = overview.value
  return [
    { key: 'todayGross', label: '今日流水', value: data ? formatCent(data.todayGrossSalesCent) : '—' },
    { key: 'todayOrders', label: '今日已支付订单', value: formatCount(data?.todayOrderCount) },
    {
      key: 'todayMembers',
      label: selectedStoreId.value ? '今日新增消费会员' : '今日新注册会员',
      value: formatCount(data?.todayNewMemberCount),
    },
    {
      key: 'todayActivity',
      label: '今日活动流水',
      value: data ? formatCent(data.todayActivityRevenueCent) : '—',
    },
  ]
})

function breakdownRows(data: ReportOverviewBreakdown | undefined, includeRecharge: boolean) {
  const rows = includeRecharge
    ? [{ key: 'recharge', label: '会员充值', total: data?.recharge ?? 0, today: data?.todayRecharge ?? 0 }]
    : []
  rows.push(
    { key: 'food', label: '餐品订单', total: data?.food ?? 0, today: data?.todayFood ?? 0 },
    { key: 'activity', label: '活动订单', total: data?.activity ?? 0, today: data?.todayActivity ?? 0 },
  )
  return rows
}

const wechatRows = computed(() => breakdownRows(overview.value?.wechatRevenue, true))
const coinRows = computed(() => breakdownRows(overview.value?.coinConsumption, false))
const trend = computed<ReportOverviewTrendPoint[]>(() => overview.value?.trend ?? [])
const trendMax = computed(() => Math.max(0, ...trend.value.map((item) => item.wechatRevenueCent)))
const trendBars = computed(() =>
  trend.value.map((item) => ({
    ...item,
    height:
      trendMax.value === 0
        ? 0
        : Math.max(5, Math.round((item.wechatRevenueCent / trendMax.value) * 100)),
  })),
)

const revenueFetcher = (query: ListQuery) => reportService.revenue(scopedQuery(query))
const catalogFetcher = (query: ListQuery) => reportService.catalogItems(scopedQuery(query))
const activityFetcher = (query: ListQuery) => reportService.activities(scopedQuery(query))
const couponFetcher = (query: ListQuery) => reportService.coupons(scopedQuery(query))
const recordFetcher = (query: ListQuery) => reportService.records(scopedQuery(query))
const memberFetcher = (query: ListQuery) => reportService.members(scopedQuery(query))
const reservationFetcher = (query: ListQuery) => reportService.reservations(scopedQuery(query))
const storeFetcher = (query: ListQuery) => reportService.stores(scopedQuery(query))

const revenueColumns = [
  textColumn<RevenueReportRow>('日期', 'date', { width: 140 }),
  textColumn<RevenueReportRow>('已支付订单', 'orderCount', { width: 120 }),
  textColumn<RevenueReportRow>('微信订单', 'wechatOrderCount', { width: 110 }),
  moneyColumn<RevenueReportRow>('微信实收', 'wechatRevenueCent'),
  textColumn<RevenueReportRow>('金币订单', 'coinOrderCount', { width: 110 }),
  renderColumn<RevenueReportRow>('金币消费', 'coinConsumption', (row) => formatCoins(row.coinConsumption), 140),
]

const catalogColumns = [
  textColumn<CatalogItemReportRow>('商品 ID', 'itemId', { width: 110 }),
  textColumn<CatalogItemReportRow>('商品', 'itemName', { width: 220 }),
  textColumn<CatalogItemReportRow>('销量', 'soldQty', { width: 120 }),
  moneyColumn<CatalogItemReportRow>('流水', 'grossCent'),
]

const activityColumns = [
  textColumn<ActivityReportRow>('活动 ID', 'activityId', { width: 110 }),
  textColumn<ActivityReportRow>('活动', 'activityName', { width: 220 }),
  textColumn<ActivityReportRow>('订单数', 'orderCount', { width: 120 }),
  textColumn<ActivityReportRow>('票数', 'ticketCount', { width: 120 }),
]

const couponColumns = [
  textColumn<CouponReportRow>('券模板 ID', 'templateId', { width: 120 }),
  textColumn<CouponReportRow>('券模板', 'name', { width: 220 }),
  textColumn<CouponReportRow>('发放数', 'issued', { width: 120 }),
  textColumn<CouponReportRow>('核销数', 'redeemed', { width: 120 }),
  renderColumn<CouponReportRow>(
    '核销率',
    'redemptionRate',
    (row) => formatRate(row.redeemed, row.issued),
    120,
  ),
]

const recordColumns = [
  textColumn<RecordReportRow>('记录 ID', 'id', { width: 120 }),
  textColumn<RecordReportRow>('类型', 'kind', { width: 140 }),
  textColumn<RecordReportRow>('核销时间', 'createdAt', { width: 180 }),
]

const memberColumns = [
  textColumn<MemberReportRow>('会员 ID', 'memberId', { width: 160 }),
  textColumn<MemberReportRow>('积分余额', 'pointsBalance', { width: 140 }),
  textColumn<MemberReportRow>('订单数', 'orderCount', { width: 120 }),
]

const reservationColumns = [
  textColumn<ReservationReportRow>('日期', 'date', { width: 140 }),
  textColumn<ReservationReportRow>('预约数', 'count', { width: 120 }),
]

const storeColumns = [
  textColumn<StoreReportRow>('门店 ID', 'storeId', { width: 100 }),
  textColumn<StoreReportRow>('门店', 'storeName', { width: 180, fixed: 'left' }),
  textColumn<StoreReportRow>('总订单', 'orderCount', { width: 100 }),
  textColumn<StoreReportRow>('已支付订单', 'paidOrderCount', { width: 110 }),
  textColumn<StoreReportRow>('消费会员', 'uniqueMemberCount', { width: 100 }),
  textColumn<StoreReportRow>('预约数', 'reservationCount', { width: 90 }),
  moneyColumn<StoreReportRow>('订单流水', 'grossCent'),
  moneyColumn<StoreReportRow>('平均订单金额', 'averageOrderCent'),
  textColumn<StoreReportRow>('餐品订单', 'foodOrderCount', { width: 100 }),
  moneyColumn<StoreReportRow>('餐品流水', 'foodGrossCent'),
  textColumn<StoreReportRow>('活动订单', 'activityOrderCount', { width: 100 }),
  moneyColumn<StoreReportRow>('活动流水', 'activityGrossCent'),
  textColumn<StoreReportRow>('券核销', 'couponRedemptionCount', { width: 90 }),
]
</script>

<template>
  <section class="reports">
    <PageHeader
      title="经营报表"
      :description="`当前范围：${selectedStoreName}。经营指标、趋势和明细均按同一门店口径统计。`"
      :breadcrumb="['报表', '经营报表']"
    />

    <NAlert
      v-if="storesError"
      type="error"
      :show-icon="false"
      class="reports__alert"
    >
      {{ storesError }}
    </NAlert>

    <div class="report-controls">
      <div class="report-controls__category">
        <span id="report-category-label">报表类型</span>
        <NSelect
          v-model:value="tab"
          :options="reportOptions"
          aria-labelledby="report-category-label"
          size="large"
        />
      </div>
      <div class="report-tabs__scope">
        <span>统计范围</span>
        <NSelect
          v-model:value="selectedStoreId"
          :options="storeOptions"
          :loading="storesLoading"
          placeholder="全部门店"
          aria-label="选择统计门店"
          filterable
          clearable
        />
      </div>
    </div>
    <p
      v-if="tab !== 'overview'"
      class="reports__scroll-hint"
    >
      左右滑动表格查看完整指标
    </p>
    <NTabs
      v-model:value="tab"
      type="line"
      class="report-tabs"
    >
      <NTabPane
        v-for="item in reportTabs"
        :key="item.key"
        :name="item.key"
        :tab="item.label"
      >
        <template v-if="item.key === 'overview'">
          <NAlert
            v-if="overviewError"
            type="error"
            :show-icon="false"
            class="reports__alert"
          >
            {{ overviewError }}
          </NAlert>

          <NSpin :show="overviewLoading">
            <template v-if="overview">
              <section class="overview-section">
                <header class="section-heading">
                  <div>
                    <h2>累计经营摘要</h2>
                    <p>{{ selectedStoreName }}的核心交易、会员和资产使用数据</p>
                  </div>
                  <span>{{ overview.storeCount }} 家门店纳入统计</span>
                </header>
                <div class="metric-grid">
                  <article
                    v-for="metric in summaryMetrics"
                    :key="metric.key"
                    class="metric-item"
                  >
                    <span>{{ metric.label }}</span>
                    <strong>{{ metric.value }}</strong>
                  </article>
                </div>
              </section>

              <section class="overview-section overview-section--divided">
                <header class="section-heading">
                  <div>
                    <h2>今日经营</h2>
                    <p>按上海时区自然日统计</p>
                  </div>
                </header>
                <div class="today-grid">
                  <article
                    v-for="metric in todayMetrics"
                    :key="metric.key"
                    class="today-item"
                  >
                    <span>{{ metric.label }}</span>
                    <strong>{{ metric.value }}</strong>
                  </article>
                </div>
              </section>

              <section class="overview-section overview-section--divided">
                <header class="section-heading">
                  <div>
                    <h2>近 7 日微信实收趋势</h2>
                    <p>柱高表示当日微信实收，柱顶为已支付订单数</p>
                  </div>
                  <strong>{{ formatCent(trendMax) }}</strong>
                </header>
                <div
                  v-if="trendBars.length"
                  class="trend-chart"
                  role="img"
                  aria-label="近七日微信实收和订单数量柱状图"
                >
                  <div
                    v-for="point in trendBars"
                    :key="point.date"
                    class="trend-day"
                    :aria-label="`${point.date}，${formatCent(point.wechatRevenueCent)}，${point.orderCount} 单`"
                  >
                    <span>{{ point.orderCount }} 单</span>
                    <div class="trend-rail">
                      <i :style="{ height: `${point.height}%` }" />
                    </div>
                    <time :datetime="point.date">{{ point.date.slice(5).replace('-', '/') }}</time>
                  </div>
                </div>
                <NEmpty
                  v-else
                  description="暂无趋势数据"
                />
              </section>

              <section class="breakdown-grid overview-section--divided">
                <div class="breakdown-block">
                  <header class="section-heading section-heading--compact">
                    <div>
                      <h2>微信支付构成</h2>
                      <p>人民币金额，单位为元</p>
                    </div>
                    <strong>{{ formatCent(overview.wechatRevenue.total) }}</strong>
                  </header>
                  <div
                    v-for="row in wechatRows"
                    :key="row.key"
                    class="breakdown-row"
                  >
                    <span>{{ row.label }}</span>
                    <strong>{{ formatCent(row.total) }}</strong>
                    <small>今日 {{ formatCent(row.today) }}</small>
                  </div>
                </div>

                <div class="breakdown-block">
                  <header class="section-heading section-heading--compact">
                    <div>
                      <h2>金币消费构成</h2>
                      <p>金币个数，不与人民币分混算</p>
                    </div>
                    <strong>{{ formatCoins(overview.coinConsumption.total) }}</strong>
                  </header>
                  <div
                    v-for="row in coinRows"
                    :key="row.key"
                    class="breakdown-row"
                  >
                    <span>{{ row.label }}</span>
                    <strong>{{ formatCoins(row.total) }}</strong>
                    <small>今日 {{ formatCoins(row.today) }}</small>
                  </div>
                </div>
              </section>
            </template>
            <NEmpty
              v-else-if="!overviewLoading"
              description="暂无经营总览数据"
            />
          </NSpin>
        </template>

        <ResourceListView
          v-else-if="item.key === 'revenue'"
          :key="`revenue-${reportScopeKey}`"
          class="report-list-pane"
          title="收款趋势"
          :description="`${selectedStoreName}按支付日期汇总的已支付订单`"
          :fields="rangeFields"
          :columns="revenueColumns"
          :scroll-x="reportWidth(revenueColumns)"
          :fetcher="revenueFetcher"
        />
        <ResourceListView
          v-else-if="item.key === 'catalog'"
          :key="`catalog-${reportScopeKey}`"
          class="report-list-pane"
          title="商品销售"
          :description="`${selectedStoreName}商品销量与销售流水`"
          :fields="rangeFields"
          :columns="catalogColumns"
          :scroll-x="reportWidth(catalogColumns)"
          :fetcher="catalogFetcher"
        />
        <ResourceListView
          v-else-if="item.key === 'activities'"
          :key="`activities-${reportScopeKey}`"
          class="report-list-pane"
          title="活动经营"
          :description="`${selectedStoreName}活动订单与售票数量`"
          :fields="rangeFields"
          :columns="activityColumns"
          :scroll-x="reportWidth(activityColumns)"
          :fetcher="activityFetcher"
        />
        <ResourceListView
          v-else-if="item.key === 'coupons'"
          :key="`coupons-${reportScopeKey}`"
          class="report-list-pane"
          title="券效率"
          :description="`${selectedStoreName}券发放、核销数量与核销率`"
          :fields="rangeFields"
          :columns="couponColumns"
          :scroll-x="reportWidth(couponColumns)"
          :fetcher="couponFetcher"
        />
        <ResourceListView
          v-else-if="item.key === 'records'"
          :key="`records-${reportScopeKey}`"
          class="report-list-pane"
          title="核销记录"
          :description="`${selectedStoreName}活动票与优惠券核销明细`"
          :fields="rangeFields"
          :columns="recordColumns"
          :scroll-x="reportWidth(recordColumns)"
          :fetcher="recordFetcher"
        />
        <ResourceListView
          v-else-if="item.key === 'members'"
          :key="`members-${reportScopeKey}`"
          class="report-list-pane"
          title="会员消费"
          :description="`${selectedStoreName}产生订单的会员及订单数量`"
          :fields="rangeFields"
          :columns="memberColumns"
          :scroll-x="reportWidth(memberColumns)"
          :fetcher="memberFetcher"
        />
        <ResourceListView
          v-else-if="item.key === 'reservations'"
          :key="`reservations-${reportScopeKey}`"
          class="report-list-pane"
          title="预约趋势"
          :description="`${selectedStoreName}每日预约数量`"
          :fields="rangeFields"
          :columns="reservationColumns"
          :scroll-x="reportWidth(reservationColumns)"
          :fetcher="reservationFetcher"
        />
        <ResourceListView
          v-else-if="item.key === 'stores'"
          :key="`stores-${reportScopeKey}`"
          class="report-list-pane"
          title="门店经营对比"
          description="对比各门店的订单、会员、预约、流水与券核销表现"
          :fields="rangeFields"
          :columns="storeColumns"
          :scroll-x="reportWidth(storeColumns)"
          :fetcher="storeFetcher"
        />
      </NTabPane>
    </NTabs>
  </section>
</template>

<style scoped>
.reports {
  max-width: 1480px;
  min-width: 0;
}

.report-controls {
  display: flex;
  justify-content: flex-end;
  padding-bottom: var(--ic-space-md);
}

.report-controls__category,
.reports__scroll-hint {
  display: none;
}

.section-heading p,
.breakdown-row small {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.reports__alert {
  margin-bottom: var(--ic-space-md);
}

.report-tabs :deep(.n-tabs-nav) {
  min-height: 54px;
  border-top: 1px solid var(--ic-color-border);
  border-bottom: 1px solid var(--ic-color-border);
}

.report-tabs :deep(.n-tabs-nav-scroll-content) {
  gap: 2px;
}

.report-tabs :deep(.n-tabs-tab) {
  padding: 16px 12px;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
  transition: color 160ms ease-out, background-color 160ms ease-out;
}

.report-tabs :deep(.n-tabs-tab:hover) {
  color: var(--ic-color-text);
  background: var(--ic-color-surface-muted);
}

.report-tabs :deep(.n-tabs-tab--active) {
  color: var(--ic-color-text);
  font-weight: 600;
}

.report-tabs__scope {
  display: flex;
  align-items: center;
  gap: var(--ic-space-sm);
  padding-left: var(--ic-space-md);
}

.report-tabs__scope > span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  white-space: nowrap;
}

.report-tabs__scope :deep(.n-select) {
  width: 260px;
}

.report-tabs__scope :deep(.n-base-selection-placeholder) {
  color: var(--ic-color-text-secondary);
}

:deep(.report-list-pane > .page-header) {
  padding-top: var(--ic-space-lg);
  padding-bottom: 4px;
}

.overview-section {
  padding: var(--ic-space-lg) 0;
}

.overview-section--divided {
  border-top: 1px solid var(--ic-color-divider);
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ic-space-md);
  margin-bottom: var(--ic-space-md);
}

.section-heading h2 {
  margin: 0;
  font-size: var(--ic-font-lg);
  font-weight: 650;
}

.section-heading p {
  margin: 5px 0 0;
}

.section-heading > span,
.section-heading > strong {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.metric-grid,
.today-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border-top: 1px solid var(--ic-color-divider);
}

.metric-item,
.today-item {
  min-width: 0;
  padding: var(--ic-space-md) var(--ic-space-md) var(--ic-space-md) 0;
  border-bottom: 1px solid var(--ic-color-divider);
}

.metric-item span,
.metric-item strong,
.today-item span,
.today-item strong {
  display: block;
}

.metric-item span,
.today-item span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.metric-item strong {
  margin-top: 8px;
  overflow: hidden;
  font-size: var(--ic-font-xl);
  font-variant-numeric: tabular-nums;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.today-item strong {
  margin-top: 6px;
  font-size: var(--ic-font-lg);
  font-variant-numeric: tabular-nums;
}

.trend-chart {
  display: grid;
  height: 240px;
  grid-template-columns: repeat(7, minmax(36px, 1fr));
  gap: clamp(10px, 3vw, 36px);
  padding: 8px 20px 0;
  border-bottom: 1px solid var(--ic-color-border);
}

.trend-day {
  display: grid;
  min-width: 0;
  grid-template-rows: 20px 1fr 30px;
  text-align: center;
}

.trend-day > span,
.trend-day > time {
  color: var(--ic-color-text-secondary);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.trend-rail {
  display: flex;
  min-height: 0;
  align-items: flex-end;
  justify-content: center;
}

.trend-rail i {
  display: block;
  width: min(42px, 76%);
  background: var(--ic-color-primary);
  border-radius: 5px 5px 0 0;
  transition: height 0.2s ease-out;
}

.trend-day > time {
  padding-top: 9px;
}

.breakdown-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--ic-space-xl);
  padding: var(--ic-space-lg) 0;
}

.breakdown-block {
  min-width: 0;
}

.section-heading--compact {
  padding-bottom: var(--ic-space-md);
  border-bottom: 1px solid var(--ic-color-divider);
}

.breakdown-row {
  display: grid;
  grid-template-columns: minmax(100px, 1fr) auto minmax(110px, auto);
  gap: var(--ic-space-md);
  align-items: baseline;
  padding: 12px 0;
  border-bottom: 1px solid var(--ic-color-divider);
  font-variant-numeric: tabular-nums;
}

.breakdown-row small {
  text-align: right;
}

@media (prefers-reduced-motion: reduce) {
  .report-tabs :deep(.n-tabs-tab),
  .trend-rail i {
    transition: none;
  }
}

@media (max-width: 900px) {
  .metric-grid,
  .today-grid,
  .breakdown-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .report-controls {
    display: grid;
    justify-content: stretch;
    gap: 12px;
  }

  .report-controls__category,
  .report-tabs__scope {
    display: grid;
    grid-template-columns: 64px minmax(0, 1fr);
    align-items: center;
    gap: 12px;
    padding: 0;
    font-size: var(--ic-font-sm);
  }

  .report-tabs :deep(.n-tabs-nav) {
    display: none;
  }

  .reports__scroll-hint {
    display: block;
    margin: 0;
    color: var(--ic-color-text-secondary);
    font-size: var(--ic-font-sm);
  }

  .section-heading {
    align-items: stretch;
    flex-direction: column;
  }

  .report-tabs__scope > span {
    font-size: var(--ic-font-sm);
  }

  .report-tabs__scope :deep(.n-select) {
    width: 100%;
  }

  .metric-grid,
  .today-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-item strong,
  .today-item strong {
    overflow: visible;
    overflow-wrap: anywhere;
    white-space: normal;
    font-size: 20px;
  }

  .breakdown-grid {
    grid-template-columns: 1fr;
  }

  .trend-chart {
    grid-template-columns: repeat(7, minmax(0, 1fr));
    gap: 6px;
    padding-inline: 0;
  }

  .breakdown-row {
    grid-template-columns: 1fr auto;
  }

  .breakdown-row small {
    grid-column: 1 / -1;
    text-align: left;
  }

  .breakdown-row strong {
    overflow-wrap: anywhere;
    min-width: 0;
  }

  :deep(.report-list-pane .filter-bar),
  :deep(.report-list-pane .data-table) {
    min-width: 0;
    padding: 12px 0;
    border-radius: 0;
    border-inline: 0;
    background: transparent;
  }

  :deep(.report-list-pane .filter-bar__fields),
  :deep(.report-list-pane .filter-bar__field) {
    width: 100%;
    min-width: 0;
  }

  :deep(.report-list-pane .n-date-picker) {
    width: 100% !important;
  }

  :deep(.report-list-pane .n-pagination) {
    flex-wrap: wrap;
    justify-content: flex-start;
    gap: 8px 0;
  }

  :deep(.n-base-selection-label),
  :deep(.report-list-pane .n-input),
  :deep(.report-list-pane .n-button) {
    min-height: 44px;
    align-items: center;
  }
}
</style>
