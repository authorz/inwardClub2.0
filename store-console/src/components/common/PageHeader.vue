<script setup lang="ts">
/** 页面标题栏：面包屑 + 标题 + 描述 + 右侧操作区插槽。统一页面头部结构。 */
import { NBreadcrumb, NBreadcrumbItem } from 'naive-ui'

defineProps<{ title: string; description?: string; breadcrumb?: string[] }>()
</script>

<template>
  <header class="page-header">
    <NBreadcrumb
      v-if="breadcrumb && breadcrumb.length"
      class="page-header__crumb"
    >
      <NBreadcrumbItem
        v-for="(item, index) in breadcrumb"
        :key="index"
      >
        {{ item }}
      </NBreadcrumbItem>
    </NBreadcrumb>
    <div class="page-header__main">
      <div class="page-header__text">
        <h1 class="page-header__title">
          {{ title }}
        </h1>
        <p
          v-if="description"
          class="page-header__desc"
        >
          {{ description }}
        </p>
      </div>
      <div class="page-header__actions">
        <slot name="actions" />
      </div>
    </div>
  </header>
</template>

<style scoped>
.page-header {
  margin-bottom: var(--ic-space-4);
}
.page-header__crumb {
  margin-bottom: var(--ic-space-1);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.page-header__main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ic-space-4);
}
.page-header__title {
  margin: 0;
  font-size: var(--ic-font-lg);
  font-weight: 600;
  color: var(--ic-color-text);
}
.page-header__desc {
  font-size: var(--ic-font-sm);
  margin: var(--ic-space-1) 0 0;
  color: var(--ic-color-text-secondary);
}
.page-header__actions {
  display: flex;
  gap: var(--ic-space-2);
  flex-shrink: 0;
}
</style>
