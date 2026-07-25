<script setup lang="ts">
/**
 * 总后台广告管理。
 * 总后台可创建全局广告或门店广告，并可选择同范围的活动作为小程序跳转目标。
 */
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
import {
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpace,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { FilterField, ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import AssetImage from '@/components/AssetImage.vue'
import AssetUpload from '@/components/AssetUpload.vue'
import { actionsColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { BANNER_STATUS_OPTIONS, type OptionItem } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { activityService, bannerService, storeService } from '@/api/services'
import type { Activity, Banner } from '@/api/models'
import type { ListQuery, ListResult } from '@/api/types'
import { toastError, toastSuccess } from '@/utils/feedback'

const ACTIVITY_LINK_PREFIX = '/pages/activity-detail/activity-detail?id='

const listRef = ref<ResourceListInstance | null>(null)
const storeOptions = ref<OptionItem[]>([])
const activities = ref<Activity[]>([])
const uploadKey = ref(0)
const scopeOptions: OptionItem[] = [
  { label: '全部门店', value: 'global' },
  { label: '指定门店', value: 'store' },
]

const fields = computed<FilterField[]>(() => [
  {
    key: 'scopeType',
    label: '投放范围',
    type: 'select',
    options: scopeOptions,
    width: 160,
  },
  {
    key: 'storeId',
    label: '所属门店',
    type: 'select',
    options: storeOptions.value,
    width: 200,
  },
  {
    key: 'keyword',
    label: '广告标题',
    type: 'input',
    placeholder: '支持标题模糊搜索',
    width: 220,
  },
  { key: 'status', label: '显示状态', type: 'select', options: BANNER_STATUS_OPTIONS },
])

function activityIDFromLink(linkUrl?: string): string | null {
  if (!linkUrl?.startsWith(ACTIVITY_LINK_PREFIX)) return null
  const activityID = linkUrl.slice(ACTIVITY_LINK_PREFIX.length)
  return /^\d+$/.test(activityID) ? activityID : null
}

function deliveryRange(row: Banner): string {
  if (row.scopeType === 'global') return '全部门店'
  return (
    storeOptions.value.find(({ value }) => String(value) === String(row.storeId ?? ''))?.label ??
    '未绑定'
  )
}

function activityName(linkUrl?: string): string {
  const activityID = activityIDFromLink(linkUrl)
  if (!activityID) return '不关联活动'
  return (
    activities.value.find(({ id }) => String(id) === activityID)?.name ?? `活动 #${activityID}`
  )
}

const columns = [
  renderColumn<Banner>(
    '广告图片',
    'imageUrl',
    (row) => h(AssetImage, { src: row.imageUrl, width: 144, height: 64 }),
    176,
  ),
  textColumn<Banner>('标题', 'title', { width: 180 }),
  renderColumn<Banner>('投放范围', 'scopeType', deliveryRange, 160),
  renderColumn<Banner>('关联活动', 'linkUrl', (row) => activityName(row.linkUrl), 180),
  textColumn<Banner>('排序', 'sortOrder', { width: 80 }),
  statusColumn<Banner>('显示状态', 'status', BANNER_STATUS_OPTIONS, 100),
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

interface BannerForm {
  scopeType: 'global' | 'store'
  storeId: string | null
  title: string
  status: string
  activityMode: 'none' | 'activity'
  activityId: string | null
  sortOrder: number
  assetId: string | null
  imageUrl: string
}

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<BannerForm>({
  scopeType: 'store',
  storeId: null,
  title: '',
  status: 'active',
  activityMode: 'none',
  activityId: null,
  sortOrder: 0,
  assetId: null,
  imageUrl: '',
})

const activityOptions = computed<OptionItem[]>(() =>
  activities.value
    .filter((activity) => {
      if (form.scopeType === 'global') {
        return activity.scopeType === 'global' || activity.storeId == null
      }
      return String(activity.storeId ?? '') === String(form.storeId ?? '')
    })
    .map((activity) => ({
      label: activity.title ?? activity.name ?? `活动 #${activity.id}`,
      value: String(activity.id),
    })),
)

const activityFieldLabel = computed(() =>
  form.scopeType === 'global' ? '全局活动' : '门店活动',
)

const activityPlaceholder = computed(() => {
  if (form.scopeType === 'global') return '请选择全局活动'
  return form.storeId ? '请选择该门店下的活动' : '请先选择门店'
})

async function loadReferences(): Promise<void> {
  try {
    const [storeResult, activityResult] = await Promise.all([
      storeService.list({ page: 1, pageSize: 100 }),
      activityService.list({ page: 1, pageSize: 100 }),
    ])
    storeOptions.value = storeResult.items.map((store) => ({
      label: store.name,
      value: String(store.id),
    }))
    activities.value = activityResult.items
  } catch (e) {
    toastError((e as { message?: string }).message ?? '门店和活动数据加载失败')
  }
}

async function fetchBanners(query: ListQuery): Promise<ListResult<Banner>> {
  const result = await bannerService.list()
  const keyword = String(query.keyword ?? '')
    .trim()
    .toLocaleLowerCase()
  const storeID = String(query.storeId ?? '')
  const scopeType = String(query.scopeType ?? '')
  const status = String(query.status ?? '')
  const filtered = result.items.filter((banner) => {
    if (scopeType && banner.scopeType !== scopeType) return false
    if (storeID && String(banner.storeId ?? '') !== storeID) return false
    if (status && banner.status !== status) return false
    return !keyword || banner.title.toLocaleLowerCase().includes(keyword)
  })
  const page = Number(query.page) || 1
  const pageSize = Number(query.pageSize) || 20
  const offset = (page - 1) * pageSize
  return {
    items: filtered.slice(offset, offset + pageSize),
    meta: { page, pageSize, total: filtered.length },
  }
}

function resetForm(): void {
  form.scopeType = 'store'
  form.storeId = null
  form.title = ''
  form.status = 'active'
  form.activityMode = 'none'
  form.activityId = null
  form.sortOrder = 0
  form.assetId = null
  form.imageUrl = ''
  uploadKey.value += 1
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

function openEdit(row: Banner): void {
  editingId.value = row.id
  resetForm()
  const activityID = activityIDFromLink(row.linkUrl)
  form.scopeType = row.scopeType === 'global' ? 'global' : 'store'
  form.storeId = row.storeId == null ? null : String(row.storeId)
  form.title = row.title
  form.status = row.status === 'inactive' ? 'inactive' : 'active'
  form.activityMode = activityID ? 'activity' : 'none'
  form.activityId = activityID
  form.sortOrder = row.sortOrder ?? 0
  form.assetId = row.assetId == null ? null : String(row.assetId)
  form.imageUrl = row.imageUrl ?? ''
  drawerShow.value = true
}

function changeScope(scopeType: 'global' | 'store'): void {
  form.scopeType = scopeType
  form.storeId = null
  form.activityId = null
}

watch(
  () => form.storeId,
  (storeID, previousStoreID) => {
    if (previousStoreID == null || storeID === previousStoreID || !form.activityId) return
    const activity = activities.value.find(({ id }) => String(id) === String(form.activityId))
    if (String(activity?.storeId ?? '') !== String(storeID ?? '')) form.activityId = null
  },
)

watch(
  () => form.activityMode,
  (mode) => {
    if (mode === 'none') form.activityId = null
  },
)

async function submit(): Promise<void> {
  if (form.scopeType === 'store' && !form.storeId) return toastError('请选择所属门店')
  if (!form.title.trim()) return toastError('请填写广告标题')
  if (form.activityMode === 'activity' && !form.activityId) {
    return toastError(`请选择要关联的${form.scopeType === 'global' ? '全局活动' : '门店活动'}`)
  }
  if (!form.assetId) return toastError('请上传广告图片')

  const payload: Partial<Banner> = {
    scopeType: form.scopeType,
    title: form.title.trim(),
    status: form.status,
    linkUrl:
      form.activityMode === 'activity' && form.activityId
        ? `${ACTIVITY_LINK_PREFIX}${form.activityId}`
        : '',
    sortOrder: form.sortOrder,
    assetId: Number(form.assetId),
  }
  if (form.scopeType === 'store') payload.storeId = Number(form.storeId)
  submitting.value = true
  try {
    if (editingId.value) await bannerService.update(editingId.value, payload)
    else await bannerService.create(payload)
    toastSuccess('广告已保存')
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
    title: '删除广告',
    content: `确认删除「${row.title}」？`,
    execute: () => bannerService.remove(row.id),
    successText: '已删除',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增广告',
    type: 'primary' as const,
    permission: PERMISSIONS.BANNER_WRITE,
    onClick: openCreate,
  },
]

onMounted(loadReferences)
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="广告管理"
      description="管理全部门店或指定门店的首页广告、显示状态和活动跳转"
      :breadcrumb="['广告管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="fetchBanners"
      :toolbar-actions="toolbarActions"
      empty-text="暂无广告，请先新增并上传图片"
    />

    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑广告' : '新增广告'"
      :width="620"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm
        label-placement="left"
        label-width="112"
        require-mark-placement="left"
      >
        <NFormItem
          label="广告标题"
          required
        >
          <NInput
            v-model:value="form.title"
            placeholder="请输入便于后台识别的标题"
          />
        </NFormItem>

        <NFormItem
          label="投放范围"
          required
        >
          <NRadioGroup
            :value="form.scopeType"
            @update:value="changeScope"
          >
            <NSpace>
              <NRadio value="global">
                全部门店
              </NRadio>
              <NRadio value="store">
                指定门店
              </NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>

        <NFormItem
          v-if="form.scopeType === 'store'"
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
          label="显示状态"
          required
        >
          <NRadioGroup v-model:value="form.status">
            <NSpace>
              <NRadio value="active">
                显示
              </NRadio>
              <NRadio value="inactive">
                不显示
              </NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>

        <NFormItem label="是否关联活动">
          <NRadioGroup v-model:value="form.activityMode">
            <NSpace>
              <NRadio value="none">
                不关联活动
              </NRadio>
              <NRadio value="activity">
                关联活动
              </NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>

        <NFormItem
          v-if="form.activityMode === 'activity'"
          :label="activityFieldLabel"
          required
        >
          <NSelect
            v-model:value="form.activityId"
            :options="activityOptions.map(({ label, value }) => ({ label, value }))"
            :disabled="form.scopeType === 'store' && !form.storeId"
            :placeholder="activityPlaceholder"
            filterable
          />
        </NFormItem>

        <NFormItem label="排序">
          <NInputNumber
            v-model:value="form.sortOrder"
            :min="0"
            :precision="0"
            placeholder="数字越小越靠前"
            style="width: 100%"
          />
        </NFormItem>

        <NFormItem
          label="广告图片"
          required
        >
          <div class="banner-upload">
            <AssetUpload
              :key="uploadKey"
              v-model:asset-id="form.assetId"
              purpose="banner"
              :preview-url="form.imageUrl || null"
              :width="320"
              :height="144"
            />
            <p class="field-help">
              建议上传横版图片，推荐比例 20:9；支持 JPG、PNG、WebP。
            </p>
          </div>
        </NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>

<style scoped>
.banner-upload {
  width: 100%;
}

.field-help {
  margin: var(--ic-space-sm) 0 0;
  color: var(--ic-color-text-secondary);
  font-size: var(--ic-font-xs);
  line-height: 1.6;
}
</style>
