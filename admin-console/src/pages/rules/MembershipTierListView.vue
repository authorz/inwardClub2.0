<script setup lang="ts">
/**
 * VIP 等级。列表 + 新增/编辑（骨架）。
 * 等级门槛/权益等运营数值以表单配置为准，不写死。
 */
import { h, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import AssetImage from '@/components/AssetImage.vue'
import AssetUpload from '@/components/AssetUpload.vue'
import { actionsColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { membershipTierService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import type { MembershipTier } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [{ key: 'keyword', label: '等级名称', type: 'input' }]

const columns = [
  renderColumn<MembershipTier>(
    'VIP 海报',
    'bannerUrl',
    (row) => h(AssetImage, { src: row.bannerUrl, width: 56, height: 32 }),
    90,
  ),
  textColumn<MembershipTier>('等级名称', 'name'),
  textColumn<MembershipTier>('等级', 'level', { width: 90 }),
  textColumn<MembershipTier>('成长值门槛', 'threshold', { width: 130 }),
  statusColumn<MembershipTier>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  actionsColumn<MembershipTier>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.RULE_PUBLISH, onClick: () => openEdit(row) },
          () => '编辑',
        ),
        h(
          PermissionButton,
          { permission: PERMISSIONS.RULE_PUBLISH, type: 'error', onClick: () => disable(row) },
          () => '禁用',
        ),
      ]),
    160,
  ),
]

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<Partial<MembershipTier>>({ name: '', level: 1, threshold: 0, bannerPath: null })
/** 编辑态已有海报的展示地址（服务端返回 bannerUrl，不含 path，仅用于预览） */
const currentBannerUrl = ref('')

function openCreate(): void {
  editingId.value = null
  form.name = ''
  form.level = 1
  form.threshold = 0
  form.bannerPath = null
  currentBannerUrl.value = ''
  drawerShow.value = true
}
function openEdit(row: MembershipTier): void {
  editingId.value = row.id
  form.name = row.name
  form.level = row.level
  form.threshold = row.threshold ?? 0
  // 详情不回传 bannerPath，仅回传 bannerUrl：未重新上传表示保持当前海报不变。
  form.bannerPath = null
  currentBannerUrl.value = row.bannerUrl ?? ''
  drawerShow.value = true
}
async function submit(): Promise<void> {
  if (!form.name) return toastError('请填写等级名称')
  submitting.value = true
  try {
    if (editingId.value) await membershipTierService.update(editingId.value, form)
    else await membershipTierService.create(form)
    toastSuccess('已保存')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}
async function disable(row: MembershipTier): Promise<void> {
  const ok = await runAudited({
    title: '禁用会员等级',
    content: `确认禁用「${row.name}」？`,
    highRisk: true,
    execute: () => membershipTierService.action(API_PATHS.membershipTiers.disable(row.id)),
    successText: '已禁用',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增等级',
    type: 'primary' as const,
    permission: PERMISSIONS.RULE_PUBLISH,
    onClick: openCreate,
  },
]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="VIP 等级"
      description="会员等级与权益门槛配置"
      :breadcrumb="['VIP / 权益规则', 'VIP 等级']"
      :fields="fields"
      :columns="columns"
      :fetcher="membershipTierService.list"
      :toolbar-actions="toolbarActions"
    />
    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑等级' : '新增等级'"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="等级名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="请输入等级名称"
          />
        </NFormItem>
        <NFormItem label="等级序号">
          <NInputNumber
            v-model:value="form.level"
            :min="1"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem label="成长值门槛">
          <NInputNumber
            v-model:value="form.threshold"
            :min="0"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem label="VIP 海报">
          <AssetUpload
            v-model:path="form.bannerPath"
            purpose="vip_banner"
            :preview-url="currentBannerUrl || null"
            :width="160"
            :height="80"
          />
        </NFormItem>
      </NForm>
      <p class="form-note">
        VIP 海报直接上传图片（直传至七牛，与 Banner / 门店 Logo 一致），保存的是对象路径（path/objectKey）而非完整地址；
        编辑时未重新上传表示保持当前海报不变。权益明细待业务规则配置后补充（TODO: 待业务规则配置）。
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
