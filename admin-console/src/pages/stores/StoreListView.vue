<script setup lang="ts">
/**
 * 门店管理（全局）。
 * 列表 + 新增/编辑（FormDrawer）。门店是全局资源，写操作展示审计提示。
 * 复用：ResourceListView（列表骨架）、FormDrawer（表单）、列工厂、PermissionButton。
 */
import { h, reactive, ref } from 'vue'
import {
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSpace,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import AssetUpload from '@/components/AssetUpload.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { STORE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { storeService } from '@/api/services'
import type { Store } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: '门店名称', type: 'input', placeholder: '搜索门店名称 / 电话' },
  { key: 'status', label: '状态', type: 'select', options: STORE_STATUS_OPTIONS },
]

const columns = [
  textColumn<Store>('ID', 'id', { width: 80 }),
  textColumn<Store>('门店名称', 'name'),
  textColumn<Store>('门店地址', 'address', { width: 260 }),
  textColumn<Store>('联系电话', 'phone', { width: 140 }),
  statusColumn<Store>('营业状态', 'status', STORE_STATUS_OPTIONS),
  dateTimeColumn<Store>('创建时间', 'createdAt'),
  actionsColumn<Store>((row) =>
    h(NSpace, {}, () => [
      h(
        PermissionButton,
        { permission: PERMISSIONS.STORE_WRITE, onClick: () => openEdit(row) },
        () => '编辑',
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
  businessHours: '',
  latitude: null,
  longitude: null,
  customerServiceQrAssetId: null,
  customerServiceQrUrl: '',
})

function resetForm(): void {
  form.name = ''
  form.phone = ''
  form.address = ''
  form.businessHours = ''
  form.latitude = null
  form.longitude = null
  form.customerServiceQrAssetId = null
  form.customerServiceQrUrl = ''
  coordPaste.value = ''
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

async function openEdit(row: Store): Promise<void> {
  editingId.value = row.id
  resetForm()
  drawerShow.value = true
  try {
    const detail = await storeService.get(row.id)
    form.name = detail.name
    form.phone = detail.phone
    form.address = detail.address
    form.businessHours = detail.businessHours ?? ''
    form.latitude = detail.latitude ?? null
    form.longitude = detail.longitude ?? null
    form.customerServiceQrAssetId = detail.customerServiceQrAssetId ?? null
    form.customerServiceQrUrl = detail.customerServiceQrUrl ?? ''
    coordPaste.value = ''
  } catch (e) {
    drawerShow.value = false
    toastError((e as { message?: string }).message ?? '加载门店资料失败')
  }
}

// —— 坐标拾取 ——
// 微信小程序 wx.getLocation/openLocation 用 GCJ-02，故坐标必须是 GCJ-02：
// 用腾讯（或高德）坐标拾取器，切勿用百度（BD-09 会偏移数百米）。
const coordPaste = ref('')

function openCoordPicker(): void {
  window.open('https://lbs.qq.com/getPoint/', '_blank')
}

// 接受拾取器复制的「纬度,经度」（腾讯输出该顺序），拆分回填经纬度输入框。
function applyPastedCoord(v: string): void {
  const nums = v
    .split(/[,，\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .map(Number)
    .filter((n) => Number.isFinite(n))
  if (nums.length >= 2) {
    form.latitude = nums[0]
    form.longitude = nums[1]
    coordPaste.value = ''
  }
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
    const payload: Partial<Store> = {
      ...form,
      customerServiceQrAssetId: form.customerServiceQrAssetId
        ? Number(form.customerServiceQrAssetId)
        : null,
    }
    if (editingId.value) {
      await storeService.update(editingId.value, payload)
      toastSuccess('门店已更新')
    } else {
      await storeService.create(payload)
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
      :width="860"
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
        <div class="form-row">
          <NFormItem
            label="客服电话 / 联系方式"
            class="form-row__item"
          >
            <NInput
              v-model:value="form.phone"
              placeholder="请输入客服电话或门店联系电话"
            />
          </NFormItem>
          <NFormItem
            label="营业时间"
            class="form-row__item"
          >
            <NInput
              v-model:value="form.businessHours"
              placeholder="如：10:00 - 22:00"
            />
          </NFormItem>
        </div>
        <NFormItem label="客服微信二维码">
          <AssetUpload
            v-model:asset-id="form.customerServiceQrAssetId"
            v-model:public-url="form.customerServiceQrUrl"
            purpose="store_contact_qr"
            :preview-url="form.customerServiceQrUrl"
            :width="150"
            :height="150"
          />
        </NFormItem>
        <NFormItem
          label="地址"
          required
        >
          <NInput
            v-model:value="form.address"
            type="textarea"
            :autosize="{ minRows: 1, maxRows: 2 }"
            placeholder="请输入门店地址"
          />
        </NFormItem>
        <NFormItem label="经纬度（GPS · GCJ-02）">
          <div class="coord">
            <div class="coord__row">
              <NInputNumber
                v-model:value="form.latitude"
                :show-button="false"
                :precision="6"
                :min="-90"
                :max="90"
                placeholder="纬度 latitude 如 31.230416"
                style="flex: 1"
              />
              <NInputNumber
                v-model:value="form.longitude"
                :show-button="false"
                :precision="6"
                :min="-180"
                :max="180"
                placeholder="经度 longitude 如 121.473701"
                style="flex: 1"
              />
            </div>
            <div class="coord__row">
              <NButton
                secondary
                @click="openCoordPicker"
              >
                打开腾讯地图拾取坐标
              </NButton>
              <NInput
                v-model:value="coordPaste"
                placeholder="粘贴「纬度,经度」自动填入"
                style="flex: 1"
                @change="applyPastedCoord"
              />
            </div>
          </div>
        </NFormItem>
      </NForm>
      <p class="form-note">
        坐标用于小程序「距离计算」与「导航前往」。请用<strong>腾讯 / 高德</strong>坐标拾取器（GCJ-02，与微信一致），<strong>勿用百度</strong>（坐标系不同，会偏移数百米）。在拾取器里搜门店地址 → 点地图 → 复制「纬度,经度」粘贴到上方即可。
      </p>
    </FormDrawer>
  </div>
</template>

<style scoped>
.form-note {
  margin-top: var(--ic-space-md);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.form-row {
  display: flex;
  gap: 12px;
}
.form-row__item {
  flex: 1;
}
.coord {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.coord__row {
  display: flex;
  gap: 8px;
  align-items: center;
}
</style>
