<script setup lang="ts">
/** 错误事件（只读）。 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, textColumn } from '@/utils/columns'
import { readonlyLists } from '@/api/services'
import type { ErrorEvent } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  { key: 'path', label: '路径', type: 'input' },
  { key: 'created', label: '时间', type: 'daterange' },
]

const columns = [
  dateTimeColumn<ErrorEvent>('时间', 'createdAt'),
  textColumn<ErrorEvent>('方法', 'method', { width: 80 }),
  textColumn<ErrorEvent>('路径', 'path', { width: 220 }),
  textColumn<ErrorEvent>('状态码', 'status', { width: 90 }),
  textColumn<ErrorEvent>('信息', 'message'),
  textColumn<ErrorEvent>('Request ID', 'requestId', { width: 200 }),
]
</script>

<template>
  <ResourceListView
    title="错误事件"
    description="服务端错误事件（只读）"
    :breadcrumb="['审计与日志', '错误事件']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.errorEvents"
  />
</template>
