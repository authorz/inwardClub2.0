<script setup lang="ts">
/** 本店报表：累计经营概览与可按时间筛选的分项分析。 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NDatePicker,
  NRadioButton,
  NRadioGroup,
  NSkeleton,
  NSpin,
  NTabPane,
  NTabs,
} from 'naive-ui'
import { reportService } from '@/api/services'
import { ApiError } from '@/api/error'
import { formatCent, formatCompactCent } from '@/utils/format'
import { EmptyState, PageHeader } from '@/components/common'
import type {
  ActivityReportRow,
  CatalogItemReportRow,
  ReportOverview,
  RevenueReportRow,
} from '@/types/models'

type RangeKey = 'today' | '7d' | '30d'
type PaymentChannel = 'wechat' | 'coin'

const tab = ref('revenue')
const paymentChannel = ref<PaymentChannel>('wechat')
const overviewLoading = ref(false)
const overview = ref<ReportOverview | null>(null)
const overviewError = ref<string | null>(null)
const revenueLoading = ref(false)
const revenueRows = ref<RevenueReportRow[]>([])
const revenueError = ref<string | null>(null)
const catalogLoading = ref(false)
const catalogRows = ref<CatalogItemReportRow[]>([])
const catalogError = ref<string | null>(null)
const activityLoading = ref(false)
const activityRows = ref<ActivityReportRow[]>([])
const activityError = ref<string | null>(null)
const quickRange = ref<RangeKey | null>('7d')
const rangeError = ref('')
let reportRequestVersion = 0

const rangeOptions = [
  { label: '今日', value: 'today' },
  { label: '近 7 天', value: '7d' },
  { label: '近 30 天', value: '30d' },
]

const countFormatter = new Intl.NumberFormat('zh-CN')
const compactFormatter = new Intl.NumberFormat('zh-CN', {
  notation: 'compact',
  maximumFractionDigits: 1,
})

function formatCount(value: number | null | undefined): string {
  return value == null ? '—' : countFormatter.format(value)
}

function formatCoins(value: number | null | undefined): string {
  return value == null ? '—' : `${countFormatter.format(value)} 金币`
}

function formatDateKey(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function atLocalDayStart(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function presetTimestamps(key: RangeKey): [number, number] {
  const end = atLocalDayStart(new Date())
  const days = key === 'today' ? 1 : key === '7d' ? 7 : 30
  const start = new Date(end)
  start.setDate(end.getDate() - (days - 1))
  return [start.getTime(), end.getTime()]
}

const dateRange = ref<[number, number]>(presetTimestamps('7d'))

function windowFromTimestamps(value: [number, number]) {
  const start = atLocalDayStart(new Date(value[0]))
  const end = atLocalDayStart(new Date(value[1]))
  const days = Math.round((end.getTime() - start.getTime()) / 86_400_000) + 1
  return { from: formatDateKey(start), to: formatDateKey(end), days }
}

const selectedWindow = computed(() => windowFromTimestamps(dateRange.value))
const mobileRangeStart = ref(selectedWindow.value.from)
const mobileRangeEnd = ref(selectedWindow.value.to)
watch(dateRange, () => {
  mobileRangeStart.value = selectedWindow.value.from
  mobileRangeEnd.value = selectedWindow.value.to
})

function applyMobileRange(): void {
  const start = new Date(`${mobileRangeStart.value}T00:00:00`).getTime()
  const end = new Date(`${mobileRangeEnd.value}T00:00:00`).getTime()
  if (!Number.isFinite(start) || !Number.isFinite(end) || start > end) {
    rangeError.value = '请选择完整日期，且开始日期不能晚于结束日期。'
    return
  }
  if (disableFutureDate(end)) {
    rangeError.value = '分析时间不能晚于今天。'
    return
  }
  updateCustomRange([start, end])
}

const rangeLabel = computed(() => (
  `${selectedWindow.value.from.replaceAll('-', '/')} – ${selectedWindow.value.to.replaceAll('-', '/')}`
))

function dateKeys(from: string, days: number): string[] {
  const cursor = new Date(`${from}T12:00:00`)
  return Array.from({ length: days }, (_, index) => {
    const date = new Date(cursor)
    date.setDate(cursor.getDate() + index)
    return formatDateKey(date)
  })
}

function dateKey(value: string): string {
  return value.slice(0, 10)
}

const revenueTrend = computed<RevenueReportRow[]>(() => {
  const byDate = new Map(revenueRows.value.map((row) => [dateKey(row.date), row]))
  return dateKeys(selectedWindow.value.from, selectedWindow.value.days).map((date) => {
    const row = byDate.get(date)
    return {
      date,
      orderCount: row?.orderCount ?? 0,
      grossCent: row?.grossCent ?? 0,
      wechatOrderCount: row?.wechatOrderCount ?? 0,
      wechatRevenueCent: row?.wechatRevenueCent ?? 0,
      coinOrderCount: row?.coinOrderCount ?? 0,
      coinConsumption: row?.coinConsumption ?? 0,
    }
  })
})

const wechatTotalCent = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.wechatRevenueCent, 0)
))
const coinTotal = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.coinConsumption, 0)
))
const wechatOrderCount = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.wechatOrderCount, 0)
))
const coinOrderCount = computed(() => (
  revenueTrend.value.reduce((sum, row) => sum + row.coinOrderCount, 0)
))
const channelTotal = computed(() => (
  paymentChannel.value === 'wechat' ? wechatTotalCent.value : coinTotal.value
))
const channelOrderCount = computed(() => (
  paymentChannel.value === 'wechat' ? wechatOrderCount.value : coinOrderCount.value
))
const channelLabel = computed(() => paymentChannel.value === 'wechat' ? '微信支付' : '金币消费')
const hasRevenue = computed(() => channelTotal.value > 0 || channelOrderCount.value > 0)

const summaryMetrics = computed(() => [
  { key: 'members', label: '累计消费会员', value: formatCount(overview.value?.memberCount) },
  { key: 'orders', label: '累计已支付订单', value: formatCount(overview.value?.orderCount) },
  { key: 'wechat', label: '微信实收', value: overview.value ? formatCent(overview.value.wechatRevenue.total) : '—' },
  { key: 'coins', label: '金币消费', value: formatCoins(overview.value?.coinConsumption.total) },
  { key: 'issued', label: '累计发券', value: formatCount(overview.value?.couponsIssued) },
  { key: 'redeemed', label: '累计核销', value: formatCount(overview.value?.couponsRedeemed) },
])

function errorMessage(error: unknown, fallback: string): string {
  return error instanceof ApiError ? error.message : fallback
}

async function loadOverview(): Promise<void> {
  overviewLoading.value = true
  overviewError.value = null
  try {
    overview.value = await reportService.overview()
  } catch (error) {
    overviewError.value = errorMessage(error, '累计经营概览加载失败')
  } finally {
    overviewLoading.value = false
  }
}

async function loadRangeReports(): Promise<void> {
  const requestVersion = ++reportRequestVersion
  const window = selectedWindow.value
  const params = { from: window.from, to: window.to, page: 1, pageSize: Math.min(window.days, 100) }
  revenueLoading.value = true
  catalogLoading.value = true
  activityLoading.value = true
  revenueError.value = null
  catalogError.value = null
  activityError.value = null

  const [revenueResult, catalogResult, activityResult] = await Promise.allSettled([
    reportService.revenue(params),
    reportService.catalogItems({ ...params, pageSize: 20 }),
    reportService.activities({ ...params, pageSize: 20 }),
  ])
  if (requestVersion !== reportRequestVersion) return

  if (revenueResult.status === 'fulfilled') {
    const rows = revenueResult.value
    const missingChannelFields = rows.some((row) => (
      !Number.isFinite(row.wechatOrderCount)
      || !Number.isFinite(row.wechatRevenueCent)
      || !Number.isFinite(row.coinOrderCount)
      || !Number.isFinite(row.coinConsumption)
    ))
    if (missingChannelFields) {
      revenueRows.value = []
      revenueError.value = '报表服务版本过旧，暂时无法区分微信支付与金币消费，请先更新服务端。'
    } else {
      revenueRows.value = rows
    }
  } else {
    revenueRows.value = []
    revenueError.value = errorMessage(revenueResult.reason, '收款趋势加载失败')
  }
  if (catalogResult.status === 'fulfilled') catalogRows.value = catalogResult.value
  else {
    catalogRows.value = []
    catalogError.value = errorMessage(catalogResult.reason, '商品销售排行加载失败')
  }
  if (activityResult.status === 'fulfilled') activityRows.value = activityResult.value
  else {
    activityRows.value = []
    activityError.value = errorMessage(activityResult.reason, '活动经营数据加载失败')
  }
  revenueLoading.value = false
  catalogLoading.value = false
  activityLoading.value = false
}

function setQuickRange(value: RangeKey): void {
  quickRange.value = value
  dateRange.value = presetTimestamps(value)
  rangeError.value = ''
  void loadRangeReports()
}

function updateCustomRange(value: [number, number] | null): void {
  if (!value) return
  const window = windowFromTimestamps(value)
  if (window.days > 90) {
    rangeError.value = '自定义时间范围最多支持 90 天，请缩短后重试。'
    return
  }
  quickRange.value = null
  dateRange.value = value
  rangeError.value = ''
  void loadRangeReports()
}

function disableFutureDate(timestamp: number): boolean {
  const tomorrow = atLocalDayStart(new Date())
  tomorrow.setDate(tomorrow.getDate() + 1)
  return timestamp >= tomorrow.getTime()
}

function formatDateLabel(date: string): string {
  return date.slice(5).replace('-', '/')
}

function formatChannelValue(value: number): string {
  return paymentChannel.value === 'wechat' ? formatCent(value) : formatCoins(value)
}

function formatCompactChannelValue(value: number): string {
  return paymentChannel.value === 'wechat' ? formatCompactCent(value) : compactFormatter.format(value)
}

const chartHost = ref<HTMLElement | null>(null)
const chartWidth = ref(960)
const chartHeight = 300
const chartPadding = { top: 24, right: 24, bottom: 48, left: 72 }
const activePointIndex = ref<number | null>(null)
const selectedPointIndex = ref<number | null>(null)
let chartResizeObserver: ResizeObserver | null = null

function niceStep(value: number): number {
  if (value <= 0) return 1
  const exponent = Math.floor(Math.log10(value))
  const magnitude = 10 ** exponent
  const fraction = value / magnitude
  const niceFraction = fraction <= 1 ? 1 : fraction <= 2 ? 2 : fraction <= 2.5 ? 2.5 : fraction <= 5 ? 5 : 10
  return niceFraction * magnitude
}

const chartSeries = computed(() => revenueTrend.value.map((row) => ({
  ...row,
  value: paymentChannel.value === 'wechat' ? row.wechatRevenueCent : row.coinConsumption,
  channelOrders: paymentChannel.value === 'wechat' ? row.wechatOrderCount : row.coinOrderCount,
})))
const chartPeak = computed(() => Math.max(0, ...chartSeries.value.map((row) => row.value)))
const chartScale = computed(() => {
  const step = niceStep((chartPeak.value * 1.12) / 4)
  const max = Math.max(step, Math.ceil((chartPeak.value * 1.08) / step) * step)
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
  const rows = chartSeries.value
  const plotWidth = chartWidth.value - chartPadding.left - chartPadding.right
  const plotHeight = chartHeight - chartPadding.top - chartPadding.bottom
  return rows.map((row, index) => ({
    ...row,
    x: rows.length === 1
      ? chartPadding.left + plotWidth / 2
      : chartPadding.left + (index / (rows.length - 1)) * plotWidth,
    y: chartHeight - chartPadding.bottom - (row.value / chartScale.value.max) * plotHeight,
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
const mobilePoint = computed(() => (
  chartPoints.value[selectedPointIndex.value ?? chartPoints.value.length - 1]
))

function selectChartPoint(index: number): void {
  selectedPointIndex.value = index
  activePointIndex.value = index
}

function selectMobilePoint(event: Event): void {
  selectChartPoint(Number((event.target as HTMLInputElement).value))
}

watch([dateRange, paymentChannel], () => {
  activePointIndex.value = null
  selectedPointIndex.value = null
})

function showDateLabel(index: number): boolean {
  const pointCount = chartPoints.value.length
  if (pointCount <= 1) return true
  const availableWidth = chartWidth.value - chartPadding.left - chartPadding.right
  const maxLabelCount = Math.max(2, Math.floor(availableWidth / 72))
  const labelCount = Math.min(pointCount, maxLabelCount)
  return Array.from({ length: labelCount }, (_, labelIndex) => (
    Math.round(labelIndex * (pointCount - 1) / (labelCount - 1))
  )).includes(index)
}

watch(chartHost, (element, previousElement) => {
  if (previousElement) chartResizeObserver?.unobserve(previousElement)
  if (!element || typeof ResizeObserver === 'undefined') return
  chartResizeObserver ??= new ResizeObserver(([entry]) => {
    if (entry?.contentRect.width) chartWidth.value = Math.floor(entry.contentRect.width)
  })
  if (element.clientWidth) chartWidth.value = Math.floor(element.clientWidth)
  chartResizeObserver.observe(element)
}, { flush: 'post' })

onMounted(() => {
  void Promise.all([loadOverview(), loadRangeReports()])
})

onBeforeUnmount(() => chartResizeObserver?.disconnect())
</script>

<template>
  <section class="reports">
    <PageHeader
      title="本店报表"
      description="累计经营概览，以及可按时间筛选的收款、商品和活动分析"
    />

    <NSpin :show="overviewLoading">
      <section
        class="summary-section"
        aria-label="累计经营概览"
      >
        <header class="section-heading">
          <div>
            <h2>累计经营概览</h2>
            <p>累计口径不受下方时间筛选影响</p>
          </div>
        </header>
        <div class="summary-strip">
          <article
            v-for="metric in summaryMetrics"
            :key="metric.key"
          >
            <span>{{ metric.label }}</span>
            <strong>{{ metric.value }}</strong>
          </article>
        </div>
        <NAlert
          v-if="overviewError"
          type="error"
          :show-icon="false"
          class="reports__alert"
        >
          {{ overviewError }}
        </NAlert>
      </section>
    </NSpin>

    <section
      class="report-controls"
      aria-label="报表时间筛选"
    >
      <div class="report-controls__copy">
        <strong>分析时间</strong>
        <span>{{ rangeLabel }}，筛选同步作用于下方三个分析页面</span>
      </div>
      <div class="report-controls__actions">
        <div
          class="range-presets"
          aria-label="快捷时间范围"
        >
          <NButton
            v-for="option in rangeOptions"
            :key="option.value"
            size="small"
            :type="quickRange === option.value ? 'primary' : 'default'"
            :secondary="quickRange !== option.value"
            :aria-pressed="quickRange === option.value"
            @click="setQuickRange(option.value as RangeKey)"
          >
            {{ option.label }}
          </NButton>
        </div>
        <NDatePicker
          class="desktop-date-range"
          :value="dateRange"
          type="daterange"
          format="yyyy-MM-dd"
          :is-date-disabled="disableFutureDate"
          :clearable="false"
          aria-label="选择自定义分析时间"
          @update:value="updateCustomRange"
        />
        <form
          class="mobile-date-range"
          @submit.prevent="applyMobileRange"
        >
          <label>
            <span>开始日期</span>
            <input
              v-model="mobileRangeStart"
              type="date"
              required
              :max="formatDateKey(new Date())"
              :aria-invalid="!!rangeError"
              :aria-describedby="rangeError ? 'report-range-error' : undefined"
            >
          </label>
          <label>
            <span>结束日期</span>
            <input
              v-model="mobileRangeEnd"
              type="date"
              required
              :min="mobileRangeStart"
              :max="formatDateKey(new Date())"
              :aria-invalid="!!rangeError"
              :aria-describedby="rangeError ? 'report-range-error' : undefined"
            >
          </label>
          <NButton attr-type="submit">
            应用日期
          </NButton>
        </form>
      </div>
    </section>

    <NAlert
      v-if="rangeError"
      id="report-range-error"
      type="warning"
      :show-icon="false"
      class="reports__alert"
    >
      {{ rangeError }}
    </NAlert>

    <NTabs
      v-model:value="tab"
      type="line"
      class="reports__tabs"
    >
      <NTabPane
        name="revenue"
        tab="收款趋势"
      >
        <section class="analysis-panel">
          <header class="analysis-panel__header">
            <div>
              <h2>收款趋势</h2>
              <p>{{ rangeLabel }} · 按支付完成日期汇总，人民币与金币分开统计</p>
            </div>
            <NRadioGroup
              v-model:value="paymentChannel"
              size="small"
              name="payment-channel"
            >
              <NRadioButton value="wechat">
                微信支付
              </NRadioButton>
              <NRadioButton value="coin">
                金币消费
              </NRadioButton>
            </NRadioGroup>
          </header>

          <div class="channel-summary">
            <span>
              <small>微信实收</small>
              <strong>{{ formatCent(wechatTotalCent) }}</strong>
              <em>{{ formatCount(wechatOrderCount) }} 单</em>
            </span>
            <span>
              <small>金币消费</small>
              <strong>{{ formatCoins(coinTotal) }}</strong>
              <em>{{ formatCount(coinOrderCount) }} 单</em>
            </span>
          </div>

          <NSkeleton
            v-if="revenueLoading"
            height="300px"
            :sharp="false"
            class="analysis-panel__state"
          />
          <NAlert
            v-else-if="revenueError"
            type="error"
            :show-icon="false"
            class="analysis-panel__state"
          >
            <div class="state-error">
              <span>{{ revenueError }}</span>
              <NButton
                size="small"
                @click="loadRangeReports"
              >
                重试
              </NButton>
            </div>
          </NAlert>
          <div
            v-else-if="hasRevenue"
            ref="chartHost"
            class="trend-chart-scroll"
            :class="`trend-chart-scroll--${paymentChannel}`"
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
                :aria-label="`${rangeLabel} ${channelLabel}趋势，合计 ${formatChannelValue(channelTotal)}，${channelOrderCount} 单`"
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
                    {{ formatCompactChannelValue(tick.value) }}
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
                  :aria-label="`${point.date}，${channelLabel} ${formatChannelValue(point.value)}，${point.channelOrders} 单`"
                  @mouseenter="activePointIndex = index"
                  @focus="selectChartPoint(index)"
                  @blur="activePointIndex = null"
                  @click="selectChartPoint(index)"
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
                    :x="point.x - (chartWidth - chartPadding.left - chartPadding.right) / Math.max(1, chartPoints.length - 1) / 2"
                    :y="chartPadding.top"
                    :width="(chartWidth - chartPadding.left - chartPadding.right) / Math.max(1, chartPoints.length - 1)"
                    :height="chartHeight - chartPadding.top - chartPadding.bottom"
                  />
                  <circle
                    class="trend-chart__point"
                    :class="{ 'trend-chart__point--zero': point.value === 0 }"
                    :cx="point.x"
                    :cy="point.y"
                    :r="activePointIndex === index ? 5 : point.value === 0 ? 2.5 : 3.5"
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
                <strong>{{ formatChannelValue(activePoint.value) }}</strong>
                <small>{{ activePoint.channelOrders }} 单{{ channelLabel }}</small>
              </div>
            </div>
            <div
              v-if="mobilePoint"
              class="mobile-chart-detail"
            >
              <label
                v-if="chartPoints.length > 1"
                for="report-trend-date"
              >拖动查看每日明细</label>
              <input
                v-if="chartPoints.length > 1"
                id="report-trend-date"
                type="range"
                min="0"
                :max="chartPoints.length - 1"
                :value="selectedPointIndex ?? chartPoints.length - 1"
                :aria-valuetext="`${mobilePoint.date}，${formatChannelValue(mobilePoint.value)}，${mobilePoint.channelOrders} 单`"
                @input="selectMobilePoint"
              >
              <div class="mobile-chart-detail__value">
                <span>{{ mobilePoint.date }} · {{ channelLabel }}</span>
                <strong>{{ formatChannelValue(mobilePoint.value) }}</strong>
                <span>{{ formatCount(mobilePoint.channelOrders) }} 单</span>
              </div>
            </div>
          </div>
          <EmptyState
            v-else
            :description="`所选时间范围内暂无${channelLabel}记录`"
          />
        </section>
      </NTabPane>

      <NTabPane
        name="items"
        tab="热门商品"
      >
        <section class="analysis-panel">
          <header class="analysis-panel__header">
            <div>
              <h2>热门商品</h2>
              <p>{{ rangeLabel }} · 按销售流水排序，展示前 20 项</p>
            </div>
            <strong>{{ catalogRows.length }} 项</strong>
          </header>
          <NSkeleton
            v-if="catalogLoading"
            height="260px"
            :sharp="false"
            class="analysis-panel__state"
          />
          <NAlert
            v-else-if="catalogError"
            type="error"
            :show-icon="false"
            class="analysis-panel__state"
          >
            <div class="state-error">
              <span>{{ catalogError }}</span>
              <NButton
                size="small"
                @click="loadRangeReports"
              >
                重试
              </NButton>
            </div>
          </NAlert>
          <div
            v-else-if="catalogRows.length"
            class="rank-table"
          >
            <div class="rank-table__head">
              <span>排名</span><span>商品</span><span>销量</span><span>销售流水</span>
            </div>
            <div
              v-for="(row, index) in catalogRows"
              :key="row.itemId"
              class="rank-table__row"
            >
              <span>{{ String(index + 1).padStart(2, '0') }}</span>
              <strong>{{ row.itemName }}</strong>
              <span><small class="rank-table__label">销量</small>{{ formatCount(row.soldQty) }} 份</span>
              <span><small class="rank-table__label">销售流水</small>{{ formatCent(row.grossCent) }}</span>
            </div>
          </div>
          <EmptyState
            v-else
            description="所选时间范围内暂无商品销售"
          />
        </section>
      </NTabPane>

      <NTabPane
        name="activities"
        tab="活动经营"
      >
        <section class="analysis-panel">
          <header class="analysis-panel__header">
            <div>
              <h2>活动经营</h2>
              <p>{{ rangeLabel }} · 按订单数量排序，展示前 20 项</p>
            </div>
            <strong>{{ activityRows.length }} 项</strong>
          </header>
          <NSkeleton
            v-if="activityLoading"
            height="260px"
            :sharp="false"
            class="analysis-panel__state"
          />
          <NAlert
            v-else-if="activityError"
            type="error"
            :show-icon="false"
            class="analysis-panel__state"
          >
            <div class="state-error">
              <span>{{ activityError }}</span>
              <NButton
                size="small"
                @click="loadRangeReports"
              >
                重试
              </NButton>
            </div>
          </NAlert>
          <div
            v-else-if="activityRows.length"
            class="rank-table"
          >
            <div class="rank-table__head">
              <span>排名</span><span>活动</span><span>订单数</span><span>售票数</span>
            </div>
            <div
              v-for="(row, index) in activityRows"
              :key="row.activityId"
              class="rank-table__row"
            >
              <span>{{ String(index + 1).padStart(2, '0') }}</span>
              <strong>{{ row.activityName }}</strong>
              <span><small class="rank-table__label">订单数</small>{{ formatCount(row.orderCount) }} 单</span>
              <span><small class="rank-table__label">售票数</small>{{ formatCount(row.ticketCount) }} 张</span>
            </div>
          </div>
          <EmptyState
            v-else
            description="所选时间范围内暂无活动订单"
          />
        </section>
      </NTabPane>
    </NTabs>
  </section>
</template>

<style scoped>
.reports {
  min-width: 0;
  max-width: 1480px;
}

.mobile-date-range,
.mobile-chart-detail,
.rank-table__label {
  display: none;
}

.summary-section {
  padding: var(--ic-space-2) 0 var(--ic-space-5);
}

.section-heading,
.analysis-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ic-space-5);
}

.section-heading {
  margin-bottom: var(--ic-space-3);
}

.section-heading h2,
.analysis-panel__header h2 {
  margin: 0;
  color: var(--ic-color-text);
  font-size: var(--ic-font-md);
  font-weight: 650;
}

.section-heading p,
.analysis-panel__header p {
  margin: var(--ic-space-1) 0 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}

.summary-strip {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  border-top: var(--ic-divider);
  border-bottom: var(--ic-divider);
}

.summary-strip article {
  min-width: 0;
  padding: var(--ic-space-4) var(--ic-space-4) var(--ic-space-4) 0;
}

.summary-strip article + article {
  padding-left: var(--ic-space-4);
  border-left: var(--ic-divider);
}

.summary-strip span,
.summary-strip strong {
  display: block;
}

.summary-strip span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.summary-strip strong {
  margin-top: var(--ic-space-2);
  color: var(--ic-color-text);
  font-size: var(--ic-font-lg);
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}

.report-controls {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-5);
  padding: var(--ic-space-3) 0;
  border-top: var(--ic-divider);
  border-bottom: var(--ic-divider);
}

.report-controls__copy {
  min-width: 0;
}

.report-controls__copy strong,
.report-controls__copy span {
  display: block;
}

.report-controls__copy strong {
  font-size: var(--ic-font-sm);
}

.report-controls__copy span {
  margin-top: 3px;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.report-controls__actions,
.range-presets {
  display: flex;
  align-items: center;
  gap: var(--ic-space-2);
}

.report-controls__actions :deep(.n-date-picker) {
  width: 260px;
}

.reports__alert {
  margin-top: var(--ic-space-3);
}

.reports__tabs {
  margin-top: var(--ic-space-5);
}

.reports__tabs :deep(.n-tabs-nav) {
  min-height: 48px;
  border-bottom: var(--ic-divider);
}

.reports__tabs :deep(.n-tabs-nav-scroll-content) {
  gap: var(--ic-space-1);
}

.reports__tabs :deep(.n-tabs-tab) {
  padding: 0 var(--ic-space-4);
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
  transition: color 160ms ease-out, background-color 160ms ease-out;
}

.reports__tabs :deep(.n-tabs-tab:hover) {
  color: var(--ic-color-text);
  background: var(--ic-color-surface-muted);
}

.reports__tabs :deep(.n-tabs-tab--active) {
  color: var(--ic-color-text);
  font-weight: 600;
}

.analysis-panel {
  padding-top: var(--ic-space-5);
}

.analysis-panel__header {
  padding-bottom: var(--ic-space-4);
  border-bottom: var(--ic-divider);
}

.analysis-panel__header > strong {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.channel-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border-bottom: var(--ic-divider);
}

.channel-summary > span {
  display: grid;
  grid-template-columns: minmax(96px, 1fr) auto auto;
  gap: var(--ic-space-3);
  align-items: baseline;
  padding: var(--ic-space-3) 0;
}

.channel-summary > span + span {
  padding-left: var(--ic-space-5);
  border-left: var(--ic-divider);
}

.channel-summary small,
.channel-summary em {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  font-style: normal;
}

.channel-summary strong {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: var(--ic-font-md);
  font-variant-numeric: tabular-nums;
}

.analysis-panel__state {
  margin-top: var(--ic-space-5);
}

.state-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-4);
}

.trend-chart-scroll {
  --chart-color: var(--ic-color-info);
  --chart-area: color-mix(in srgb, var(--ic-color-info) 10%, transparent);
  overflow-x: auto;
  margin-top: var(--ic-space-4);
}

.trend-chart-scroll--coin {
  --chart-color: var(--ic-color-primary);
  --chart-area: color-mix(in srgb, var(--ic-color-primary) 7%, transparent);
}

.trend-chart-stage {
  position: relative;
  height: 300px;
  max-width: 100%;
}

.trend-chart {
  display: block;
  max-width: 100%;
  overflow: hidden;
}

.trend-chart__grid line {
  stroke: var(--ic-color-border);
  stroke-width: 1;
  shape-rendering: crispEdges;
}

.trend-chart__grid text,
.trend-chart__date {
  fill: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  font-variant-numeric: tabular-nums;
}

.trend-chart__area {
  fill: var(--chart-area);
  pointer-events: none;
}

.trend-chart__line {
  fill: none;
  stroke: var(--chart-color);
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
  fill: var(--chart-color);
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

.rank-table {
  overflow-x: auto;
}

.rank-table__head,
.rank-table__row {
  display: grid;
  min-width: 680px;
  grid-template-columns: 72px minmax(260px, 1fr) 140px 160px;
  gap: var(--ic-space-4);
  align-items: center;
  padding: 13px var(--ic-space-2);
  border-bottom: var(--ic-divider);
}

.rank-table__head {
  color: var(--ic-color-text-secondary);
  background: var(--ic-color-surface-muted);
  font-size: var(--ic-font-xs);
}

.rank-table__row {
  font-size: var(--ic-font-sm);
  font-variant-numeric: tabular-nums;
}

.rank-table__row > span:first-child {
  color: var(--ic-color-text-tertiary);
}

.rank-table__row strong {
  overflow: hidden;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1100px) {
  .summary-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .summary-strip article:nth-child(4) {
    padding-left: 0;
    border-left: 0;
  }

  .summary-strip article:nth-child(n + 4) {
    border-top: var(--ic-divider);
  }
}

@media (max-width: 860px) {
  .report-controls,
  .analysis-panel__header {
    align-items: stretch;
    flex-direction: column;
  }

  .report-controls__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .report-controls__actions :deep(.n-date-picker) {
    width: 100%;
  }

  .channel-summary {
    grid-template-columns: 1fr;
  }

  .channel-summary > span + span {
    padding-left: 0;
    border-top: var(--ic-divider);
    border-left: 0;
  }
}

@media (max-width: 640px) {
  .summary-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .summary-strip article:nth-child(odd) {
    padding-left: 0;
    border-left: 0;
  }

  .summary-strip article:nth-child(even) {
    padding-left: var(--ic-space-3);
    border-left: var(--ic-divider);
  }

  .summary-strip article:nth-child(n + 3) {
    border-top: var(--ic-divider);
  }

  .range-presets {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
  }

  .channel-summary > span {
    grid-template-columns: 1fr minmax(0, 2fr);
  }

  .channel-summary strong {
    text-align: right;
  }

  .channel-summary em {
    grid-column: 1 / -1;
  }
}

@media (max-width: 768px) {
  .report-controls__actions .desktop-date-range {
    display: none;
  }

  .range-presets {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .range-presets :deep(.n-button),
  .mobile-date-range :deep(.n-button) {
    min-height: 44px;
  }

  .mobile-date-range {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--ic-space-3);
    min-width: 0;
  }

  .mobile-date-range label {
    display: grid;
    gap: var(--ic-space-1);
    min-width: 0;
    color: var(--ic-color-text-secondary);
    font-size: var(--ic-font-xs);
  }

  .mobile-date-range input {
    box-sizing: border-box;
    width: 100%;
    min-width: 0;
    min-height: 44px;
    padding: 8px;
    border: 1px solid var(--ic-color-border-strong);
    border-radius: var(--ic-radius-sm);
    background: var(--ic-color-surface);
    color: var(--ic-color-text);
    font: inherit;
    font-size: 16px;
    color-scheme: light;
  }

  .mobile-date-range input:focus-visible,
  .mobile-chart-detail input:focus-visible {
    outline: 2px solid var(--ic-color-primary);
    outline-offset: 2px;
  }

  .mobile-date-range :deep(.n-button) {
    grid-column: 1 / -1;
  }

  .reports__tabs :deep(.n-tabs-tab) {
    padding-inline: var(--ic-space-3);
  }

  .analysis-panel__header :deep(.n-radio-group) {
    display: flex;
  }

  .analysis-panel__header :deep(.n-radio-button) {
    flex: 1;
    min-height: 44px;
    line-height: 42px;
    text-align: center;
  }

  .trend-chart__tooltip {
    display: none;
  }

  .mobile-chart-detail {
    display: grid;
    gap: var(--ic-space-2);
    padding-bottom: var(--ic-space-3);
    color: var(--ic-color-text-secondary);
    font-size: var(--ic-font-xs);
  }

  .mobile-chart-detail input {
    width: 100%;
    min-width: 0;
    height: 44px;
    margin: 0;
    accent-color: var(--ic-color-primary);
  }

  .mobile-chart-detail__value {
    display: grid;
    gap: var(--ic-space-1);
    padding-top: var(--ic-space-3);
    border-top: var(--ic-divider);
    font-variant-numeric: tabular-nums;
  }

  .mobile-chart-detail__value strong {
    color: var(--ic-color-text);
    font-size: var(--ic-font-md);
    overflow-wrap: anywhere;
  }

  .rank-table__head {
    display: none;
  }

  .rank-table__row {
    min-width: 0;
    grid-template-columns: 28px minmax(0, 1fr);
    gap: var(--ic-space-2) var(--ic-space-3);
    padding: var(--ic-space-4) 0;
    align-items: baseline;
  }

  .rank-table__row strong {
    overflow: visible;
    white-space: normal;
    overflow-wrap: anywhere;
  }

  .rank-table__row > span:nth-child(n + 3) {
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: var(--ic-space-1) var(--ic-space-3);
    grid-column: 2;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .rank-table__label {
    display: inline;
    color: var(--ic-color-text-secondary);
    font-size: var(--ic-font-xs);
  }
}

@media (max-width: 480px) {
  .summary-strip {
    grid-template-columns: minmax(0, 1fr);
  }

  .summary-strip article:nth-child(n) {
    display: grid;
    grid-template-columns: minmax(88px, 1fr) minmax(0, 2fr);
    align-items: baseline;
    gap: var(--ic-space-3);
    padding: var(--ic-space-3) 0;
    border-left: 0;
    border-top: 0;
  }

  .summary-strip article:nth-child(n + 2) {
    border-top: var(--ic-divider);
  }

  .summary-strip strong {
    margin-top: 0;
    text-align: right;
    font-size: var(--ic-font-md);
  }
}

@media (prefers-reduced-motion: reduce) {
  .reports__tabs :deep(.n-tabs-tab),
  .trend-chart__point {
    transition: none;
  }
}
</style>
