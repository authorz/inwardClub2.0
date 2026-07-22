<script setup lang="ts">
/**
 * 审计日志（只读）。记录所有后台写操作的 actor / 角色 / scope / 对象 / 前后差异 / request ID。
 */
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, textColumn } from '@/utils/columns'
import { readonlyLists } from '@/api/services'
import type { AuditLog } from '@/api/models'
import type { FilterField } from '@/components/ui-types'

const fields: FilterField[] = [
  { key: 'actorType', label: '操作者类型', type: 'input' },
  { key: 'action', label: '动作', type: 'input' },
  { key: 'targetType', label: '对象类型', type: 'input' },
  { key: 'created', label: '时间', type: 'daterange' },
]

const columns = [
  dateTimeColumn<AuditLog>('时间', 'createdAt'),
  textColumn<AuditLog>('操作者类型', 'actorType', { width: 140 }),
  textColumn<AuditLog>('操作者 ID', 'actorId', { width: 120 }),
  textColumn<AuditLog>('门店', 'storeId', { width: 100 }),
  textColumn<AuditLog>('动作', 'action', { width: 180 }),
  textColumn<AuditLog>('对象类型', 'targetType', { width: 140 }),
  textColumn<AuditLog>('对象 ID', 'targetId', { width: 140 }),
  textColumn<AuditLog>('Request ID', 'requestId', { width: 200 }),
]
</script>

<template>
  <ResourceListView
    title="审计日志"
    description="所有后台写操作的可追溯记录（只读）"
    :breadcrumb="['审计与日志', '审计日志']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.auditLogs"
  />
</template>
