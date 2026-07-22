<script setup lang="ts">
/**
 * Banner 管理。列表 + 新增/编辑（骨架）。
 * 图片仅接受 assetId；接入资产选择器后补充。
 */
import { h, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import AssetImage from '@/components/AssetImage.vue'
import { actionsColumn, dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { bannerService } from '@/api/services'
import type { Banner } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: 'Banner 标题', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const columns = [
  renderColumn<Banner>(
    '图片',
    'assetId',
    (row) => h(AssetImage, { assetId: row.assetId, width: 56, height: 32 }),
    80,
  ),
  textColumn<Banner>('标题', 'title'),
  textColumn<Banner>('排序', 'sortOrder', { width: 90 }),
  statusColumn<Banner>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<Banner>('创建时间', 'createdAt'),
  actionsColumn<Banner>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.BANNER_WRITE, onClick: () => openEdit(row) },
          () => '编辑',
        ),
        h(
          PermissionButton,
          { permission: PERMISSIONS.BANNER_WRITE, type: 'error', onClick: () => remove(row) },
          () => '删除',
        ),
      ]),
    160,
  ),
]

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<Partial<Banner>>({ title: '', sortOrder: 0 })

function openCreate(): void {
  editingId.value = null
  form.title = ''
  form.sortOrder = 0
  drawerShow.value = true
}
function openEdit(row: Banner): void {
  editingId.value = row.id
  form.title = row.title
  form.sortOrder = row.sortOrder ?? 0
  drawerShow.value = true
}
async function submit(): Promise<void> {
  if (!form.title) return toastError('请填写标题')
  submitting.value = true
  try {
    if (editingId.value) await bannerService.update(editingId.value, form)
    else await bannerService.create(form)
    toastSuccess('已保存')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}
async function remove(row: Banner): Promise<void> {
  const ok = await runAudited({
    title: '删除 Banner',
    content: `确认删除「${row.title}」？`,
    execute: () => bannerService.remove(row.id),
    successText: '已删除',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增 Banner',
    type: 'primary' as const,
    permission: PERMISSIONS.BANNER_WRITE,
    onClick: openCreate,
  },
]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="Banner 管理"
      :breadcrumb="['Banner 管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="bannerService.list"
      :toolbar-actions="toolbarActions"
    />
    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑 Banner' : '新增 Banner'"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="标题"
          required
        >
          <NInput
            v-model:value="form.title"
            placeholder="请输入标题"
          />
        </NFormItem>
        <NFormItem label="排序">
          <NInputNumber
            v-model:value="form.sortOrder"
            :min="0"
            style="width: 100%"
          />
        </NFormItem>
      </NForm>
      <p class="form-note">
        Banner 图片仅接受 assetId，接入资产选择器后补充上传。
      </p>
    </FormDrawer>
  </div>
</template>

<style scoped>
.form-note {
  margin-top: var(--ic-space-sm);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
</style>
