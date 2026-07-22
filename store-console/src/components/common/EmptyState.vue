<script setup lang="ts">
/** 统一空状态。列表、详情、无权限区块统一使用。 */
import { NEmpty } from 'naive-ui'
withDefaults(defineProps<{ description?: string }>(), {
  description: '暂无数据',
})
</script>

<template>
  <div class="empty-state">
    <NEmpty :description="description">
      <template #extra>
        <slot name="extra" />
      </template>
    </NEmpty>
  </div>
</template>

<style scoped>
.empty-state {
  padding: var(--ic-space-7) 0;
  display: flex;
  justify-content: center;
}
</style>
