<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import { actionsColumn, dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { PRINTER_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { printerService, storeService } from '@/api/services'
import type { PrinterDevice } from '@/api/models'
import { toastError, toastSuccess } from '@/utils/feedback'

interface PrinterForm {
  storeId: string | null
  name: string
  deviceSn: string
  deviceKey: string
  status: 'active' | 'disabled'
  reason: string
}

const listRef = ref<ResourceListInstance | null>(null)
const storeOptions = ref<Array<{ label: string; value: string }>>([])
const storeNames = ref(new Map<string, string>())
const formShow = ref(false)
const formSubmitting = ref(false)
const editingId = ref<string | null>(null)
const deleteShow = ref(false)
const deleteSubmitting = ref(false)
const deleteTarget = ref<PrinterDevice | null>(null)
const deleteReason = ref('')
const form = reactive<PrinterForm>({
  storeId: null,
  name: '',
  deviceSn: '',
  deviceKey: '',
  status: 'active',
  reason: '',
})
const printerFormStatusOptions = PRINTER_STATUS_OPTIONS.map(({ label, value }) => ({ label, value }))

const fields = computed<FilterField[]>(() => [
  {
    key: 'storeId',
    label: '所属门店',
    type: 'select',
    options: storeOptions.value,
    width: 220,
  },
  {
    key: 'keyword',
    label: '打印机',
    type: 'input',
    placeholder: '搜索名称 / 设备 SN',
    width: 240,
  },
  { key: 'status', label: '状态', type: 'select', options: PRINTER_STATUS_OPTIONS },
])

function storeName(storeId: string | number): string {
  return storeNames.value.get(String(storeId)) ?? `门店 #${storeId}`
}

const columns = [
  textColumn<PrinterDevice>('ID', 'id', { width: 80 }),
  renderColumn<PrinterDevice>('所属门店', 'storeId', (row) => storeName(row.storeId), 180),
  textColumn<PrinterDevice>('打印机名称', 'name', { width: 170 }),
  textColumn<PrinterDevice>('设备 SN', 'deviceSn', { width: 180 }),
  renderColumn<PrinterDevice>('服务商', 'provider', (row) =>
    row.provider === 'xpyun' ? '芯烨云' : row.provider, 110),
  statusColumn<PrinterDevice>('状态', 'status', PRINTER_STATUS_OPTIONS, 110),
  dateTimeColumn<PrinterDevice>('更新时间', 'updatedAt'),
  actionsColumn<PrinterDevice>(
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
            onClick: () => openDelete(row),
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
    storeNames.value = new Map(result.items.map((store) => [String(store.id), store.name]))
  } catch (error) {
    toastError((error as { message?: string }).message ?? '门店列表加载失败')
  }
}

function resetForm(): void {
  form.storeId = null
  form.name = ''
  form.deviceSn = ''
  form.deviceKey = ''
  form.status = 'active'
  form.reason = ''
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  formShow.value = true
}

function openEdit(row: PrinterDevice): void {
  editingId.value = String(row.id)
  form.storeId = String(row.storeId)
  form.name = row.name
  form.deviceSn = row.deviceSn
  form.deviceKey = ''
  form.status = row.status
  form.reason = ''
  formShow.value = true
}

async function submitForm(): Promise<void> {
  if (!form.storeId) return toastError('请选择所属门店')
  if (!form.name.trim()) return toastError('请输入打印机名称')
  if (!form.deviceSn.trim()) return toastError('请输入设备 SN')
  if (!form.reason.trim()) return toastError('请填写操作原因')

  formSubmitting.value = true
  try {
    if (editingId.value) {
      await printerService.update(editingId.value, {
        name: form.name.trim(),
        deviceSn: form.deviceSn.trim(),
        ...(form.deviceKey ? { deviceKey: form.deviceKey } : {}),
        status: form.status,
        reason: form.reason.trim(),
      })
      toastSuccess('打印机已更新')
    } else {
      await printerService.create({
        storeId: Number(form.storeId),
        name: form.name.trim(),
        provider: 'xpyun',
        deviceSn: form.deviceSn.trim(),
        ...(form.deviceKey ? { deviceKey: form.deviceKey } : {}),
        status: form.status,
        reason: form.reason.trim(),
      })
      toastSuccess('打印机已新增')
    }
    formShow.value = false
    await listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    formSubmitting.value = false
  }
}

function openDelete(row: PrinterDevice): void {
  deleteTarget.value = row
  deleteReason.value = ''
  deleteShow.value = true
}

async function submitDelete(): Promise<void> {
  if (!deleteTarget.value) return
  if (!deleteReason.value.trim()) return toastError('请填写操作原因')
  deleteSubmitting.value = true
  try {
    await printerService.remove(String(deleteTarget.value.id), deleteReason.value.trim())
    toastSuccess('打印机已删除')
    deleteShow.value = false
    deleteTarget.value = null
    await listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '删除失败')
  } finally {
    deleteSubmitting.value = false
  }
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增打印机',
    type: 'primary' as const,
    permission: PERMISSIONS.STORE_WRITE,
    onClick: openCreate,
  },
]

