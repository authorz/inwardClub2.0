<script setup lang="ts">
/**
 * 总后台活动管理。
 * 列表使用摘要接口；编辑时拉取完整详情，避免整体 PUT 覆盖未回填字段。
 */
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NDatePicker,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import AssetImage from '@/components/AssetImage.vue'
import AssetUpload from '@/components/AssetUpload.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import {
  actionsColumn,
  renderColumn,
  statusColumn,
  textColumn,
} from '@/utils/columns'
import {
  PAY_CHANNEL_OPTIONS,
  RESOURCE_STATUS,
  RESOURCE_STATUS_OPTIONS,
  type OptionItem,
} from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { activityService, storeService } from '@/api/services'
import type { Activity } from '@/api/models'
import { runAudited } from '@/composables/useAuditedAction'
import { toastError, toastSuccess } from '@/utils/feedback'
import { formatDateTime } from '@/utils/format'

const listRef = ref<ResourceListInstance | null>(null)
const uploadKey = ref(0)
const storeOptions = ref<OptionItem[]>([])

const activityStatusOptions = RESOURCE_STATUS_OPTIONS.filter(({ value }) =>
  [RESOURCE_STATUS.DRAFT, RESOURCE_STATUS.PUBLISHED].includes(
    value as typeof RESOURCE_STATUS.DRAFT | typeof RESOURCE_STATUS.PUBLISHED,
  ),
)
const onlinePayChannelOptions = PAY_CHANNEL_OPTIONS

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
    label: '活动标题',
    type: 'input',
    placeholder: '支持标题模糊搜索',
    width: 240,
  },
  { key: 'status', label: '状态', type: 'select', options: activityStatusOptions },
])

const activityScopeOptions = computed<Array<{ label: string; value: string }>>(() => [
  { label: '全部门店（全局活动）', value: 'global' },
  ...storeOptions.value.map(({ label, value }) => ({ label, value: String(value) })),
])

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

function activityTitle(row: Activity): string {
  return row.title || row.name || '未命名活动'
}

function activityStore(row: Activity): string {
  if (row.scopeType === 'global' || row.storeId == null) return '全部门店'
  return (
    storeOptions.value.find(({ value }) => String(value) === String(row.storeId))?.label ??
    `门店 #${row.storeId}`
  )
}

function activityAuditTime(row: Activity): string {
  const createdTime = row.createdAt ? Date.parse(row.createdAt) : Number.NaN
  const updatedTime = row.updatedAt ? Date.parse(row.updatedAt) : Number.NaN
  const isUpdated =
    Number.isFinite(updatedTime) &&
    (!Number.isFinite(createdTime) || updatedTime > createdTime)
  const value = isUpdated ? row.updatedAt : row.createdAt
  return value ? `${isUpdated ? '更新' : '创建'} ${formatDateTime(value)}` : '-'
}

function activityCover(row: Activity) {
  const previewable = Boolean(row.imageUrl || row.assetId)
  return h(
    'div',
    {
      class: ['activity-cover', previewable && 'activity-cover--previewable'],
      title: previewable ? '点击查看大图' : undefined,
    },
    [
      h(AssetImage, {
        src: row.imageUrl,
        assetId: row.assetId == null ? null : String(row.assetId),
        width: 96,
        height: 54,
      }),
    ],
  )
}

function activityPeriod(row: Activity) {
  if (!row.startAt && !row.endAt) return '-'
  return h('div', { class: 'activity-period' }, [
    h('span', {}, row.startAt ? formatDateTime(row.startAt) : '未设置开始时间'),
    h(
      'span',
      { class: 'activity-period__end' },
      `至 ${row.endAt ? formatDateTime(row.endAt) : '未设置结束时间'}`,
    ),
  ])
}

