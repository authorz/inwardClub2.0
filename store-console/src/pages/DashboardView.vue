<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NSkeleton } from 'naive-ui'
import { useRouter } from 'vue-router'
import { reportService } from '@/api/services'
import type {
  ReportOverview,
  ReportOverviewBreakdown,
  ReportOverviewTrendPoint,
} from '@/types/models'
import { ApiError } from '@/api/error'
import { PermissionButton } from '@/components/common'
import { PERM } from '@/constants/permissions'

const router = useRouter()
const overview = ref<ReportOverview | null>(null)
const loading = ref(true)
const errorMessage = ref('')
const updatedAt = ref('')

const countFormatter = new Intl.NumberFormat('zh-CN')
const currencyFormatter = new Intl.NumberFormat('zh-CN', {
  style: 'currency',
  currency: 'CNY',
  minimumFractionDigits: 2,
})

const quickEntries = [
  { label: '本店订单', path: '/orders' },
  { label: '支付与退款', path: '/payments' },
  { label: '会员管理', path: '/members' },
  { label: '本店报表', path: '/reports' },
]

const metricIcons: Record<string, string[]> = {
  revenue: ['M7 4l5 8 5-8', 'M6 13h12', 'M6 17h12', 'M12 12v8'],
  offlineCollection: ['M7 4l5 8 5-8', 'M6 13h12', 'M6 17h12', 'M12 12v8'],
  members: [
    'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2',
    'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8',
    'M22 21v-2a4 4 0 0 0-3-3.87',
    'M16 3.13a4 4 0 0 1 0 7.75',
  ],
  orders: [
    'M6 3h12a1 1 0 0 1 1 1v17l-3-2-4 2-4-2-3 2V4a1 1 0 0 1 1-1Z',
    'M9 8h6',
    'M9 12h6',
    'M9 16h3',
  ],
  todayRevenue: ['M7 4l5 8 5-8', 'M6 13h12', 'M6 17h12', 'M12 12v8'],
  todayOrders: [
    'M6 3h12a1 1 0 0 1 1 1v17l-3-2-4 2-4-2-3 2V4a1 1 0 0 1 1-1Z',
    'M9 8h6',
    'M9 12h6',
    'M9 16h3',
  ],
  todayMembers: [
    'M15 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2',
    'M8.5 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8',
    'M19 8v6',
    'M16 11h6',
  ],
  activityRevenue: [
    'M3 8a2 2 0 0 0 0 4v7h18v-7a2 2 0 0 0 0-4V5H3v3Z',
    'M13 5v2',
    'M13 10v2',
    'M13 15v2',
  ],
}

const summaryMetrics = computed(() => [
  {
    key: 'revenue',
    label: '本店营业收入',
    value: formatCurrency(overview.value?.wechatRevenue?.total),
    detail: '仅统计微信实收',
    primary: true,
  },
  {
    key: 'offlineCollection',
    label: '线下收款',
    value: formatCurrency(overview.value?.offlineCollectionRevenueCent),
    detail: '本店累计已支付线下聚合收款',
    actionPath: '/collection',
  },
  {
    key: 'members',
    label: '总用户数量',
    value: formatCount(overview.value?.memberCount),
    detail: '平台累计注册用户（全部门店）',
  },
  {
    key: 'orders',
    label: '本店订单数量',
    value: formatCount(overview.value?.orderCount),
    detail: '累计已支付订单',
  },
])

const todayMetrics = computed(() => [
  {
    key: 'todayRevenue',
    label: '今日收入',
    value: formatCurrency(overview.value?.todayGrossSalesCent),
    note: '微信实收',
    tone: 'green',
  },
  {
    key: 'todayOrders',
    label: '今日订单',
    value: formatCount(overview.value?.todayOrderCount),
    note: '已支付订单',
    tone: 'blue',
  },
  {
    key: 'todayMembers',
    label: '今日新注册用户',
    value: formatCount(overview.value?.todayNewMemberCount),
    note: '自然日新增',
    tone: 'amber',
  },
  {
    key: 'activityRevenue',
    label: '活动收入',
    value: formatCurrency(overview.value?.activityRevenueCent),
    note: `今日 ${formatCurrency(overview.value?.todayActivityRevenueCent)}`,
    tone: 'violet',
  },
])