onMounted(loadStores)
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="打印机管理"
      description="统一维护各门店的小票打印设备；跨店写操作将记录目标门店、操作原因和前后差异"
      :breadcrumb="['门店管理', '打印机管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="printerService.list"
      :toolbar-actions="toolbarActions"
      empty-text="暂无打印机，可选择门店后新增设备"
    />

    <FormDrawer
      v-model:show="formShow"
      :title="editingId ? '编辑打印机' : '新增打印机'"
      :submitting="formSubmitting"
      :submit-text="editingId ? '保存修改' : '确认新增'"
      high-risk
      cross-store
      @submit="submitForm"
    >
      <NForm
        label-placement="top"
        autocomplete="off"
      >
        <NFormItem
          label="所属门店"
          required
        >
          <NSelect
            v-model:value="form.storeId"
            :options="storeOptions"
            :disabled="Boolean(editingId)"
            filterable
            placeholder="请选择打印机所属门店"
          />
        </NFormItem>
        <NFormItem
          label="打印机名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="如：前台小票机"
            maxlength="64"
          />
        </NFormItem>
        <NFormItem label="服务商">
          <NSelect
            :value="'xpyun'"
            :options="[{ label: '芯烨云', value: 'xpyun' }]"
            disabled
          />
        </NFormItem>
        <NFormItem
          label="设备 SN"
          required
        >
          <NInput
            v-model:value="form.deviceSn"
            :input-props="{ name: 'printer-device-sn', autocomplete: 'off' }"
            placeholder="请输入设备序列号"
            maxlength="64"
          />
        </NFormItem>
        <NFormItem label="设备密钥">
          <NInput
            v-model:value="form.deviceKey"
            type="password"
            :input-props="{ name: 'printer-device-key', autocomplete: 'new-password' }"
            show-password-on="click"
            :placeholder="editingId ? '留空则不修改' : '请输入设备密钥（可选）'"
            maxlength="64"
          />
        </NFormItem>
        <NFormItem
          label="设备状态"
          required
        >
          <NSelect
            v-model:value="form.status"
            :options="printerFormStatusOptions"
          />
        </NFormItem>
        <NFormItem
          label="操作原因"
          required
        >
          <NInput
            v-model:value="form.reason"
            type="textarea"
            placeholder="请说明新增或修改原因"
            maxlength="200"
            show-count
          />
        </NFormItem>
      </NForm>
    </FormDrawer>

    <FormDrawer
      v-model:show="deleteShow"
      title="删除打印机"
      :submitting="deleteSubmitting"
      submit-text="确认删除"
      high-risk
      cross-store
      @submit="submitDelete"
    >
      <NForm label-placement="top">
        <NFormItem label="打印机">
          <NInput
            :value="deleteTarget ? `${storeName(deleteTarget.storeId)} · ${deleteTarget.name}` : ''"
            disabled
          />
        </NFormItem>
        <NFormItem
          label="操作原因"
          required
        >
          <NInput
            v-model:value="deleteReason"
            type="textarea"
            placeholder="请说明删除原因"
            maxlength="200"
            show-count
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>
