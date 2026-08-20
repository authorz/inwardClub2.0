<script setup lang="ts">
/**
 * 管理员管理：超级管理员维护本店普通管理员账号。
 * 服务端仅接受用户名(username)+显示名(displayName)；初始密码由服务端生成，
 * 在新增/重置密码响应的 initialPassword 中一次性返回，展示后不可再查询。
 * 停用为不可逆写操作，均带 Idempotency-Key。
 */
import { computed, h, reactive, ref } from 'vue'
import { NButton, NInput, NModal, NSpace, type DataTableColumns } from 'naive-ui'
import { cashierService } from '@/api/services'
import { useAsyncList } from '@/composables/useAsyncList'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { ACTIVE_STATUS, toOptions } from '@/constants/enums'
import { PERM } from '@/constants/permissions'
import { statusColumn, textColumn } from '@/utils/columns'
import { DataTable, PageHeader, PermissionButton, StatusFilterBar } from '@/components/common'
import type { Cashier } from '@/types/models'

const list = useAsyncList<Cashier>((params) => cashierService.list(params), {
  initialFilters: { status: '', keyword: '' },
})
const action = useAsyncAction()

const editShow = ref(false)
const form = reactive<{ id: string | number | null; username: string; displayName: string }>({
  id: null,
  username: '',
  displayName: '',
})

// 一次性初始密码展示（新增 / 重置密码后）。
const credentialShow = ref(false)
const credential = reactive<{ title: string; username: string; password: string }>({
  title: '',
  username: '',
  password: '',
})

function showCredential(title: string, row: Cashier | null) {
  if (!row?.initialPassword) return
  credential.title = title
  credential.username = row.username ?? ''
  credential.password = row.initialPassword
  credentialShow.value = true
}

function openCreate() {
  form.id = null
  form.username = ''
  form.displayName = ''
  editShow.value = true
}

function openEdit(row: Cashier) {
  form.id = row.id
  form.username = row.username ?? ''
  form.displayName = row.displayName
  editShow.value = true
}

function save() {
  void action.run(
    () =>
      form.id == null
        ? cashierService.create({ username: form.username, displayName: form.displayName })
        : cashierService.update(form.id, { displayName: form.displayName }),
    {
      successMessage: '已保存',
      onSuccess: (res) => {
        const creating = form.id == null
        editShow.value = false
        list.refresh()
        if (creating) showCredential('新增管理员成功', res)
      },
    },
  )
}

function disable(row: Cashier) {
  void action.run(() => cashierService.disable(row.id), {
    confirm: { content: `确认停用管理员「${row.displayName}」？停用后该账号将无法登录`, danger: true },
    successMessage: '已停用',
    onSuccess: () => list.refresh(),
  })
}

function resetPassword(row: Cashier) {
  void action.run(() => cashierService.resetPassword(row.id), {
    confirm: { content: `确认重置管理员「${row.displayName}」的登录密码？将生成新的初始密码` },
    successMessage: '密码已重置',
    onSuccess: (res) => showCredential('密码已重置', res),
  })
}

const columns = computed<DataTableColumns<Cashier>>(() => [
  textColumn<Cashier>('姓名', (r) => r.displayName),
  textColumn<Cashier>('登录账号', (r) => r.username),
  statusColumn<Cashier>('状态', ACTIVE_STATUS, (r) => r.status, { width: 90 }),
  {
    title: '操作',
    key: 'actions',
    width: 200,
    fixed: 'right',
    render: (row: Cashier) =>
      h(NSpace, { size: 4 }, {
        default: () => [
          h(
            PermissionButton,
            { permissions: [PERM.staffWrite], type: 'primary', text: true, onClick: () => openEdit(row) },
            { default: () => '编辑' },
          ),
          h(
            PermissionButton,
            { permissions: [PERM.staffWrite], text: true, onClick: () => resetPassword(row) },
            { default: () => '重置密码' },
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
      title="管理员管理"
      description="超级管理员可维护本店普通管理员账号"
    >
      <template #actions>
        <PermissionButton
          :permissions="[PERM.staffWrite]"
          type="primary"
          @click="openCreate"
        >
          新增管理员
        </PermissionButton>
      </template>
    </PageHeader>

    <StatusFilterBar
      :status-options="toOptions(ACTIVE_STATUS)"
      :status="(list.filters.status as string) ?? null"
      :keyword="(list.filters.keyword as string) ?? ''"
      :loading="list.loading.value"
      search-placeholder="搜索姓名 / 登录账号"
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
      empty-text="暂无普通管理员"
      @update:page="list.setPage"
      @update:page-size="list.setPageSize"
    />

    <NModal
      v-model:show="editShow"
      preset="card"
      :title="form.id == null ? '新增管理员' : '编辑管理员'"
      style="width: 400px"
    >
      <div class="cashier-form">
        <label>
          <span class="ic-muted">登录账号</span>
          <NInput
            v-model:value="form.username"
            placeholder="登录账号"
            :disabled="form.id != null"
          />
        </label>
        <label>
          <span class="ic-muted">姓名</span>
          <NInput
            v-model:value="form.displayName"
            placeholder="管理员姓名"
          />
        </label>
        <p class="ic-muted cashier-form__hint">
          初始密码由系统生成，保存后仅展示一次，请及时交给管理员。
        </p>
      </div>
      <template #footer>
        <div class="cashier-form__footer">
          <NButton @click="editShow = false">
            取消
          </NButton>
          <NButton
            type="primary"
            :loading="action.running.value"
            :disabled="!form.displayName || (form.id == null && !form.username)"
            @click="save"
          >
            保存
          </NButton>
        </div>
      </template>
    </NModal>

    <NModal
      v-model:show="credentialShow"
      preset="card"
      :title="credential.title"
      style="width: 360px"
    >
      <div class="cashier-form">
        <div>
          <span class="ic-muted">登录账号：</span>{{ credential.username }}
        </div>
        <div>
          <span class="ic-muted">初始密码：</span><strong>{{ credential.password }}</strong>
        </div>
        <p class="ic-muted cashier-form__hint">
          该密码仅展示一次，请立即复制并交给管理员，关闭后无法再次查看。
        </p>
      </div>
      <template #footer>
        <div class="cashier-form__footer">
          <NButton
            type="primary"
            @click="credentialShow = false"
          >
            我已保存
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.cashier-form {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-4);
}
.cashier-form label {
  display: flex;
  flex-direction: column;
  gap: var(--ic-space-2);
  font-size: var(--ic-font-sm);
}
.cashier-form__hint {
  font-size: var(--ic-font-xs);
}
.cashier-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--ic-space-2);
}
</style>
