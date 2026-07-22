<script setup lang="ts">
/**
 * 账号列表通用视图（公共组件，配置驱动）。
 *
 * 总后台账号 / 门店管理员 / 员工三类账号后端契约并不一致：
 *  - super（总后台账号，admin_accounts role=super_admin）：仅支持「列表 + 禁用」，
 *    无新增 / 编辑接口；字段 username/displayName/role。
 *  - store-admin（门店管理员，admin_accounts）：列表 + 新增（storeId/username/displayName）
 *    + 编辑（仅 displayName）+ 禁用；字段含绑定门店 storeName。
 *  - staff（员工，staff_accounts）：列表 + 新增（storeId/name）+ 编辑（仅 name）+ 禁用；
 *    只有 name，无 username/role。
 *
 * 因此本组件按 variant 决定列、可写性与表单形状，三个路由页只做薄封装。
 */
import { h, onMounted, reactive, ref } from 'vue'
import { NForm, NFormItem, NInput, NSelect, NSpace } from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance, TableColumnList } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { storeService } from '@/api/services'
import type { AccountEntity } from '@/api/models'
import type { ResourceService } from '@/api/resource'
import type { FilterField } from '@/components/ui-types'
import { getDialog, toastError, toastSuccess } from '@/utils/feedback'

const props = defineProps<{
  title: string
  description?: string
  service: ResourceService<AccountEntity>
  /** 账号类型：决定列、可写性与表单字段 */
  variant: 'super' | 'store-admin' | 'staff'
  /** 禁用接口路径构造器 */
  disablePath: (id: string) => string
}>()

// variant 在路由维度固定，setup 期求值一次即可。
const isStaff = props.variant === 'staff'
const showStore = props.variant !== 'super'
// super_admin 账号后端没有新增 / 编辑接口，本页对其只读（仅禁用）。
const canWrite = props.variant !== 'super'

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: '账号 / 姓名', type: 'input', placeholder: '搜索账号' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

const baseColumns: TableColumnList<AccountEntity> = isStaff
  ? [textColumn<AccountEntity>('姓名', 'name', { width: 160 })]
  : [
      textColumn<AccountEntity>('账号', 'username', { width: 160 }),
      textColumn<AccountEntity>('姓名', 'displayName', { width: 140 }),
      textColumn<AccountEntity>('角色', 'role', { width: 140 }),
    ]

const columns: TableColumnList<AccountEntity> = [
  ...baseColumns,
  ...(showStore ? [textColumn<AccountEntity>('绑定门店', 'storeName', { width: 160 })] : []),
  statusColumn<AccountEntity>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<AccountEntity>('创建时间', 'createdAt'),
  actionsColumn<AccountEntity>(
    (row) =>
      h(NSpace, {}, () => [
        ...(canWrite
          ? [
              h(
                PermissionButton,
                { permission: PERMISSIONS.ACCOUNT_WRITE, onClick: () => openEdit(row) },
                () => '编辑',
              ),
            ]
          : []),
        h(
          PermissionButton,
          {
            permission: PERMISSIONS.ACCOUNT_WRITE,
            type: 'error',
            onClick: () => disable(row),
          },
          () => '禁用',
        ),
      ]),
    160,
  ),
]

// 门店下拉（新增门店管理员 / 员工时选择绑定门店）
const storeOptions = ref<{ label: string; value: number }[]>([])
onMounted(async () => {
  if (!showStore) return
  try {
    const res = await storeService.list({ pageSize: 100 })
    storeOptions.value = res.items.map((s) => ({ label: s.name, value: Number(s.id) }))
  } catch {
    // 门店下拉加载失败不阻塞列表本身
  }
})

const drawerShow = ref(false)
const submitting = ref(false)
const editingId = ref<string | null>(null)
const form = reactive<{
  storeId: number | null
  username: string
  displayName: string
  name: string
}>({ storeId: null, username: '', displayName: '', name: '' })

