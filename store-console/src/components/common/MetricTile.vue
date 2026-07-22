<script setup lang="ts">
/** 指标块：工作台/报表复用的单个数值展示（通栏轻背景，非重阴影卡片）。 */
defineProps<{ label: string; value: string | number; hint?: string }>()
</script>

<template>
  <div class="metric ic-band">
    <div class="metric__label ic-muted">
      {{ label }}
    </div>
    <div class="metric__value ic-mono">
      {{ value }}
    </div>
    <div
      v-if="hint"
      class="metric__hint ic-muted"
    >
      {{ hint }}
    </div>
  </div>
</template>

<style scoped>
.metric {
  padding: var(--ic-space-4) var(--ic-space-5);
}
.metric__label {
  font-size: var(--ic-font-sm);
}
.metric__value {
  font-size: var(--ic-font-xl);
  font-weight: 700;
  margin-top: var(--ic-space-2);
}
.metric__hint {
  font-size: var(--ic-font-xs);
  margin-top: var(--ic-space-1);
}
</style>