const trend = computed<ReportOverviewTrendPoint[]>(() => overview.value?.trend ?? [])
const trendMax = computed(() => Math.max(0, ...trend.value.map((item) => item.wechatRevenueCent)))
const trendBars = computed(() =>
  trend.value.map((item) => ({
    ...item,
    height:
      trendMax.value === 0
        ? 0
        : Math.max(6, Math.round((item.wechatRevenueCent / trendMax.value) * 100)),
  })),
)

const compositionItems = computed(() => {
  const data = overview.value?.wechatRevenue
  const items = [
    { key: 'recharge', label: '快捷充值', value: data?.recharge ?? 0, color: '#1a1a1a' },
    { key: 'food', label: '餐品订单', value: data?.food ?? 0, color: '#3a6ea5' },
    { key: 'activity', label: '活动订单', value: data?.activity ?? 0, color: '#2f8f4e' },
  ]
  const total = items.reduce((sum, item) => sum + item.value, 0)
  return items.map((item) => ({ ...item, ratio: total > 0 ? item.value / total : 0 }))
})

const wechatRows = computed(() => breakdownRows(overview.value?.wechatRevenue, true))
const coinRows = computed(() => breakdownRows(overview.value?.coinConsumption, false))

function breakdownRows(data: ReportOverviewBreakdown | undefined, includeRecharge: boolean) {
  const rows = includeRecharge
    ? [
        {
          key: 'recharge',
          label: '快捷充值',
          value: data?.recharge ?? 0,
          today: data?.todayRecharge ?? 0,
        },
      ]
    : []
  rows.push(
    { key: 'food', label: '餐品订单', value: data?.food ?? 0, today: data?.todayFood ?? 0 },
    {
      key: 'activity',
      label: '活动订单',
      value: data?.activity ?? 0,
      today: data?.todayActivity ?? 0,
    },
  )
  return rows
}

function formatCount(value: number | null | undefined): string {
  return value == null ? '—' : countFormatter.format(value)
}

function formatCurrency(value: number | null | undefined): string {
  return value == null ? '—' : currencyFormatter.format(value / 100)
}

function formatCompactCurrency(valueCent: number): string {
  const yuan = valueCent / 100
  if (yuan >= 100_000_000) return `¥${(yuan / 100_000_000).toFixed(1)}亿`
  if (yuan >= 10_000) return `¥${(yuan / 10_000).toFixed(1)}万`
  return `¥${Math.round(yuan)}`
}

function formatDateLabel(date: string): string {
  return date.slice(5).replace('-', '/')
}

