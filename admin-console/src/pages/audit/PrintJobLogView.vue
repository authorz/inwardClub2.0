<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue'
import { NTag } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import { dateTimeColumn, renderColumn, textColumn } from '@/utils/columns'
import { readonlyLists, storeService } from '@/api/services'
import type { PrintJobLog } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError } from '@/utils/feedback'

const storeOptions = ref<Array<{ label: string; value: string }>>([])

const statusOptions = [
  { label: '等待打印', value: 'pending' },
  { label: '正在打印', value: 'processing' },
  { label: '打印成功', value: 'printed' },
  { label: '打印失败', value: 'failed' },
]

const statusLabels = Object.fromEntries(statusOptions.map((item) => [item.value, item.label]))
const statusTypes: Record<PrintJobLog['status'], 'default' | 'info' | 'success' | 'error'> = {
  pending: 'default',
  processing: 'info',
  printed: 'success',
  failed: 'error',
}

const fields = computed<FilterField[]>(() => [
  { key: 'storeId', label: '所属门店', type: 'select', options: storeOptions.value, width: 220 },
  { key: 'status', label: '打印状态', type: 'select', options: statusOptions },
  {
    key: 'keyword',
    label: '关键词',
    type: 'input',
    placeholder: '订单号 / 门店 / 打印机 / SN',
    width: 260,
  },
  { key: 'created', label: '创建时间', type: 'daterange' },
])

const columns = [
  dateTimeColumn<PrintJobLog>('创建时间', 'createdAt', 170),
  textColumn<PrintJobLog>('所属门店', 'storeName', { width: 170 }),
  renderColumn<PrintJobLog>(
    '打印机',
    'deviceName',
    (row) =>
      h('div', { class: 'print-log__device' }, [
        h('strong', row.deviceName || '已删除的打印机'),
        h('span', `SN：${row.deviceSn || '—'}`),
      ]),
    210,
  ),
  textColumn<PrintJobLog>('业务订单号', 'businessOrderNo', { width: 210 }),
  renderColumn<PrintJobLog>(
    '状态',
    'status',
    (row) =>
      h(
        NTag,
        { size: 'small', bordered: false, type: statusTypes[row.status] ?? 'default' },
        { default: () => statusLabels[row.status] ?? row.status },
      ),
    110,
  ),
  textColumn<PrintJobLog>('尝试次数', 'attempts', { width: 100 }),
  renderColumn<PrintJobLog>(
    '失败原因',
    'lastError',
    (row) => row.lastError || '—',
    300,
  ),
  dateTimeColumn<PrintJobLog>('更新时间', 'updatedAt', 170),
]

async function loadStores(): Promise<void> {
  try {
    const result = await storeService.list({ page: 1, pageSize: 100 })
    storeOptions.value = result.items.map((store) => ({
      label: store.name,
      value: String(store.id),
    }))
  } catch (error) {
    toastError((error as { message?: string }).message ?? '门店列表加载失败')
  }
}

onMounted(loadStores)
</script>

<template>
  <ResourceListView
    title="打印日志"
    description="查看各门店小票打印任务的执行状态、尝试次数和失败原因"
    :breadcrumb="['数据与日志', '打印日志']"
    :fields="fields"
    :columns="columns"
    :fetcher="readonlyLists.printJobs"
    empty-text="暂无打印任务"
  />
</template>

<style scoped>
.print-log__device {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.print-log__device strong {
  font-weight: 600;
}

.print-log__device span {
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}
</style>
