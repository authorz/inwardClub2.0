<script setup lang="ts">
/** 页面标题栏：标题 + 描述 + 右侧操作区插槽。统一页面头部结构。 */
defineProps<{ title: string; description?: string }>()
</script>

<template>
  <header class="page-header">
    <div class="page-header__text">
      <h2 class="page-header__title">
        {{ title }}
      </h2>
      <p
        v-if="description"
        class="page-header__desc ic-muted"
      >
        {{ description }}
      </p>
    </div>
    <div class="page-header__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ic-space-4);
  margin-bottom: var(--ic-space-4);
}
.page-header__title {
  font-size: var(--ic-font-lg);
  margin: 0;
}
.page-header__desc {
  font-size: var(--ic-font-sm);
  margin: 4px 0 0;
}
.page-header__actions {
  display: flex;
  gap: var(--ic-space-2);
  flex-shrink: 0;
}
</style>
