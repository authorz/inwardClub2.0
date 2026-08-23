<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import {
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NSelect,
  NSpace,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { SCOPE_TYPE_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { couponTemplateService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import type { CouponTemplate } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)
const couponTypes = [
  { label: '赛事门票券', value: 'event_ticket' },
  { label: '小吃券', value: 'snack' },
  { label: '酒水券', value: 'alcohol' },
  { label: '饮料券', value: 'beverage' },
  { label: '饮品或啤酒券', value: 'drink' },
  { label: '餐食券', value: 'meal' },
  { label: '礼品券', value: 'gift' },
]
const couponStatuses = [
  { label: '草稿', value: 'draft', tone: 'default' as const },
  { label: '已发布', value: 'published', tone: 'success' as const },
  { label: '已停用', value: 'disabled', tone: 'warning' as const },
]
const fields: FilterField[] = [
  { key: 'keyword', label: '券名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: couponStatuses },
]

const columns = [
  textColumn<CouponTemplate>('ID', 'id', { width: 72 }),
  textColumn<CouponTemplate>('券名称', 'name', { minWidth: 160 }),
  statusColumn<CouponTemplate>('类型', 'couponType', couponTypes, 100),
  statusColumn<CouponTemplate>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  statusColumn<CouponTemplate>('状态', 'status', couponStatuses, 100),
  dateTimeColumn<CouponTemplate>('更新时间', 'updatedAt', 170),
  actionsColumn<CouponTemplate>((row) => h(NSpace, { size: 6, wrap: false }, () => [
    h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: () => openEdit(row) }, () => '编辑'),
    row.status !== 'published'
      ? h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, type: 'primary', onClick: () => publish(row) }, () => '发布')
      : h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, type: 'warning', onClick: () => disable(row) }, () => '停用'),
    h(PermissionButton, { permission: PERMISSIONS.COUPON_GLOBAL_WRITE, type: 'error', onClick: () => remove(row) }, () => '删除'),
  ]), 220),
]

const show = ref(false)
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({
  name: '',
  description: '',
  couponType: 'alcohol',
})

function resetForm(): void {
  Object.assign(form, {
    name: '', description: '', couponType: 'alcohol',
  })
}
function openCreate(): void {
  editingId.value = null
  resetForm()
  show.value = true
}
async function openEdit(row: CouponTemplate): Promise<void> {
  try {
    const detail = await couponTemplateService.get(String(row.id))
    editingId.value = String(row.id)
    Object.assign(form, {
      name: detail.name,
      description: detail.description ?? '',
      couponType: detail.couponType,
    })
    show.value = true
  } catch (error) {
    toastError((error as { message?: string }).message ?? '读取优惠券详情失败')
  }
}
async function save(): Promise<void> {
  if (!form.name.trim()) return toastError('请填写券名称')
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(), description: form.description.trim(), couponType: form.couponType,
    }
    if (editingId.value) await couponTemplateService.update(editingId.value, payload)
    else await couponTemplateService.create(payload)
    toastSuccess('优惠券已保存')
    show.value = false
    listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    saving.value = false
  }
}
async function publish(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '启用券类型', content: `启用“${row.name}”后可用于 VIP 福利和商品关联。`,
    highRisk: true,
    execute: () => couponTemplateService.action(API_PATHS.coupons.publishTemplate(String(row.id))),
    successText: '优惠券已发布',
  })
  if (ok) listRef.value?.reload()
}
async function disable(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '停用优惠券', content: `确认停用“${row.name}”？已发出的券不受影响。`,
    highRisk: true,
    execute: () => couponTemplateService.action(API_PATHS.coupons.disableTemplate(String(row.id))),
    successText: '优惠券已停用',
  })
  if (ok) listRef.value?.reload()
}
async function remove(row: CouponTemplate): Promise<void> {
  const ok = await runAudited({
    title: '删除优惠券', content: `确认删除“${row.name}”？`, highRisk: true,
    execute: () => couponTemplateService.remove(String(row.id)), successText: '优惠券已删除',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [{
  key: 'create', label: '新增优惠券', type: 'primary' as const,
  permission: PERMISSIONS.COUPON_GLOBAL_WRITE, onClick: openCreate,
}]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="券管理"
      description="维护平台券类型；一张券只能兑换一种商品或一张活动门票"
      :breadcrumb="['权益规则', '券管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="couponTemplateService.list"
      :toolbar-actions="toolbarActions"
      empty-text="暂无券类型，点击右上角新增"
    />
    <FormDrawer
      v-model:show="show"
      :title="editingId ? '编辑优惠券' : '新增优惠券'"
      :submitting="saving"
      :width="720"
      @submit="save"
    >
      <NForm label-placement="top">
        <NGrid
          :cols="2"
          :x-gap="16"
        >
          <NFormItemGi
            label="券名称"
            required
          >
            <NInput
              v-model:value="form.name"
              placeholder="例如：50 元代金券"
            />
          </NFormItemGi>
          <NFormItemGi
            label="券类型"
            required
          >
            <NSelect
              v-model:value="form.couponType"
              :options="couponTypes"
            />
          </NFormItemGi>
          <NFormItemGi
            label="使用说明"
            :span="2"
          >
            <NInput
              v-model:value="form.description"
              type="textarea"
              :rows="3"
              placeholder="说明适用范围和使用条件"
            />
          </NFormItemGi>
        </NGrid>
      </NForm>
    </FormDrawer>
  </div>
</template>
