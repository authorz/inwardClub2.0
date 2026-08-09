<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { NButton, NDatePicker, NForm, NFormItem, NInput, NModal, NSelect, NSpace, type DataTableColumns } from 'naive-ui'
import { tournamentEventService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { PUBLISH_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { statusColumn, textColumn } from '@/utils/columns'
import { formatDateTime } from '@/utils/format'
import { AssetImage, AssetUpload, DataTable, PageHeader, PermissionButton, RichTextEditor } from '@/components/common'
import { feedback } from '@/utils/feedback'
import type { TournamentEvent } from '@/types/models'

const list = useAsyncList<TournamentEvent>((params) => tournamentEventService.list(params), { initialFilters: { status: '', keyword: '' } })
const action = useAsyncAction()
const show = ref(false)
const statusOptions = toOptions(PUBLISH_STATUS).map(({ label, value }) => ({ label, value }))
const form = reactive({
  id: null as string | number | null, title: '', summary: '', content: '', assetId: null as string | null,
  imageUrl: '', timeRange: null as [number, number] | null, status: 'published',
})

async function open(row?: TournamentEvent): Promise<void> {
  let target = row
  if (row) {
    try { target = await tournamentEventService.detail(row.id) }
    catch (error) { feedback.message.error((error as { message?: string }).message ?? '赛事详情加载失败'); return }
  }
  Object.assign(form, target ? {
    id: target.id, title: target.title, summary: target.summary ?? '', content: target.content ?? '',
    assetId: target.assetId == null ? null : String(target.assetId), imageUrl: target.imageUrl ?? '',
    timeRange: target.startAt && target.endAt ? [Date.parse(target.startAt), Date.parse(target.endAt)] : null,
    status: target.status,
  } : { id: null, title: '', summary: '', content: '', assetId: null, imageUrl: '', timeRange: null, status: 'published' })
  show.value = true
}

async function save(): Promise<void> {
  if (!form.title.trim()) return void feedback.message.error('请填写赛事标题')
  if (!form.timeRange) return void feedback.message.error('请设置赛事开始和结束时间')
  if (form.timeRange[0] >= form.timeRange[1]) return void feedback.message.error('赛事结束时间必须晚于开始时间')
  await action.run(async () => {
    const payload = {
      title: form.title.trim(), summary: form.summary.trim(), content: form.content, assetId: form.assetId ? Number(form.assetId) : null,
      startAt: new Date(form.timeRange![0]).toISOString(), endAt: new Date(form.timeRange![1]).toISOString(), status: form.status,
    }
    if (form.id == null) await tournamentEventService.create(payload)
    else await tournamentEventService.update(form.id, payload)
  }, { successMessage: '赛事活动已保存', onSuccess: () => { show.value = false; list.refresh() } })
}

function remove(row: TournamentEvent): void {
  void action.run(() => tournamentEventService.remove(row.id), {
    confirm: { title: '删除赛事活动', content: `确认删除「${row.title}」？删除后无法恢复。` },
    successMessage: '赛事活动已删除', onSuccess: () => list.refresh(),
  })
}

function period(row: TournamentEvent) {
  return h('div', { class: 'event-period' }, [h('span', {}, row.startAt ? formatDateTime(row.startAt) : '-'), h('span', { class: 'event-period__end' }, `至 ${row.endAt ? formatDateTime(row.endAt) : '-'}`)])
}

const columns = computed<DataTableColumns<TournamentEvent>>(() => [
  textColumn<TournamentEvent>('ID', (row) => row.id, { width: 70 }),
  { title: '赛事图片', key: 'imageUrl', width: 130, render: (row) => h(AssetImage, { src: row.imageUrl ?? null, assetId: row.assetId ?? null, width: 104, height: 60 }) },
  textColumn<TournamentEvent>('赛事标题', (row) => row.title, { width: 220 }),
  { title: '赛事时间', key: 'period', width: 190, render: period },
  statusColumn<TournamentEvent>('状态', PUBLISH_STATUS, (row) => row.status, { width: 96 }),
  { title: '操作', key: 'actions', width: 160, render: (row) => h(NSpace, { size: 8 }, () => [
    h(PermissionButton, { permissions: [PERM.activityWrite], onClick: () => void open(row) }, () => '编辑'),
    h(PermissionButton, { permissions: [PERM.activityWrite], type: 'error', onClick: () => remove(row) }, () => '删除'),
  ]) },
])
</script>

<template>
  <div>
    <PageHeader title="赛事活动" description="发布本店赛事预告与图文详情；赛事活动不包含票档、支付和报名">
      <template #actions><PermissionButton :permissions="[PERM.activityWrite]" type="primary" @click="open()">新增赛事活动</PermissionButton></template>
    </PageHeader>
    <div class="event-filters">
      <div class="event-filter"><label>赛事标题</label><NInput :value="list.filters.keyword as string" placeholder="支持标题模糊搜索" clearable @update:value="list.filters.keyword = $event" @keyup.enter="list.applyFilters({})" /></div>
      <div class="event-filter event-filter--status"><label>状态</label><NSelect :value="list.filters.status as string" :options="[{ label: '全部', value: '' }, ...statusOptions]" @update:value="list.filters.status = $event" /></div>
      <div class="event-filter-actions"><NButton type="primary" :loading="list.loading.value" @click="list.applyFilters({})">查询</NButton><NButton @click="list.reset()">重置</NButton></div>
    </div>
    <DataTable :columns="columns" :data="list.rows.value" :loading="list.loading.value" :page="list.page.value" :page-size="list.pageSize.value" :total="list.total.value" empty-text="暂无赛事活动" @update:page="list.setPage" @update:page-size="list.setPageSize" />
    <NModal v-model:show="show" preset="card" :title="form.id == null ? '新增赛事活动' : '编辑赛事活动'" style="width: min(820px, calc(100vw - 32px))">
      <NForm label-placement="top">
        <div class="event-form__grid"><NFormItem label="发布状态" required><NSelect v-model:value="form.status" :options="statusOptions" /></NFormItem><NFormItem label="赛事时间" required><NDatePicker v-model:value="form.timeRange" type="datetimerange" style="width: 100%" /></NFormItem></div>
        <NFormItem label="赛事标题" required><NInput v-model:value="form.title" maxlength="128" show-count /></NFormItem>
        <NFormItem label="赛事简介"><NInput v-model:value="form.summary" type="textarea" :autosize="{ minRows: 2, maxRows: 4 }" maxlength="500" show-count /></NFormItem>
        <NFormItem label="小 Banner"><AssetUpload v-model:asset-id="form.assetId" v-model:preview-url="form.imageUrl" purpose="activity" :width="320" :height="128" /></NFormItem>
        <NFormItem label="图文详情"><RichTextEditor v-model="form.content" placeholder="编辑赛事规则、流程和说明" /></NFormItem>
      </NForm>
      <template #footer><NSpace justify="end"><NButton @click="show = false">取消</NButton><NButton type="primary" :loading="action.running.value" @click="save">保存</NButton></NSpace></template>
    </NModal>
  </div>
</template>

<style scoped>
.event-filters { display: flex; align-items: end; gap: 16px; margin: 18px 0; }
.event-filter { width: 280px; }.event-filter--status { width: 180px; }.event-filter label { display: block; margin-bottom: 7px; color: #606266; font-size: 13px; }.event-filter-actions { display: flex; gap: 10px; }
.event-form__grid { display: grid; grid-template-columns: 1fr 2fr; gap: 16px; }.event-period { display: flex; flex-direction: column; line-height: 1.45; }.event-period__end { color: #8a8f98; }
@media (max-width: 720px) { .event-filters { align-items: stretch; flex-direction: column; }.event-filter, .event-filter--status { width: 100%; }.event-form__grid { grid-template-columns: 1fr; gap: 0; } }
</style>
