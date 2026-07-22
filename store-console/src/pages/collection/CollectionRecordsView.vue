<script setup lang="ts">
/**
 * 收款记录：本店线下聚合收款单列表。
 *
 * 说明：服务端无收款单列表读端点（GET /store/offline-collection-orders 不存在，
 * 仅 POST 创建、GET/:id 详情、POST/:id/cancel 取消），故记录列表退化为空状态。
 * 创建收款码见「线下聚合收款」页；创建后可在返回的收款码弹窗内查看/取消当次单据。
 */
import { EmptyState, PageHeader } from '@/components/common'
</script>

<template>
  <div>
    <PageHeader
      title="收款记录"
      description="本店线下聚合收款单"
    />

    <EmptyState description="暂无收款记录（收款单列表功能待后端支持），请在「线下聚合收款」页创建收款码" />
  </div>
</template>
