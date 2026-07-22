<script setup lang="ts">
/**
 * 登录日志（只读）。
 * 后端无 /admin/login-events 接口，本页已从路由与菜单移除（残留死文件，未接入导航）。
 * 为通过构建且不调用不存在的接口，fetcher 降级为空列表。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, textColumn } from '@/utils/columns'
import type { LoginEvent } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import type { ListResult } from '@/api/types'

const fields: FilterField[] = [
  { key: 'actor', label: '账号', type: 'input' },
  { key: 'created', label: '时间', type: 'daterange' },
]

const emptyFetcher = (): Promise<ListResult<LoginEvent>> =>
  Promise.resolve({ items: [], meta: { page: 1, pageSize: 0, total: 0 } })

const columns = [
  dateTimeColumn<LoginEvent>('时间', 'createdAt'),
  textColumn<LoginEvent>('账号', 'actor', { width: 160 }),
  textColumn<LoginEvent>('IP', 'ip', { width: 140 }),
  textColumn<LoginEvent>('结果', 'result', { width: 100 }),
  textColumn<LoginEvent>('User-Agent', 'userAgent'),
]
</script>

<template>
  <ResourceListView
    title="登录日志"
    description="总后台登录事件（只读）"
    :breadcrumb="['审计与日志', '登录日志']"
    :fields="fields"
    :columns="columns"
    :fetcher="emptyFetcher"
  />
</template>
