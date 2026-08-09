<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import {
  NForm,
  NFormItemGi,
  NGrid,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, moneyColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
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
  { label: '兑换券', value: 'exchange' },
  { label: '折扣券', value: 'discount' },
  { label: '代金券', value: 'cash' },
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

function stockOf(row: CouponTemplate): number {
  return row.stockQuantity ?? row.totalStock ?? 0
}
function issuedOf(row: CouponTemplate): number {
  return row.issuedQuantity ?? row.issuedCount ?? 0
}

const columns = [
  textColumn<CouponTemplate>('ID', 'id', { width: 72 }),
  textColumn<CouponTemplate>('券名称', 'name', { minWidth: 160 }),
  statusColumn<CouponTemplate>('类型', 'couponType', couponTypes, 100),
  moneyColumn<CouponTemplate>('面额', 'valueCent', 100),
  statusColumn<CouponTemplate>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  renderColumn<CouponTemplate>('库存 / 已发', 'stock', (row) => `${stockOf(row)} / ${issuedOf(row)}`, 120),
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
  couponType: 'cash',
  valueYuan: 0,
  pointsPrice: 0,
  stockQuantity: 0,
  perMemberLimit: 1,
})

function resetForm(): void {
  Object.assign(form, {
    name: '', description: '', couponType: 'cash', valueYuan: 0,
    pointsPrice: 0, stockQuantity: 0, perMemberLimit: 1,
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
      valueYuan: detail.valueCent / 100,
      pointsPrice: detail.pointsPrice ?? 0,
      stockQuantity: stockOf(detail),
      perMemberLimit: detail.perMemberLimit ?? 1,
    })
    show.value = true
  } catch (error) {
    toastError((error as { message?: string }).message ?? '读取优惠券详情失败')
  }
}
async function save(): Promise<void> {
  if (!form.name.trim()) return toastError('请填写券名称')
  if (form.valueYuan < 0) return toastError('券面额不能小于 0')
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(), description: form.description.trim(), couponType: form.couponType,
      valueCent: Math.round(form.valueYuan * 100), pointsPrice: form.pointsPrice,
      stockQuantity: form.stockQuantity, perMemberLimit: form.perMemberLimit,
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
    title: '发布优惠券', content: `发布“${row.name}”后可用于会员领取和充值赠送。`,
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
      description="创建、发布和维护平台及门店优惠券；已发布券可绑定到充值奖励"
      :breadcrumb="['权益规则', '券管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="couponTemplateService.list"
      :toolbar-actions="toolbarActions"
      empty-text="暂无优惠券，点击右上角新增"
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
          <NFormItemGi label="面额（元）">
            <NInputNumber
              v-model:value="form.valueYuan"
              :min="0"
              :precision="2"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi label="兑换积分">
            <NInputNumber
              v-model:value="form.pointsPrice"
              :min="0"
              :precision="0"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi label="库存（0 表示不限量）">
            <NInputNumber
              v-model:value="form.stockQuantity"
              :min="0"
              :precision="0"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi label="每人限领（0 表示不限量）">
            <NInputNumber
              v-model:value="form.perMemberLimit"
              :min="0"
              :precision="0"
              style="width: 100%"
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
