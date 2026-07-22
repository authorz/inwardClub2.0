<script setup lang="ts">
/**
 * 今日概览：本店当日关键指标 + 待处理事项快捷入口。
 * 仅本店范围，不做跨店大屏。
 */
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NSpin } from 'naive-ui'
import { reportService } from '@/api/services'
import { ApiError } from '@/api/error'
import { formatCent } from '@/utils/format'
import { MetricTile, PageHeader } from '@/components/common'
import type { ReportOverview } from '@/types/models'

const router = useRouter()
const loading = ref(false)
const overview = ref<ReportOverview | null>(null)
const errorMsg = ref<string | null>(null)

async function load() {
  loading.value = true
  errorMsg.value = null
  try {
    overview.value = await reportService.overview()
  } catch (err) {
    // 后端未就绪时不阻塞页面，展示占位。
    errorMsg.value = err instanceof ApiError ? err.message : '概览数据加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(load)

const quickLinks = [
  { title: '点餐订单处理', name: 'food-orders' },
  { title: '活动核销', name: 'activity-verify' },
  { title: '积分审核', name: 'point-review' },
  { title: '线下聚合收款', name: 'collection' },
]
</script>

<template>
  <div>
    <PageHeader
      title="今日概览"
      description="本店当日经营与待办概况"
    >
      <template #actions>
        <NButton
          :loading="loading"
          @click="load"
        >
          刷新
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
        class="dashboard__hint ic-muted"
      >
        {{ errorMsg }}（等待服务端接口就绪）
      </p>
    </NSpin>

    <section class="quick">
      <h3 class="quick__title">
        快捷操作
      </h3>
      <div class="quick__grid">
        <button
          v-for="link in quickLinks"
          :key="link.name"
          class="quick__item ic-band"
          @click="router.push({ name: link.name })"
        >
          {{ link.title }}
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.metrics {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--ic-space-3);
}
.dashboard__hint {
  font-size: var(--ic-font-xs);
  margin-top: var(--ic-space-3);
}
.quick {
  margin-top: var(--ic-space-6);
}
.quick__title {
  font-size: var(--ic-font-md);
  margin-bottom: var(--ic-space-3);
}
.quick__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--ic-space-3);
}
.quick__item {
  padding: var(--ic-space-4);
  text-align: left;
  font-size: var(--ic-font-base);
  cursor: pointer;
  transition: border-color 0.15s;
}
.quick__item:hover {
  border-color: var(--ic-color-border-strong);
}
</style>
