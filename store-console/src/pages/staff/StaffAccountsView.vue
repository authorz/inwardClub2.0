<script setup lang="ts">
/**
 * 员工账号管理：本店 staff_accounts 绑定/改名/删除。
 * 员工须先在小程序注册；新增时按手机号查到会员后绑定为员工（不再手输姓名）。
 * 删除仅移除员工权限绑定，不删除其小程序会员账号。写操作带 Idempotency-Key。
 * 收银员账号在独立页面管理。
 */
import { computed, h, reactive, ref } from 'vue'
import {
  NAvatar,
  NButton,
  NCheckbox,
  NCheckboxGroup,
  NInput,
  NInputGroup,
  NModal,
  NSpace,
  type DataTableColumns,
} from 'naive-ui'
import { memberService, staffAccountService } from '@/api/services'
import { ApiError } from '@/api/error'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { ACTIVE_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import { feedback } from '@/utils/feedback'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { Member, StaffAccount } from '@/types/models'

const list = useAsyncList<StaffAccount>((params) => staffAccountService.list(params), {
  initialFilters: { status: '', keyword: '' },
})
const action = useAsyncAction()

const editShow = ref(false)
const mode = ref<'create' | 'edit'>('create')
const form = reactive<{ id: string | number | null; name: string }>({ id: null, name: '' })

// 新增（绑定）态：按手机号查会员
const searchPhone = ref('')
const searching = ref(false)
const lookupError = ref<string | null>(null)
const foundMembers = ref<Member[]>([])
const selectedMemberIds = ref<string[]>([])
const selectedMembers = computed(() => {
  const selectedIds = new Set(selectedMemberIds.value)
  return foundMembers.value.filter((member) => selectedIds.has(String(member.id)))
})

interface BatchBindResult {
  successIds: string[]
  failures: Array<{ id: string; name: string; message: string }>
}

function memberName(member: Member) {
  return member.nickname?.trim() || '会员'
}

function memberInitial(member: Member) {
  return memberName(member).slice(0, 1)
}

function bindMembers(members: Member[]): Promise<BatchBindResult> {
  return Promise.allSettled(
    members.map((member) =>
      staffAccountService.create({ memberId: member.id, name: memberName(member) }),
    ),
  ).then((outcomes) => {
    const result: BatchBindResult = { successIds: [], failures: [] }
    outcomes.forEach((outcome, index) => {
      const member = members[index]
      const id = String(member.id)
      if (outcome.status === 'fulfilled') {
        result.successIds.push(id)
        return
      }
      result.failures.push({
        id,
        name: memberName(member),
        message: outcome.reason instanceof ApiError ? outcome.reason.message : '绑定失败',
      })
    })
    return result
  })
}

function handleBatchBindResult(result: BatchBindResult) {
  void list.refresh()
  if (result.failures.length === 0) {
    feedback.message.success(`已绑定 ${result.successIds.length} 名员工`)
    editShow.value = false
    return
  }

  const successIds = new Set(result.successIds)
  foundMembers.value = foundMembers.value.filter(
    (member) => !successIds.has(String(member.id)),
  )
  selectedMemberIds.value = result.failures.map((failure) => failure.id)
  const failureDetails = result.failures
    .map((failure) => `${failure.name}（${failure.message}）`)
    .join('、')
  lookupError.value = result.successIds.length
    ? `已绑定 ${result.successIds.length} 名；以下员工绑定失败：${failureDetails}`
    : `绑定失败：${failureDetails}`
  feedback.message.warning(
    result.successIds.length
      ? `已绑定 ${result.successIds.length} 名，${result.failures.length} 名失败`
      : `${result.failures.length} 名员工绑定失败`,
  )
}

function openCreate() {
  mode.value = 'create'
  form.id = null
  form.name = ''
  searchPhone.value = ''
  foundMembers.value = []
  selectedMemberIds.value = []
  lookupError.value = null
  editShow.value = true
}

function openEdit(row: StaffAccount) {
  mode.value = 'edit'
  form.id = row.id
  form.name = row.name
  editShow.value = true
}

async function lookupMember() {
  const phone = searchPhone.value.trim()
  if (!phone) {
    lookupError.value = '请输入手机号'
    return
  }
  searching.value = true
  lookupError.value = null
  foundMembers.value = []
  selectedMemberIds.value = []
  try {
    foundMembers.value = await memberService.lookupByPhone(phone)
    if (foundMembers.value.length === 0) {
      lookupError.value = '未找到匹配的已注册会员'
    }
  } catch (err) {
    lookupError.value =
      err instanceof ApiError ? err.message : '查询失败，请重试'
  } finally {
    searching.value = false
  }
}

function clearLookupSelection() {
  foundMembers.value = []
  selectedMemberIds.value = []
  lookupError.value = null
}

function save() {
  if (mode.value === 'create') {
    const members = selectedMembers.value
    if (members.length === 0) {
      lookupError.value = '请先按手机号查询并选择至少一名会员'
      return
    }
    void action.run(() => bindMembers(members), {
      onSuccess: handleBatchBindResult,
    })
    return
  }
  if (form.id == null) return
  const id = form.id
  void action.run(() => staffAccountService.update(id, { name: form.name }), {
    successMessage: '已保存',
    onSuccess: () => {
      editShow.value = false
      list.refresh()
    },
  })
}

function remove(row: StaffAccount) {
  void action.run(() => staffAccountService.remove(row.id), {
    confirm: {
      content: `确认删除员工「${row.name}」？仅移除其员工权限，不影响其小程序会员账号`,
      danger: true,
    },
    successMessage: '已删除',
    onSuccess: () => list.refresh(),
  })
}

const columns = computed<DataTableColumns<StaffAccount>>(() => [
  textColumn<StaffAccount>('姓名', (r) => r.name),
  textColumn<StaffAccount>('手机号', (r) => r.phone || '—', { width: 150 }),
  statusColumn<StaffAccount>('状态', ACTIVE_STATUS, (r) => r.status, { width: 90 }),
  dateColumn<StaffAccount>('创建时间', (r) => r.createdAt, { width: 150 }),
  {
    title: '操作',
    key: 'actions',
    width: 160,
    fixed: 'right',
    render: (row: StaffAccount) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(
            PermissionButton,
            { permissions: [PERM.staffWrite], type: 'primary', text: true, onClick: () => openEdit(row) },
            { default: () => '编辑' },
          ),
          h(
            PermissionButton,
            { permissions: [PERM.staffWrite], type: 'error', text: true, onClick: () => remove(row) },
            { default: () => '删除' },
          ),
        ],
      }),
  },
])
</script>

