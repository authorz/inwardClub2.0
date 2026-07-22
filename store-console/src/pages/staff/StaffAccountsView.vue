<script setup lang="ts">
/**
 * 员工账号管理：本店 staff_accounts 增改与停用。
 * 服务端仅接受/返回姓名(name)+状态；登录凭证与角色不在本模型内。
 * 停用为不可逆写操作，带 Idempotency-Key。收银员账号在独立页面管理。
 */
import { computed, h, reactive, ref } from 'vue'
import { NButton, NInput, NSpace, NModal, type DataTableColumns } from 'naive-ui'
import { staffAccountService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { ACTIVE_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { dateColumn, statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { StaffAccount } from '@/types/models'

const list = useAsyncList<StaffAccount>((params) => staffAccountService.list(params), {
  initialFilters: { status: '', keyword: '' },
})
const action = useAsyncAction()

const editShow = ref(false)
const form = reactive<{ id: string | number | null; name: string }>({ id: null, name: '' })

function openCreate() {
  form.id = null
  form.name = ''
  editShow.value = true
}

function openEdit(row: StaffAccount) {
  form.id = row.id
  form.name = row.name
  editShow.value = true
}

function save() {
  void action.run(
    () =>
      form.id == null
        ? staffAccountService.create({ name: form.name })
        : staffAccountService.update(form.id, { name: form.name }),
    {
      successMessage: '已保存',
      onSuccess: () => {
        editShow.value = false
        list.refresh()
      },
    },
  )
}

function disable(row: StaffAccount) {
  void action.run(() => staffAccountService.disable(row.id), {
    confirm: { content: `确认停用员工账号「${row.name}」？停用后该账号将无法登录`, danger: true },
    successMessage: '已停用',
    onSuccess: () => list.refresh(),
  })
}

const columns = computed<DataTableColumns<StaffAccount>>(() => [
  textColumn<StaffAccount>('姓名', (r) => r.name),
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
            {
              permissions: [PERM.staffWrite],
              type: 'error',
              text: true,
              disabled: row.status === 'disabled',
              onClick: () => disable(row),
            },
            { default: () => '停用' },
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
      description="维护本店员工账号"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.staffWrite]"
          type="primary"
          @click="openCreate"
        >
          新增员工账号
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
      :title="form.id == null ? '新增员工账号' : '编辑员工账号'"
      style="width: 400px"
    >
      <div class="staff-form">
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
.staff-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
