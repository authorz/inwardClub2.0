<script setup lang="ts">
/**
 * 全局商品。列表 + 发布/下架（可发布资源）。
 * 商品创建/规格/投放门店等复杂表单待服务端接口就绪后补充。
 */
import { h, ref } from 'vue'
import { NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import PermissionButton from '@/components/PermissionButton.vue'
import {
  actionsColumn,
  dateTimeColumn,
  moneyColumn,
  statusColumn,
  textColumn,
} from '@/utils/columns'
import {
  ITEM_TYPE_OPTIONS,
  RESOURCE_STATUS,
  RESOURCE_STATUS_OPTIONS,
  SCOPE_TYPE_OPTIONS,
} from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { catalogItemService } from '@/api/services'
import { usePublishableActions } from '@/composables/usePublishableActions'
import type { CatalogItem } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const listRef = ref<ResourceListInstance | null>(null)
// 无独立 publish 接口：发布=整体 PUT 改写 status。小程序端商品读取以 status='published' 为门槛。
const { publish, unpublish } = usePublishableActions(
  catalogItemService,
  { publishedStatus: RESOURCE_STATUS.PUBLISHED, unpublishedStatus: RESOURCE_STATUS.DRAFT },
  () => listRef.value?.reload(),
)

const fields: FilterField[] = [
  { key: 'keyword', label: '商品名称', type: 'input' },
  { key: 'itemType', label: '商品类型', type: 'select', options: ITEM_TYPE_OPTIONS },
  { key: 'scopeType', label: '范围', type: 'select', options: SCOPE_TYPE_OPTIONS },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const columns = [
  textColumn<CatalogItem>('商品名称', 'name'),
  statusColumn<CatalogItem>('类型', 'itemType', ITEM_TYPE_OPTIONS, 100),
  statusColumn<CatalogItem>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  moneyColumn<CatalogItem>('价格', 'priceCent'),
  textColumn<CatalogItem>('库存', 'stockQuantity', { width: 90 }),
  statusColumn<CatalogItem>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<CatalogItem>('创建时间', 'createdAt'),
  actionsColumn<CatalogItem>(
    (row) =>
      h(NSpace, {}, () => [
        row.status === RESOURCE_STATUS.PUBLISHED
          ? h(
              PermissionButton,
              {
                permission: PERMISSIONS.CATALOG_GLOBAL_WRITE,
                type: 'error',
                onClick: () => unpublish(row, row.name),
              },
              () => '下架',
            )
          : h(
              PermissionButton,
              {
                permission: PERMISSIONS.CATALOG_GLOBAL_WRITE,
                type: 'primary',
                onClick: () => publish(row, row.name),
              },
              () => '发布',
            ),
      ]),
    120,
  ),
]
</script>

<template>
  <ResourceListView
    ref="listRef"
    title="全局商品"
    description="全局商品模板、投放与全局上下架；跨店写操作写入审计"
    :breadcrumb="['商品与分类', '全局商品']"
    :fields="fields"
    :columns="columns"
    :fetcher="catalogItemService.list"
  />
</template>
