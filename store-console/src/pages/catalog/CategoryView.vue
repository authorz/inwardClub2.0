<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NModal, NSelect, NSpace, type DataTableColumns } from 'naive-ui'
import { catalogService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { confirm } from '@/composables/useConfirm'
import { feedback } from '@/utils/feedback'
import { PERM } from '@/constants/permissions'
import { ACTIVE_STATUS, toOptions } from '@/constants/enums'
import { AssetImage, AssetUpload, DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import { statusColumn, textColumn } from '@/utils/columns'
import type { CatalogCategory } from '@/types/models'

const list = useAsyncList<CatalogCategory>((params) => catalogService.categories(params), { initialFilters: { keyword: '', status: '' } })
const statuses = toOptions(ACTIVE_STATUS)
const selectStatuses = statuses.map(({ label, value }) => ({ label, value }))
const show = ref(false)
const saving = ref(false)
const form = reactive({
  id: null as string | number | null,
  name: '',
  assetId: null as string | null,
  imageUrl: '',
  sortOrder: 0,
  status: 'active',
})

function open(row?: CatalogCategory): void {
  Object.assign(form, row ? {
    id: row.id,
    name: row.name,
    assetId: row.assetId == null ? null : String(row.assetId),
    imageUrl: row.imageUrl ?? '',
    sortOrder: row.sortOrder ?? 0,
    status: row.status ?? 'active',
  } : {
    id: null,
    name: '',
    assetId: null,
    imageUrl: '',
    sortOrder: 0,
    status: 'active',
  })
  show.value = true
}
async function save(): Promise<void> {
  if (!form.name.trim()) return void feedback.message.error('请填写分类名称')
  saving.value = true
  try {
    const body = {
      name: form.name.trim(),
      parentId: null,
      assetId: form.assetId ? Number(form.assetId) : null,
      sortOrder: form.sortOrder,
      status: form.status,
    }
    if (form.id == null) await catalogService.createCategory(body)
    else await catalogService.updateCategory(form.id, body)
    feedback.message.success('分类已保存'); show.value = false; list.refresh()
  } catch (error) { feedback.message.error((error as { message?: string }).message ?? '保存失败') }
  finally { saving.value = false }
}
async function remove(row: CatalogCategory): Promise<void> {
  if (!await confirm({ content: `确认删除分类“${row.name}”？`, danger: true })) return
  try { await catalogService.deleteCategory(row.id); feedback.message.success('分类已删除'); list.refresh() }
  catch (error) { feedback.message.error((error as { message?: string }).message ?? '删除失败') }
}
const columns = computed<DataTableColumns<CatalogCategory>>(() => [
  textColumn<CatalogCategory>('分类 ID', (row) => row.id, { width: 90 }),
  {
    title: '图标',
    key: 'icon',
    width: 72,
    render: (row) => h(AssetImage, { src: row.imageUrl, assetId: row.assetId, width: 44, height: 44 }),
  },
  textColumn<CatalogCategory>('分类名称', (row) => row.name),
  textColumn<CatalogCategory>('排序', (row) => row.sortOrder ?? 0, { width: 90 }),
  statusColumn<CatalogCategory>('状态', ACTIVE_STATUS, (row) => row.status, { width: 100 }),
  { title: '操作', key: 'actions', width: 130, render: (row) => h(NSpace, { size: 4 }, () => [
    h(PermissionButton, { permissions: [PERM.catalogWrite], text: true, onClick: () => open(row) }, () => '编辑'),
    h(PermissionButton, { permissions: [PERM.catalogWrite], text: true, type: 'error', onClick: () => remove(row) }, () => '删除'),
  ]) },
])
</script>

<template>
  <div>
    <PageHeader
      title="商品分类"
      description="维护当前门店自有商品分类"
    />
    <StatusFilterBar
      :status-options="statuses"
      :status="list.filters.status as string"
      :keyword="list.filters.keyword as string"
      :loading="list.loading.value"
      search-placeholder="搜索分类名称"
      @update:status="list.filters.status = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.catalogWrite]"
          type="primary"
          @click="open()"
        >
          新增分类
        </PermissionButton>
      </template>
    </StatusFilterBar>
    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无分类"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
    <NModal
      v-model:show="show"
      preset="card"
      :title="form.id == null ? '新增分类' : '编辑分类'"
      style="width: 480px"
    >
      <NForm label-placement="top">
        <NFormItem
          label="分类名称"
          required
        >
          <NInput v-model:value="form.name" />
        </NFormItem>
        <NFormItem label="分类图标">
          <AssetUpload
            v-model:asset-id="form.assetId"
            v-model:preview-url="form.imageUrl"
            purpose="category"
            :width="64"
            :height="64"
            compact
            clearable
          />
        </NFormItem>
        <NFormItem label="排序">
          <NInputNumber
            v-model:value="form.sortOrder"
            :min="0"
          />
        </NFormItem><NFormItem label="状态">
          <NSelect
            v-model:value="form.status"
            :options="selectStatuses"
          />
        </NFormItem>
      </NForm>
      <template #footer>
        <NSpace justify="end">
          <NButton @click="show = false">
            取消
          </NButton><NButton
            type="primary"
            :loading="saving"
            @click="save"
          >
            保存
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>