async function loadOverview(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    overview.value = await reportService.overview()
    updatedAt.value = new Intl.DateTimeFormat('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }).format(new Date())
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '经营数据加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadOverview)
</script>

<template>
  <section class="dashboard">
    <div class="dashboard-toolbar">
      <span>{{ updatedAt ? `数据更新于 ${updatedAt}` : '本店经营数据与近 7 日趋势' }}</span>
      <NButton
        size="small"
        :loading="loading"
        @click="loadOverview"
      >
        刷新
      </NButton>
    </div>

    <NAlert
      v-if="errorMessage"
      type="error"
      :show-icon="false"
      class="dashboard__error"
    >
      {{ errorMessage }}
    </NAlert>

    <div
      v-if="loading && !overview"
      class="summary-grid"
    >
      <NSkeleton
        v-for="index in 4"
        :key="index"
        height="132px"
        :sharp="false"
      />
    </div>

    <template v-else>
      <div class="summary-grid">
        <article
          v-for="metric in summaryMetrics"
          :key="metric.key"
          class="summary-card"
          :class="{ 'summary-card--primary': metric.primary }"
        >
          <div class="summary-card__heading">
            <span class="summary-card__label">{{ metric.label }}</span>
            <span class="metric-icon">
              <svg
                viewBox="0 0 24 24"
                aria-hidden="true"
              >
                <path
                  v-for="path in metricIcons[metric.key]"
                  :key="path"
                  :d="path"
                />
              </svg>
            </span>
          </div>
          <strong class="summary-card__value">{{ metric.value }}</strong>
          <div class="summary-card__footer">
            <span class="summary-card__detail">{{ metric.detail }}</span>
            <PermissionButton
              v-if="metric.actionPath"
              :permissions="[PERM.collectionCreate]"
              type="primary"
              @click="router.push(metric.actionPath)"
            >
              发起线下收款
            </PermissionButton>
          </div>
        </article>
      </div>

      <div class="today-strip">
        <article
          v-for="metric in todayMetrics"
          :key="metric.key"
          class="today-metric"
        >
          <span
            class="metric-icon today-metric__icon"
            :class="`today-metric__icon--${metric.tone}`"
          >
            <svg
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                v-for="path in metricIcons[metric.key]"
                :key="path"
                :d="path"
              />
            </svg>
          </span>
          <div>
            <span class="today-metric__label">{{ metric.label }}</span>
            <strong class="today-metric__value">{{ metric.value }}</strong>
            <span class="today-metric__note">{{ metric.note }}</span>
          </div>
        </article>
      </div>

      <article class="panel trend-panel">
        <header class="panel__header">
          <div>
            <h2>近 7 日经营趋势</h2>
            <p>柱高表示微信实收，柱顶显示当日已支付订单数</p>
          </div>
          <strong class="trend-panel__peak">峰值 {{ formatCompactCurrency(trendMax) }}</strong>
        </header>
        <div
          v-if="trendBars.length"
          class="bar-chart"
          role="img"
          aria-label="近七日微信收入和订单数量柱状图"
        >
          <div
            v-for="item in trendBars"
            :key="item.date"
            class="bar-chart__day"
            :aria-label="`${item.date}，微信实收 ${formatCurrency(item.wechatRevenueCent)}，${item.orderCount} 单`"
          >
            <span class="bar-chart__orders">{{ item.orderCount }} 单</span>
            <div class="bar-chart__rail">
              <i :style="{ height: `${item.height}%` }" />
            </div>
            <span class="bar-chart__date">{{ formatDateLabel(item.date) }}</span>
          </div>
        </div>
        <div
          v-else
          class="panel__empty"
        >
          暂无趋势数据
        </div>
      </article>

      <div class="breakdown-grid">
        <article class="panel breakdown-panel">
          <header class="panel__header panel__header--totals">
            <div>
              <h2>微信支付收入</h2>
              <p>真实资金流入，不包含金币消费</p>
            </div>
            <div class="panel-total">
              <span>总收入</span>
              <strong>{{ formatCurrency(overview?.wechatRevenue?.total) }}</strong>
              <small>今日 {{ formatCurrency(overview?.wechatRevenue?.today) }}</small>
            </div>
          </header>

          <div class="composition-bar">
            <i
              v-for="item in compositionItems"
              :key="item.key"
              :style="{ width: `${item.ratio * 100}%`, backgroundColor: item.color }"
            />
          </div>
          <div class="composition-legend">
            <span
              v-for="item in compositionItems"
              :key="item.key"
            >
              <i :style="{ backgroundColor: item.color }" />
              {{ item.label }} {{ (item.ratio * 100).toFixed(1) }}%
            </span>
          </div>

          <div class="breakdown-list">
            <div
              v-for="row in wechatRows"
              :key="row.key"
              class="breakdown-row"
            >
              <span>{{ row.label }}</span>
              <strong>{{ formatCurrency(row.value) }}</strong>
              <small>今日 {{ formatCurrency(row.today) }}</small>
            </div>
          </div>
        </article>

        <article class="panel breakdown-panel">
          <header class="panel__header panel__header--totals">
            <div>
              <h2>金币消费</h2>
              <p>按金币账本实际扣款统计</p>
            </div>
            <div class="panel-total">
              <span>总消费</span>
              <strong>{{ formatCount(overview?.coinConsumption?.total) }}</strong>
              <small>今日 {{ formatCount(overview?.coinConsumption?.today) }}</small>
            </div>
          </header>
          <div class="breakdown-list">
            <div
              v-for="row in coinRows"
              :key="row.key"
              class="breakdown-row"
            >
              <span>{{ row.label }}</span>
              <strong>{{ formatCount(row.value) }}</strong>
              <small>今日 {{ formatCount(row.today) }}</small>
            </div>
          </div>
        </article>
      </div>

      <nav
        class="quick-links"
        aria-label="常用功能"
      >
        <span>常用功能</span>
        <NButton
          v-for="entry in quickEntries"
          :key="entry.path"
          size="small"
          secondary
          @click="router.push(entry.path)"
        >
          {{ entry.label }}
        </NButton>
      </nav>
    </template>
  </section>
</template>

<style scoped>
.dashboard {
  max-width: 1480px;
}

.dashboard-toolbar {
  min-height: 28px;
  margin-bottom: 12px;
  display: flex;
  gap: var(--ic-space-4);
  align-items: center;
  justify-content: space-between;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.dashboard__error {
  margin-bottom: var(--ic-space-4);
}

.summary-grid,
.breakdown-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--ic-space-4);
}

.summary-card,
.panel {
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-lg);
}

