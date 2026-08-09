<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from 'vue'
import { NDatePicker, NForm, NFormItem, NInput, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import AssetImage from '@/components/AssetImage.vue'
import AssetUpload from '@/components/AssetUpload.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import { actionsColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS, RESOURCE_STATUS_OPTIONS, type OptionItem } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { storeService, tournamentEventService } from '@/api/services'
import type { TournamentEvent } from '@/api/models'
import { runAudited } from '@/composables/useAuditedAction'
import { toastError, toastSuccess } from '@/utils/feedback'
import { formatDateTime } from '@/utils/format'

const listRef = ref<ResourceListInstance | null>(null)
const storeOptions = ref<OptionItem[]>([])
const show = ref(false)
const saving = ref(false)
const editingId = ref<string | null>(null)
const uploadKey = ref(0)
const statusOptions = RESOURCE_STATUS_OPTIONS.filter(({ value }) =>
  [RESOURCE_STATUS.DRAFT, RESOURCE_STATUS.PUBLISHED].includes(value as 'draft' | 'published'),
)
const fields = computed<FilterField[]>(() => [
  { key: 'storeId', label: '所属门店', type: 'select', options: storeOptions.value, width: 200 },
  { key: 'keyword', label: '赛事标题', type: 'input', placeholder: '支持标题模糊搜索', width: 240 },
  { key: 'status', label: '状态', type: 'select', options: statusOptions },
])

const form = reactive({
  storeId: '' as string,
  title: '',
  summary: '',
  content: '',
  assetId: null as string | null,
  imageUrl: '',
  timeRange: null as [number, number] | null,
  status: RESOURCE_STATUS.PUBLISHED as string,
})

function period(row: TournamentEvent) {
  return h('div', { class: 'event-period' }, [
    h('span', {}, row.startAt ? formatDateTime(row.startAt) : '-'),
    h('span', { class: 'event-period__end' }, `至 ${row.endAt ? formatDateTime(row.endAt) : '-'}`),
  ])
}

const columns = [
  textColumn<TournamentEvent>('ID', 'id', { width: 70 }),
  renderColumn<TournamentEvent>('赛事图片', 'imageUrl', (row) => h(AssetImage, {
    src: row.imageUrl, assetId: row.assetId == null ? null : String(row.assetId), width: 112, height: 64,
  }), 140),
  textColumn<TournamentEvent>('赛事标题', 'title', { width: 220 }),
  textColumn<TournamentEvent>('所属门店', 'storeName', { width: 160 }),
  renderColumn<TournamentEvent>('赛事时间', 'period', period, 190),
  statusColumn<TournamentEvent>('状态', 'status', statusOptions, 100),
  actionsColumn<TournamentEvent>((row) => h(NSpace, {}, () => [
    h(PermissionButton, { permission: PERMISSIONS.ACTIVITY_GLOBAL_WRITE, onClick: () => void openEdit(row) }, () => '编辑'),
    h(PermissionButton, { permission: PERMISSIONS.ACTIVITY_GLOBAL_WRITE, type: 'error', onClick: () => void remove(row) }, () => '删除'),
  ]), 150),
]

async function loadStores(): Promise<void> {
  try {
    const result = await storeService.list({ page: 1, pageSize: 100 })
    storeOptions.value = result.items.map((row) => ({ label: row.name, value: String(row.id) }))
  } catch (error) {
    toastError((error as { message?: string }).message ?? '门店列表加载失败')
  }
}

function reset(): void {
  Object.assign(form, { storeId: '', title: '', summary: '', content: '', assetId: null, imageUrl: '', timeRange: null, status: RESOURCE_STATUS.PUBLISHED })
  uploadKey.value += 1
}

function openCreate(): void {
  editingId.value = null
  reset()
  show.value = true
}

async function openEdit(row: TournamentEvent): Promise<void> {
  try {
    const detail = await tournamentEventService.get(String(row.id))
    editingId.value = String(row.id)
    reset()
    Object.assign(form, {
      storeId: String(detail.storeId), title: detail.title, summary: detail.summary ?? '', content: detail.content ?? '',
      assetId: detail.assetId == null ? null : String(detail.assetId), imageUrl: detail.imageUrl ?? '',
      timeRange: detail.startAt && detail.endAt ? [Date.parse(detail.startAt), Date.parse(detail.endAt)] : null,
      status: detail.status ?? RESOURCE_STATUS.PUBLISHED,
    })
    show.value = true
  } catch (error) {
    toastError((error as { message?: string }).message ?? '赛事详情加载失败')
  }
}

async function submit(): Promise<void> {
  if (!form.storeId) return toastError('请选择所属门店')
  if (!form.title.trim()) return toastError('请填写赛事标题')
  if (!form.timeRange) return toastError('请设置赛事开始和结束时间')
  if (form.timeRange[0] >= form.timeRange[1]) return toastError('赛事结束时间必须晚于开始时间')
  saving.value = true
  try {
    const payload: Partial<TournamentEvent> = {
      storeId: Number(form.storeId), title: form.title.trim(), summary: form.summary.trim(), content: form.content,
      assetId: form.assetId ? Number(form.assetId) : null,
      startAt: new Date(form.timeRange[0]).toISOString(), endAt: new Date(form.timeRange[1]).toISOString(), status: form.status,
    }
    if (editingId.value) await tournamentEventService.update(editingId.value, payload)
    else await tournamentEventService.create(payload)
    toastSuccess(editingId.value ? '赛事活动已更新' : '赛事活动已创建')
    show.value = false
    listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally { saving.value = false }
}

async function remove(row: TournamentEvent): Promise<void> {
  const ok = await runAudited({
    title: '删除赛事活动', content: `确认删除「${row.title}」？`, highRisk: true, positiveText: '确认删除',
    execute: () => tournamentEventService.remove(String(row.id)), successText: '赛事活动已删除',
  })
  if (ok) listRef.value?.reload()
}

onMounted(loadStores)
</script>

<template>
  <div>
    <ResourceListView ref="listRef" title="赛事活动" description="发布门店赛事预告与图文详情；赛事活动不包含票档、支付和报名" :breadcrumb="['赛事活动']" :fields="fields" :columns="columns" :fetcher="tournamentEventService.list" :toolbar-actions="[{ key: 'create', label: '新增赛事活动', type: 'primary', permission: PERMISSIONS.ACTIVITY_GLOBAL_WRITE, onClick: openCreate }]" empty-text="暂无赛事活动" />
    <FormDrawer v-model:show="show" :title="editingId ? '编辑赛事活动' : '新增赛事活动'" :width="820" :submitting="saving" cross-store @submit="submit">
      <NForm label-placement="top">
        <div class="event-form__grid">
          <NFormItem label="所属门店" required><NSelect v-model:value="form.storeId" :options="storeOptions.map(({ label, value }) => ({ label, value }))" filterable /></NFormItem>
          <NFormItem label="发布状态" required><NSelect v-model:value="form.status" :options="statusOptions.map(({ label, value }) => ({ label, value }))" /></NFormItem>
        </div>
        <NFormItem label="赛事标题" required><NInput v-model:value="form.title" maxlength="128" show-count /></NFormItem>
        <NFormItem label="赛事时间" required><NDatePicker v-model:value="form.timeRange" type="datetimerange" style="width: 100%" /></NFormItem>
        <NFormItem label="赛事简介"><NInput v-model:value="form.summary" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" maxlength="500" show-count /></NFormItem>
        <NFormItem label="小 Banner"><AssetUpload :key="uploadKey" v-model:asset-id="form.assetId" purpose="activity" :preview-url="form.imageUrl || null" :width="320" :height="128" /></NFormItem>
        <NFormItem label="图文详情"><RichTextEditor v-model="form.content" placeholder="编辑赛事规则、流程和说明" /></NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>

<style scoped>
.event-form__grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--ic-space-md); }
.event-period { display: flex; flex-direction: column; line-height: 1.45; white-space: nowrap; }
.event-period__end { color: var(--ic-color-text-secondary); }
@media (max-width: 720px) { .event-form__grid { grid-template-columns: 1fr; gap: 0; } }
</style>
