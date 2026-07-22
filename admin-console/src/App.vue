<script setup lang="ts">
/**
 * 应用根组件。
 * 挂载 Naive UI 全局 Provider（主题、message、dialog、loading bar），
 * 主题使用集中定义的黑白灰 overrides。
 * 反馈句柄注册在 AppShell 内完成（需在 Provider 上下文里）。
 */
import { NConfigProvider, NDialogProvider, NLoadingBarProvider, NMessageProvider } from 'naive-ui'
import { zhCN, dateZhCN } from 'naive-ui'
import { adminThemeOverrides } from './styles/theme'
import AppShell from './AppShell.vue'
</script>

<template>
  <NConfigProvider
    :theme-overrides="adminThemeOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
  >
    <NLoadingBarProvider>
      <NMessageProvider>
        <NDialogProvider>
          <AppShell />
        </NDialogProvider>
      </NMessageProvider>
    </NLoadingBarProvider>
  </NConfigProvider>
</template>
