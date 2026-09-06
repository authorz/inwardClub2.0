<script setup lang="ts">
/**
 * 筛选区（公共组件，schema 驱动）。
 *
 * 通过 fields 配置渲染 input / select / daterange，
 * v-model 绑定一个筛选对象；点击查询 / 重置触发事件。
 * 所有列表页复用，不再各自手写筛选布局。
 */
import { NButton, NDatePicker, NInput, NSelect, NSpace } from 'naive-ui'
import { computed } from 'vue'
import type { FilterField } from './ui-types'

const props = defineProps<{
  fields: FilterField[]
  modelValue: Record<string, unknown>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
  search: []
  reset: []
}>()

function setValue(key: string, value: unknown): void {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function setRange(field: FilterField, value: [number, number] | null): void {
  const next = { ...props.modelValue }
  if (value) {
    next[`${field.key}From`] = new Date(value[0]).toISOString()
    next[`${field.key}To`] = new Date(value[1]).toISOString()
  } else {
    delete next[`${field.key}From`]
    delete next[`${field.key}To`]
  }
  emit('update:modelValue', next)
}

function rangeValue(field: FilterField): [number, number] | null {
  const from = props.modelValue[`${field.key}From`]
  const to = props.modelValue[`${field.key}To`]
  if (typeof from !== 'string' || typeof to !== 'string') return null
  const fromTime = Date.parse(from)
  const toTime = Date.parse(to)
  return Number.isNaN(fromTime) || Number.isNaN(toTime) ? null : [fromTime, toTime]
}

function dateValue(field: FilterField, boundary: 'From' | 'To'): string {
  const value = props.modelValue[`${field.key}${boundary}`]
  if (typeof value !== 'string') return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function setDate(field: FilterField, boundary: 'From' | 'To', event: Event): void {
  const value = (event.target as HTMLInputElement).value
  const next = { ...props.modelValue }
  const key = `${field.key}${boundary}`
  if (value) next[key] = new Date(`${value}T00:00:00`).toISOString()
  else delete next[key]
  emit('update:modelValue', next)
}

const invalidDateRange = computed(() => props.fields.some((field) => {
  if (!field.mobileNative || field.type !== 'daterange') return false
  const from = dateValue(field, 'From')
  const to = dateValue(field, 'To')
  return Boolean(from || to) && (!from || !to || from > to)
}))
</script>

<template>
  <div class="filter-bar">
    <div class="filter-bar__fields">
      <div
        v-for="field in fields"
        :key="field.key"
        class="filter-bar__field"
      >
        <span class="filter-bar__label">{{ field.label }}</span>
        <NInput
          v-if="field.type === 'input'"
          :value="(modelValue[field.key] as string) ?? null"
          :placeholder="field.placeholder ?? '请输入'"
          clearable
          :style="{ width: (field.width ?? 180) + 'px' }"
          @update:value="(v) => setValue(field.key, v)"
          @keyup.enter="emit('search')"
        />
        <NSelect
          v-else-if="field.type === 'select'"
          :value="(modelValue[field.key] as string) ?? null"
          :options="(field.options ?? []).map((o) => ({ label: o.label, value: o.value }))"
          :placeholder="field.placeholder ?? '全部'"
          clearable
          :style="{ width: (field.width ?? 160) + 'px' }"
          @update:value="(v) => setValue(field.key, v)"
        />
        <NDatePicker
          v-else-if="field.type === 'daterange'"
          :value="rangeValue(field)"
          type="daterange"
          clearable
          :class="{ 'filter-bar__desktop-date': field.mobileNative }"
          :style="{ width: (field.width ?? 260) + 'px' }"
          @update:value="(v) => setRange(field, v as [number, number] | null)"
        />
        <div
          v-if="field.type === 'daterange' && field.mobileNative"
          class="filter-bar__mobile-date"
        >
          <label>
            <span>开始日期</span>
            <input
              type="date"
              :value="dateValue(field, 'From')"
              :max="dateValue(field, 'To') || undefined"
              @change="setDate(field, 'From', $event)"
            >
          </label>
          <label>
            <span>结束日期</span>
            <input
              type="date"
              :value="dateValue(field, 'To')"
              :min="dateValue(field, 'From') || undefined"
              @change="setDate(field, 'To', $event)"
            >
          </label>
        </div>
      </div>
    </div>
    <NSpace class="filter-bar__actions">
      <NButton
        type="primary"
        size="small"
        :disabled="invalidDateRange"
        @click="emit('search')"
      >
        查询
      </NButton>
      <NButton
        size="small"
        @click="emit('reset')"
      >
        重置
      </NButton>
    </NSpace>
    <p
      v-if="invalidDateRange"
      class="filter-bar__range-error"
      role="status"
    >
      请选择完整时间范围，开始日期不能晚于结束日期。
    </p>
  </div>
</template>

<style scoped>
.filter-bar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--ic-space-md);
  padding: var(--ic-space-md);
  background: var(--ic-color-surface);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-md);
  margin-bottom: var(--ic-space-md);
  flex-wrap: wrap;
}
.filter-bar__fields {
  display: flex;
  flex-wrap: wrap;
  gap: var(--ic-space-md);
}
.filter-bar__field {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-xs);
}
.filter-bar__label {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-secondary);
}
.filter-bar__actions {
  flex-shrink: 0;
}
.filter-bar__mobile-date {
  display: none;
}
.filter-bar__range-error {
  width: 100%;
  margin: 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-sm);
}
@media (max-width: 768px) {
  .filter-bar__desktop-date {
    display: none;
  }
  .filter-bar__mobile-date {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }
  .filter-bar__mobile-date label {
    display: grid;
    min-width: 0;
    gap: 6px;
    font-size: var(--ic-font-sm);
    color: var(--ic-color-text-secondary);
  }
  .filter-bar__mobile-date input {
    box-sizing: border-box;
    width: 100%;
    min-width: 0;
    min-height: 44px;
    padding: 8px;
    border: 1px solid var(--ic-color-border);
    border-radius: var(--ic-radius-sm);
    background: var(--ic-color-surface);
    color: var(--ic-color-text);
    font: inherit;
    font-size: 16px;
    color-scheme: light;
  }
  .filter-bar__mobile-date input:focus-visible {
    outline: 2px solid var(--ic-color-primary);
    outline-offset: 2px;
  }
}
</style>
