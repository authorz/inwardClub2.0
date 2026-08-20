<script setup lang="ts">
/**
 * 账号列表通用视图（公共组件，配置驱动）。
 *
 * 总后台账号 / 门店管理员 / 员工三类账号后端契约并不一致：
 *  - super（总后台账号，admin_accounts role=super_admin）：支持新增、编辑、
 *    修改密码、禁用和删除；系统管理员不可禁用或删除。
 *  - store-admin（门店管理员，admin_accounts）：列表 + 新增
 *    （storeId/username/password/displayName）+ 编辑（仅 displayName）+ 禁用；
 *    字段含绑定门店 storeName。
 *  - staff（员工，staff_accounts）：列表 + 新增（storeId/name）+ 编辑（仅 name）+ 禁用；
 *    只有 name，无 username/role。
 *
 * 因此本组件按 variant 决定列、可写性与表单形状，三个路由页只做薄封装。
 */
import { computed, h, onMounted, reactive, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NForm,
  NFormItem,
  NInput,
  NInputGroup,
  NRadio,
  NRadioGroup,
  NSelect,
  NSpace,
} from 'naive-ui'
import ResourceListView from '@/components/ResourceListView.vue'
import type { ResourceListInstance, TableColumnList } from '@/components/ui-types'
import FormDrawer from '@/components/FormDrawer.vue'
import PermissionButton from '@/components/PermissionButton.vue'
import { actionsColumn, dateTimeColumn, statusColumn, textColumn } from '@/utils/columns'
import { RESOURCE_STATUS_OPTIONS } from '@/constants/enums'
import { PERMISSIONS } from '@/constants/permissions'
import { runAudited } from '@/composables/useAuditedAction'
import { storeService } from '@/api/services'
import { http } from '@/api/http'
import { API_PATHS } from '@/constants/api-paths'
import type { AccountEntity, Member } from '@/api/models'
import type { ResourceService } from '@/api/resource'
import type { FilterField } from '@/components/ui-types'
import { toastError, toastSuccess } from '@/utils/feedback'

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

const listRef = ref<ResourceListInstance | null>(null)

const fields: FilterField[] = [
  { key: 'keyword', label: '账号 / 姓名', type: 'input', placeholder: '搜索账号' },
  { key: 'status', label: '状态', type: 'select', options: RESOURCE_STATUS_OPTIONS },
]

function staffNickname(row: AccountEntity): string {
  return row.nickname?.trim() || row.name?.trim() || '会员'
}

function staffInitial(row: AccountEntity): string {
  return staffNickname(row).slice(0, 1)
}

function staffAvatar(row: AccountEntity) {
  const props = { size: 36, round: true, objectFit: 'cover' as const }
  return row.avatarUrl
    ? h(NAvatar, { ...props, src: row.avatarUrl }, { fallback: () => staffInitial(row) })
    : h(NAvatar, props, { default: () => staffInitial(row) })
}

const baseColumns: TableColumnList<AccountEntity> = isStaff
  ? [
      {
        title: '用户',
        key: 'member',
        width: 200,
        render: (row) =>
          h('div', { class: 'staff-user' }, [
            staffAvatar(row),
            h('div', { class: 'staff-user__meta' }, [
              h('span', { class: 'staff-user__nickname' }, staffNickname(row)),
              row.name && row.name !== staffNickname(row)
                ? h('span', { class: 'staff-user__name' }, `员工名：${row.name}`)
                : null,
            ]),
          ]),
      },
      textColumn<AccountEntity>('手机号', 'phone', { width: 150 }),
    ]
  : [
      textColumn<AccountEntity>('账号', 'username', { width: 160 }),
      textColumn<AccountEntity>('姓名', 'displayName', { width: 140 }),
      {
        title: '账号类型',
        key: 'accountType',
        width: 140,
        render: (row) => (row.isSystem ? '系统管理员' : props.variant === 'super' ? '管理员' : row.role),
      },
    ]

