<script setup lang="ts">
/**
 * 页头（公共组件）：面包屑 + 页标题 + 描述 + 右侧操作区。
 */
import { NBreadcrumb, NBreadcrumbItem } from 'naive-ui'

defineProps<{
  title: string
  description?: string
  /** 面包屑层级文案，从上级到当前 */
  breadcrumb?: string[]
}>()
</script>

<template>
  <header class="page-header">
    <NBreadcrumb
      v-if="breadcrumb && breadcrumb.length"
      class="page-header__crumb"
    >
      <NBreadcrumbItem
        v-for="(c, i) in breadcrumb"
        :key="i"
      >
        {{ c }}
      </NBreadcrumbItem>
    </NBreadcrumb>
    <div class="page-header__main">
      <div>
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
  margin-bottom: var(--ic-space-md);
}
.page-header__crumb {
  margin-bottom: var(--ic-space-xs);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.page-header__main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--ic-space-md);
}
.page-header__title {
  margin: 0;
  font-size: var(--ic-font-xl);
  font-weight: 600;
  color: var(--ic-color-text);
}
.page-header__desc {
  margin: var(--ic-space-xs) 0 0;
  font-size: var(--ic-font-sm);
  color: var(--ic-color-text-secondary);
}
.page-header__actions {
  display: flex;
  gap: var(--ic-space-sm);
  flex-shrink: 0;
}
</style>
