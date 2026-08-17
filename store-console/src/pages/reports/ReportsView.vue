<script setup lang="ts">
/**
 * 本店报表：本店经营概览与分项报表入口（仅本店范围，无跨店维度）。
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
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
  return dateKeys(selectedWindow.value.from, selectedWindow.value.days).map((date) => {
    const row = byDate.get(date)
    return {
      date,
      orderCount: row?.orderCount ?? 0,
      grossCent: row?.grossCent ?? 0,
    }
  })
})
const revenueTotalCent = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.grossCent, 0)
))
const revenueOrderCount = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.orderCount, 0)
))
const revenuePeakCent = computed(() => Math.max(0, ...revenueTrend.value.map((row) => row.grossCent)))
const hasRevenue = computed(() => revenueTotalCent.value > 0 || revenueOrderCount.value > 0)
const rangeLabel = computed(() => (
  `${selectedWindow.value.from.replaceAll('-', '/')} – ${selectedWindow.value.to.replaceAll('-', '/')}`
))

function formatDateLabel(date: string): string {
  return date.slice(5).replace('-', '/')
}

const chartHost = ref<HTMLElement | null>(null)
const chartWidth = ref(960)
const chartHeight = 300
const chartPadding = { top: 24, right: 24, bottom: 48, left: 68 }
const activePointIndex = ref<number | null>(null)
let chartResizeObserver: ResizeObserver | null = null

function niceStep(value: number): number {
  if (value <= 0) return 100
  const exponent = Math.floor(Math.log10(value))
  const magnitude = 10 ** exponent
  const fraction = value / magnitude
  const niceFraction = fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 2.5 ? 2.5 : fraction <= 5 ? 5 : 10
  return niceFraction * magnitude
}

const chartScale = computed(() => {
  const step = niceStep((revenuePeakCent.value * 1.12) / 4)
  const max = Math.max(step, Math.ceil((revenuePeakCent.value * 1.08) / step) * step)
  return { max, step }
})

const chartTicks = computed(() => {
  const count = Math.round(chartScale.value.max / chartScale.value.step)
  const plotHeight = chartHeight - chartPadding.top - chartPadding.bottom
  return Array.from({ length: count + 1 }, (_, index) => {
    const value = index * chartScale.value.step
    return {
      value,
      y: chartHeight - chartPadding.bottom - (value / chartScale.value.max) * plotHeight,
    }
  }).reverse()
})

const chartPoints = computed(() => {
  const rows = revenueTrend.value
  const plotWidth = chartWidth.value - chartPadding.left - chartPadding.right
  const plotHeight = chartHeight - chartPadding.top - chartPadding.bottom
  return rows.map((row, index) => ({
    ...row,
    x: rows.length === 1
      ? chartPadding.left + plotWidth / 2
      : chartPadding.left + (index / (rows.length - 1)) * plotWidth,
    y: chartHeight - chartPadding.bottom - (row.grossCent / chartScale.value.max) * plotHeight,
  }))
})

const revenueLinePath = computed(() => chartPoints.value
  .map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x} ${point.y}`)
  .join(' '))

const revenueAreaPath = computed(() => {
  const points = chartPoints.value
  if (points.length < 2) return ''
  const baseline = chartHeight - chartPadding.bottom
  const line = points.map((point) => `L ${point.x} ${point.y}`).join(' ')
  return `M ${points[0].x} ${baseline} ${line} L ${points.at(-1)?.x ?? 0} ${baseline} Z`
})

const activePoint = computed(() => (
  activePointIndex.value == null ? null : chartPoints.value[activePointIndex.value] ?? null
))

function showDateLabel(index: number): boolean {
  const pointCount = chartPoints.value.length
  if (pointCount <= 1) return true
  const availableWidth = chartWidth.value - chartPadding.left - chartPadding.right
  const maxLabelCount = Math.max(2, Math.floor(availableWidth / 72))
  const step = Math.max(1, Math.ceil((pointCount - 1) / (maxLabelCount - 1)))
  return index === 0 || index === pointCount - 1 || index % step === 0
}

watch(chartHost, (element, previousElement) => {
  if (previousElement) chartResizeObserver?.unobserve(previousElement)
  if (!element || typeof ResizeObserver === 'undefined') return
  chartResizeObserver ??= new ResizeObserver(([entry]) => {
    if (entry) chartWidth.value = Math.max(680, Math.floor(entry.contentRect.width))
  })
  chartWidth.value = Math.max(680, Math.floor(element.clientWidth))
  chartResizeObserver.observe(element)
}, { flush: 'post' })

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

onBeforeUnmount(() => chartResizeObserver?.disconnect())
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
            ref="chartHost"
            class="trend-chart-scroll"
            @mouseleave="activePointIndex = null"
          >
            <div
              class="trend-chart-stage"
              :style="{ width: `${chartWidth}px` }"
            >
              <svg
                class="trend-chart"
                :viewBox="`0 0 ${chartWidth} ${chartHeight}`"
                :width="chartWidth"
                :height="chartHeight"
                role="img"
                :aria-label="`${rangeLabel} 收款趋势，区间收款 ${formatCent(revenueTotalCent)}，${revenueOrderCount} 单`"
              >
                <g
                  v-for="tick in chartTicks"
                  :key="tick.value"
                  class="trend-chart__grid"
                >
                  <line
                    :x1="chartPadding.left"
                    :x2="chartWidth - chartPadding.right"
                    :y1="tick.y"
                    :y2="tick.y"
                  />
                  <text
                    :x="chartPadding.left - 12"
                    :y="tick.y + 4"
                    text-anchor="end"
                  >
                    {{ formatCompactCent(tick.value) }}
                  </text>
                </g>

                <path
                  v-if="revenueAreaPath"
                  class="trend-chart__area"
                  :d="revenueAreaPath"
                />
                <path
                  class="trend-chart__line"
                  :d="revenueLinePath"
                />

                <g
                  v-for="(point, index) in chartPoints"
                  :key="point.date"
                  class="trend-chart__point-group"
                  role="img"
                  tabindex="0"
                  :aria-label="`${point.date}，收款 ${formatCent(point.grossCent)}，已支付订单 ${point.orderCount} 单`"
                  @mouseenter="activePointIndex = index"
                  @focus="activePointIndex = index"
                  @blur="activePointIndex = null"
                  @click="activePointIndex = index"
                >
                  <line
                    v-if="activePointIndex === index"
                    class="trend-chart__crosshair"
                    :x1="point.x"
                    :x2="point.x"
                    :y1="chartPadding.top"
                    :y2="chartHeight - chartPadding.bottom"
                  />
                  <rect
                    class="trend-chart__hit-area"
                    :x="point.x - Math.max(16, (chartWidth - chartPadding.left - chartPadding.right) / chartPoints.length / 2)"
                    :y="chartPadding.top"
                    :width="Math.max(32, (chartWidth - chartPadding.left - chartPadding.right) / chartPoints.length)"
                    :height="chartHeight - chartPadding.top - chartPadding.bottom"
                  />
                  <circle
                    class="trend-chart__point"
                    :class="{ 'trend-chart__point--zero': point.grossCent === 0 }"
                    :cx="point.x"
                    :cy="point.y"
                    :r="activePointIndex === index ? 5 : point.grossCent === 0 ? 2.5 : 3.5"
                  />
                  <text
                    v-if="showDateLabel(index)"
                    class="trend-chart__date"
                    :x="point.x"
                    :y="chartHeight - 17"
                    text-anchor="middle"
                  >
                    {{ formatDateLabel(point.date) }}
                  </text>
                </g>
              </svg>

              <div
                v-if="activePoint"
                class="trend-chart__tooltip"
                :style="{
                  left: `${Math.min(chartWidth - 108, Math.max(108, activePoint.x))}px`,
                  top: `${Math.max(4, activePoint.y - 80)}px`,
                }"
              >
                <span>{{ activePoint.date }}</span>
                <strong>{{ formatCent(activePoint.grossCent) }}</strong>
                <small>{{ activePoint.orderCount }} 单已支付</small>
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
.trend-chart-scroll {
  overflow-x: auto;
  margin-top: var(--ic-space-4);
}
.trend-chart-stage {
  position: relative;
  min-width: 680px;
  height: 300px;
}
.trend-chart {
  display: block;
  overflow: visible;
}
.trend-chart__grid line {
  stroke: var(--ic-color-border);
  stroke-width: 1;
  shape-rendering: crispEdges;
}
.trend-chart__grid text,
.trend-chart__date {
  fill: var(--ic-color-text-tertiary);
  font-size: var(--ic-font-xs);
  font-variant-numeric: tabular-nums;
}
.trend-chart__area {
  fill: rgba(64, 115, 158, 0.08);
  pointer-events: none;
}
.trend-chart__line {
  fill: none;
  stroke: var(--ic-color-info);
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 2.5;
  pointer-events: none;
}
.trend-chart__hit-area {
  fill: transparent;
  cursor: crosshair;
}
.trend-chart__crosshair {
  stroke: var(--ic-color-border-strong);
  stroke-dasharray: 3 4;
  stroke-width: 1;
  pointer-events: none;
}
.trend-chart__point {
  fill: var(--ic-color-info);
  stroke: var(--ic-color-surface);
  stroke-width: 2;
  transition: r 120ms ease-out;
  pointer-events: none;
}
.trend-chart__point--zero {
  fill: var(--ic-color-surface);
  stroke: var(--ic-color-border-strong);
  stroke-width: 1.5;
}
.trend-chart__point-group:focus {
  outline: none;
}
.trend-chart__point-group:focus .trend-chart__point {
  stroke: var(--ic-color-text);
  stroke-width: 2.5;
}
.trend-chart__tooltip {
  position: absolute;
  z-index: 1;
  display: grid;
  min-width: 156px;
  padding: 10px 12px;
  border: 1px solid var(--ic-color-border-strong);
  border-radius: var(--ic-radius-md);
  background: var(--ic-color-surface);
  box-shadow: var(--ic-shadow-sm);
  pointer-events: none;
  transform: translateX(-50%);
}
.trend-chart__tooltip span,
.trend-chart__tooltip small {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}
.trend-chart__tooltip strong {
  margin: 2px 0;
  color: var(--ic-color-text);
  font-size: var(--ic-font-md);
  font-variant-numeric: tabular-nums;
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
  .trend-chart__point {
    transition: none;
  }
}
</style>
