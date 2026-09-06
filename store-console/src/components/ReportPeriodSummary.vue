<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NAlert, NButton, NSkeleton } from 'naive-ui'
import { formatCent } from '@/utils/format'
import { periodDays, reportPreset, reportToday, type ReportPeriod, type ReportPreset } from '@/utils/report-period'

const props = defineProps<{
  modelValue: ReportPeriod
  grossCent: number
  wechatRevenueCent: number
  coinConsumption: number
  orderCount: number
  loading: boolean
  error?: string | null
}>()
const emit = defineEmits<{
  'update:modelValue': [period: ReportPeriod]
  retry: []
}>()
const presets: Array<{ label: string; value: ReportPreset }> = [
  { label: '今天', value: 'today' },
  { label: '昨天', value: 'yesterday' },
  { label: '上个月', value: 'lastMonth' },
]
const customOpen = ref(false)
const draftFrom = ref(props.modelValue.from)
const draftTo = ref(props.modelValue.to)
const rangeError = ref('')
const countFormatter = new Intl.NumberFormat('zh-CN')
const rangeLabel = computed(() => props.modelValue.from === props.modelValue.to
  ? props.modelValue.from.replaceAll('-', '/')
  : `${props.modelValue.from.replaceAll('-', '/')} - ${props.modelValue.to.replaceAll('-', '/')}`)
const activePreset = computed(() => presets.find(({ value }) => {
  const period = reportPreset(value)
  return period.from === props.modelValue.from && period.to === props.modelValue.to
})?.value)

watch(() => props.modelValue, (period) => {
  draftFrom.value = period.from
  draftTo.value = period.to
})

function selectPreset(preset: ReportPreset): void {
  customOpen.value = false
  rangeError.value = ''
  emit('update:modelValue', reportPreset(preset))
}

function applyCustom(): void {
  const period = { from: draftFrom.value, to: draftTo.value }
  const days = periodDays(period)
  if (!Number.isFinite(days) || days < 1 || period.to > reportToday()) {
    rangeError.value = '请选择有效日期，开始日期不能晚于结束日期，且不能晚于今天。'
    return
  }
  if (days > 90) {
    rangeError.value = '自定义时间范围最多支持 90 天。'
    return
  }
  rangeError.value = ''
  customOpen.value = false
  emit('update:modelValue', period)
}
</script>

<template>
  <section
    class="period-summary"
    aria-label="经营数据汇总"
  >
    <div
      class="period-presets"
      aria-label="报表日期"
    >
      <NButton
        v-for="preset in presets"
        :key="preset.value"
        :type="activePreset === preset.value ? 'primary' : 'default'"
        :aria-pressed="activePreset === preset.value"
        @click="selectPreset(preset.value)"
      >
        {{ preset.label }}
      </NButton>
      <NButton
        :type="!activePreset ? 'primary' : 'default'"
        :aria-expanded="customOpen"
        @click="customOpen = !customOpen"
      >
        自定义
      </NButton>
    </div>
    <form
      v-if="customOpen"
      class="period-custom"
      @submit.prevent="applyCustom"
    >
      <label>
        <span>开始日期</span>
        <input
          v-model="draftFrom"
          type="date"
          required
          :max="reportToday()"
        >
      </label>
      <label>
        <span>结束日期</span>
        <input
          v-model="draftTo"
          type="date"
          required
          :min="draftFrom"
          :max="reportToday()"
        >
      </label>
      <p
        v-if="rangeError"
        role="alert"
      >
        {{ rangeError }}
      </p>
      <NButton
        attr-type="submit"
        type="primary"
      >
        应用日期
      </NButton>
    </form>
    <div class="period-label">
      <time>{{ rangeLabel }}</time>
      <span>上海时间</span>
    </div>
    <NAlert
      v-if="error && !loading"
      type="error"
      :show-icon="false"
    >
      <div class="period-error">
        <span>{{ error }}</span>
        <NButton
          size="small"
          @click="emit('retry')"
        >
          重试
        </NButton>
      </div>
    </NAlert>
    <div
      class="period-metrics"
      :aria-busy="loading"
      aria-live="polite"
    >
      <article class="period-metric period-metric--gross">
        <span>营业额 <small>（元）</small></span>
        <NSkeleton
          v-if="loading"
          height="36px"
          width="70%"
        />
        <strong v-else>{{ error ? '—' : formatCent(grossCent) }}</strong>
        <small>{{ error || loading ? '已支付订单' : `已支付 ${countFormatter.format(orderCount)} 单` }}</small>
      </article>
      <article class="period-metric period-metric--wechat">
        <span>微信实收 <small>（元）</small></span>
        <NSkeleton
          v-if="loading"
          height="28px"
        />
        <strong
          v-else
          :class="{ 'period-value--long': formatCent(wechatRevenueCent).length > 10 }"
        >{{ error ? '—' : formatCent(wechatRevenueCent) }}</strong>
      </article>
      <article class="period-metric period-metric--coin">
        <span>金币消费 <small>（个）</small></span>
        <NSkeleton
          v-if="loading"
          height="28px"
        />
        <strong v-else>{{ error ? '—' : countFormatter.format(coinConsumption) }}</strong>
      </article>
    </div>
  </section>
</template>

<style scoped>
.period-summary {
  min-width: 0;
  margin-bottom: 24px;
}

.period-presets {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  max-width: 560px;
}

.period-presets :deep(.n-button) {
  min-height: 44px;
  padding-inline: 8px;
}

.period-label {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin: 12px 0;
  font-size: 12px;
  color: var(--ic-color-text-secondary);
}

.period-metrics {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr;
  gap: 12px;
  margin-top: 12px;
}

.period-metric {
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 12px;
  padding: 20px;
  border-radius: 8px;
  background: var(--ic-color-surface);
}

.period-metric > span {
  font-size: 14px;
  color: var(--ic-color-text-secondary);
}

.period-metric small {
  font-size: 12px;
}

.period-metric > strong {
  font-size: 24px;
  line-height: 1.3;
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}

.period-metric--gross {
  background: var(--ic-color-primary);
  color: #fff;
}

.period-metric--gross > span, .period-metric--gross > small {
  color: #e0e0e0;
}

.period-metric--gross > strong {
  font-size: 32px;
}

.period-metric--wechat > strong {
  color: #247344;
}

.period-custom {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding-top: 12px;
  max-width: 560px;
}

.period-custom label {
  display: grid;
  gap: 6px;
  min-width: 0;
  font-size: 13px;
  color: var(--ic-color-text-secondary);
}

.period-custom input {
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  min-height: 44px;
  padding: 8px;
  border: 1px solid var(--ic-color-border);
  border-radius: 4px;
  background: var(--ic-color-surface);
  color: var(--ic-color-text);
  font: inherit;
  font-size: 16px;
  color-scheme: light;
}

.period-custom input:focus-visible {
  outline: 2px solid var(--ic-color-primary);
  outline-offset: 2px;
}

.period-custom > p, .period-custom > :deep(.n-button) {
  grid-column: 1 / -1;
  margin: 0;
  min-height: 44px;
}

.period-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

@media (max-width: 768px) {

  .period-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .period-metric {
    padding: 16px;
    gap: 10px;
    min-height: 76px;
  }

  .period-metric--gross {
    grid-column: 1 / -1;
  }

  .period-metric > strong {
    font-size: 22px;
  }

  .period-metric--gross > strong {
    font-size: 30px;
  }

  .period-metric > .period-value--long {
    font-size: 18px;
  }
}
</style>
