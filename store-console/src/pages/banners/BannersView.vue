<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { NButton, NForm, NFormItem, NInput, NInputNumber, NModal, NSelect, NSpace, type DataTableColumns } from 'naive-ui'
import { bannerService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { confirm } from '@/composables/useConfirm'
import { feedback } from '@/utils/feedback'
import { PERM } from '@/constants/permissions'
import type { EnumOption } from '@/constants/enums'
import { AssetImage, AssetUpload, DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import type { StoreBanner } from '@/types/models'

const statusMap: Record<string, EnumOption> = {
  active: { value: 'active', label: '显示中', tone: 'success' },
  inactive: { value: 'inactive', label: '已隐藏', tone: 'default' },
}
const statusOptions = Object.values(statusMap)
const statusSelectOptions = statusOptions.map(({ label, value }) => ({ label, value }))

const list = useAsyncList<StoreBanner>(async (params) => {
  const all = await bannerService.list()
  const keyword = String(params.keyword ?? '').trim().toLowerCase()
  const status = String(params.status ?? '')
  const filtered = all.filter((row) => (!status || row.status === status) && (!keyword || row.title.toLowerCase().includes(keyword)))
  const page = Number(params.page ?? 1)
  const pageSize = Number(params.pageSize ?? 20)
  return { rows: filtered.slice((page - 1) * pageSize, page * pageSize), page, pageSize, total: filtered.length }
}, { initialFilters: { keyword: '', status: '' } })

const show = ref(false)
const saving = ref(false)
const form = reactive({
  id: null as string | number | null, title: '', assetId: null as string | null,
  imageUrl: '', linkUrl: '', sortOrder: 0, status: 'active',
})

function open(row?: StoreBanner): void {
  Object.assign(form, row ? {
    id: row.id, title: row.title, assetId: String(row.assetId), imageUrl: row.imageUrl ?? '',
    linkUrl: row.linkUrl ?? '', sortOrder: row.sortOrder, status: row.status,
  } : { id: null, title: '', assetId: null, imageUrl: '', linkUrl: '', sortOrder: 0, status: 'active' })
  show.value = true
}

async function save(): Promise<void> {
  if (!form.title.trim()) return void feedback.message.error('请填写广告标题')
  if (!form.assetId) return void feedback.message.error('请上传广告图片')
  saving.value = true
  try {
    const body = {
      title: form.title.trim(), assetId: Number(form.assetId), linkUrl: form.linkUrl.trim(),
      sortOrder: form.sortOrder, status: form.status,
    }
    if (form.id == null) await bannerService.create(body)
    else await bannerService.update(form.id, body)
    feedback.message.success('广告已保存')
    show.value = false
    list.refresh()
  } catch (error) {
    feedback.message.error((error as { message?: string }).message ?? '保存失败')
  } finally { saving.value = false }
}

async function remove(row: StoreBanner): Promise<void> {
  if (!await confirm({ content: `确认删除广告“${row.title}”？`, danger: true })) return
  try { await bannerService.remove(row.id); feedback.message.success('广告已删除'); list.refresh() }
  catch (error) { feedback.message.error((error as { message?: string }).message ?? '删除失败') }
}

const columns = computed<DataTableColumns<StoreBanner>>(() => [
  { title: '广告图', key: 'imageUrl', width: 170, render: (row) => h(AssetImage, { src: row.imageUrl, width: 144, height: 64 }) },
  textColumn<StoreBanner>('标题', (row) => row.title),
  textColumn<StoreBanner>('跳转地址', (row) => row.linkUrl || '不跳转'),
  textColumn<StoreBanner>('排序', (row) => row.sortOrder, { width: 70 }),
  statusColumn<StoreBanner>('状态', statusMap, (row) => row.status, { width: 100 }),
  dateColumn<StoreBanner>('更新时间', (row) => row.updatedAt, { width: 150 }),
  { title: '操作', key: 'actions', width: 130, render: (row) => h(NSpace, { size: 4 }, () => [
    h(PermissionButton, { permissions: [PERM.activityWrite], text: true, onClick: () => open(row) }, () => '编辑'),
    h(PermissionButton, { permissions: [PERM.activityWrite], text: true, type: 'error', onClick: () => remove(row) }, () => '删除'),
  ]) },
])
</script>

<template>
  <div>
    <PageHeader
      title="本店广告"
      description="管理当前门店在小程序首页展示的广告和跳转地址"
    />
    <StatusFilterBar
      :status-options="statusOptions"
      :status="list.filters.status as string"
      :keyword="list.filters.keyword as string"
      :loading="list.loading.value"
      search-placeholder="搜索广告标题"
      @update:status="list.filters.status = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.activityWrite]"
          type="primary"
          @click="open()"
        >
          新增广告
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
      empty-text="暂无本店广告"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />
    <NModal
      v-model:show="show"
      preset="card"
      :title="form.id == null ? '新增广告' : '编辑广告'"
      style="width: 580px"
    >
      <NForm label-placement="top">
        <NFormItem
          label="广告标题"
          required
        >
          <NInput v-model:value="form.title" />
        </NFormItem>
        <NFormItem
          label="广告图片"
          required
        >
          <AssetUpload
            v-model:asset-id="form.assetId"
            v-model:preview-url="form.imageUrl"
            purpose="banner"
            :width="288"
            :height="128"
          />
        </NFormItem>
        <NFormItem label="跳转地址">
          <NInput
            v-model:value="form.linkUrl"
            placeholder="不填写则仅展示图片"
          />
        </NFormItem>
        <NFormItem label="排序">
          <NInputNumber
            v-model:value="form.sortOrder"
            :min="0"
          />
        </NFormItem>
        <NFormItem label="状态">
          <NSelect
            v-model:value="form.status"
            :options="statusSelectOptions"
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