.summary-card {
  min-height: 132px;
  padding: 22px;
  display: flex;
  flex-direction: column;
}

.summary-card__heading {
  display: flex;
  gap: var(--ic-space-4);
  align-items: center;
  justify-content: space-between;
}

.summary-card--primary {
  color: #fff;
  background: var(--ic-color-primary);
  border-color: var(--ic-color-primary);
}

.summary-card__label,
.today-metric__label,
.panel-total span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}

.summary-card--primary .summary-card__label,
.summary-card--primary .summary-card__detail {
  color: #cfcfcf;
}

.metric-icon {
  width: 34px;
  height: 34px;
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  color: var(--ic-color-text-secondary);
  background: var(--ic-color-surface-muted);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
}

.metric-icon svg {
  width: 18px;
  height: 18px;
  fill: none;
  stroke: currentcolor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 1.8;
}

.summary-card--primary .metric-icon {
  color: #fff;
  background: #262626;
  border-color: #404040;
}

.summary-card__value {
  margin-top: 12px;
  font-size: 30px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.025em;
}

.summary-card__footer {
  margin-top: auto;
  display: flex;
  gap: 12px;
  align-items: flex-end;
  justify-content: space-between;
}

.summary-card__detail {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.today-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-top: var(--ic-space-4);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-lg);
}

