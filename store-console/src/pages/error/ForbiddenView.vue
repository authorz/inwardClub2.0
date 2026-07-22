<script setup lang="ts">
import { NButton, NResult } from 'naive-ui'
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="error-page">
    <NResult
      status="403"
      title="无操作权限"
      description="当前账号没有访问该页面的权限。"
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
  min-height: 60vh;
}
</style>
