<script setup lang="ts">
/**
 * 门店管理（全局）。
 * 列表 + 新增/编辑（FormDrawer）。门店是全局资源，写操作展示审计提示。
 * 复用：ResourceListView（列表骨架）、FormDrawer（表单）、列工厂、PermissionButton。
 */
import { h, reactive, ref } from 'vue'
import {
  NDescriptions,
  NDescriptionsItem,
  NForm,
  NFormItem,
  NInput,
  NSpace,
  NSpin,
  NSwitch,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { storeService, storeSettingsService } from '@/api/services'
import type { Store, StoreSettingsData } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: '门店名称', type: 'input', placeholder: '搜索门店名称 / 电话' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const columns = [
  textColumn<Store>('门店名称', 'name'),
  textColumn<Store>('联系电话', 'phone', { width: 140 }),
  statusColumn<Store>('营业状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<Store>('创建时间', 'createdAt'),
  actionsColumn<Store>((row) =>
    h(NSpace, {}, () => [
      h(
        PermissionButton,
        { permission: PERMISSIONS.STORE_WRITE, onClick: () => openEdit(row) },
        () => '编辑',
      ),
      h(
        PermissionButton,
        { permission: PERMISSIONS.STORE_READ, onClick: () => openSettings(row) },
        () => '详情/设置',
      ),
    ]),
  ),
]

// —— 新增 / 编辑表单 ——
const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<Partial<Store>>({
  name: '',
  phone: '',
  address: '',
})

function resetForm(): void {
  form.name = ''
  form.phone = ''
  form.address = ''
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

function openEdit(row: Store): void {
  editingId.value = row.id
  form.name = row.name
  form.phone = row.phone
  form.address = row.address
  drawerShow.value = true
}

async function submit(): Promise<void> {
  if (!form.name) {
    toastError('请填写门店名称')
    return
  }
  if (!form.address) {
    toastError('请填写门店地址')
    return
  }
  submitting.value = true
  try {
    if (editingId.value) {
      await storeService.update(editingId.value, form)
      toastSuccess('门店已更新')
    } else {
      await storeService.create(form)
      toastSuccess('门店已创建')
    }
    drawerShow.value = false
    listRef.value?.reload()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}

// —— 详情 / 设置抽屉 ——
const settingsDrawerShow = ref(false)
const settingsLoading = ref(false)
const settingsSaving = ref(false)
const settingsStoreId = ref<string | null>(null)
const detail = ref<Store | null>(null)
const settingsForm = reactive<StoreSettingsData>({
  businessHoursNote: '',
  autoAcceptOrders: false,
  tableReservationEnabled: false,
})

async function openSettings(row: Store): Promise<void> {
  settingsStoreId.value = row.id
  detail.value = null
  settingsDrawerShow.value = true
  settingsLoading.value = true
  try {
    const [detailData, settingsData] = await Promise.all([
      storeService.get(row.id),
      storeSettingsService.get(row.id),
    ])
    detail.value = detailData
    Object.assign(settingsForm, settingsData.settings ?? {})
  } catch (e) {
    toastError((e as { message?: string }).message ?? '加载门店详情/设置失败')
    settingsDrawerShow.value = false
  } finally {
    settingsLoading.value = false
  }
}

async function submitSettings(): Promise<void> {
  if (!settingsStoreId.value) return
  settingsSaving.value = true
  try {
    await storeSettingsService.update(settingsStoreId.value, { ...settingsForm })
    toastSuccess('门店设置已更新')
    settingsDrawerShow.value = false
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    settingsSaving.value = false
  }
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增门店',
    type: 'primary' as const,
    permission: PERMISSIONS.STORE_WRITE,
    onClick: openCreate,
  },
]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="门店管理"
      description="管理全局门店，写操作将写入审计日志"
      :breadcrumb="['门店管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="storeService.list"
      :toolbar-actions="toolbarActions"
    />

    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑门店' : '新增门店'"
      :submitting="submitting"
      high-risk
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="门店名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="请输入门店名称"
          />
        </NFormItem>
        <NFormItem label="联系电话">
          <NInput
            v-model:value="form.phone"
            placeholder="请输入联系电话"
          />
        </NFormItem>
        <NFormItem
          label="地址"
          required
        >
          <NInput
            v-model:value="form.address"
            type="textarea"
            placeholder="请输入门店地址"
          />
        </NFormItem>
      </NForm>
      <p class="form-note">
        Logo 等资产字段仅接受 assetId，接入资产服务后补充。
      </p>
    </FormDrawer>

    <FormDrawer
      v-model:show="settingsDrawerShow"
      title="门店详情 / 设置"
      :submitting="settingsSaving"
      high-risk
      submit-text="保存设置"
      @submit="submitSettings"
    >
      <NSpin :show="settingsLoading">
        <NDescriptions
          v-if="detail"
          label-placement="left"
          :column="1"
          bordered
        >
          <NDescriptionsItem label="门店名称">
            {{ detail.name }}
          </NDescriptionsItem>
          <NDescriptionsItem label="联系电话">
            {{ detail.phone }}
          </NDescriptionsItem>
          <NDescriptionsItem label="地址">
            {{ detail.address }}
          </NDescriptionsItem>
          <NDescriptionsItem label="营业时间">
            {{ detail.businessHours }}
          </NDescriptionsItem>
          <NDescriptionsItem label="营业状态">
            {{ detail.status }}
          </NDescriptionsItem>
        </NDescriptions>

        <NForm
          label-placement="top"
          style="margin-top: var(--ic-space-md)"
        >
          <NFormItem label="营业时间备注">
            <NInput
              v-model:value="settingsForm.businessHoursNote"
              placeholder="请输入营业时间备注"
            />
          </NFormItem>
          <NFormItem label="自动接单">
            <NSwitch v-model:value="settingsForm.autoAcceptOrders" />
          </NFormItem>
          <NFormItem label="启用桌位预订">
            <NSwitch v-model:value="settingsForm.tableReservationEnabled" />
          </NFormItem>
        </NForm>
      </NSpin>
    </FormDrawer>
  </div>
</template>

<style scoped>
.form-note {
  margin-top: var(--ic-space-md);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
</style>