.today-metric {
  min-width: 0;
  padding: 18px 20px;
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.today-metric + .today-metric {
  border-left: 1px solid var(--ic-color-border);
}

.today-metric__icon {
  width: 32px;
  height: 32px;
}

.today-metric__icon--green {
  color: var(--ic-color-success);
}

.today-metric__icon--blue {
  color: var(--ic-color-info);
}

.today-metric__icon--amber {
  color: var(--ic-color-warning);
}

.today-metric__icon--violet {
  color: #735b8f;
}

.today-metric__label,
.today-metric__value,
.today-metric__note {
  display: block;
}

.today-metric__value {
  margin-top: 7px;
  overflow: hidden;
  font-size: var(--ic-font-lg);
  font-variant-numeric: tabular-nums;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.today-metric__note,
.panel__header p,
.panel-total small,
.breakdown-row small {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.today-metric__note {
  margin-top: 5px;
}

.panel {
  min-width: 0;
  padding: var(--ic-space-5);
}

.trend-panel {
  margin-top: var(--ic-space-4);
}

.panel__header {
  display: flex;
  gap: var(--ic-space-4);
  align-items: flex-start;
  justify-content: space-between;
}

.panel__header h2 {
  margin: 0;
  font-size: var(--ic-font-md);
  font-weight: 650;
}

.panel__header p {
  margin: 6px 0 0;
}

.trend-panel__peak {
  font-size: var(--ic-font-sm);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.bar-chart {
  height: 250px;
  margin-top: 20px;
  padding: 8px 18px 0;
  display: grid;
  grid-template-columns: repeat(7, minmax(36px, 1fr));
  gap: clamp(10px, 3vw, 36px);
  border-bottom: 1px solid var(--ic-color-border-strong);
}

.bar-chart__day {
  min-width: 0;
  height: 100%;
  display: grid;
  grid-template-rows: 20px 1fr 30px;
  text-align: center;
}

.bar-chart__orders,
.bar-chart__date {
  color: var(--ic-color-text-secondary);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.bar-chart__rail {
  min-height: 0;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.bar-chart__rail i {
  width: min(42px, 76%);
  display: block;
  background: var(--ic-color-primary);
  border-radius: 5px 5px 0 0;
  transition: height 0.2s ease-out;
}

.bar-chart__date {
  padding-top: 9px;
}

.panel__empty {
  min-height: 230px;
  display: grid;
  place-items: center;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}

.breakdown-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: var(--ic-space-4);
}

.panel__header--totals {
  padding-bottom: 20px;
  border-bottom: 1px solid var(--ic-color-border);
}

.panel-total {
  text-align: right;
}

.panel-total span,
.panel-total strong,
.panel-total small {
  display: block;
}

.panel-total strong {
  margin-top: 4px;
  font-size: var(--ic-font-lg);
  font-variant-numeric: tabular-nums;
}

.panel-total small {
  margin-top: 4px;
}

.composition-bar {
  height: 10px;
  margin-top: 20px;
  display: flex;
  overflow: hidden;
  background: var(--ic-color-border);
  border-radius: 5px;
}

.composition-bar i {
  height: 100%;
  display: block;
}

.composition-legend {
  margin-top: 10px;
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

.composition-legend span {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}

.composition-legend i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.breakdown-list {
  margin-top: 8px;
}

.breakdown-row {
  padding: 15px 0;
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 5px 16px;
  align-items: baseline;
}

.breakdown-row + .breakdown-row {
  border-top: 1px solid var(--ic-color-border);
}

.breakdown-row > span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}

.breakdown-row strong {
  font-size: var(--ic-font-base);
  font-variant-numeric: tabular-nums;
}

.breakdown-row small {
  grid-column: 2;
}

.quick-links {
  margin-top: var(--ic-space-4);
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.quick-links > span {
  margin-right: 4px;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

@media (max-width: 1100px) {
  .summary-grid,
  .today-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .today-metric:nth-child(3) {
    border-top: 1px solid var(--ic-color-border);
    border-left: 0;
  }

  .today-metric:nth-child(4) {
    border-top: 1px solid var(--ic-color-border);
  }
}

@media (max-width: 720px) {
  .summary-grid,
  .today-strip,
  .breakdown-grid {
    grid-template-columns: 1fr;
  }

  .today-metric + .today-metric {
    border-top: 1px solid var(--ic-color-border);
    border-left: 0;
  }

  .panel__header {
    flex-direction: column;
  }

  .panel-total {
    text-align: left;
  }

  .bar-chart {
    gap: 8px;
    padding-inline: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .bar-chart__rail i {
    transition: none;
  }
}
</style>
