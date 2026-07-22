<script setup lang="ts">
/**
 * 规则中心（签到 / 邀请 / 低消 / 充值等版本化规则）。列表 + 新增 + 发布/禁用（高风险）。
 * 规则数值不写死；草稿/预览/发布/禁用/版本历史/命中记录待详情页补充。
 */
import { h, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS, SCOPE_TYPE_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { ruleDefinitionService } from '@/api/services'
import { API_PATHS } from '@/constants/api-paths'
import type { RuleDefinition } from '@/api/models'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: '规则名称 / Key', type: 'input' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const columns = [
  textColumn<RuleDefinition>('规则 Key', 'ruleKey', { width: 200 }),
  statusColumn<RuleDefinition>('范围', 'scopeType', SCOPE_TYPE_OPTIONS, 90),
  textColumn<RuleDefinition>('版本', 'version', { width: 90 }),
  statusColumn<RuleDefinition>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<RuleDefinition>('更新时间', 'updatedAt'),
  actionsColumn<RuleDefinition>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.RULE_PUBLISH, type: 'primary', onClick: () => publish(row) },
          () => '发布',
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

async function publish(row: RuleDefinition): Promise<void> {
  const ok = await runAudited({
    title: '发布规则',
    content: `确认发布规则「${row.ruleKey}」？发布将改变线上运营数值，写入审计并生成新版本。`,
    highRisk: true,
    positiveText: '确认发布',
    execute: () => ruleDefinitionService.action(API_PATHS.ruleDefinitions.publish(row.id), undefined, true),
    successText: '规则已发布',
  })
  if (ok) listRef.value?.reload()
}
async function disable(row: RuleDefinition): Promise<void> {
  const ok = await runAudited({
    title: '禁用规则',
    content: `确认禁用规则「${row.ruleKey}」？`,
    highRisk: true,
    positiveText: '确认禁用',
    execute: () => ruleDefinitionService.action(API_PATHS.ruleDefinitions.disable(row.id), undefined, true),
    successText: '规则已禁用',
  })
  if (ok) listRef.value?.reload()
}

const drawerShow = ref(false)
const submitting = ref(false)
const jsonPlaceholder = '规则参数 JSON，例如 {"points": 10}'
const form = reactive<{ ruleKey: string; scopeType: string; configJson: string }>({
  ruleKey: '',
  scopeType: 'global',
  configJson: '{}',
})

function openCreate(): void {
  form.ruleKey = ''
  form.scopeType = 'global'
  form.configJson = '{}'
  drawerShow.value = true
}
async function submit(): Promise<void> {
  if (!form.ruleKey) return toastError('请填写规则 Key')
  let config: unknown
  try {
    config = JSON.parse(form.configJson || '{}')
  } catch {
    return toastError('规则配置必须是合法 JSON')
  }
  submitting.value = true
  try {
    await ruleDefinitionService.create({
      ruleKey: form.ruleKey,
      scopeType: form.scopeType,
      configJson: config,
    })
    toastSuccess('已创建草稿')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '创建失败')
  } finally {
    submitting.value = false
  }
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增规则',
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
      title="规则中心"
      description="版本化运营规则；新增为草稿，发布/禁用为高风险操作，写入审计"
      :breadcrumb="['VIP / 权益规则', '规则中心']"
      :fields="fields"
      :columns="columns"
      :fetcher="ruleDefinitionService.list"
      :toolbar-actions="toolbarActions"
    />
    <FormDrawer
      v-model:show="drawerShow"
      title="新增规则"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          label="规则 Key"
          required
        >
          <NInput
            v-model:value="form.ruleKey"
            placeholder="如 sign_in / invitation / min_consumption"
          />
        </NFormItem>
        <NFormItem label="适用范围">
          <NSelect
            v-model:value="form.scopeType"
            :options="SCOPE_TYPE_OPTIONS.map((o) => ({ label: o.label, value: o.value }))"
          />
        </NFormItem>
        <NFormItem
          label="规则配置（JSON）"
          required
        >
          <NInput
            v-model:value="form.configJson"
            type="textarea"
            :placeholder="jsonPlaceholder"
          />
        </NFormItem>
      </NForm>
      <p class="form-note">
        规则配置为 JSON 对象；新增后为草稿状态，需发布后生效。
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