const columns = [
  textColumn<Activity>('ID', 'id', { width: 80 }),
  renderColumn<Activity>('活动封面', 'imageUrl', activityCover, 120),
  renderColumn<Activity>('活动标题', 'name', activityTitle, 220),
  renderColumn<Activity>('所属门店', 'storeId', activityStore, 180),
  renderColumn<Activity>('活动时间', 'activityPeriod', activityPeriod, 190),
  renderColumn<Activity>('创建/更新时间', 'auditTime', activityAuditTime, 190),
  statusColumn<Activity>('状态', 'status', activityStatusOptions, 100),
  actionsColumn<Activity>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.ACTIVITY_GLOBAL_WRITE,
            onClick: () => void openEdit(row),
          },
          () => '编辑详情',
        ),
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.ACTIVITY_GLOBAL_WRITE,
            type: 'error',
            onClick: () => void remove(row),
          },
          () => '删除',
        ),
      ]),
    180,
  ),
]

interface ActivityForm {
  storeTarget: string
  title: string
  description: string
  content: string
  assetId: string | null
  imageUrl: string
  timeRange: [number, number] | null
  payChannels: string[]
  purchaseLimitPerMember: number
  status: string
}

const drawerShow = ref(false)
const submitting = ref(false)
const detailLoading = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<ActivityForm>({
  storeTarget: 'global',
  title: '',
  description: '',
  content: '',
  assetId: null,
  imageUrl: '',
  timeRange: null,
  payChannels: ['wechat'],
  purchaseLimitPerMember: 0,
  status: RESOURCE_STATUS.DRAFT,
})