function resetForm(): void {
  form.storeId = null
  form.username = ''
  form.displayName = ''
  form.name = ''
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

function openEdit(row: AccountEntity): void {
  editingId.value = row.id
  resetForm()
  if (isStaff) {
    form.name = row.name ?? ''
  } else {
    form.username = row.username ?? ''
    form.displayName = row.displayName ?? ''
  }
  drawerShow.value = true
}

async function submit(): Promise<void> {
  if (isStaff && !form.name) return toastError('请填写员工姓名')
  if (!isStaff && !editingId.value && !form.username) return toastError('请填写账号')
  if (!editingId.value && showStore && form.storeId == null) return toastError('请选择绑定门店')

  let payload: Record<string, unknown>
  if (isStaff) {
    payload = editingId.value ? { name: form.name } : { storeId: form.storeId, name: form.name }
  } else {
    payload = editingId.value
      ? { displayName: form.displayName }
      : { storeId: form.storeId, username: form.username, displayName: form.displayName }
  }

  submitting.value = true
  try {
    if (editingId.value) {
      await props.service.update(editingId.value, payload as Partial<AccountEntity>)
    } else {
      const created = await props.service.create(payload as Partial<AccountEntity>)
      // 门店管理员创建时后端一次性返回初始密码，必须显式展示给操作者。
      if (created?.initialPassword) {
        getDialog()?.success({
          title: '账号已创建',
          content: `账号「${created.username ?? form.username}」的初始密码：${created.initialPassword}。该密码仅显示一次，请立即妥善保存并交给账号持有人。`,
          positiveText: '我已保存',
        })
      }
    }
    toastSuccess('已保存')
    drawerShow.value = false
    listRef.value?.reload()
  } catch (e) {
    toastError((e as { message?: string }).message ?? '保存失败')
  } finally {
    submitting.value = false
  }
}

async function disable(row: AccountEntity): Promise<void> {
  const label = row.username ?? row.name ?? row.id
  const ok = await runAudited({
    title: '禁用账号',
    content: `确认禁用账号「${label}」？该操作将写入审计日志。`,
    highRisk: true,
    positiveText: '确认禁用',
    execute: () => props.service.action(props.disablePath(row.id)),
    successText: '账号已禁用',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = canWrite
  ? [
      {
        key: 'create',
        label: '新增账号',
        type: 'primary' as const,
        permission: PERMISSIONS.ACCOUNT_WRITE,
        onClick: openCreate,
      },
    ]
  : []
</script>

<template>
  <div>
    <ResourceListView
      ref="listRef"
      :title="title"
      :description="description"
      :breadcrumb="['账号与权限', title]"
      :fields="fields"
      :columns="columns"
      :fetcher="service.list"
      :toolbar-actions="toolbarActions"
    />

    <FormDrawer
      v-model:show="drawerShow"
      :title="editingId ? '编辑账号' : '新增账号'"
      :submitting="submitting"
      @submit="submit"
    >
      <NForm label-placement="top">
        <NFormItem
          v-if="!editingId && showStore"
          label="绑定门店"
          required
        >
          <NSelect
            v-model:value="form.storeId"
            :options="storeOptions"
            placeholder="选择绑定门店"
          />
        </NFormItem>

        <template v-if="isStaff">
          <NFormItem
            label="姓名"
            required
          >
            <NInput
              v-model:value="form.name"
              placeholder="请输入员工姓名"
            />
          </NFormItem>
        </template>

        <template v-else>
          <NFormItem
            label="账号"
            required
          >
            <NInput
              v-model:value="form.username"
              :disabled="!!editingId"
              placeholder="请输入账号"
            />
          </NFormItem>
          <NFormItem label="姓名">
            <NInput
              v-model:value="form.displayName"
              placeholder="请输入姓名"
            />
          </NFormItem>
        </template>
      </NForm>
      <p
        v-if="showStore"
        class="form-note"
      >
        同一员工 / 门店管理员只能绑定一个门店；编辑时不可更换绑定门店。
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
