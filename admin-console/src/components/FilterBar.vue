<script setup lang="ts">
/**
 * 筛选区（公共组件，schema 驱动）。
 *
 * 通过 fields 配置渲染 input / select / daterange，
 * v-model 绑定一个筛选对象；点击查询 / 重置触发事件。
 * 所有列表页复用，不再各自手写筛选布局。
 */
import { NButton, NDatePicker, NInput, NSelect, NSpace } from 'naive-ui'
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
          type="daterange"
          clearable
          :style="{ width: (field.width ?? 260) + 'px' }"
          @update:value="(v) => setRange(field, v as [number, number] | null)"
        />
      </div>
    </div>
    <NSpace class="filter-bar__actions">
      <NButton
        type="primary"
        size="small"
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
</style>
