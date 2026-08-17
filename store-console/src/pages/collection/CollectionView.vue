<script setup lang="ts">
import { CollectionCreatePanel } from '@/components/collection'
import { PageHeader } from '@/components/common'
</script>

<template>
  <div>
    <PageHeader
      title="线下聚合收款"
      description="创建固定金额微信收款码，顾客扫码支付后自动确认收款结果"
    />

    <div class="collect-form ic-band">
      <CollectionCreatePanel />
    </div>
  </div>
</template>

<style scoped>
.collect-form {
  max-width: 480px;
  padding: var(--ic-space-5);
}
</style>
