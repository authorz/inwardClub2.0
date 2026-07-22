<script setup lang="ts">
/**
 * 状态标签（公共组件）。
 * 依据 options 的 tone 映射到 Naive UI tag type，同时始终展示文本，
 * 满足「不依赖颜色单独表达状态」的可访问性要求。
 */
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import { buildOptionMap, type OptionItem } from '@/constants/enums'

const props = defineProps<{
  value: string | null | undefined
  options: OptionItem[]
  /** 找不到映射时的兜底文案 */
  fallback?: string
}>()

const map = computed(() => buildOptionMap(props.options))
const current = computed(() => (props.value != null ? map.value[props.value] : undefined))

const tagType = computed(() => {
  switch (current.value?.tone) {
    case 'success':
      return 'success'
    case 'warning':
      return 'warning'
    case 'error':
      return 'error'
    case 'info':
      return 'info'
    default:
      return 'default'
  }
})

const text = computed(() => current.value?.label ?? props.fallback ?? props.value ?? '-')
</script>

<template>
  <NTag
    :type="tagType"
    :bordered="false"
    size="small"
  >
    {{ text }}
  </NTag>
</template>
