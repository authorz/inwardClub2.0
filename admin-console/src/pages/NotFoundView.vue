<script setup lang="ts">
import { useRouter } from 'vue-router'
import { NButton, NResult } from 'naive-ui'
const router = useRouter()
</script>

<template>
  <div class="not-found">
    <NResult
      status="404"
      title="页面不存在"
      description="请检查地址或返回工作台"
    >
      <template #footer>
        <NButton
          type="primary"
          @click="router.replace('/dashboard')"
        >
          返回工作台
        </NButton>
      </template>
    </NResult>
  </div>
</template>

<style scoped>
.not-found {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
