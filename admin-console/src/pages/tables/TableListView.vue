<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import { actionsColumn, dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { TABLE_SEAT_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { storeService, tableService } from '@/api/services'
import type { VenueTable } from '@/api/models'
import { runAudited } from '@/composables/useAuditedAction'
import { toastError, toastSuccess } from '@/utils/feedback'

interface TableForm {
  storeId: string | null
  name: string
  code: string
  capacity: number | null
  basePoints: number | null
  status: string
}

const listRef = ref<ResourceListInstance | null>(null)
const storeOptions = ref<Array<{ label: string; value: string }>>([])
const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<TableForm>({
  storeId: null,
  name: '',
  code: '',
  capacity: 0,
  basePoints: 0,
  status: 'available',
})

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
    label: '桌子',
    type: 'input',
    placeholder: '搜索桌子名称 / 编号',
    width: 220,
  },
  { key: 'status', label: '状态', type: 'select', options: TABLE_SEAT_STATUS_OPTIONS },
])

const columns = [
  textColumn<VenueTable>('桌子 ID', 'id', { width: 100 }),
  textColumn<VenueTable>('桌子名称', 'name', { width: 160 }),
  textColumn<VenueTable>('桌子编号', 'code', { width: 120 }),
  textColumn<VenueTable>('所属门店', 'storeName', { width: 180 }),
  renderColumn<VenueTable>(
    '座位',
    'seatCount',
    (row) => `${row.seatCount ?? 0} / ${row.capacity ?? 0}`,
    100,
  ),
  textColumn<VenueTable>('基础积分', 'basePoints', { width: 100 }),
  statusColumn<VenueTable>('状态', 'status', TABLE_SEAT_STATUS_OPTIONS, 110),
  dateTimeColumn<VenueTable>('更新时间', 'updatedAt'),
  actionsColumn<VenueTable>(
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

function resetForm(): void {
  form.storeId = null
  form.name = ''
  form.code = ''
  form.capacity = 0
  form.basePoints = 0
  form.status = 'available'
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

function openEdit(row: VenueTable): void {
  editingId.value = row.id
  form.storeId = row.storeId == null ? null : String(row.storeId)
  form.name = row.name
  form.code = row.code
  form.capacity = row.capacity
  form.basePoints = row.basePoints
  form.status = row.status ?? 'available'
  drawerShow.value = true
}

async function submit(): Promise<void> {
  if (!form.name.trim()) return toastError('请输入桌子名称')
  if (!form.code.trim()) return toastError('请输入桌子编号')
  if (form.capacity == null || form.capacity <= 0) return toastError('座位数量必须大于 0')
  if (form.basePoints == null || form.basePoints < 0) return toastError('请输入有效的基础积分')
  if (!form.storeId) return toastError('请选择所属门店')

  const payload = {
    storeId: Number(form.storeId),
    name: form.name.trim(),
    code: form.code.trim(),
    capacity: form.capacity,
    basePoints: form.basePoints,
    status: form.status,
  }
  submitting.value = true
  try {
    if (editingId.value) await tableService.update(editingId.value, payload)
    else await tableService.create(payload)
    toastSuccess(editingId.value ? '桌子已更新' : '桌子已创建')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}

async function remove(row: VenueTable): Promise<void> {
  const success = await runAudited({
    title: '删除桌子',
    content: `确认删除“${row.name}”吗？已有座位或预约记录的桌子不能删除。`,
    highRisk: true,
    positiveText: '确认删除',
    successText: '桌子已删除',
    execute: () => tableService.remove(row.id),
  })
  if (success) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增桌子',
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
    title="桌子管理"
    description="按门店维护可预约桌子；座位数量是容量上限，具体座位请在座位管理中添加"
    :breadcrumb="['桌位管理', '桌子管理']"
    :fields="fields"
    :columns="columns"
    :fetcher="tableService.list"
    :toolbar-actions="toolbarActions"
    empty-text="暂无桌子，请先新增桌子后再维护座位"
  />

  <FormDrawer
    v-model:show="drawerShow"
    :title="editingId ? '编辑桌子' : '新增桌子'"
    :submitting="submitting"
    :width="680"
    high-risk
    cross-store
    :submit-text="editingId ? '保存' : '创建'"
    @submit="submit"
  >
    <NForm label-placement="top">
      <div class="form-grid">
        <NFormItem
          label="桌子名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="请输入桌子名称"
            maxlength="64"
          />
        </NFormItem>
        <NFormItem
          label="桌子编号"
          required
        >
          <NInput
            v-model:value="form.code"
            placeholder="如：A-01"
            maxlength="64"
          />
        </NFormItem>
        <NFormItem
          label="座位数量"
          required
        >
          <NInputNumber
            v-model:value="form.capacity"
            :min="1"
            :precision="0"
            placeholder="请输入座位数量"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="基础积分"
          required
        >
          <NInputNumber
            v-model:value="form.basePoints"
            :min="0"
            :precision="0"
            placeholder="如：50"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="所属门店"
          required
        >
          <NSelect
            v-model:value="form.storeId"
            :options="storeOptions"
            placeholder="请选择门店"
            filterable
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
      </div>
    </NForm>
  </FormDrawer>
</template>

<style scoped>
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  column-gap: 20px;
}

@media (max-width: 640px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
