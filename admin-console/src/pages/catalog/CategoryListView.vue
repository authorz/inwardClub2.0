<script setup lang="ts">
/**
 * 全局分类。列表 + 新增/编辑（骨架）。
 * 分类为全局模板；门店可引用为父级/展示分组。
 */
import { h, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS, SCOPE_TYPE_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { categoryService } from '@/api/services'
import type { CatalogCategory } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: '分类名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const columns = [
  textColumn<CatalogCategory>('分类名称', 'name'),
  statusColumn<CatalogCategory>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  textColumn<CatalogCategory>('排序', 'sortOrder', { width: 90 }),
  statusColumn<CatalogCategory>('状态', 'status', RESOURCE_STATUS_OPTIONS),
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
// status 为服务端必填字段（CategoryInput.Status binding:required），默认已发布。
const form = reactive<Partial<CatalogCategory>>({ name: '', sortOrder: 0, status: 'published' })

function openCreate(): void {
  editingId.value = null
  form.name = ''
  form.sortOrder = 0
  form.status = 'published'
  drawerShow.value = true
}
function openEdit(row: CatalogCategory): void {
  editingId.value = row.id
  form.name = row.name
  form.sortOrder = row.sortOrder ?? 0
  form.status = row.status ?? 'published'
  drawerShow.value = true
}
async function submit(): Promise<void> {
  if (!form.name) return toastError('请填写分类名称')
  submitting.value = true
  try {
    if (editingId.value) await categoryService.update(editingId.value, form)
    else await categoryService.create(form)
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
    label: '新增全局分类',
    type: 'primary' as const,
    permission: PERMISSIONS.CATALOG_GLOBAL_WRITE,
    onClick: openCreate,
  },
]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="全局分类"
      description="全局商品分类模板"
      :breadcrumb="['商品与分类', '全局分类']"
      :fields="fields"
      :columns="columns"
      :fetcher="categoryService.list"
      :toolbar-actions="toolbarActions"
    />
    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑分类' : '新增分类'"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="分类名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="请输入分类名称"
          />
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
            :options="RESOURCE_STATUS_OPTIONS.map((o) => ({ label: o.label, value: o.value }))"
            placeholder="请选择状态"
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>
