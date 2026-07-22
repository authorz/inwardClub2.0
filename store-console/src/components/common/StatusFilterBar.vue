<script setup lang="ts">
/**
 * 通用筛选栏：状态筛选 + 关键字搜索 + 额外筛选插槽 + 刷新。
 * 列表页统一复用，避免每页重复搭建筛选 UI 与事件。
 */
import { computed } from 'vue'
import { NButton, NInput, NSelect, type SelectOption } from 'naive-ui'
import type { EnumOption } from '@/constants/enums'

const props = withDefaults(
  defineProps<{
    /** 状态选项字典值（来自集中枚举）。为空则不展示状态筛选。 */
    statusOptions?: EnumOption[]
    /** 当前状态值。 */
    status?: string | null
    /** 关键字。 */
    keyword?: string
    /** 是否展示关键字搜索。 */
    searchable?: boolean
    searchPlaceholder?: string
    loading?: boolean
  }>(),
  { searchable: true, searchPlaceholder: '搜索订单号/会员', loading: false, status: null, keyword: '' },
)

const emit = defineEmits<{
  'update:status': [value: string | null]
  'update:keyword': [value: string]
  apply: []
  reset: []
}>()

const statusSelectOptions = computed<SelectOption[]>(() => [
  { label: '全部状态', value: '' },
  ...(props.statusOptions ?? []).map((o) => ({ label: o.label, value: o.value })),
])
</script>

<template>
  <div class="filter-bar">
    <NSelect
      v-if="statusOptions && statusOptions.length"
      class="filter-bar__status"
      :value="status ?? ''"
      :options="statusSelectOptions"
      @update:value="emit('update:status', $event === '' ? null : $event)"
    />

    <NInput
      v-if="searchable"
      class="filter-bar__search"
      :value="keyword"
      :placeholder="searchPlaceholder"
      clearable
      @update:value="emit('update:keyword', $event)"
      @keyup.enter="emit('apply')"
    />

    <div class="filter-bar__extra">
      <slot name="filters" />
    </div>

    <div class="filter-bar__actions">
      <NButton
        type="primary"
        :loading="loading"
        @click="emit('apply')"
      >
        查询
      </NButton>
      <NButton
        quaternary
        @click="emit('reset')"
      >
        重置
      </NButton>
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.filter-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--ic-space-2);
  margin-bottom: var(--ic-space-4);
}
.filter-bar__status {
  width: 150px;
}
.filter-bar__search {
  width: 240px;
}
.filter-bar__extra {
  display: flex;
  gap: var(--ic-space-2);
  align-items: center;
}
.filter-bar__actions {
  display: flex;
  gap: var(--ic-space-2);
  margin-left: auto;
}
</style>
