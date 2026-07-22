<script setup lang="ts">
import { NConfigProvider, zhCN, dateZhCN } from 'naive-ui'
import { RouterView } from 'vue-router'
import { themeOverrides } from '@/theme'
</script>

<template>
  <NConfigProvider
    :theme-overrides="themeOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
  >
    <RouterView />
  </NConfigProvider>
</template>
