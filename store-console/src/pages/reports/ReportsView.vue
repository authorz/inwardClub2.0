<script setup lang="ts">
/**
 * 本店报表：本店经营概览与分项报表入口（仅本店范围，无跨店维度）。
 */
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NSkeleton, NSpin, NTabPane, NTabs } from 'naive-ui'
import { reportService } from '@/api/services'
import { ApiError } from '@/api/error'
import { formatCent, formatCompactCent } from '@/utils/format'
import { EmptyState, MetricTile, PageHeader } from '@/components/common'
import type { ReportOverview, RevenueReportRow } from '@/types/models'

type RangeKey = 'today' | '7d' | '30d'

const overviewLoading = ref(false)
const overview = ref<ReportOverview | null>(null)
const overviewError = ref<string | null>(null)
const revenueLoading = ref(false)
const revenueRows = ref<RevenueReportRow[]>([])
const revenueError = ref<string | null>(null)
const range = ref<RangeKey>('7d')
let revenueRequestVersion = 0

const rangeOptions = [
  { label: '今日', value: 'today' },
  { label: '近 7 天', value: '7d' },
  { label: '近 30 天', value: '30d' },
]

function formatDateKey(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function dateKey(value: string): string {
  return value.slice(0, 10)
}

function rangeWindow(key: RangeKey): { from: string; to: string; days: number } {
  const end = new Date()
  const days = key === 'today' ? 1 : key === '7d' ? 7 : 30
  const start = new Date(end)
  start.setDate(end.getDate() - (days - 1))
  return { from: formatDateKey(start), to: formatDateKey(end), days }
}

function dateKeys(from: string, days: number): string[] {
  const cursor = new Date(`${from}T12:00:00`)
  return Array.from({ length: days }, (_, index) => {
    const date = new Date(cursor)
    date.setDate(cursor.getDate() + index)
    return formatDateKey(date)
  })
}

const selectedWindow = computed(() => rangeWindow(range.value))
const revenueTrend = computed<RevenueReportRow[]>(() => {
  const byDate = new Map(revenueRows.value.map((row) => [dateKey(row.date), row]))
  return dateKeys(selectedWindow.value.from, selectedWindow.value.days).map((date) => (
    byDate.get(date) ?? { date, orderCount: 0, grossCent: 0 }
  ))
})
const revenueTotalCent = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.grossCent, 0)
))
const revenueOrderCount = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.orderCount, 0)
))
const revenuePeakCent = computed(() => (
  Math.max(0, ...revenueTrend.value.map((row) => row.grossCent))
))
const revenueBars = computed(() => revenueTrend.value.map((row) => ({
  ...row,
  height: revenuePeakCent.value > 0 && row.grossCent > 0
    ? Math.max(4, (row.grossCent / revenuePeakCent.value) * 100)
    : 0,
})))
const hasRevenue = computed(() => revenueTotalCent.value > 0 || revenueOrderCount.value > 0)
const rangeLabel = computed(() => (
  `${selectedWindow.value.from.replaceAll('-', '/')} – ${selectedWindow.value.to.replaceAll('-', '/')}`
))

function formatDateLabel(date: string): string {
  return date.slice(5).replace('-', '/')
}

function showDateLabel(index: number): boolean {
  if (range.value !== '30d') return true
  return index === 0 || index === revenueBars.value.length - 1 || index % 5 === 0
}

async function loadOverview() {
  overviewLoading.value = true
  overviewError.value = null
  try {
    overview.value = await reportService.overview()
  } catch (err) {
    overviewError.value = err instanceof ApiError ? err.message : '经营概览加载失败'
  } finally {
    overviewLoading.value = false
  }
}

async function loadRevenue() {
  const requestVersion = ++revenueRequestVersion
  const window = rangeWindow(range.value)
  revenueLoading.value = true
  revenueError.value = null
  try {
    const rows = await reportService.revenue({
      from: window.from,
      to: window.to,
      page: 1,
      pageSize: window.days,
    })
    if (requestVersion === revenueRequestVersion) revenueRows.value = rows
  } catch (err) {
    if (requestVersion === revenueRequestVersion) {
      revenueRows.value = []
      revenueError.value = err instanceof ApiError ? err.message : '收款趋势加载失败'
    }
  } finally {
    if (requestVersion === revenueRequestVersion) revenueLoading.value = false
  }
}

function setRange(value: RangeKey) {
  if (range.value === value && revenueRows.value.length > 0) return
  range.value = value
  void loadRevenue()
}

onMounted(() => {
  void Promise.all([loadOverview(), loadRevenue()])
})
</script>

