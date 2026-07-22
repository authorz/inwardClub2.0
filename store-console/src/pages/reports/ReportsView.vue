<script setup lang="ts">
/**
 * 本店报表：本店经营概览与分项报表入口（仅本店范围，无跨店维度）。
 * 趋势图表待服务端报表接口就绪后接入。
 */
import { onMounted, ref } from 'vue'
import { NButton, NSpin, NTabPane, NTabs } from 'naive-ui'
import { reportService } from '@/api/services'
import { ApiError } from '@/api/error'
import { formatCent } from '@/utils/format'
import { EmptyState, MetricTile, PageHeader } from '@/components/common'
import type { ReportOverview } from '@/types/models'

const loading = ref(false)
const overview = ref<ReportOverview | null>(null)
const errorMsg = ref<string | null>(null)
const range = ref<'today' | '7d' | '30d'>('7d')

const rangeOptions = [
  { label: '今日', value: 'today' },
  { label: '近 7 天', value: '7d' },
  { label: '近 30 天', value: '30d' },
]

async function load() {
  loading.value = true
  errorMsg.value = null
  try {
    overview.value = await reportService.overview()
  } catch (err) {
    errorMsg.value = err instanceof ApiError ? err.message : '报表加载失败'
  } finally {
    loading.value = false
  }
}

function setRange(value: 'today' | '7d' | '30d') {
  range.value = value
  void load()
}

onMounted(load)
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
          @click="setRange(opt.value as 'today' | '7d' | '30d')"
        >
          {{ opt.label }}
        </NButton>
      </template>
    </PageHeader>

    <NSpin :show="loading">
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
        v-if="errorMsg"
        class="ic-muted reports__hint"
      >
        {{ errorMsg }}（等待服务端接口就绪）
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
        <EmptyState description="收款趋势图表待服务端 /store/reports/revenue 接口就绪后接入" />
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
</style>