function resetForm(): void {
  form.storeTarget = 'global'
  form.title = ''
  form.description = ''
  form.content = ''
  form.assetId = null
  form.imageUrl = ''
  form.timeRange = null
  form.payChannels = ['wechat']
  form.purchaseLimitPerMember = 0
  form.status = RESOURCE_STATUS.DRAFT
  uploadKey.value += 1
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

async function openEdit(row: Activity): Promise<void> {
  if (detailLoading.value) return
  detailLoading.value = true
  try {
    const detail = await activityService.get(row.id)
    editingId.value = row.id
    resetForm()
    form.storeTarget = detail.storeId == null ? 'global' : String(detail.storeId)
    form.title = detail.title ?? detail.name ?? ''
    form.description = detail.description ?? ''
    form.content = detail.content ?? ''
    form.assetId = detail.assetId == null ? null : String(detail.assetId)
    form.imageUrl = detail.imageUrl ?? ''
    form.timeRange =
      detail.startAt && detail.endAt
        ? [new Date(detail.startAt).getTime(), new Date(detail.endAt).getTime()]
        : null
    form.payChannels = detail.payChannels?.length ? [...detail.payChannels] : ['wechat']
    form.purchaseLimitPerMember = detail.purchaseLimitPerMember ?? 0
    form.status = detail.status ?? RESOURCE_STATUS.DRAFT
    drawerShow.value = true
  } catch (e) {
    toastError((e as { message?: string }).message ?? '活动详情加载失败')
  } finally {
    detailLoading.value = false
  }
}

async function submit(): Promise<void> {
  if (!form.title.trim()) return toastError('请填写活动标题')
  if (!form.payChannels.length) return toastError('请选择至少一种支付方式')
  if (form.timeRange && form.timeRange[0] >= form.timeRange[1]) {
    return toastError('活动结束时间必须晚于开始时间')
  }

  const payload: Partial<Activity> = {
    storeId: form.storeTarget === 'global' ? null : Number(form.storeTarget),
    title: form.title.trim(),
    description: form.description.trim(),
    content: form.content,
    assetId: form.assetId ? Number(form.assetId) : null,
    startAt: form.timeRange ? new Date(form.timeRange[0]).toISOString() : undefined,
    endAt: form.timeRange ? new Date(form.timeRange[1]).toISOString() : undefined,
    payChannels: form.payChannels,
    purchaseLimitPerMember: form.purchaseLimitPerMember,
    status: form.status,
  }

  submitting.value = true
  try {
    if (editingId.value) await activityService.update(editingId.value, payload)
    else await activityService.create(payload)
    toastSuccess(editingId.value ? '活动已更新' : '活动已创建')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}

async function remove(row: Activity): Promise<void> {
  const ok = await runAudited({
    title: '删除活动',
    content: `确认删除「${activityTitle(row)}」？删除后无法恢复。`,
    highRisk: true,
    positiveText: '确认删除',
    execute: () => activityService.remove(row.id),
    successText: '活动已删除',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增活动',
    type: 'primary' as const,
    permission: PERMISSIONS.ACTIVITY_GLOBAL_WRITE,
    onClick: openCreate,
  },
]

onMounted(loadStores)
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="活动管理"
      description="维护活动基本信息、封面、图文详情、时间、支付方式和发布状态"
      :breadcrumb="['活动管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="activityService.list"
      :toolbar-actions="toolbarActions"
      empty-text="暂无活动，请先新增活动"
    />

    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑活动' : '新增活动'"
      :width="880"
      :submitting="submitting"
      :high-risk="Boolean(editingId)"
      cross-store
      @submit="submit"
    >
      <NForm label-placement="top">
        <div class="activity-form__grid">
          <NFormItem
            label="活动范围"
            required
          >
            <NSelect
              v-model:value="form.storeTarget"
              :options="activityScopeOptions"
              placeholder="请选择全部门店或具体门店"
              filterable
            />
          </NFormItem>

          <NFormItem
            label="发布状态"
            required
          >
            <NSelect
              v-model:value="form.status"
              :options="activityStatusOptions.map(({ label, value }) => ({ label, value }))"
            />
          </NFormItem>
        </div>

        <NFormItem
          label="活动标题"
          required
        >
          <NInput
            v-model:value="form.title"
            placeholder="请输入活动标题"
            maxlength="128"
            show-count
          />
        </NFormItem>

        <div class="activity-form__grid">
          <NFormItem
            label="活动时间"
          >
            <NDatePicker
              v-model:value="form.timeRange"
              type="datetimerange"
              clearable
              style="width: 100%"
            />
          </NFormItem>

          <NFormItem label="每人限购">
            <NInputNumber
              v-model:value="form.purchaseLimitPerMember"
              :min="0"
              :precision="0"
              placeholder="0 表示不限制"
              style="width: 100%"
            />
          </NFormItem>
        </div>

        <NFormItem label="活动摘要">
          <NInput
            v-model:value="form.description"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            placeholder="用于活动列表和分享摘要"
            maxlength="500"
            show-count
          />
        </NFormItem>

        <NFormItem
          label="支付方式"
          required
        >
          <NSelect
            v-model:value="form.payChannels"
            multiple
            :options="onlinePayChannelOptions.map(({ label, value }) => ({ label, value }))"
            placeholder="请选择支付方式"
          />
        </NFormItem>

        <NFormItem label="活动封面">
          <div class="activity-form__asset">
            <AssetUpload
              :key="uploadKey"
              v-model:asset-id="form.assetId"
              purpose="activity"
              :preview-url="form.imageUrl || null"
              :width="240"
              :height="320"
            />
            <p>建议上传 3:4 竖版海报（如 750×1000px），支持 JPG、PNG、WebP。</p>
          </div>
        </NFormItem>

        <NFormItem label="图文详情">
          <RichTextEditor
            v-model="form.content"
            placeholder="编辑活动详情，可插入标题、列表、链接和图片"
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>

<style scoped>
.activity-cover {
  width: 96px;
  line-height: 0;
}

.activity-cover--previewable {
  cursor: zoom-in;
}

.activity-period {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.45;
  white-space: nowrap;
}

.activity-period__end {
  color: var(--ic-color-text-secondary);
}

.activity-form__grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(220px, 1fr);
  gap: var(--ic-space-md);
}

.activity-form__asset p {
  margin: var(--ic-space-sm) 0 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
}

@media (max-width: 720px) {
  .activity-form__grid {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