<template>
  <div>
    <PageHeader
      title="本店报表"
      description="本店经营概览与分项统计"
    >
      <template #actions>
        <NButton
          v-for="opt in rangeOptions"
          :key="opt.value"
          size="small"
          :type="range === opt.value ? 'primary' : 'default'"
          quaternary
          @click="setRange(opt.value as RangeKey)"
        >
          {{ opt.label }}
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="overviewLoading">
      <div class="metrics">
        <MetricTile
          label="门店数"
          :value="overview?.storeCount ?? '—'"
        />
        <MetricTile
          label="会员数"
          :value="overview?.memberCount ?? '—'"
        />
        <MetricTile
          label="订单数"
          :value="overview?.orderCount ?? '—'"
        />
        <MetricTile
          label="销售额"
          :value="formatCent(overview?.grossSalesCent ?? null)"
        />
        <MetricTile
          label="发券数"
          :value="overview?.couponsIssued ?? '—'"
        />
        <MetricTile
          label="核销券数"
          :value="overview?.couponsRedeemed ?? '—'"
        />
      </div>
      <p
        v-if="overviewError"
        class="ic-muted reports__hint"
      >
        {{ overviewError }}
      </p>
    </NSpin>

    <NTabs
      type="line"
      default-value="revenue"
      class="reports__tabs"
    >
      <NTabPane
        name="revenue"
        tab="收款趋势"
      >
        <section class="trend-panel">
          <header class="trend-panel__header">
            <div>
              <h2>收款趋势</h2>
              <p>{{ rangeLabel }} · 按支付完成日期汇总</p>
            </div>
            <div class="trend-summary">
              <span>
                <small>区间收款</small>
                <strong>{{ formatCent(revenueTotalCent) }}</strong>
              </span>
              <span>
                <small>已支付订单</small>
                <strong>{{ revenueOrderCount }} 单</strong>
              </span>
            </div>
          </header>

          <NSkeleton
            v-if="revenueLoading"
            height="300px"
            :sharp="false"
          />
          <NAlert
            v-else-if="revenueError"
            type="error"
            :show-icon="false"
          >
            <div class="trend-error">
              <span>{{ revenueError }}</span>
              <NButton
                size="small"
                @click="loadRevenue"
              >
                重试
              </NButton>
            </div>
          </NAlert>
          <div
            v-else-if="hasRevenue"
            class="bar-chart-scroll"
          >
            <div
              class="bar-chart"
              :class="`bar-chart--${range}`"
              role="img"
              :aria-label="`${rangeLabel} 收款趋势，区间收款 ${formatCent(revenueTotalCent)}，${revenueOrderCount} 单`"
            >
              <div
                v-for="(item, index) in revenueBars"
                :key="item.date"
                class="bar-chart__day"
                :title="`${item.date}：${formatCent(item.grossCent)}，${item.orderCount} 单`"
              >
                <span
                  v-if="range !== '30d'"
                  class="bar-chart__amount"
                >
                  {{ formatCompactCent(item.grossCent) }}
                </span>
                <div class="bar-chart__rail">
                  <i :style="{ height: `${item.height}%` }" />
                </div>
                <span class="bar-chart__date">
                  {{ showDateLabel(index) ? formatDateLabel(item.date) : '' }}
                </span>
              </div>
            </div>
          </div>
          <EmptyState
            v-else
            description="所选时间范围内暂无已支付收款"
          />
        </section>
      </NTabPane>
      <NTabPane
        name="items"
        tab="热门商品"
      >
        <EmptyState description="热门商品统计待服务端 /store/reports/catalog-items 接入" />
      </NTabPane>
      <NTabPane
        name="activities"
        tab="活动核销"
      >
        <EmptyState description="活动核销统计待服务端 /store/reports/activities 接入" />
      </NTabPane>
    </NTabs>
  </div>
</template>

<style scoped>
.metrics {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--ic-space-3);
}
.reports__hint {
  font-size: var(--ic-font-xs);
  margin-top: var(--ic-space-3);
}
.reports__tabs {
  margin-top: var(--ic-space-6);
}
.trend-panel {
  padding-top: var(--ic-space-2);
}
.trend-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ic-space-5);
  padding-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
}
.trend-panel__header h2 {
  margin: 0;
  font-size: var(--ic-font-lg);
}
.trend-panel__header p {
  margin: var(--ic-space-1) 0 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
.trend-summary {
  display: flex;
  align-items: center;
  gap: var(--ic-space-5);
  flex-shrink: 0;
}
.trend-summary span {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: var(--ic-space-1);
}
.trend-summary small {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}
.trend-summary strong {
  font-size: var(--ic-font-md);
  font-variant-numeric: tabular-nums;
}
.trend-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-4);
}
.trend-panel > .n-skeleton,
.trend-panel > .n-alert {
  margin-top: var(--ic-space-5);
}
.bar-chart-scroll {
  overflow-x: auto;
  padding-top: var(--ic-space-5);
}
.bar-chart {
  display: grid;
  grid-template-columns: repeat(7, minmax(64px, 1fr));
  gap: var(--ic-space-3);
  min-width: 560px;
  height: 300px;
}
.bar-chart--today {
  grid-template-columns: minmax(96px, 140px);
  min-width: 0;
}
.bar-chart--30d {
  grid-template-columns: repeat(30, minmax(24px, 1fr));
  gap: var(--ic-space-2);
  min-width: 880px;
}
.bar-chart__day {
  min-width: 0;
  display: grid;
  grid-template-rows: 24px minmax(0, 1fr) 22px;
  align-items: end;
  text-align: center;
}
.bar-chart__amount,
.bar-chart__date {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.bar-chart__rail {
  position: relative;
  width: min(42px, 70%);
  height: 100%;
  justify-self: center;
  overflow: hidden;
  background: var(--ic-color-surface-muted);
  border-bottom: 1px solid var(--ic-color-border-strong);
}
.bar-chart__rail i {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  background: var(--ic-color-primary);
  transition: height 180ms ease-out;
}
.bar-chart--30d .bar-chart__day {
  grid-template-rows: minmax(0, 1fr) 22px;
}
.bar-chart--30d .bar-chart__rail {
  width: min(18px, 75%);
}
@media (max-width: 720px) {
  .trend-panel__header {
    flex-direction: column;
  }
  .trend-summary {
    width: 100%;
    justify-content: space-between;
  }
  .trend-summary span {
    align-items: flex-start;
  }
}
@media (prefers-reduced-motion: reduce) {
  .bar-chart__rail i {
    transition: none;
  }
}
</style>
