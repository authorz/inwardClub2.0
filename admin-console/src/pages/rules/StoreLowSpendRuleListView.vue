<script setup lang="ts">
import { h, reactive, ref } from 'vue'
import { NForm, NFormItemGi, NGrid, NInputNumber, NSpace, NSwitch, NTag, NTimePicker } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import type { ResourceListInstance, FilterField } from '@/components/ui-types'
import { dateTimeColumn, renderColumn, textColumn } from '@/utils/columns'
import { PERMISSIONS } from '@/constants/permissions'
import { systemService } from '@/api/services'
import type { StoreLowSpendRule } from '@/api/models'
import { runAudited } from '@/composables/useAuditedAction'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)
const show = ref(false)
const saving = ref(false)
const editingStoreId = ref<number | null>(null)
const editingStoreName = ref('')

const fields: FilterField[] = [
  { key: 'storeName', label: '门店名称', type: 'input', placeholder: '支持模糊搜索' },
]

const form = reactive({
  enabled: true,
  reservationCutoff: '20:00',
  consumptionCutoff: '20:30',
  minimumAmount: 88,
  rewardPoints: 2000,
})

function openEdit(row: StoreLowSpendRule): void {
  editingStoreId.value = row.storeId
  editingStoreName.value = row.storeName
  Object.assign(form, {
    enabled: row.enabled,
    reservationCutoff: row.reservationCutoff || '20:00',
    consumptionCutoff: row.consumptionCutoff || '20:30',
    minimumAmount: row.minimumAmount || 88,
    rewardPoints: row.rewardPoints || 2000,
  })
  show.value = true
}

function validate(): boolean {
  if (!form.reservationCutoff || !form.consumptionCutoff) {
    toastError('请选择完整的规则时间')
    return false
  }
  if (form.reservationCutoff >= form.consumptionCutoff) {
    toastError('低消截止时间必须晚于预约或候桌截止时间')
    return false
  }
  if (!Number.isInteger(form.minimumAmount) || form.minimumAmount <= 0) {
    toastError('低消金额必须是大于 0 的整数')
    return false
  }
  if (!Number.isInteger(form.rewardPoints) || form.rewardPoints <= 0) {
    toastError('赠送积分必须是大于 0 的整数')
    return false
  }
  return true
}

async function save(): Promise<void> {
  if (!editingStoreId.value || !validate()) return
  saving.value = true
  try {
    await systemService.updateStoreLowSpendRule(editingStoreId.value, { ...form })
    toastSuccess('门店奖励规则已保存')
    show.value = false
    listRef.value?.reload()
  } catch (error) {
    toastError((error as { message?: string }).message ?? '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(row: StoreLowSpendRule): Promise<void> {
  if (!row.configured) return
  const ok = await runAudited({
    title: '删除门店奖励规则',
    content: `删除“${row.storeName}”的规则后，该门店将停止发放预约低消奖励。`,
    highRisk: true,
    positiveText: '确认删除',
    execute: () => systemService.deleteStoreLowSpendRule(row.storeId),
    successText: '门店奖励规则已删除',
  })
  if (ok) listRef.value?.reload()
}

const columns = [
  textColumn<StoreLowSpendRule>('门店 ID', 'storeId', { width: 90 }),
  textColumn<StoreLowSpendRule>('门店', 'storeName', { minWidth: 150 }),
  renderColumn<StoreLowSpendRule>('状态', 'enabled', (row) =>
    h(NTag, { type: row.configured && row.enabled ? 'success' : 'default', bordered: false }, () =>
      !row.configured ? '未配置' : row.enabled ? '已开启' : '已关闭',
    ), 100),
  renderColumn<StoreLowSpendRule>('预约/候桌截止', 'reservationCutoff', (row) => row.reservationCutoff, 130),
  renderColumn<StoreLowSpendRule>('低消截止', 'consumptionCutoff', (row) => row.consumptionCutoff, 110),
  renderColumn<StoreLowSpendRule>('低消金额', 'minimumAmount', (row) => `¥${row.minimumAmount}`, 110),
  renderColumn<StoreLowSpendRule>('奖励积分', 'rewardPoints', (row) => row.rewardPoints.toLocaleString(), 110),
  dateTimeColumn<StoreLowSpendRule>('更新时间', 'updatedAt', 170),
  renderColumn<StoreLowSpendRule>('操作', 'actions', (row) =>
    h(NSpace, { size: 8, wrap: false }, () => [
      h(PermissionButton, {
        permission: PERMISSIONS.RULE_PUBLISH,
        onClick: () => openEdit(row),
      }, () => row.configured ? '编辑' : '配置'),
      row.configured
        ? h(PermissionButton, {
            permission: PERMISSIONS.RULE_PUBLISH,
            type: 'error',
            onClick: () => remove(row),
          }, () => '删除')
        : null,
    ]), 150),
]
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      title="预约低消奖励"
      description="按门店配置预约或候桌截止时间、低消门槛和达标积分；总部可跨店查看与维护"
      :breadcrumb="['权益规则', '预约低消奖励']"
      :fields="fields"
      :columns="columns"
      :fetcher="systemService.listStoreLowSpendRules"
      empty-text="暂无门店"
    />

    <FormDrawer
      v-model:show="show"
      :title="`配置奖励规则 · ${editingStoreName}`"
      :submitting="saving"
      :width="680"
      @submit="save"
    >
      <NForm label-placement="top">
        <NGrid :cols="2" :x-gap="16">
          <NFormItemGi label="规则状态" :span="2">
            <NSwitch v-model:value="form.enabled">
              <template #checked>已开启</template>
              <template #unchecked>已关闭</template>
            </NSwitch>
          </NFormItemGi>
          <NFormItemGi label="预约 / 候桌截止" required>
            <NTimePicker
              v-model:formatted-value="form.reservationCutoff"
              format="HH:mm"
              :clearable="false"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi label="完成低消截止" required>
            <NTimePicker
              v-model:formatted-value="form.consumptionCutoff"
              format="HH:mm"
              :clearable="false"
              style="width: 100%"
            />
          </NFormItemGi>
          <NFormItemGi label="累计低消金额" required>
            <NInputNumber v-model:value="form.minimumAmount" :min="1" :precision="0" style="width: 100%">
              <template #suffix>元</template>
            </NInputNumber>
          </NFormItemGi>
          <NFormItemGi label="达标赠送积分" required>
            <NInputNumber v-model:value="form.rewardPoints" :min="1" :precision="0" style="width: 100%">
              <template #suffix>积分</template>
            </NInputNumber>
          </NFormItemGi>
        </NGrid>
      </NForm>
    </FormDrawer>
  </div>
</template>
