<script setup lang="ts">
/**
 * 工作台（运营视角，非装饰型大屏）。
 * 展示关键运营指标概览与待处理入口。指标来自 /admin/reports/overview；
 * 服务端未就绪时展示占位，不影响其余功能。
 */
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NGrid, NGridItem, NThing } from 'naive-ui'
import PageHeader from '@/components/PageHeader.vue'
import { reportService } from '@/api/services'
import type { ReportOverview } from '@/api/models'
import type { NormalizedError } from '@/api/types'

const router = useRouter()

interface OverviewMetric {
  key: keyof ReportOverview
  label: string
  value: string
}

const metrics = ref<OverviewMetric[]>([
  { key: 'storeCount', label: '门店总数', value: '—' },
  { key: 'memberCount', label: '会员总数', value: '—' },
  { key: 'orderCount', label: '订单总数', value: '—' },
  { key: 'couponsRedeemed', label: '已核销券数', value: '—' },
])
const loaded = ref(false)
const errored = ref(false)

const quickEntries = [
  { label: '门店管理', path: '/stores' },
  { label: '订单中心', path: '/orders' },
  { label: '退款审批', path: '/payments/refunds' },
  { label: '审计日志', path: '/audit/logs' },
]

onMounted(async () => {
  try {
    const data = await reportService.overview()
    if (data) {
      metrics.value = metrics.value.map((m) => ({
        ...m,
        value: data[m.key] != null ? String(data[m.key]) : '—',
      }))
    }
    loaded.value = true
  } catch (e) {
    // 服务端未就绪属预期，仅标记，不打断页面
    errored.value = (e as NormalizedError).status !== 401
  }
})
</script>

<template>
  <section>
    <PageHeader
      title="工作台"
      description="总部运营与治理概览"
      :breadcrumb="['工作台']"
    />

    <NGrid
      :cols="4"
      :x-gap="16"
      :y-gap="16"
      responsive="screen"
    >
      <NGridItem
        v-for="m in metrics"
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

    <div class="quick">
      <h2 class="quick__title">
        快捷入口
      </h2>
      <NGrid
        :cols="4"
        :x-gap="16"
        :y-gap="16"
        responsive="screen"
      >
        <NGridItem
          v-for="q in quickEntries"
          :key="q.path"
        >
          <div
            class="quick__item"
            role="button"
            tabindex="0"
            @click="router.push(q.path)"
          >
            <NThing :title="q.label">
              <template #description>
                <span class="quick__desc">进入 {{ q.label }}</span>
              </template>
            </NThing>
          </div>
        </NGridItem>
      </NGrid>
    </div>

    <p
      v-if="errored"
      class="dashboard__notice"
    >
      指标接口尚未就绪（/admin/reports/overview），当前展示占位数据。
    </p>
  </section>
</template>

<style scoped>
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
.quick {
  margin-top: var(--ic-space-xl);
}
.quick__title {
  font-size: var(--ic-font-lg);
  font-weight: 600;
  margin-bottom: var(--ic-space-md);
}
.quick__item {
  padding: var(--ic-space-md) var(--ic-space-lg);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
  cursor: pointer;
  transition: border-color 0.15s ease;
}
.quick__item:hover {
  border-color: var(--ic-color-border-strong);
}
.quick__desc {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.dashboard__notice {
  margin-top: var(--ic-space-lg);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
</style>
