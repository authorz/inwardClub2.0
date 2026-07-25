<script setup lang="ts">
import { computed, h, reactive, ref } from 'vue'
import { NForm, NFormItem, NInputNumber, NSpace } from 'naive-ui'
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
import { formatCent, toCent } from '@/utils/format'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]
const numberFormatter = new Intl.NumberFormat('zh-CN')

const columns = [
  moneyColumn<RechargeProduct>('充值金额', 'amountCent'),
  renderColumn<RechargeProduct>('到账金币', 'coinAmount', (row) => numberFormatter.format(row.coinAmount), 120),
  renderColumn<RechargeProduct>('赠送积分', 'pointsAmount', (row) => numberFormatter.format(row.pointsAmount), 120),
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
const form = reactive({
  coinAmount: null as number | null,
  pointsAmount: 0 as number | null,
  sortOrder: 0,
})
const previewText = computed(() => {
  const amountCent = toCent(amountYuan.value)
  if (amountCent == null || amountCent <= 0 || form.coinAmount == null || form.coinAmount <= 0) {
    return '填写金额和到账金币后，可在这里确认用户实际获得的权益。'
  }
  const points = form.pointsAmount ?? 0
  return `支付 ${formatCent(amountCent)}，到账 ${numberFormatter.format(form.coinAmount)} 金币${
    points > 0 ? `，赠送 ${numberFormatter.format(points)} 积分` : ''
  }`
})

function openCreate(): void {
  editingId.value = null
  form.coinAmount = null
  form.pointsAmount = 0
  form.sortOrder = 0
  amountYuan.value = null
  drawerShow.value = true
}
function openEdit(row: RechargeProduct): void {
  editingId.value = row.id
  form.coinAmount = row.coinAmount
  form.pointsAmount = row.pointsAmount ?? 0
  form.sortOrder = row.sortOrder ?? 0
  amountYuan.value = row.amountCent != null ? row.amountCent / 100 : null
  drawerShow.value = true
}
async function submit(): Promise<void> {
  const cent = toCent(amountYuan.value)
  if (cent == null || cent <= 0) return toastError('请填写大于 0 的充值金额')
  if (form.coinAmount == null || form.coinAmount <= 0) return toastError('请填写大于 0 的到账金币')
  if (form.pointsAmount == null || form.pointsAmount < 0) return toastError('赠送积分不能小于 0')
  const payload = {
    amountCent: cent,
    coinAmount: form.coinAmount,
    pointsAmount: form.pointsAmount,
    sortOrder: form.sortOrder,
  }
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
    title: '禁用快捷充值档位',
    content: `确认禁用“${formatCent(row.amountCent)} 到账 ${numberFormatter.format(row.coinAmount)} 金币”档位？`,
    highRisk: true,
    execute: () => rechargeProductService.action(API_PATHS.rechargeProducts.disable(row.id)),
    successText: '已禁用',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增充值档位',
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
      title="快捷充值"
      description="配置固定充值金额，以及支付成功后到账的金币和赠送积分"
      :breadcrumb="['快捷充值']"
      :fields="fields"
      :columns="columns"
      :fetcher="rechargeProductService.list"
      :toolbar-actions="toolbarActions"
    />
    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑快捷充值档位' : '新增快捷充值档位'"
      :submitting="submitting"
      high-risk
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="支付金额（元）"
          required
        >
          <NInputNumber
            v-model:value="amountYuan"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem
          label="到账金币"
          required
        >
          <NInputNumber
            v-model:value="form.coinAmount"
            :min="1"
            :precision="0"
            style="width: 100%"
          />
        </NFormItem>
        <NFormItem label="赠送积分">
          <NInputNumber
            v-model:value="form.pointsAmount"
            :min="0"
            :precision="0"
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
        <div class="recharge-preview">
          <span class="recharge-preview__label">用户到账预览</span>
          <span>{{ previewText }}</span>
        </div>
      </NForm>
    </FormDrawer>
  </div>
</template>

<style scoped>
.recharge-preview {
  display: grid;
  gap: 6px;
  padding: 14px 16px;
  color: #333;
  background: #f5f5f3;
  border-left: 3px solid #1d1d1f;
}

.recharge-preview__label {
  color: #777;
  font-size: 12px;
}
</style>
