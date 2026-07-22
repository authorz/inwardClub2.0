<script setup lang="ts">
/**
 * 券模板。只读列表。
 * 服务端券模板更新 DTO（TemplateInput）不含 status 字段，UpdateTemplate 也不改写 status，
 * 现阶段没有任何接口可切换券模板发布状态，故不提供发布/下架快捷操作（残留后端缺口）。
 * 适用商品/分类、有效期规则、跨店发券、发券/核销记录待详情页补充。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, moneyColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS, SCOPE_TYPE_OPTIONS } from '@/constants/enums'
import { couponTemplateService } from '@/api/services'
import type { CouponTemplate } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  { key: 'keyword', label: '券名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const columns = [
  textColumn<CouponTemplate>('券名称', 'name'),
  textColumn<CouponTemplate>('券类型', 'couponType', { width: 120 }),
  moneyColumn<CouponTemplate>('面额', 'valueCent'),
  statusColumn<CouponTemplate>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  statusColumn<CouponTemplate>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<CouponTemplate>('创建时间', 'createdAt'),
]
</script>

<template>
  <ResourceListView
    title="券模板"
    description="全局券模板与发券规则；跨店发券写入审计并记录原因"
    :breadcrumb="['券管理']"
    :fields="fields"
    :columns="columns"
    :fetcher="couponTemplateService.list"
  />
</template>
