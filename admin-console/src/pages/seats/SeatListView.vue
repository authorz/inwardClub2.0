<script setup lang="ts">
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import { NAlert, NForm, NFormItem, NInput, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { TABLE_SEAT_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { seatService, storeService, tableService } from '@/api/services'
import type { VenueSeat, VenueTable } from '@/api/models'
import { runAudited } from '@/composables/useAuditedAction'
import { toastError, toastSuccess } from '@/utils/feedback'

interface SeatForm {
  storeId: string | null
  tableId: string | null
  name: string
  status: string
}

const listRef = ref<ResourceListInstance | null>(null)
const storeOptions = ref<Array<{ label: string; value: string }>>([])
const tables = ref<VenueTable[]>([])
const drawerShow = ref(false)
const submitting = ref(false)
const loadingTables = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<SeatForm>({
  storeId: null,
  tableId: null,
  name: '',
  status: 'available',
})

const tableOptions = computed<Array<{ label: string; value: string }>>(() =>
  tables.value.map((table) => ({
    label: `${table.name}（${table.code}，${table.seatCount}/${table.capacity}）`,
    value: String(table.id),
  })),
)

const fields = computed<FilterField[]>(() => [
  {
    key: 'storeId',
    label: '所属门店',
    type: 'select',
    options: storeOptions.value,
    width: 200,
  },
  {
    key: 'keyword',
    label: '座位名称',
    type: 'input',
    placeholder: '搜索座位名称',
    width: 220,
  },
  { key: 'status', label: '状态', type: 'select', options: TABLE_SEAT_STATUS_OPTIONS },
])

const columns = [
  textColumn<VenueSeat>('座位 ID', 'id', { width: 100 }),
  textColumn<VenueSeat>('座位名称', 'name', { width: 160 }),
  textColumn<VenueSeat>('所属桌子', 'tableName', { width: 180 }),
  textColumn<VenueSeat>('所属门店', 'storeName', { width: 180 }),
  statusColumn<VenueSeat>('状态', 'status', TABLE_SEAT_STATUS_OPTIONS, 110),
  dateTimeColumn<VenueSeat>('更新时间', 'updatedAt'),
  actionsColumn<VenueSeat>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.STORE_WRITE, onClick: () => openEdit(row) },
          () => '编辑',
        ),
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.STORE_WRITE,
            type: 'error',
            onClick: () => remove(row),
          },
          () => '删除',
        ),
      ]),
    150,
  ),
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

async function loadTables(storeId: string | null, selectedTableId?: string): Promise<void> {
  tables.value = []
  if (!storeId) return
  loadingTables.value = true
  try {
    const result = await tableService.list({ page: 1, pageSize: 100, storeId })
    tables.value = result.items
    if (selectedTableId && !result.items.some((item) => String(item.id) === selectedTableId)) {
      form.tableId = null
    }
  } catch (error) {
    toastError((error as { message?: string }).message ?? '桌子列表加载失败')
  } finally {
    loadingTables.value = false
  }
}

function openCreate(): void {
  editingId.value = null
  form.storeId = null
  form.tableId = null
  form.name = ''
  form.status = 'available'
  tables.value = []
  drawerShow.value = true
}

async function openEdit(row: VenueSeat): Promise<void> {
  editingId.value = row.id
  form.storeId = row.storeId == null ? null : String(row.storeId)
  form.tableId = null
  form.name = row.name
  form.status = row.status ?? 'available'
  drawerShow.value = true
  await loadTables(form.storeId)
  form.tableId = String(row.tableId)
}

watch(
  () => form.storeId,
  async (storeId, previousStoreId) => {
    if (storeId === previousStoreId) return
    form.tableId = null
    await loadTables(storeId)
  },
)

async function submit(): Promise<void> {
  if (!form.storeId) return toastError('请先选择所属门店')
  if (!form.tableId) return toastError('请选择所属桌子；添加座位前必须先添加桌子')
  if (!form.name.trim()) return toastError('请输入座位名称')

  submitting.value = true
  try {
    const payload = {
      tableId: Number(form.tableId),
      name: form.name.trim(),
      status: form.status,
    }
    if (editingId.value) await seatService.update(editingId.value, payload)
    else await seatService.create(payload)
    toastSuccess(editingId.value ? '座位已更新' : '座位已创建')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}

async function remove(row: VenueSeat): Promise<void> {
  const success = await runAudited({
    title: '删除座位',
    content: `确认删除“${row.name}”吗？已有预约记录的座位不能删除。`,
    highRisk: true,
    positiveText: '确认删除',
    successText: '座位已删除',
    execute: () => seatService.remove(row.id),
  })
  if (success) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增座位',
    type: 'primary' as const,
    permission: PERMISSIONS.STORE_WRITE,
    onClick: openCreate,
  },
]

onMounted(loadStores)
</script>

<template>
  <ResourceListView
    ref="listRef"
    title="座位管理"
    description="座位必须绑定已有桌子，所属门店由桌子自动确定"
    :breadcrumb="['桌位管理', '座位管理']"
    :fields="fields"
    :columns="columns"
    :fetcher="seatService.list"
    :toolbar-actions="toolbarActions"
    empty-text="暂无座位，请先在桌子管理中新增桌子"
  />

  <FormDrawer
    v-model:show="drawerShow"
    :title="editingId ? '编辑座位' : '新增座位'"
    :submitting="submitting"
    high-risk
    cross-store
    :submit-text="editingId ? '保存' : '创建'"
    @submit="submit"
  >
    <NAlert
      type="info"
      :bordered="false"
      class="dependency-hint"
    >
      添加座位前必须先添加桌子；座位的所属门店将跟随桌子。
    </NAlert>
    <NForm label-placement="top">
      <NFormItem
        label="所属门店"
        required
      >
        <NSelect
          v-model:value="form.storeId"
          :options="storeOptions"
          placeholder="请先选择门店"
          filterable
        />
      </NFormItem>
      <NFormItem
        label="所属桌子"
        required
      >
        <NSelect
          v-model:value="form.tableId"
          :options="tableOptions"
          :loading="loadingTables"
          :disabled="!form.storeId"
          :placeholder="form.storeId ? '请选择桌子' : '请先选择门店'"
          filterable
        />
      </NFormItem>
      <NFormItem
        label="座位名称"
        required
      >
        <NInput
          v-model:value="form.name"
          placeholder="如：1号位"
          maxlength="64"
        />
      </NFormItem>
      <NFormItem
        label="状态"
        required
      >
        <NSelect
          v-model:value="form.status"
          :options="TABLE_SEAT_STATUS_OPTIONS.map(({ label, value }) => ({ label, value }))"
          placeholder="请选择状态"
        />
      </NFormItem>
    </NForm>
  </FormDrawer>
</template>

<style scoped>
.dependency-hint {
  margin-bottom: 18px;
}
</style>
