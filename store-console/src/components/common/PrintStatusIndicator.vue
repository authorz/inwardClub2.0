<script setup lang="ts">
/**
 * 打印状态提示：展示打印任务/打印机状态的小圆点 + 文案。
 * 打印机管理、订单打印结果复用。
 */
import { computed } from 'vue'
import { resolveEnum, type EnumOption } from '@/constants/enums'
import { PRINT_JOB_STATUS, PRINTER_STATUS } from '@/constants/enums'

const props = withDefaults(
  defineProps<{
    value: string | null | undefined
    /** 使用打印机状态字典还是打印任务字典。 */
    kind?: 'printer' | 'job'
  }>(),
  { kind: 'printer' },
)

const dict = computed<Record<string, EnumOption>>(() =>
  props.kind === 'printer' ? PRINTER_STATUS : PRINT_JOB_STATUS,
)
const option = computed(() => resolveEnum(dict.value, props.value))
const dotColor = computed(() => {
  switch (option.value.tone) {
    case 'success':
      return 'var(--ic-color-success)'
    case 'warning':
      return 'var(--ic-color-warning)'
    case 'error':
      return 'var(--ic-color-danger)'
    case 'info':
      return 'var(--ic-color-info)'
    default:
      return 'var(--ic-color-neutral)'
  }
})
</script>

<template>
  <span class="print-status">
    <span
      class="print-status__dot"
      :style="{ background: dotColor }"
    />
    <span class="print-status__label">{{ option.label }}</span>
  </span>
</template>

<style scoped>
.print-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--ic-font-sm);
}
.print-status__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
}
</style>
