<script setup lang="ts">
/**
 * 应用外壳：在 Provider 上下文内注册全局 message/dialog 句柄，
 * 使 http client / store / composable 等非组件环境也能统一弹提示与二次确认。
 */
import { useDialog, useMessage } from 'naive-ui'
import { registerFeedback } from '@/utils/feedback'

registerFeedback(useMessage(), useDialog())
</script>

<template>
  <router-view />
</template>