<template>
  <div>
    <PageHeader
      title="员工账号管理"
      description="员工须先在小程序注册，按手机号搜索后绑定为本店员工"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.staffWrite]"
          type="primary"
          @click="openCreate"
        >
          新增员工
        </PermissionButton>
      </template>
    </PageHeader>

    <StatusFilterBar
      :status-options="toOptions(ACTIVE_STATUS)"
      :status="(list.filters.status as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索姓名"
      @update:status="list.filters.status = $event ?? ''"
      @update:keyword="list.filters.keyword = $event"
      @apply="list.applyFilters({})"
      @reset="list.reset()"
    />

    <DataTable
      :columns="columns"
      :data="list.rows.value"
      :loading="list.loading.value"
      :page="list.page.value"
      :page-size="list.pageSize.value"
      :total="list.total.value"
      empty-text="暂无员工账号"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="editShow"
      preset="card"
      :title="mode === 'create' ? '新增员工' : '编辑员工'"
      style="width: 460px"
    >
      <!-- 新增：按手机号查会员 → 绑定 -->
      <div
        v-if="mode === 'create'"
        class="staff-form"
      >
        <label>
          <span class="ic-muted">员工手机号（须已在小程序注册）</span>
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
        </label>

        <p
          v-if="lookupError"
          class="staff-form__err"
        >
          {{ lookupError }}
        </p>

        <NCheckboxGroup
          v-if="foundMembers.length"
          v-model:value="selectedMemberIds"
          class="staff-form__results"
        >
          <div
            v-for="member in foundMembers"
            :key="String(member.id)"
            class="staff-form__result"
          >
            <span class="staff-form__member">
              <NAvatar
                v-if="member.avatarUrl"
                :size="40"
                round
                :src="member.avatarUrl"
                object-fit="cover"
              >
                <template #fallback>
                  {{ memberInitial(member) }}
                </template>
              </NAvatar>
              <NAvatar
                v-else
                :size="40"
                round
              >
                {{ memberInitial(member) }}
              </NAvatar>
              <span class="staff-form__member-info">
                <span class="staff-form__found-name">{{ memberName(member) }}</span>
                <span class="ic-muted">{{ member.phone || '未绑定手机号' }}</span>
              </span>
            </span>
            <NCheckbox
              :value="String(member.id)"
              :aria-label="`选择员工${memberName(member)}`"
              class="staff-form__selection"
            />
          </div>
        </NCheckboxGroup>
      </div>

      <!-- 编辑：改显示名 -->
      <div
        v-else
        class="staff-form"
      >
        <label>
          <span class="ic-muted">姓名</span>
          <NInput
            v-model:value="form.name"
            placeholder="员工姓名"
          />
        </label>
      </div>

      <template #footer>
        <div class="staff-form__footer">
          <NButton @click="editShow = false">
            取消
          </NButton>
          <NButton
            v-if="mode === 'create'"
            type="primary"
            :loading="action.running.value"
            :disabled="selectedMembers.length === 0"
            @click="save"
          >
            {{ selectedMembers.length ? `绑定 ${selectedMembers.length} 名员工` : '绑定为员工' }}
          </NButton>
          <NButton
            v-else
            type="primary"
            :loading="action.running.value"
            :disabled="!form.name"
            @click="save"
          >
            保存
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.staff-form {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-4);
}
.staff-form label {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
.staff-form__err {
  margin: 0;
  color: var(--ic-color-error, #d03050);
  font-size: var(--ic-font-sm);
}
.staff-form__results {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
}
.staff-form__result {
  display: flex;
  align-items: center;
  gap: var(--ic-space-3);
  width: 100%;
  padding: var(--ic-space-3);
  border: 1px solid var(--ic-color-border, #efeff5);
  border-radius: 8px;
  background: var(--ic-color-fill-2, #fafafc);
}
.staff-form__member {
  display: flex;
  flex: 1;
  align-items: center;
  min-width: 0;
  gap: var(--ic-space-3);
}
.staff-form__selection {
  flex: 0 0 auto;
  margin-left: auto;
}
.staff-form__member-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: var(--ic-space-1);
}
.staff-form__found-name {
  font-weight: 600;
}
.staff-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
