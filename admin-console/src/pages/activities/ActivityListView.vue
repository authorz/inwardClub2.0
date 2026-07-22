<script setup lang="ts">
/**
 * 全局活动。只读列表。
 * 服务端活动更新 DTO（ActivityInput）含 purchaseLimitPerMember，但 GET 详情视图
 * （ConsoleActivityView）不返回该字段；若以「取详情+整体 PUT」方式切换 status，
 * 会把 purchaseLimitPerMember 静默重置为 0，属数据损坏。故不提供发布/下架快捷操作，
 * 状态改由编辑抽屉在操作者可见全部字段时显式设置（残留：编辑抽屉待补）。
 * 场次、票档、投放门店、分享资产生成待服务端接口就绪后在详情页补充。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS, SCOPE_TYPE_OPTIONS } from '@/constants/enums'
import { activityService } from '@/api/services'
import type { Activity } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  { key: 'keyword', label: '活动标题', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const columns = [
  textColumn<Activity>('活动标题', 'name'),
  statusColumn<Activity>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  dateTimeColumn<Activity>('开始时间', 'startAt'),
  dateTimeColumn<Activity>('结束时间', 'endAt'),
  statusColumn<Activity>('状态', 'status', RESOURCE_STATUS_OPTIONS),
]
</script>

<template>
  <ResourceListView
    title="全局活动"
    description="全局活动模板、场次、票档与投放；发布为跨店操作写入审计"
    :breadcrumb="['活动管理']"
    :fields="fields"
    :columns="columns"
    :fetcher="activityService.list"
  />
</template>
