<script setup lang="ts">
/** 错误事件（只读）。 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, textColumn } from '@/utils/columns'
import { readonlyLists } from '@/api/services'
import type { ErrorEvent } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  {
    key: 'requestId',
    label: '错误 ID',
    type: 'input',
    placeholder: '输入完整或部分错误 ID',
    width: 260,
  },
  { key: 'path', label: '路径', type: 'input' },
  { key: 'created', label: '时间', type: 'daterange' },
]

const columns = [
  dateTimeColumn<ErrorEvent>('时间', 'createdAt'),
  textColumn<ErrorEvent>('Request ID', 'requestId', { width: 240 }),
  textColumn<ErrorEvent>('会员 ID', 'memberId', { width: 100 }),
  textColumn<ErrorEvent>('OpenID', 'wechatOpenId', { width: 260 }),
  textColumn<ErrorEvent>('方法', 'method', { width: 80 }),
  textColumn<ErrorEvent>('路径', 'path', { width: 220 }),
  textColumn<ErrorEvent>('状态码', 'status', { width: 90 }),
  textColumn<ErrorEvent>('信息', 'message'),
]
</script>

<template>
  <ResourceListView
    title="错误事件"
    description="服务端错误事件（只读），支持按错误 ID 模糊查询并追踪预注册用户身份"
    :breadcrumb="['审计与日志', '错误事件']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.errorEvents"
  />
</template>
