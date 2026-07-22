<script setup lang="ts">
/**
 * 充值产品。列表 + 新增/编辑（骨架）。
 * 金额为整数分；赠送金币/积分等运营数值不写死，以表单配置为准。
 */
import { h, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NInputNumber, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, moneyColumn, renderColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { rechargeProductService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import type { RechargeProduct } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'
import { toCent } from '@/utils/format'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: '产品名称', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

/** 充值赠送资产类型（服务端取值 coin / point） */
const RECHARGE_ASSET_OPTIONS = [
  { label: '金币', value: 'coin' },
  { label: '积分', value: 'point' },
]
function assetTypeLabel(value: string | undefined): string {
  return RECHARGE_ASSET_OPTIONS.find((o) => o.value === value)?.label ?? value ?? '-'
}

const columns = [
  textColumn<RechargeProduct>('产品名称', 'name'),
  moneyColumn<RechargeProduct>('充值金额', 'amount'),
  textColumn<RechargeProduct>('赠送数量', 'bonusAmount', { width: 100 }),
  renderColumn<RechargeProduct>('赠送资产', 'assetType', (row) => assetTypeLabel(row.assetType), 100),
  textColumn<RechargeProduct>('排序', 'sortOrder', { width: 80 }),
  statusColumn<RechargeProduct>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  actionsColumn<RechargeProduct>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.RECHARGE_WRITE, onClick: () => openEdit(row) },
          () => '编辑',
        ),
        h(
          PermissionButton,
          { permission: PERMISSIONS.RECHARGE_WRITE, type: 'error', onClick: () => disable(row) },
          () => '禁用',
        ),
      ]),
    160,
  ),
]

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const amountYuan = ref<number | null>(null)
const form = reactive<Partial<RechargeProduct>>({
  name: '',
  bonusAmount: 0,
  assetType: 'coin',
  sortOrder: 0,
})

function openCreate(): void {
  editingId.value = null
  form.name = ''
  form.bonusAmount = 0
  form.assetType = 'coin'
  form.sortOrder = 0
  amountYuan.value = null
  drawerShow.value = true
}
function openEdit(row: RechargeProduct): void {
  editingId.value = row.id
  form.name = row.name
  form.bonusAmount = row.bonusAmount ?? 0
  form.assetType = row.assetType ?? 'coin'
  form.sortOrder = row.sortOrder ?? 0
  amountYuan.value = row.amount != null ? row.amount / 100 : null
  drawerShow.value = true
}
async function submit(): Promise<void> {
  if (!form.name) return toastError('请填写产品名称')
  const cent = toCent(amountYuan.value)
  if (cent == null) return toastError('请填写正确的充值金额')
  const payload = { ...form, amount: cent }
  submitting.value = true
  try {
    if (editingId.value) await rechargeProductService.update(editingId.value, payload)
    else await rechargeProductService.create(payload)
    toastSuccess('已保存')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}
async function disable(row: RechargeProduct): Promise<void> {
  const ok = await runAudited({
    title: '禁用充值产品',
    content: `确认禁用「${row.name}」？`,
    highRisk: true,
    execute: () => rechargeProductService.action(API_PATHS.rechargeProducts.disable(row.id)),
    successText: '已禁用',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增充值产品',
    type: 'primary' as const,
    permission: PERMISSIONS.RECHARGE_WRITE,
    onClick: openCreate,
  },
]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="充值产品"
      description="金币充值档位；金额以元填写，提交转为整数分"
      :breadcrumb="['充值产品']"
      :fields="fields"
      :columns="columns"
      :fetcher="rechargeProductService.list"
      :toolbar-actions="toolbarActions"
    />
    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑充值产品' : '新增充值产品'"
      :submitting="submitting"
      high-risk
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="产品名称"
          required
        >
          <NInput
            v-model:value="form.name"
            placeholder="请输入产品名称"
          />
        </NFormItem>
        <NFormItem
          label="充值金额（元）"
          required
        >
          <NInputNumber
            v-model:value="amountYuan"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem label="赠送资产类型">
          <NSelect
            v-model:value="form.assetType"
            :options="RECHARGE_ASSET_OPTIONS"
          />
        </NFormItem>
        <NFormItem label="赠送数量">
          <NInputNumber
            v-model:value="form.bonusAmount"
            :min="0"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem label="排序（小在前）">
          <NInputNumber
            v-model:value="form.sortOrder"
            :min="0"
            style="width: 100%"
          />
        </NFormItem>
      </NForm>
    </FormDrawer>
  </div>
</template>
