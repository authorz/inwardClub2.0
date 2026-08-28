<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import { NForm, NFormItemGi, NGrid, NInput, NInputNumber, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { couponCategoryService } from '@/api/services'
import type { CouponCategory } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)

const usageOptions = [
  { label: '赛事使用', value: 'event_ticket' },
  { label: '活动门票兑换', value: 'admission_ticket' },
  { label: '小吃兑换', value: 'snack' },
  { label: '酒水兑换', value: 'alcohol' },
  { label: '饮料兑换', value: 'beverage' },
  { label: '饮品兑换', value: 'drink' },
  { label: '餐食兑换', value: 'meal' },
  { label: '礼品兑换', value: 'gift' },
]
const statusOptions = [
  { label: '启用', value: 'active', tone: 'success' as const },
  { label: '停用', value: 'disabled', tone: 'warning' as const },
]
const fields: FilterField[] = [
  { key: 'keyword', label: '类型名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: statusOptions },
]

const columns = [
  textColumn<CouponCategory>('ID', 'id', { width: 72 }),
  textColumn<CouponCategory>('券种名称', 'name', { minWidth: 180 }),
  statusColumn<CouponCategory>('使用方式', 'businessType', usageOptions, 160),
  renderColumn<CouponCategory>('默认有效期', 'defaultValidityDays', (row) => `${row.defaultValidityDays} 天`, 110),
  textColumn<CouponCategory>('排序', 'sortOrder', { width: 90 }),
  statusColumn<CouponCategory>('状态', 'status', statusOptions, 90),
  dateTimeColumn<CouponCategory>('更新时间', 'updatedAt', 170),
  actionsColumn<CouponCategory>((row) => h(NSpace, { size: 6 }, () => [
    h(PermissionButton, {
      permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
      onClick: () => openEdit(row),
    }, () => '编辑'),
    h(PermissionButton, {
      permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
      type: 'error',
      onClick: () => remove(row),
    }, () => '删除'),
  ]), 130),
]

const drawerShow = ref(false)
const saving = ref(false)
const editingId = ref<string | null>(null)
const form = reactive({
  name: '', businessType: 'alcohol', description: '', admissionCount: 1,
  defaultValidityDays: 30, sortOrder: 0, status: 'active',
})

function openCreate(): void {
  editingId.value = null
  Object.assign(form, {
    name: '', businessType: 'alcohol', description: '', admissionCount: 1,
    defaultValidityDays: 30, sortOrder: 0, status: 'active',
  })
  drawerShow.value = true
}

function openEdit(row: CouponCategory): void {
  editingId.value = String(row.id)
  Object.assign(form, {
    name: row.name,
    businessType: row.businessType,
    description: row.description ?? '',
    admissionCount: row.admissionCount || 1,
    defaultValidityDays: row.defaultValidityDays || 30,
    sortOrder: row.sortOrder,
    status: row.status,
  })
  drawerShow.value = true
}

async function save(): Promise<void> {
  if (!form.name.trim()) return toastError('请填写券种名称')
  saving.value = true
  try {
    const payload = { ...form, name: form.name.trim() }
    if (editingId.value) await couponCategoryService.update(editingId.value, payload)
    else await couponCategoryService.create(payload)
    toastSuccess('券种已保存')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(row: CouponCategory): Promise<void> {
  const ok = await runAudited({
    title: '删除券种',
    content: `确认删除“${row.name}”？已被使用的券种不能删除，只能停用。`,
    highRisk: true,
    execute: () => couponCategoryService.remove(String(row.id)),
    successText: '券种已删除',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [{
  key: 'create',
  label: '新增券种',
  type: 'primary' as const,
  permission: PERMISSIONS.COUPON_GLOBAL_WRITE,
  onClick: openCreate,
}]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="券种"
      description="统一管理可购买、可赠送和可核销的券种；系统内部发放定义自动维护"
      :breadcrumb="['权益规则', '券管理']"
      :fields="fields"
      :columns="columns"
      :fetcher="couponCategoryService.list"
      :toolbar-actions="toolbarActions"
      empty-text="暂无券种，点击右上角新增"
    />

    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑券种' : '新增券种'"
      :submitting="saving"
      :width="640"
      @submit="save"
    >
      <NForm label-placement="top">
        <NGrid
          :cols="2"
          :x-gap="16"
        >
          <NFormItemGi
            label="券种名称"
            required
          >
            <NInput
              v-model:value="form.name"
              placeholder="例如：周赛赛事券"
            />
          </NFormItemGi>
          <NFormItemGi
            label="默认有效期（天）"
            required
          >
            <NInputNumber
              v-model:value="form.defaultValidityDays"
              :min="1"
              :max="3650"
              :precision="0"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi
            v-if="form.businessType === 'admission_ticket'"
            label="每张可兑人数"
            required
          >
            <NInputNumber
              v-model:value="form.admissionCount"
              :min="1"
              :max="99"
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
              placeholder="说明该券种可以兑换什么或如何使用"
            />
          </NFormItemGi>
          <NFormItemGi
            label="使用方式"
            required
          >
            <NSelect
              v-model:value="form.businessType"
              :options="usageOptions"
              :disabled="Boolean(editingId)"
            />
          </NFormItemGi>
          <NFormItemGi label="排序">
            <NInputNumber
              v-model:value="form.sortOrder"
              :min="0"
              :precision="0"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi
            label="状态"
            required
          >
            <NSelect
              v-model:value="form.status"
              :options="statusOptions"
            />
          </NFormItemGi>
        </NGrid>
      </NForm>
    </FormDrawer>
  </div>
</template>
