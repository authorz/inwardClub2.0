<script setup lang="ts">
/**
 * 员工账号管理：本店 staff_accounts 绑定/改名/删除。
 * 员工须先在小程序注册；新增时按手机号查到会员后绑定为员工（不再手输姓名）。
 * 删除仅移除员工权限绑定，不删除其小程序会员账号。写操作带 Idempotency-Key。
 * 收银员账号在独立页面管理。
 */
import { computed, h, reactive, ref } from 'vue'
import {
  NButton,
  NInput,
  NInputGroup,
  NModal,
  NRadio,
  NRadioGroup,
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
const selectedMemberId = ref<string | null>(null)
const selectedMember = computed(
  () => foundMembers.value.find((member) => String(member.id) === selectedMemberId.value) ?? null,
)

function openCreate() {
  mode.value = 'create'
  form.id = null
  form.name = ''
  searchPhone.value = ''
  foundMembers.value = []
  selectedMemberId.value = null
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
  selectedMemberId.value = null
  form.name = ''
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
  selectedMemberId.value = null
  lookupError.value = null
  form.name = ''
}

function selectMember(value: string) {
  selectedMemberId.value = value
  const member = selectedMember.value
  form.name = member?.nickname?.trim() || '会员'
}

function save() {
  if (mode.value === 'create') {
    if (!selectedMember.value) {
      lookupError.value = '请先按手机号查询并选择会员'
      return
    }
    const memberId = selectedMember.value.id
    const name = form.name.trim()
    void action.run(() => staffAccountService.create({ memberId, name }), {
      successMessage: '已绑定为员工',
      onSuccess: () => {
        editShow.value = false
        list.refresh()
      },
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

        <NRadioGroup
          v-if="foundMembers.length"
          :value="selectedMemberId"
          class="staff-form__results"
          @update:value="selectMember"
        >
          <NRadio
            v-for="member in foundMembers"
            :key="String(member.id)"
            :value="String(member.id)"
            class="staff-form__result"
          >
            <span class="staff-form__found-name">{{ member.nickname || '会员' }}</span>
            <span class="ic-muted">{{ member.phone || '未绑定手机号' }}</span>
          </NRadio>
        </NRadioGroup>

        <label v-if="selectedMember">
          <span class="ic-muted">员工姓名（默认取会员昵称，可修改）</span>
          <NInput
            v-model:value="form.name"
            placeholder="员工姓名"
          />
        </label>
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
            :disabled="!selectedMember"
            @click="save"
          >
            绑定为员工
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
  width: 100%;
  padding: var(--ic-space-3);
  border: 1px solid var(--ic-color-border, #efeff5);
  border-radius: 8px;
  background: var(--ic-color-fill-2, #fafafc);
}
.staff-form__result :deep(.n-radio__label) {
  display: flex;
  flex: 1;
  justify-content: space-between;
  gap: var(--ic-space-3);
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
