<script setup lang="ts">
/**
 * 状态标签。根据集中枚举字典解析出标签文案与语义色调。
 * 全站状态展示统一走此组件，禁止散落字符串与颜色。
 */
import { computed } from 'vue'
import { NTag } from 'naive-ui'
import { resolveEnum, type EnumOption, type StatusTone } from '@/constants/enums'

const props = defineProps<{
  /** 集中枚举字典。 */
  dict: Record<string, EnumOption>
  /** 当前值。 */
  value: string | null | undefined
}>()

const TONE_TO_TYPE: Record<StatusTone, 'default' | 'success' | 'warning' | 'error' | 'info'> = {
  default: 'default',
  success: 'success',
  warning: 'warning',
  error: 'error',
  info: 'info',
}

const option = computed(() => resolveEnum(props.dict, props.value))
const tagType = computed(() => TONE_TO_TYPE[option.value.tone ?? 'default'])
</script>

<template>
  <NTag
    :type="tagType"
    size="small"
    :bordered="false"
    round
  >
    {{ option.label }}
  </NTag>
</template>