const columns: TableColumnList<AccountEntity> = [
  ...baseColumns,
  ...(showStore ? [textColumn<AccountEntity>('绑定门店', 'storeName', { width: 160 })] : []),
  statusColumn<AccountEntity>('状态', 'status', RESOURCE_STATUS_OPTIONS),
  dateTimeColumn<AccountEntity>('创建时间', 'createdAt'),
  actionsColumn<AccountEntity>(
    (row) =>
      h(NSpace, {}, () => [
        h(
          PermissionButton,
          { permission: PERMISSIONS.ACCOUNT_WRITE, onClick: () => openEdit(row) },
          () => '编辑',
        ),
        ...(isStaff
          ? [
              h(
                PermissionButton,
                {
                  permission: PERMISSIONS.ACCOUNT_WRITE,
                  type: 'error',
                  onClick: () => removeStaff(row),
                },
                () => '删除',
              ),
            ]
          : props.variant === 'super'
            ? row.isSystem
              ? []
              : [
                  h(
                    PermissionButton,
                    {
                      permission: PERMISSIONS.ACCOUNT_WRITE,
                      onClick: () => disable(row),
                    },
                    () => '禁用',
                  ),
                  h(
                    PermissionButton,
                    {
                      permission: PERMISSIONS.ACCOUNT_WRITE,
                      type: 'error',
                      onClick: () => removeAdmin(row),
                    },
                    () => '删除',
                  ),
                ]
            : [
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
      ]),
    220,
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
  password: string
  displayName: string
  name: string
}>({ storeId: null, username: '', password: '', displayName: '', name: '' })

// 员工绑定态：按手机号查会员（仅 staff 新增用）
const searchPhone = ref('')
const searching = ref(false)
const lookupError = ref<string | null>(null)
const foundMembers = ref<Member[]>([])
const selectedMemberId = ref<string | null>(null)
const selectedMember = computed(
  () => foundMembers.value.find((member) => String(member.id) === selectedMemberId.value) ?? null,
)

function resetForm(): void {
  form.storeId = null
  form.username = ''
  form.password = ''
  form.displayName = ''
  form.name = ''
  searchPhone.value = ''
  foundMembers.value = []
  selectedMemberId.value = null
  lookupError.value = null
}

function openCreate(): void {
  editingId.value = null
  resetForm()
  drawerShow.value = true
}

async function lookupMember(): Promise<void> {
  const phone = searchPhone.value.trim()
  if (!phone) {
    lookupError.value = '请输入手机号'
    return
  }
  searching.value = true
  lookupError.value = null
  foundMembers.value = []
  selectedMemberId.value = null
  form.name = ''
  try {
    foundMembers.value = await http.get<Member[]>(API_PATHS.members.lookup, { phone })
    if (foundMembers.value.length === 0) {
      lookupError.value = '未找到匹配的已注册会员'
    }
  } catch (e) {
    const err = e as { status?: number; message?: string }
    lookupError.value = err.message ?? '查询失败，请重试'
  } finally {
    searching.value = false
  }
}

function clearLookupSelection(): void {
  foundMembers.value = []
  selectedMemberId.value = null
  lookupError.value = null
  form.name = ''
}

function selectMember(value: string): void {
  selectedMemberId.value = value
  form.name = selectedMember.value?.nickname?.trim() || '会员'
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
  if (isStaff && editingId.value && !form.name) return toastError('请填写员工姓名')
  if (isStaff && !editingId.value && !selectedMember.value)
    return toastError('请先按手机号查询并选择会员')
  if (!isStaff && !editingId.value && !form.username) return toastError('请填写账号')
  if (
    (props.variant === 'super' || props.variant === 'store-admin') &&
    !editingId.value &&
    !form.password
  )
    return toastError('请填写初始密码')
  if (!editingId.value && showStore && form.storeId == null) return toastError('请选择绑定门店')

  let payload: Record<string, unknown>
  if (isStaff) {
    payload = editingId.value
      ? { name: form.name }
      : {
          storeId: form.storeId,
          memberId: Number(selectedMember.value!.id),
          name: form.name.trim(),
        }
  } else if (props.variant === 'super') {
    payload = editingId.value
      ? {
          displayName: form.displayName,
          ...(form.password ? { password: form.password } : {}),
        }
      : {
          username: form.username,
          password: form.password,
          displayName: form.displayName,
        }
  } else {
    payload = editingId.value
      ? {
          displayName: form.displayName,
          ...(props.variant === 'store-admin' && form.password
            ? { password: form.password }
            : {}),
        }
      : {
          storeId: form.storeId,
          username: form.username,
          password: form.password,
          displayName: form.displayName,
        }
  }

  submitting.value = true
  try {
    if (editingId.value) {
      await props.service.update(editingId.value, payload as Partial<AccountEntity>)
    } else {
      await props.service.create(payload as Partial<AccountEntity>)
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

// 员工删除：仅移除 staff_accounts 绑定（撤销员工权限），不删除其小程序会员账号。
async function removeStaff(row: AccountEntity): Promise<void> {
  const label = row.name ?? row.id
  const ok = await runAudited({
    title: '删除员工',
    content: `确认删除员工「${label}」？仅移除其员工权限，不影响其小程序会员账号。该操作将写入审计日志。`,
    highRisk: true,
    positiveText: '确认删除',
    execute: () => props.service.remove(row.id),
    successText: '员工已删除',
  })
  if (ok) listRef.value?.reload()
}

async function removeAdmin(row: AccountEntity): Promise<void> {
  const label = row.username ?? row.id
  const ok = await runAudited({
    title: '删除总后台账号',
    content: `确认永久删除账号「${label}」？删除后该账号将立即无法登录。该操作将写入审计日志。`,
    highRisk: true,
    positiveText: '确认删除',
    execute: () => props.service.remove(row.id),
    successText: '账号已删除',
  })
  if (ok) listRef.value?.reload()
}

const toolbarActions = [
  {
    key: 'create',
    label: '新增账号',
    type: 'primary' as const,
    permission: PERMISSIONS.ACCOUNT_WRITE,
    onClick: openCreate,
  },
]
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
          <!-- 编辑：仅改显示名 -->
          <NFormItem
            v-if="editingId"
            label="姓名"
            required
          >
            <NInput
              v-model:value="form.name"
              placeholder="请输入员工姓名"
            />
          </NFormItem>
          <!-- 新增：员工须先在小程序注册，按手机号查到会员后绑定 -->
          <template v-else>
            <NFormItem
              label="员工手机号（须已在小程序注册）"
              required
            >
              <NInputGroup>
                <NInput
                  v-model:value="searchPhone"
                  placeholder="输入至少 3 位，支持尾号"
                  @update:value="clearLookupSelection"
                  @keyup.enter="lookupMember"
                />
                <NButton
                  type="primary"
                  :loading="searching"
                  @click="lookupMember"
                >
                  查询
                </NButton>
              </NInputGroup>
            </NFormItem>
            <p
              v-if="lookupError"
              class="form-err"
            >
              {{ lookupError }}
            </p>
            <NFormItem
              v-if="foundMembers.length"
              label="选择员工账号"
              required
              class="lookup-field"
            >
              <div class="lookup-content">
                <span class="lookup-meta">找到 {{ foundMembers.length }} 个匹配结果</span>
                <NRadioGroup
                  :value="selectedMemberId"
                  class="lookup-results"
                  @update:value="selectMember"
                >
                  <NRadio
                    v-for="member in foundMembers"
                    :key="String(member.id)"
                    :value="String(member.id)"
                    class="lookup-result"
                    :class="{
                      'lookup-result--selected': selectedMemberId === String(member.id),
                    }"
                  >
                    <span class="found__name">{{ member.nickname || '会员' }}</span>
                    <span class="found__phone">{{ member.phone || '未绑定手机号' }}</span>
                  </NRadio>
                </NRadioGroup>
              </div>
            </NFormItem>
            <NFormItem
              v-if="selectedMember"
              label="员工姓名（默认取会员昵称，可修改）"
            >
              <NInput
                v-model:value="form.name"
                placeholder="员工姓名"
              />
            </NFormItem>
          </template>
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
          <NFormItem
            v-if="variant === 'super' || variant === 'store-admin'"
            :label="editingId ? '新密码（留空则不修改）' : '初始密码'"
            :required="!editingId"
          >
            <NInput
              v-model:value="form.password"
              type="password"
              show-password-on="click"
              autocomplete="new-password"
              placeholder="请输入初始密码"
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
.staff-user {
  display: flex;
  align-items: center;
  gap: var(--ic-space-sm);
  min-width: 0;
}
.staff-user__meta {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.staff-user__nickname,
.staff-user__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.staff-user__nickname {
  font-weight: 600;
  color: var(--ic-color-text);
}
.staff-user__name {
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.form-note {
  margin-top: var(--ic-space-sm);
  font-size: var(--ic-font-xs);
  color: var(--ic-color-text-tertiary);
}
.form-err {
  margin: var(--ic-space-xs) 0 0;
  font-size: var(--ic-font-sm);
  color: var(--ic-color-danger, #d03050);
}
.lookup-field :deep(.n-form-item-blank) {
  width: 100%;
}
.lookup-content {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-sm);
  width: 100%;
}
.lookup-meta {
  font-size: var(--ic-font-xs);
  line-height: 1.5;
  color: var(--ic-color-text-tertiary);
}
.lookup-results {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-sm);
  width: 100%;
}
.lookup-result {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 52px;
  padding: 12px var(--ic-space-md);
  border: 1px solid var(--ic-color-border);
  border-radius: var(--ic-radius-lg);
  background: var(--ic-color-surface);
  cursor: pointer;
  transition:
    border-color 180ms cubic-bezier(0.22, 1, 0.36, 1),
    background-color 180ms cubic-bezier(0.22, 1, 0.36, 1);
}
.lookup-result:hover {
  border-color: var(--ic-color-border-strong);
  background: var(--ic-color-surface-muted);
}
.lookup-result:focus-within {
  outline: 2px solid var(--ic-color-primary);
  outline-offset: 2px;
}
.lookup-result--selected {
  border-color: var(--ic-color-primary);
  background: var(--ic-color-surface-muted);
}
.lookup-result :deep(.n-radio__label) {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: space-between;
  gap: var(--ic-space-sm);
  min-width: 0;
}
.found__name {
  overflow: hidden;
  font-weight: 600;
  color: var(--ic-color-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.found__phone {
  flex: 0 0 auto;
  color: var(--ic-color-text-tertiary);
}
@media (prefers-reduced-motion: reduce) {
  .lookup-result {
    transition: none;
  }
}
</style>
