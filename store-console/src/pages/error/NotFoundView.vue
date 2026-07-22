<script setup lang="ts">
import { NButton, NResult } from 'naive-ui'
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="error-page">
    <NResult
      status="404"
      title="页面不存在"
      description="你访问的页面不存在或已被移除。"
    >
      <template #footer>
        <NButton @click="router.replace('/dashboard')">
          返回工作台
        </NButton>
      </template>
    </NResult>
  </div>
</template>

<style scoped>
.error-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
}
</style>
