<script setup lang="ts">
/**
 * 总后台商品分类：每个分类必须归属一个门店。
 * 列表支持门店、名称和状态筛选；编辑旧全局数据时必须补齐门店。
 */
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace, NText } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import AssetImage from '@/components/AssetImage.vue'
import AssetUpload from '@/components/AssetUpload.vue'
import { actionsColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS, type OptionItem } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { categoryService, storeService } from '@/api/services'
import type { CatalogCategory } from '@/api/models'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)
const storeOptions = ref<OptionItem[]>([])
const categoryStatusOptions = RESOURCE_STATUS_OPTIONS.filter(({ value }) =>
  ['active', 'disabled'].includes(value),
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
    label: '分类名称',
    type: 'input',
    placeholder: '支持名称模糊搜索',
    width: 220,
  },
  { key: 'status', label: '状态', type: 'select', options: categoryStatusOptions },
])

const columns = [
  textColumn<CatalogCategory>('分类 ID', 'id', { width: 90 }),
  renderColumn<CatalogCategory>(
    '分类名称',
    'name',
    (row) =>
      h(NSpace, { align: 'center', size: 10, wrap: false }, () => [
        row.imageUrl
          ? h(AssetImage, { src: row.imageUrl, width: 36, height: 36 })
          : null,
        h(NText, {}, () => row.name),
      ]),
  ),
  renderColumn<CatalogCategory>(
    '所属门店',
    'storeName',
    (row) =>
      row.storeName
        ? row.storeName
        : h(NText, { type: 'error', depth: 1 }, () => '未绑定（需修复）'),
    180,
  ),
  textColumn<CatalogCategory>('排序', 'sortOrder', { width: 90 }),
  statusColumn<CatalogCategory>('状态', 'status', categoryStatusOptions),
  actionsColumn<CatalogCategory>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.CATALOG_GLOBAL_WRITE, onClick: () => openEdit(row) },
          () => '编辑',
        ),
      ]),
    120,
  ),
]

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const currentIconUrl = ref('')
const uploadKey = ref(0)
const form = reactive<Partial<CatalogCategory>>({
  storeId: null,
  name: '',
  assetId: null,
  sortOrder: 0,
  status: 'active',
})

async function loadStores(): Promise<void> {
  try {
    const result = await storeService.list({ page: 1, pageSize: 100 })
    storeOptions.value = result.items.map((store) => ({
      label: store.name,
      value: String(store.id),
    }))
  } catch (e) {
    toastError((e as { message?: string }).message ?? '门店列表加载失败')
  }
}

function openCreate(): void {
  editingId.value = null
  form.storeId = null
  form.name = ''
  form.assetId = null
  form.sortOrder = 0
  form.status = 'active'
  currentIconUrl.value = ''
  uploadKey.value += 1
  drawerShow.value = true
}

function openEdit(row: CatalogCategory): void {
  editingId.value = row.id
  form.storeId = row.storeId == null ? null : String(row.storeId)
  form.name = row.name
  form.assetId = row.assetId ?? null
  form.sortOrder = row.sortOrder ?? 0
  form.status = row.status ?? 'active'
  currentIconUrl.value = row.imageUrl ?? ''
  uploadKey.value += 1
  drawerShow.value = true
}

function clearIcon(): void {
  form.assetId = null
  currentIconUrl.value = ''
  uploadKey.value += 1
}

async function submit(): Promise<void> {
  if (!form.storeId) return toastError('请选择所属门店')
  if (!form.name?.trim()) return toastError('请填写分类名称')

  const payload = {
    storeId: Number(form.storeId),
    name: form.name.trim(),
    assetId: form.assetId == null ? null : Number(form.assetId),
    sortOrder: form.sortOrder ?? 0,
    status: form.status ?? 'active',
  }
  submitting.value = true
  try {
    if (editingId.value) await categoryService.update(editingId.value, payload)
    else await categoryService.create(payload)
    toastSuccess('已保存')
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
    label: '新增商品分类',
    type: 'primary' as const,
    permission: PERMISSIONS.CATALOG_GLOBAL_WRITE,
    onClick: openCreate,
  },
]

onMounted(loadStores)
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="商品分类"
      description="按门店维护商品分类；商品只能选择同一门店下的分类"
      :breadcrumb="['商品管理', '商品分类']"
      :fields="fields"
      :columns="columns"
      :fetcher="categoryService.list"
      :toolbar-actions="toolbarActions"
      empty-text="暂无商品分类，请先选择门店后新增"
    />
    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑商品分类' : '新增商品分类'"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm label-placement="top">
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
          label="分类名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="请输入分类名称"
          />
        </NFormItem>
        <NFormItem label="分类图标">
          <NSpace
            vertical
            style="width: 100%"
          >
            <AssetUpload
              :key="uploadKey"
              v-model:asset-id="form.assetId"
              v-model:public-url="currentIconUrl"
              purpose="category"
              :preview-url="currentIconUrl || null"
              :width="80"
              :height="80"
            />
            <NButton
              v-if="form.assetId || currentIconUrl"
              text
              type="error"
              @click="clearIcon"
            >
              清除图标
            </NButton>
            <NText depth="3">
              建议上传正方形 PNG、JPG 或 WebP 图片。
            </NText>
          </NSpace>
        </NFormItem>
        <NFormItem label="排序">
          <NInputNumber
            v-model:value="form.sortOrder"
            :min="0"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="状态"
          required
        >
          <NSelect
            v-model:value="form.status"
            :options="categoryStatusOptions.map(({ label, value }) => ({ label, value }))"
            placeholder="请选择状态"
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>
